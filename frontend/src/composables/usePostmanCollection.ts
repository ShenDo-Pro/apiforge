// Postman Collection v2.1 与 Apiforge 树形结构的双向映射。
// 导入/导出逻辑全部在前端完成，不新增后端接口：导出即序列化 JSON 下载，
// 导入即把解析出的 folders/requests 顺序复用现有 createCollection / saveRequest 建树。
import type { Collection, SavedRequest } from "@/types/project";
import type { AuthConfig } from "@/composables/useAuthHeader";
import type { SaveRequestPayload } from "@/composables/useRequestSaver";

// 导入计划：按 DFS 先序排列的操作列表。
// folder 操作创建集合后把 tempId 映射成真实 id；request 操作挂到 parentTempId 对应集合下。
// parentTempId 为 null 表示挂在根集合下（由调用方创建根集合后填充 idMap）。
export interface ImportOp {
  tempId: string;
  parentTempId: string | null;
  kind: "folder" | "request";
  name: string;
  payload?: SaveRequestPayload;
}
export interface PostmanPlan {
  name: string;
  ops: ImportOp[];
  unsupportedAuth: boolean; // 是否存在 Apiforge 不支持的鉴权类型（已降级为 none）
}

const POSTMAN_SCHEMA = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json";

function genId(): string {
  try {
    return crypto.randomUUID();
  } catch {
    return "xxxxxxxx-xxxx-4xxx".replace(/x/g, () => ((Math.random() * 16) | 0).toString(16));
  }
}

// ---- 鉴权转换 ----

function authVal(list: any[] | undefined, key: string): string {
  const item = (list || []).find((x) => x && x.key === key);
  return item ? String(item.value ?? "") : "";
}

// Postman 鉴权 → Apiforge AuthConfig。digest/ntlm/awsv4 等缺失类型降级为 none 并标记。
function postmanAuthToApiforge(auth: any): { config: AuthConfig; unsupported: boolean } {
  if (!auth || !auth.type || auth.type === "noauth") {
    return { config: { type: "none" }, unsupported: false };
  }
  switch (auth.type) {
    case "bearer":
      return { config: { type: "bearer", token: authVal(auth.bearer, "token") }, unsupported: false };
    case "basic":
      return {
        config: {
          type: "basic",
          username: authVal(auth.basic, "username"),
          password: authVal(auth.basic, "password"),
        },
        unsupported: false,
      };
    case "apikey":
      return {
        config: {
          type: "apikey",
          keyName: authVal(auth.apikey, "key"),
          keyValue: authVal(auth.apikey, "value"),
          keyIn: authVal(auth.apikey, "in") === "query" ? "query" : "header",
        },
        unsupported: false,
      };
    case "oauth2": {
      const o = auth.oauth2 || {};
      return {
        config: {
          type: "oauth2",
          grantType: o.grantType === "password" ? "password" : "client_credentials",
          tokenUrl: o.accessTokenUrl || o.tokenUrl || "",
          clientId: o.clientId || "",
          clientSecret: o.clientSecret || "",
          scope: o.scope || "",
          username2: o.username || "",
          password2: o.password || "",
          varName: "oauth_access_token",
        },
        unsupported: false,
      };
    }
    default:
      // digest / edge / ntlm / awsv4 等在 Apiforge 暂无对应实现
      return { config: { type: "none" }, unsupported: true };
  }
}

