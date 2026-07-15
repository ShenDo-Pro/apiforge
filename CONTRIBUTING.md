# 贡献指南 (Contributing)

感谢你考虑为 **Apiforge** 做贡献！这是一个开源的 API 客户端，对标 Postman，支持
HTTP/HTTP2、WebSocket、MQTT、GraphQL、gRPC、TCP/UDP 等多协议一体化调试，并计划提供
AI 原生调试、轻量自托管协作与本地优先（Git 友好）的桌面客户端。

## 开发环境

- 后端：Go 1.26（`backend/`）
- 前端：Node 20 + Vue 3 + Vite（`frontend/`）

```bash
# 后端
cd backend && go run ./cmd/server        # 监听 :8080，首次启动 seed admin/admin123

# 前端（开发）
cd frontend && npm install && npm run dev # http://localhost:5173，代理 /api 到 :8080
```

## 提交规范

- 分支：`feat/`、`fix/`、`docs/`、`refactor/`、`chore/` 前缀。
- 提交信息使用祈使句，例如 `feat(proxy): 支持 Cookie Jar`。
- 保持 PR 聚焦单一改动，描述「为什么」而非「做了什么」。

## 代码约定

- 后端遵循 `handler / service / model` 分层，不引入新的 Web 框架（当前为零依赖自研路由）。
- 前端遵循 `composables / stores / views` 分层；脚本在浏览器沙箱中执行（`useScriptRunner`），
  不要在服务端执行用户脚本。
- 新增协议中继请复用 `internal/relay` 的 `Register` + `protocol.go` 机制。

## 测试

- 后端：`cd backend && go test ./...`
- 前端：`cd frontend && npm run test`（如已配置）

## 行为准则

参与本项目的所有人须遵守 [Code of Conduct](CODE_OF_CONDUCT.md)。
