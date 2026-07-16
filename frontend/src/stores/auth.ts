import { defineStore } from "pinia";
import { login as apiLogin, logout as apiLogout } from "@/api/auth";
import { useEnvironmentStore } from "@/stores/environment";
import { useCollectionStore } from "@/stores/collection";
import { useProjectStore } from "@/stores/project";
import type { AuthResult, User } from "@/types/user";

// 认证状态：token 持久化到 localStorage，刷新页面后自动恢复登录态。
export const useAuthStore = defineStore("auth", {
  state: () => ({
    user: (() => {
      try {
        const raw = localStorage.getItem("auth_user");
        return raw ? (JSON.parse(raw) as User) : null;
      } catch {
        return null;
      }
    })(),
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
      if (r.user) localStorage.setItem("auth_user", JSON.stringify(r.user));
    },
    async login(username: string, password: string) {
      this.setSession(await apiLogin(username, password));
    },
    async logout() {
      // 先通知后端注销 refresh token，使其立即失效（M2）
      if (this.refreshToken) {
        await apiLogout(this.refreshToken);
      }
      this.accessToken = "";
      this.refreshToken = "";
      this.user = null;
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      localStorage.removeItem("auth_user");
      // 重置其它 store，避免上一个用户的数据残留（M31）
      useEnvironmentStore().$reset();
      useCollectionStore().$reset();
      useProjectStore().$reset();
    },
  },
});
