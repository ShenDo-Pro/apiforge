import { useEnvironmentStore } from "@/stores/environment";
import { proxySend } from "@/api/proxy";
import type { EnvVar } from "@/types/project";

// 预请求/测试脚本沙箱：构造 Postman 风格的 pm 对象，使用浏览器原生
// `new Function` 在本地执行用户脚本（无服务端 RCE 风险）。脚本写回的变量
// 经 writes 缓冲，由 useRequestRuntime 在脚本结束后统一落库。

export interface ScriptScope {
  get(key: string): string | undefined;
  set(key: string, value: string): void;
  has(key: string): boolean;
}

export interface PmResponseLike {
  code: number;
  status: string;
  responseTime: number;
  headers: Record<string, string>;
  json(): any;
  text(): string;
}

export interface RequestLike {
  url: string;
  method: string;
  headers: Record<string, string>;
  body: string;
}

export interface ScriptWrites {
  environment: Record<string, string>;
  globals: Record<string, string>;
  collection: Record<string, string>;
  local: Record<string, string>;
}

export interface Assertion {
  name: string;
  passed: boolean;
  message?: string;
}

export interface ScriptResult {
  logs: string[];
  error?: string;
  assertions: Assertion[];
}

// 轻量断言链：支持 equal/include/be.ok/be.a/have.property。
function buildExpect(target: unknown) {
  const fail = (msg: string): never => {
    throw new Error(msg);
  };
  const chain: any = {
    to: {},
  };
  const eq = (x: unknown) => {
    const a = target as any;
    const b = x as any;
    const ok = a === b || JSON.stringify(a) === JSON.stringify(b);
    if (!ok) fail(`期望 ${JSON.stringify(a)} 等于 ${JSON.stringify(b)}`);
  };
  const inc = (x: unknown) => {
    const a = target as any;
    if (typeof a === "string" || Array.isArray(a)) {
      if (!a.includes(x as any)) fail(`期望 ${JSON.stringify(a)} 包含 ${JSON.stringify(x)}`);
    } else if (typeof a === "object" && a !== null) {
      if (!(x as any).every((k: string) => k in a)) fail(`期望对象包含键 ${JSON.stringify(x)}`);
    } else fail("include 不支持的类型");
  };
  const prop = (name: string) => {
    if (typeof target !== "object" || target === null || !(name in (target as any)))
      fail(`期望对象包含属性 ${name}`);
  };
  const isType = (t: string) => {
    const actual = Array.isArray(target) ? "array" : typeof target;
    if (actual !== t) fail(`期望类型为 ${t}，实际为 ${actual}`);
  };
  chain.to.equal = eq;
  chain.to.not = { equal: (x: unknown) => {
    const a = target as any;
    const b = x as any;
    if (a === b || JSON.stringify(a) === JSON.stringify(b)) fail(`期望不等于 ${JSON.stringify(b)}`);
  } };
  chain.to.include = inc;
  chain.to.have = { property: prop };
  chain.to.be = {
    ok: () => {
      if (!target) fail(`期望为真值，实际为 ${JSON.stringify(target)}`);
    },
    a: isType,
    an: isType,
  };
  return chain;
}

// 构造一个变量作用域对象：get 从 store 读取，set 写入缓冲（脚本结束统一落库）。
function makeScope(
  read: () => Record<string, string>,
  buffer: Record<string, string>,
  live: (k: string, v: string) => void
): ScriptScope {
  return {
    get: (k) => read()[k],
    has: (k) => Object.prototype.hasOwnProperty.call(read(), k),
    set: (k, v) => {
      buffer[k] = v;
      live(k, v);
    },
  };
}

