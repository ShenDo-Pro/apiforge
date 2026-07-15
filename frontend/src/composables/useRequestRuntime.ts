import { ref } from "vue";
import { useEnvironmentStore } from "@/stores/environment";
import { useCollectionStore } from "@/stores/collection";
import { resolveTemplate, parseEnvVars } from "@/lib/vars";
import {
  createPm,
  runUserScript,
  type PmResponseLike,
  type RequestLike,
  type ScriptWrites,
  type Assertion,
} from "./useScriptRunner";
import type { EnvScope, EnvVar } from "@/types/project";

// 请求运行时：将「变量替换 → 预请求脚本 → 发送 → 响应提取/测试脚本」串成闭环。
// 所有协议客户端共用：发送前调用 prepare() 得到解析后且可被脚本改写的请求，
// 收到响应后调用 finalize() 应用 GUI 提取规则与测试脚本并写回变量。

export interface RuntimeRequest {
  url: string;
  method: string;
  headers: Record<string, string>;
  body: string;
}

export interface ExtractRuleResolved {
  source: "body" | "header";
  expr: string; // JSONPath（$.token）或 header 名
  targetVar: string;
  scope: EnvScope;
  enabled: boolean;
}

export function useRequestRuntime(
  projectId: number,
  getScripts: () => { pre: string; test: string; rules: ExtractRuleResolved[] }
) {
  const envStore = useEnvironmentStore();
  const collectionStore = useCollectionStore();

  const scriptLogs = ref<string[]>([]);
  const scriptError = ref<string | undefined>();
  const assertions = ref<Assertion[]>([]);
  const extracted = ref<{ varName: string; value: string }[]>([]);

  // 准备：合并变量替换 + 运行预请求脚本（可改写请求、写变量）
  async function prepare(req: RuntimeRequest): Promise<RuntimeRequest> {
    const vars = envStore.mergedVars;
    const resolved: RuntimeRequest = {
      url: resolveTemplate(req.url, vars),
      method: req.method,
      headers: Object.fromEntries(
        Object.entries(req.headers).map(([k, v]) => [
          resolveTemplate(k, vars),
          resolveTemplate(v, vars),
        ])
      ),
      body: resolveTemplate(req.body, vars),
    };
    const pre = getScripts().pre;
    if (pre && pre.trim()) {
      const writes: ScriptWrites = { environment: {}, globals: {}, collection: {}, local: {} };
      const { pm } = createPm({
        request: resolved,
        writes,
        envName: envStore.activeEnv?.name ?? "",
      });
      const res = await runUserScript(pre, pm);
      scriptLogs.value = res.logs;
      scriptError.value = res.error;
      await applyWrites(writes);
      // 脚本可能改写了请求
      resolved.url = pm.request.url;
      resolved.method = pm.request.method;
      resolved.headers = pm.request.headers;
      resolved.body = pm.request.body;
    }
    return resolved;
  }

  // 收尾：应用 GUI 提取规则 + 测试脚本，写回变量
  async function finalize(resp: PmResponseLike) {
    const { test, rules } = getScripts();
    for (const r of rules) {
      if (!r.enabled || !r.targetVar) continue;
      let val: unknown;
      if (r.source === "header") val = resp.headers[r.expr];
      else val = jsonPath(resp.json?.() ?? {}, r.expr);
      if (val !== undefined && val !== null && val !== "") {
        setVarByScope(r.scope, r.targetVar, String(val));
        extracted.value.push({ varName: r.targetVar, value: String(val) });
      }
    }
    if (test && test.trim()) {
      const writes: ScriptWrites = { environment: {}, globals: {}, collection: {}, local: {} };
      const { pm } = createPm({
        request: { url: "", method: "", headers: {}, body: "" },
        response: resp,
        writes,
        envName: envStore.activeEnv?.name ?? "",
      });
      const res = await runUserScript(test, pm);
      scriptLogs.value = [...scriptLogs.value, ...res.logs];
      if (res.error) scriptError.value = res.error;
      assertions.value = res.assertions;
      await applyWrites(writes);
    }
  }

  function setVarByScope(scope: EnvScope, key: string, value: string) {
    if (scope === "local") {
      envStore.setLocalVar(key, value);
    } else if (scope === "environment") {
      const env = envStore.activeEnv;
      if (env) {
        const vars = parseEnvVars(env.values);
        const i = vars.findIndex((x) => x.key === key);
        if (i >= 0) vars[i].value = value;
        else vars.push({ key, value, enabled: true, secret: false });
        env.values = JSON.stringify(vars);
        envStore.persistEnvironment(env.id, env.name, vars);
      }
    } else if (scope === "global") {
      const g = envStore.globalEnv;
      if (g) {
        const vars = parseEnvVars(g.values);
        const i = vars.findIndex((x) => x.key === key);
        if (i >= 0) vars[i].value = value;
        else vars.push({ key, value, enabled: true, secret: false });
        g.values = JSON.stringify(vars);
        envStore.persistEnvironment(g.id, g.name, vars);
      }
    } else if (scope === "collection" && envStore.activeCollectionId != null) {
      const c = collectionStore.collections.find((x) => x.id === envStore.activeCollectionId);
      if (c) {
        const vars: EnvVar[] = c.variables ? parseEnvVars(c.variables) : [];
        const i = vars.findIndex((x) => x.key === key);
        if (i >= 0) vars[i].value = value;
        else vars.push({ key, value, enabled: true, secret: false });
        collectionStore.saveCollectionVariables(projectId, c.id, vars);
      }
    }
  }

  // 将脚本写回缓冲应用到各作用域并持久化
  async function applyWrites(w: ScriptWrites) {
    const env = envStore.activeEnv;
    if (env && Object.keys(w.environment).length) {
      const vars = parseEnvVars(env.values);
      for (const [k, v] of Object.entries(w.environment)) {
        const i = vars.findIndex((x) => x.key === k);
        if (i >= 0) vars[i].value = v;
        else vars.push({ key: k, value: v, enabled: true, secret: false });
      }
      env.values = JSON.stringify(vars);
      await envStore.persistEnvironment(env.id, env.name, vars);
    }
    const g = envStore.globalEnv;
    if (g && Object.keys(w.globals).length) {
      const vars = parseEnvVars(g.values);
      for (const [k, v] of Object.entries(w.globals)) {
        const i = vars.findIndex((x) => x.key === k);
        if (i >= 0) vars[i].value = v;
        else vars.push({ key: k, value: v, enabled: true, secret: false });
      }
      g.values = JSON.stringify(vars);
      await envStore.persistEnvironment(g.id, g.name, vars);
    }
    if (envStore.activeCollectionId != null && Object.keys(w.collection).length) {
      const c = collectionStore.collections.find((x) => x.id === envStore.activeCollectionId);
      if (c) {
        const vars: EnvVar[] = c.variables ? parseEnvVars(c.variables) : [];
        for (const [k, v] of Object.entries(w.collection)) {
          const i = vars.findIndex((x) => x.key === k);
          if (i >= 0) vars[i].value = v;
          else vars.push({ key: k, value: v, enabled: true, secret: false });
        }
        await collectionStore.saveCollectionVariables(projectId, c.id, vars);
      }
    }
    for (const [k, v] of Object.entries(w.local)) envStore.setLocalVar(k, v);
  }

  // 清空本轮运行时结果（切换请求时调用）
  function resetRuntime() {
    scriptLogs.value = [];
    scriptError.value = undefined;
    assertions.value = [];
    extracted.value = [];
  }

  return { prepare, finalize, scriptLogs, scriptError, assertions, extracted, resetRuntime };
}

// 极简 JSONPath：支持 $.a.b、$.a[0].b、$ 根。
function jsonPath(obj: any, path: string): unknown {
  if (!path) return undefined;
  const p = path.replace(/^\$\.?/, "");
  if (!p) return obj;
  const segs = p.split(/[\.\[\]]/).filter((s) => s !== "");
  let cur: any = obj;
  for (const seg of segs) {
    if (cur == null) return undefined;
    const key = seg.replace(/["']/g, "");
    cur = /^\d+$/.test(key) ? cur[parseInt(key, 10)] : cur[key];
  }
  return cur;
}
