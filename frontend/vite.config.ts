import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import path from "path";

// 开发期前端跑在 5173，后端跑在 8080，这里把 /api 与 /ws 反代到后端，避免跨域。
// 生产期 build 产物直接输出到 backend/frontend/dist，由 Go 静态托管同源部署。
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    allowedHosts: true,
    proxy: {
      "/api": "http://localhost:8080",
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
      },
    },
  },
  build: {
    outDir: "../backend/frontend/dist",
    emptyOutDir: true,
  },
});
