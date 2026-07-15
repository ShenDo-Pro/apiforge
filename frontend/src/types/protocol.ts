// 与后端 proxy.ProxyRequest / ProxyResponse 对齐
export interface ProxyRequest {
  method: string;
  url: string;
  headers: Record<string, string>;
  body: string;
  forceHttp2: boolean;
  timeoutMs?: number;
  followRedirect?: boolean;
  sslVerify?: boolean;
}

export interface Timing {
  dns: number;
  tls: number;
  connect: number;
  ttfb: number;
  total: number;
}

export interface ProxyResponse {
  proto: string;
  status: number;
  headers: Record<string, string>;
  body: string;
  cookies?: RespCookie[];
  timings: Timing;
  error?: string;
}

export interface RespCookie {
  name: string;
  value: string;
  domain?: string;
  path?: string;
  expires?: string;
  httpOnly?: boolean;
  secure?: boolean;
  sameSite?: string;
}
