import { proxySend } from "@/api/proxy";
import { useEnvironmentStore } from "@/stores/environment";
import { resolveTemplate } from "@/lib/vars";

// 鉴权配置：覆盖主流 API 工具的常用鉴权方式。
// 配置以 JSON 持久化在 SavedRequest.auth 字段，发送前解析为请求头/查询参数。
export type AuthType = "none" | "bearer" | "basic" | "apikey" | "oauth2";

export interface AuthConfig {
  type: AuthType;
  // bearer：可直接填 token，或留空改用 oauth2 写入的变量
  token?: string;
  // basic
  username?: string;
  password?: string;
  // apikey
  keyName?: string;
  keyValue?: string;
  keyIn?: "header" | "query";
  // oauth2（client_credentials / password 走 tokenUrl POST）
  grantType?: "client_credentials" | "password";
  tokenUrl?: string;
  clientId?: string;
  clientSecret?: string;
  scope?: string;
  username2?: string; // password 模式的资源拥有者账号
  password2?: string;
  varName?: string; // 取到的 token 写入的本地变量名，默认 oauth_access_token
}

export interface ResolvedAuth {
  headers: Record<string, string>;
  query: Record<string, string>;
}

// 将任意 Unicode 字符串安全编码为 Base64（btoa 仅支持 Latin1，中文/emoji 会抛错）（L10）
function utf8ToBase64(s: string): string {
  const bytes = new TextEncoder().encode(s);
  let bin = "";
  bytes.forEach((b) => (bin += String.fromCharCode(b)));
  return btoa(bin);
}

// OAuth2 token 内存缓存：避免每次请求都打 token 端点（含过期判断）。
const tokenCache = new Map<string, { token: string; expiresAt: number }>();

// 解析鉴权配置为实际请求头 / 查询参数。OAuth2 为异步（需向 tokenUrl 申请）。
export async function resolveAuth(auth?: AuthConfig | null): Promise<ResolvedAuth> {
  const empty: ResolvedAuth = { headers: {}, query: {} };
  if (!auth || auth.type === "none") return empty;

  const envStore = useEnvironmentStore();
  const vars = envStore.mergedVars;
  const r = (s?: string) => (s ? resolveTemplate(s, vars) : "");

  switch (auth.type) {
    case "bearer":
      return { headers: { Authorization: `Bearer ${r(auth.token)}` }, query: {} };
    case "basic": {
      const up = `${r(auth.username)}:${r(auth.password)}`;
      return { headers: { Authorization: `Basic ${utf8ToBase64(up)}` }, query: {} };
    }
    case "apikey": {
      const name = r(auth.keyName);
      const value = r(auth.keyValue);
      if (auth.keyIn === "query") return { headers: {}, query: { [name]: value } };
      return { headers: { [name]: value }, query: {} };
    }
    case "oauth2":
      return await resolveOAuth2(auth);
    default:
      return empty;
  }
}

async function resolveOAuth2(auth: AuthConfig): Promise<ResolvedAuth> {
  const varName = auth.varName || "oauth_access_token";
  const cached = tokenCache.get(varName);
  let token = cached && cached.expiresAt > Date.now() ? cached.token : "";

  if (!token) {
    const envStore = useEnvironmentStore();
    // 先尝试直接读已写入的变量（跨会话/手动设置）
    token = envStore.mergedVars[varName] || "";
    if (!token && auth.tokenUrl) {
      const params = new URLSearchParams();
      params.set("grant_type", auth.grantType || "client_credentials");
      params.set("client_id", auth.clientId || "");
      params.set("client_secret", auth.clientSecret || "");
      if (auth.scope) params.set("scope", auth.scope);
      if (auth.grantType === "password") {
        params.set("username", auth.username2 || "");
        params.set("password", auth.password2 || "");
      }
      const res = await proxySend({
        method: "POST",
        url: auth.tokenUrl,
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: params.toString(),
        forceHttp2: false,
      });
      let json: any = {};
      try {
        json = JSON.parse(res.body);
      } catch {
        /* ignore */
      }
      token = json.access_token || json.token || "";
      if (token && json.expires_in) {
        // 提前 30s 失效，避免边界过期
        tokenCache.set(varName, {
          token,
          expiresAt: Date.now() + Number(json.expires_in) * 1000 - 30000,
        });
      }
      // 写入本地变量，供脚本/后续请求复用
      if (token) envStore.setLocalVar(varName, token);
    }
  }

  if (!token) return { headers: {}, query: {} };
  return { headers: { Authorization: `Bearer ${token}` }, query: {} };
}

export function defaultAuth(): AuthConfig {
  return { type: "none", keyIn: "header", grantType: "client_credentials", varName: "oauth_access_token" };
}
