import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import router from "./router";
import { i18n } from "./locales";
import { useThemeStore } from "./stores/theme";
import "./index.css";

// 组装应用：Pinia 状态 + vue-router 路由 + vue-i18n 多语言
const app = createApp(App);
app.use(createPinia());
app.use(router);
app.use(i18n);

// 挂载前同步主题 class，避免首屏闪烁（持久化 + 系统偏好已在 store 内处理）
useThemeStore().init();

app.mount("#app");
