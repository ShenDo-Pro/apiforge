# ApiToolX

> 📖 English: [README.md](README.md)

一个基于 Web 的接口调试工具。后端用 Go（GORM / JWT）编写并直接托管前端静态文件，
前端用 Vue 3 + Vite。支持多种协议、项目成员权限、保存请求的合集管理，
以及暗色 / 亮色主题和中英文界面。

## 功能

- 协议客户端：HTTP / HTTP2、WebSocket、MQTT、GraphQL、Socket.IO、gRPC，
  以及 TCP / UDP 透传。
- 协作：系统 `admin`、项目 `owner`、`developer` 三种角色；项目成员可分别授予
  `新增 / 修改 / 删除` 权限。
- 请求合集：可嵌套的文件夹与保存的请求，按项目组织。
- 主题与语言：暗色 / 亮色，中文 / 英文，设置持久化到 `localStorage`。
- 数据库：默认 SQLite，可切换 PostgreSQL / MySQL。

计划后续完成：MCP 与 AI 接口调试。

## 截图

![ApiToolX — 多协议接口调试工作台](docs/screenshot.png)

## 技术栈

| 层     | 选型                                     |
| ------ | ---------------------------------------- |
| 后端   | Go · GORM · JWT · 静态托管               |
| 前端   | Vue 3 · Vite · Pinia · Tailwind CSS · vue-i18n |
| 数据库 | SQLite（默认）· PostgreSQL · MySQL       |

## 快速开始

### 后端

```bash
cd backend
go run ./cmd/server
```

默认监听 `:8080`。首次启动会创建管理员账号 `admin / admin123`。
生产模式下直接托管 `frontend/dist` 下的前端。

