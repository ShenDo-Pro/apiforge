import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { getMyMembership } from "@/api";
import { useToast } from "@/composables/useToast";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", name: "login", component: () => import("@/views/LoginView.vue") },
    { path: "/projects", name: "projects", component: () => import("@/views/ProjectListView.vue"), meta: { requiresAuth: true } },
    { path: "/project/:projectId", name: "project", component: () => import("@/views/ProjectView.vue"), meta: { requiresAuth: true } },
    { path: "/account", name: "account", component: () => import("@/views/AccountView.vue"), meta: { requiresAuth: true } },
    { path: "/", redirect: "/projects" },
    { path: "/:pathMatch(.*)*", redirect: "/projects" },
  ],
});

// 路由守卫：未登录跳登录；已登录访问登录页跳项目列表；
// 进入项目页时校验当前用户是否为成员（admin 放行），非成员跳回项目列表（L18）。
router.beforeEach(async (to) => {
  const auth = useAuthStore();
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: "login", query: { redirect: to.fullPath } };
  }
  if (to.name === "login" && auth.isAuthenticated) {
    return { name: "projects" };
  }
  if (to.name === "project") {
    const pid = Number(to.params.projectId);
    // 系统管理员对所有项目有权限，无需查成员表
    if (auth.user?.role !== "admin") {
      try {
        await getMyMembership(pid);
      } catch {
        useToast().error("您不是该项目的成员");
        return { name: "projects" };
      }
    }
  }
});

export default router;
