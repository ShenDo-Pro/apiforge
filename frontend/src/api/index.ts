import axios from "axios";

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
          const { data } = await axios.post<ApiResp<{ access_token: string }>>(
            "/api/auth/refresh",
            { refresh_token: refresh }
          );
          const next = data.data.access_token;
          localStorage.setItem("access_token", next);
          error.config.headers.Authorization = `Bearer ${next}`;
          return http(error.config);
        } catch {
          // 续期失败，清除登录态并跳转
          localStorage.removeItem("access_token");
          localStorage.removeItem("refresh_token");
          window.location.href = "/login";
          return Promise.reject(error);
        }
      }
    }
    return Promise.reject(error);
  }
);

export default http;
