import axios from "axios";
import { refreshToken } from "./auth";

// 统一响应结构，与后端 response.APIResponse 对齐
export interface ApiResp<T> {
  code: number;
  message: string;
  data: T;
}

// 前端所有请求走 /api 前缀，开发期由 Vite 反代到后端，生产期同源。
const http = axios.create({
  baseURL: "/api",
  headers: { "Content-Type": "application/json" },
});

// 请求拦截：注入 Bearer token
http.interceptors.request.use((config) => {
  const token = localStorage.getItem("access_token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// 并发刷新去重：多个请求同时 401 时只发一次 refresh（M30）
let refreshPromise: Promise<string> | null = null;

// 响应拦截：401 尝试用 refresh token 续期，失败则跳登录
http.interceptors.response.use(
  (res) => res,
  async (error) => {
    const status = error.response?.status;
    if (status === 401 && !error.config._retry) {
      error.config._retry = true;
      const refresh = localStorage.getItem("refresh_token");
      if (refresh) {
        try {
          if (!refreshPromise) {
            // 复用 api/auth 的 refreshToken（M33），避免两处实现分叉
            refreshPromise = refreshToken(refresh);
          }
          const next = await refreshPromise;
          localStorage.setItem("access_token", next);
          error.config.headers.Authorization = `Bearer ${next}`;
          return http(error.config);
        } catch (e) {
          // 续期失败，清除登录态并跳转
          refreshPromise = null;
          localStorage.removeItem("access_token");
          localStorage.removeItem("refresh_token");
          window.location.href = "/login";
          return Promise.reject(error);
        } finally {
          refreshPromise = null;
        }
      }
    }
    return Promise.reject(error);
  }
);

export default http;

// 当前用户在项目中的成员身份，供路由守卫校验（L18）。
export interface MyMembership {
  id: number;
  userId: number;
  username: string;
  role: string;
  permissions: string;
}

export async function getMyMembership(projectId: number): Promise<MyMembership> {
  const { data } = await http.get<ApiResp<MyMembership>>(
    `/project/${projectId}/members/me`,
  );
  return data.data;
}