export function createPm(opts: {
  request: RequestLike;
  response?: PmResponseLike;
  writes: ScriptWrites;
  envName: string;
}): { pm: any; assertions: Assertion[] } {
  const envStore = useEnvironmentStore();
  const assertions: Assertion[] = [];

  const liveLocal = (k: string, v: string) => envStore.setLocalVar(k, v);

  const pm: any = {
    environment: Object.assign(
      makeScope(
        () => envStore.environmentVars,
        opts.writes.environment,
        (k, v) => {
          // 实时同步到活动环境，便于脚本内后续 get 读到
          const env = envStore.activeEnv;
          if (env) {
            const vars = JSON.parse(env.values || "[]") as EnvVar[];
            const i = vars.findIndex((x) => x.key === k);
            if (i >= 0) vars[i].value = v;
            else vars.push({ key: k, value: v, enabled: true, secret: false });
            env.values = JSON.stringify(vars);
          }
        }
      ),
      { name: opts.envName }
    ),
    globals: makeScope(
      () => envStore.globalVars,
      opts.writes.globals,
      (k, v) => {
        const g = envStore.globalEnv;
        if (g) {
          const vars = JSON.parse(g.values || "[]") as EnvVar[];
          const i = vars.findIndex((x) => x.key === k);
          if (i >= 0) vars[i].value = v;
          else vars.push({ key: k, value: v, enabled: true, secret: false });
          g.values = JSON.stringify(vars);
        }
      }
    ),
    collectionVariables: makeScope(
      () => envStore.collectionVars,
      opts.writes.collection,
      () => {
        /* 集合变量落库在 runtime 内统一处理 */
      }
    ),
    variables: makeScope(() => envStore.localVars, opts.writes.local, liveLocal),
    request: opts.request,
    response: opts.response,
    test(name: string, fn: () => void) {
      try {
        fn();
        assertions.push({ name, passed: true });
      } catch (e) {
        assertions.push({ name, passed: false, message: (e as Error).message });
      }
    },
    expect(target: unknown) {
      return buildExpect(target);
    },
    // 异步请求原语：脚本内可 `await pm.sendRequest(...)` 进行动态登录取 token 等。
    // 支持 (url, cb) 与 ({url,method,headers,body}, cb) 两种签名，始终返回 Promise。
    sendRequest(
      urlOrOpts: string | { url: string; method?: string; headers?: Record<string, string>; body?: string },
      callback?: (err: any, res: any) => void
    ): Promise<any> {
      const o =
        typeof urlOrOpts === "string"
          ? { url: urlOrOpts, method: "GET", headers: {}, body: "" }
          : urlOrOpts;
      return proxySend({
        method: o.method || "GET",
        url: o.url,
        headers: o.headers || {},
        body: o.body || "",
        forceHttp2: false,
      })
        .then((r) => {
          const res = {
            code: r.status,
            status: String(r.status),
            responseTime: r.timings?.total ?? 0,
            headers: r.headers,
            json: () => {
              try {
                return JSON.parse(r.body);
              } catch {
                return {};
              }
            },
            text: () => r.body,
          };
          if (callback) callback(null, res);
          return res;
        })
        .catch((e) => {
          if (callback) callback(e, null);
          throw e;
        });
    },
  };
  pm.__assertions = assertions;
  return { pm, assertions };
}

// 执行用户脚本，捕获日志与异常。返回结果与断言列表。
// 包成 async IIFE，使脚本内可使用 top-level await（如 `await pm.sendRequest(...)`）。
export async function runUserScript(
  code: string,
  pm: any
): Promise<{ logs: string[]; error?: string; assertions: Assertion[] }> {
  const logs: string[] = [];
  const consoleProxy = {
    log: (...a: unknown[]) => logs.push(a.map(String).join(" ")),
    error: (...a: unknown[]) => logs.push("[error] " + a.map(String).join(" ")),
    warn: (...a: unknown[]) => logs.push("[warn] " + a.map(String).join(" ")),
  };
  try {
    // 用户在自有浏览器运行脚本，捕获异常不阻断主流程；await 等待脚本内异步完成
    const fn = new Function("pm", "console", `return (async () => {\n${code}\n})();`);
    await fn(pm, consoleProxy);
    return { logs, assertions: (pm.__assertions as Assertion[]) || [] };
  } catch (e) {
    return { logs, error: (e as Error).message, assertions: [] };
  }
}