// Apiforge AuthConfig → Postman 鉴权结构。
function apiforgeAuthToPostman(authRaw: string): any {
  let cfg: AuthConfig = { type: "none" };
  if (authRaw) {
    try {
      cfg = JSON.parse(authRaw);
    } catch {
      /* 损坏值按 none 处理 */
    }
  }
  switch (cfg.type) {
    case "none":
      return { type: "noauth" };
    case "bearer":
      return { type: "bearer", bearer: [{ key: "token", value: cfg.token || "", type: "string" }] };
    case "basic":
      return {
        type: "basic",
        basic: [
          { key: "username", value: cfg.username || "", type: "string" },
          { key: "password", value: cfg.password || "", type: "string" },
        ],
      };
    case "apikey":
      return {
        type: "apikey",
        apikey: [
          { key: "key", value: cfg.keyName || "", type: "string" },
          { key: "value", value: cfg.keyValue || "", type: "string" },
          { key: "in", value: cfg.keyIn || "header", type: "string" },
        ],
      };
    case "oauth2":
      return {
        type: "oauth2",
        oauth2: {
          grantType: cfg.grantType || "client_credentials",
          accessTokenUrl: cfg.tokenUrl || "",
          clientId: cfg.clientId || "",
          clientSecret: cfg.clientSecret || "",
          scope: cfg.scope || "",
          username: cfg.username2 || "",
          password: cfg.password2 || "",
        },
      };
    default:
      return { type: "noauth" };
  }
}

// ---- 请求头转换 ----

function apiforgeHeadersToPostman(headersRaw: string): any[] {
  let obj: Record<string, string> = {};
  if (headersRaw) {
    try {
      obj = JSON.parse(headersRaw);
    } catch {
      return [];
    }
  }
  return Object.entries(obj).map(([key, value]) => ({ key, value }));
}

function postmanHeadersToApiforge(header: any): string {
  const obj: Record<string, string> = {};
  for (const h of header || []) {
    if (!h.disabled && h.key) obj[h.key] = h.value ?? "";
  }
  return JSON.stringify(obj);
}

// ---- Body 转换 ----

// Apiforge 的 body 为 {_v:1, mode, rawLang, raw, form, urlencoded} 的 JSON 字符串。
function mapRawLang(l: string): string {
  return ["json", "text", "xml", "javascript", "html"].includes(l) ? l : "json";
}

function apiforgeBodyToPostman(bodyRaw: string): any | null {
  if (!bodyRaw) return null;
  let o: any = null;
  try {
    o = JSON.parse(bodyRaw);
  } catch {
    return { mode: "raw", raw: bodyRaw };
  }
  if (!o || o._v !== 1) return { mode: "raw", raw: bodyRaw };
  const mode = o.mode || "raw";
  if (mode === "raw") {
    return { mode: "raw", raw: o.raw || "", options: { raw: { language: mapRawLang(o.rawLang) } } };
  }
  if (mode === "urlencoded") {
    return {
      mode: "urlencoded",
      urlencoded: (o.urlencoded || []).map((u: any) => ({
        key: u.key || "",
        value: u.value || "",
        disabled: !u.enabled,
      })),
    };
  }
  if (mode === "formdata") {
    return {
      mode: "formdata",
      formdata: (o.form || []).map((f: any) => ({
        key: f.key || "",
        value: f.value || "",
        type: f.type || "text",
        fileName: f.fileName || "",
      })),
    };
  }
  return null;
}

function postmanBodyToApiforge(body: any): string {
  if (!body || !body.mode) return JSON.stringify({ _v: 1, mode: "none" });
  if (body.mode === "raw") {
    return JSON.stringify({
      _v: 1,
      mode: "raw",
      rawLang: mapRawLang(body.options?.raw?.language),
      raw: body.raw || "",
    });
  }
  if (body.mode === "urlencoded") {
    return JSON.stringify({
      _v: 1,
      mode: "urlencoded",
      urlencoded: (body.urlencoded || []).map((u: any) => ({
        key: u.key || "",
        value: u.value || "",
        enabled: !u.disabled,
      })),
    });
  }
  if (body.mode === "formdata") {
    return JSON.stringify({
      _v: 1,
      mode: "formdata",
      form: (body.formdata || []).map((f: any) => ({
        key: f.key || "",
        value: f.value || "",
        type: f.type === "file" ? "file" : "text",
        fileName: f.fileName || "",
        enabled: !f.disabled,
      })),
    });
  }
  return JSON.stringify({ _v: 1, mode: "none" });
}

