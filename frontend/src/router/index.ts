import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", name: "login", component: () => import("@/views/LoginView.vue") },
    { path: "/projects", name: "projects", component: () => import("@/views/ProjectListView.vue"), meta: { requiresAuth: true } },
    { path: "/project/:projectId", name: "project", component: () => import("@/views/ProjectView.vue"), meta: { requiresAuth: true } },
    { path: "/", redirect: "/projects" },
    { path: "/:pathMatch(.*)*", redirect: "/projects" },
  ],
});

// 路由守卫：未登录跳登录；已登录访问登录页跳项目列表
router.beforeEach((to) => {
  const auth = useAuthStore();
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: "login", query: { redirect: to.fullPath } };
  }
  if (to.name === "login" && auth.isAuthenticated) {
    return { name: "projects" };
  }
});

export default router;
