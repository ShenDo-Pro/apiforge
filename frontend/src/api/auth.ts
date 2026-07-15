import http, { type ApiResp } from "./index";
import type { AuthResult } from "@/types/user";

export function login(username: string, password: string) {
  return http
    .post<ApiResp<AuthResult>>("/auth/login", { username, password })
    .then((r) => r.data.data);
}

export function refreshToken(refresh_token: string) {
  return http
    .post<ApiResp<{ access_token: string }>>("/auth/refresh", { refresh_token })
    .then((r) => r.data.data.access_token);
}