### 前端（开发）

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173，/api 代理到 :8080
```

### 生产构建

```bash
cd frontend
npm run build      # 输出到 ../backend/frontend/dist
cd ../backend
go run ./cmd/server # 访问 http://localhost:8080
```

## 配置

`backend/config.yaml` 可配置端口、JWT 有效期、代理响应大小上限、CORS 白名单。
敏感项可用环境变量覆盖：

| 环境变量              | 说明                                 |
| --------------------- | ------------------------------------ |
| `APITOOLX_JWT_SECRET`| JWT 签名密钥（生产环境务必修改）     |
| `DB_DRIVER`           | `sqlite`（默认）/ `pg` / `mysql`     |
| `DB_DSN`              | 数据库连接串                          |
| `SERVER_PORT`         | 监听端口                             |

示例（切换 PostgreSQL）：

```bash
DB_DRIVER=pg DB_DSN="host=localhost user=app dbname=apitoolx sslmode=disable" ./apitoolx
```

## 权限模型

- `admin` — 系统全部权限。
- 项目 `owner` — 项目内完全控制（创建者自动成为 owner）。
- `maintainer` — 可管理合集与请求，但不能删除项目或管理成员。
- `developer` — 按 owner 授予的 `新增 / 修改 / 删除` 权限操作。

## 实现路线图

ApiToolX 按协议成熟度与协作能力分阶段推进。下表列出各阶段目标与当前状态。

| 阶段 | 目标 | 状态 |
| --- | --- | --- |
| 阶段 1 · 核心请求协议 | HTTP / HTTP2、WebSocket、MQTT | 已完成 |
| 阶段 2 · 实时与消息型协议 | Socket.IO、TCP / UDP 透传 | 已完成 |
| 阶段 3 · 结构化与 RPC 协议 | GraphQL、gRPC | 已完成 |
| 阶段 4 · 协作与工程化 | 成员与角色权限、请求合集、主题与多语言、多数据库 | 已完成 |
| 阶段 5 · 规划中 | MCP 协议调试、AI / LLM 接口调试 | 规划中 |

### 阶段 1 · 核心请求协议（已完成）

覆盖最常见的请求-响应与长连接调试场景：

- **HTTP / HTTP2**：请求构造、参数、请求头、请求体、环境变量与鉴权。
- **WebSocket**：连接管理、消息收发与帧查看。
- **MQTT**：发布 / 订阅、QoS 与主题管理。

### 阶段 2 · 实时与消息型协议（已完成）

- **Socket.IO**：事件收发、命名空间与房间。
- **TCP / UDP 透传**：基于 `/ws/relay` 的二进制收发与编码。

### 阶段 3 · 结构化与 RPC 协议（已完成）

- **GraphQL**：查询、变量与 Schema 探索。
- **gRPC**：proto 加载、一元与流式调用。

### 阶段 4 · 协作与工程化（已完成）

- **成员与角色权限**：系统 `admin`，项目 `owner` / `maintainer` / `developer` 的细粒度 `新增 / 修改 / 删除` 权限。
- **请求合集**：可嵌套的文件夹与保存的请求，按项目组织。
- **主题与多语言**：暗色 / 亮色，中文 / 英文，持久化到 `localStorage`。
- **多数据库**：默认 SQLite，可切换 PostgreSQL / MySQL。

### 阶段 5 · 规划中

- **MCP 协议调试**：连接 MCP Server，调用工具与资源并查看结果。
- **AI / LLM 接口调试**：面向大模型的请求构造、流式响应与调试。

### 未来计划

以下能力已列入规划，尚未实现：

- **导入 / 导出格式扩展**：在现有 Postman v2.1 基础上，补充 OpenAPI / Swagger 与 Bruno（.bru）的导入与导出，降低从其它工具迁移的成本。
- **GraphQL 内省与文档浏览器**：支持 Schema introspection、字段自动补全与文档浏览。
- **鉴权方式补全**：新增 Digest、AWS Signature V4、OAuth 1.0a、NTLM；OAuth 2.0 增加授权码模式与 PKCE。
- **WebSocket 体验增强**：自动重连、消息历史导入 / 导出。
- **form-data 文件上传**：支持可靠的多部分二进制文件上传。
- **环境作用域扩展**：新增全局环境作用域，并支持集合级鉴权继承。
- **响应对比**：响应 diff 与「存为示例」能力。

### 部署说明

- **非根路径部署**：前端为 SPA，若部署在子路径（如 `/apitoolx/`），需在前端 `vite.config.ts`
  设置 `base: "/apitoolx/"` 并重新 `npm run build`，同时后端 `SpaHandler` 的静态目录与回退路由需匹配该前缀。
- **生产环境强制 HTTPS**：建议在反向代理（Nginx / Caddy / LB）层终止 TLS，并将
  `config.yaml` 的 `proxy.require_https: true`（或环境变量 `APITOOLX_REQUIRE_HTTPS=true`）。
  开启后中继（`/ws/relay`）握手被强制要求走 TLS，避免 WebSocket/Socket.IO 的 token 经 query
  在非加密信道泄露（L1）。
- **管理员口令**：首次启动用 `APITOOLX_ADMIN_USERNAME` / `APITOOLX_ADMIN_PASSWORD` 覆盖默认
  管理员（默认用户名 `admin`）；未设置 `APITOOLX_ADMIN_PASSWORD` 时会生成随机强口令并在日志告警
  （不再硬编码弱口令）。无论口令来源，默认管理员均被标记「首次登录必须改密」，登录后会强制跳转
  到改密界面（H6）。强烈建议部署时通过环境变量指定口令，并在首次登录后立即修改。

## 安全审计后续项（已知技术债务）

下表来自 `bug.md` 复核（详见 `bug_rep.md`）。状态图例：**✅ 已解决** / **🔵 仍建议专项或保留**。

> 全量 70 项已分两批处理：前一批（H1–H4/H7、M1/M3/M4/M6/M7/M9/M10/M11/M12/M14/M16/M17/M18/M19 及前端 M26–M31/M33/L9/L10/L16 等）已修复；
> 本表仅列仍待处理项。

### 安全加固

| 来源 | 项 | 状态 | 说明 |
| --- | --- | --- | --- |
| H8 | 前端脚本无彻底沙箱 | 🔵 | 已缓解：遮蔽 `window`/`localStorage`/`Function`/`eval` 等危险全局 + 严格模式；彻底 Worker/iframe 隔离留专项 |

### 代码重构 / 可维护性

| 来源 | 项 | 状态 | 说明 |
| --- | --- | --- | --- |
| H9 | 8 个协议视图复制粘贴 | 🔵 | 仍建议独立 PR：抽「协议注册表 + `useSavedRequest`」统一（已抽 `useRequestSaver` 共用保存逻辑） |
| L6 | monolithic `main.go` | 🔵 | 仍建议独立 PR：路由 DI 拆分（当前运行正常） |
| L8 | 前端 `any` 滥用 | 🔵 | 类型美化项，非缺陷，留后续 |

### 工程化 / 部署文档

| 来源 | 项 | 状态 | 说明 |
| --- | --- | --- | --- |
| M15 | 列表接口无分页 | ✅(部分) | 项目列表已分页（后端 `page`/`perPage` + 前端 pager）；集合/请求列表分页留后续 |

## 许可证

ApiToolX 采用 [GNU Affero 通用公共许可证 v3.0](LICENSE) 开源。如果你将 ApiToolX 作为
网络服务提供，AGPL 要求你向用户开放修改后的源代码。参见 [CONTRIBUTING.md](CONTRIBUTING.md)
开始贡献。
