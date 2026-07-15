import http, { type ApiResp } from "./index";
import type { ProxyRequest, ProxyResponse } from "@/types/protocol";

// 经后端代理发送 HTTP/HTTP2 请求，获取协议版本与计时。
export function proxySend(req: ProxyRequest) {
  return http
    .post<ApiResp<ProxyResponse>>("/proxy", req)
    .then((r) => r.data.data);
}
