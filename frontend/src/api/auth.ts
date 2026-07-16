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

// resetPassword 首次登录强制改密或主动修改密码（H6）。
export function resetPassword(old_password: string, new_password: string) {
  return http
    .post<ApiResp<null>>("/auth/reset-password", { old_password, new_password })
    .then(() => {});
}

// logout 注销 refresh token，使其服务端立即失效（M2）。
export function logout(refresh_token: string) {
  return http
    .post<ApiResp<null>>("/auth/logout", { refresh_token })
    .then(() => {})
    .catch(() => {});
}
