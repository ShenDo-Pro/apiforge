import { defineStore } from "pinia";
import { login as apiLogin } from "@/api/auth";
import type { AuthResult, User } from "@/types/user";

// 认证状态：token 持久化到 localStorage，刷新页面后自动恢复登录态。
export const useAuthStore = defineStore("auth", {
  state: () => ({
    user: null as User | null,
    accessToken: localStorage.getItem("access_token") || "",
    refreshToken: localStorage.getItem("refresh_token") || "",
  }),
  getters: {
    isAuthenticated: (s) => !!s.accessToken,
  },
  actions: {
    setSession(r: AuthResult) {
      this.accessToken = r.access_token;
      this.refreshToken = r.refresh_token;
      this.user = r.user;
      localStorage.setItem("access_token", r.access_token);
      localStorage.setItem("refresh_token", r.refresh_token);
    },
    async login(username: string, password: string) {
      this.setSession(await apiLogin(username, password));
    },
    logout() {
      this.accessToken = "";
      this.refreshToken = "";
      this.user = null;
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
    },
  },
});