function postmanUrlToString(url: any): string {
  if (!url) return "";
  if (typeof url === "string") return url;
  return url.raw || "";
}

// ---- 导出（Apiforge → Postman v2.1）----

function requestToPostmanItem(r: SavedRequest): any {
  const item: any = {
    name: r.name || "Untitled",
    request: {
      method: r.method || "GET",
      header: apiforgeHeadersToPostman(r.headers),
      url: r.url,
    },
  };
  const pmBody = apiforgeBodyToPostman(r.body);
  if (pmBody) item.request.body = pmBody;
  const pmAuth = apiforgeAuthToPostman(r.auth);
  if (pmAuth) item.request.auth = pmAuth;

  const events: any[] = [];
  if (r.preRequestScript) {
    events.push({ listen: "prerequest", script: { type: "text/javascript", exec: r.preRequestScript.split("\n") } });
  }
  if (r.testScript) {
    events.push({ listen: "test", script: { type: "text/javascript", exec: r.testScript.split("\n") } });
  }
  if (events.length) item.event = events;
  return item;
}

// 把单个集合节点递归展开为 Postman 的 folder 对象（含 name + item[]）。
function toPostmanItem(node: Collection, all: Collection[], requestsByCollection: Record<number, SavedRequest[]>): any {
  const childFolders = all
    .filter((c) => c.parentId === node.id)
    .sort((a, b) => a.sortOrder - b.sortOrder);
  const reqs = requestsByCollection[node.id] || [];
  const items: any[] = [];
  for (const f of childFolders) items.push(toPostmanItem(f, all, requestsByCollection));
  for (const r of reqs) items.push(requestToPostmanItem(r));
  return { name: node.name, item: items };
}

// 导出整个集合（含子文件夹与请求）为 Postman v2.1 集合 JSON 对象。
export function exportCollection(
  node: Collection,
  all: Collection[],
  requestsByCollection: Record<number, SavedRequest[]>,
): object {
  const inner = toPostmanItem(node, all, requestsByCollection);
  return {
    info: {
      name: node.name,
      schema: POSTMAN_SCHEMA,
      _postman_id: genId(),
    },
    item: inner.item,
  };
}

// ---- 导入（Postman v2.1 → Apiforge）----

function postmanRequestToPayload(request: any): { payload: SaveRequestPayload; unsupportedAuth: boolean } {
  const method = request.method || "GET";
  const url = postmanUrlToString(request.url);
  const headers = postmanHeadersToApiforge(request.header);
  const body = postmanBodyToApiforge(request.body);
  const authRes = postmanAuthToApiforge(request.auth);
  const auth = JSON.stringify(authRes.config);
  return {
    payload: { protocol: "http", name: "", method, url, headers, body, auth },
    unsupportedAuth: authRes.unsupported,
  };
}

// 解析 Postman 集合 JSON 为可顺序执行的导入计划。
export function fromPostman(json: any): PostmanPlan | null {
  if (!json || typeof json !== "object") return null;
  const name = json.info?.name || json.name || "Imported Collection";
  const items = json.item;
  if (!Array.isArray(items)) return null;

  const ops: ImportOp[] = [];
  let seq = 0;
  let unsupportedAuth = false;

  function walk(list: any[], parentTempId: string | null) {
    for (const it of list) {
      if (!it || typeof it !== "object") continue;
      const tempId = `n${seq++}`;
      if (Array.isArray(it.item)) {
        // 文件夹（含子项）
        ops.push({ tempId, parentTempId, kind: "folder", name: it.name || "Folder" });
        walk(it.item, tempId);
      } else if (it.request) {
        // 请求
        const { payload, unsupportedAuth: ua } = postmanRequestToPayload(it.request);
        if (ua) unsupportedAuth = true;
        ops.push({
          tempId,
          parentTempId,
          kind: "request",
          name: it.name || "Untitled",
          payload: { ...payload, name: it.name || "Untitled" },
        });
      }
    }
  }

  walk(items, null);
  return { name, ops, unsupportedAuth };
}
