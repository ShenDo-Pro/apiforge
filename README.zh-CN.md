# Apiforge

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
| `APIFORGE_JWT_SECRET`| JWT 签名密钥（生产环境务必修改）     |
| `DB_DRIVER`           | `sqlite`（默认）/ `pg` / `mysql`     |
| `DB_DSN`              | 数据库连接串                          |
| `SERVER_PORT`         | 监听端口                             |

示例（切换 PostgreSQL）：

```bash
DB_DRIVER=pg DB_DSN="host=localhost user=app dbname=apiforge sslmode=disable" ./apiforge
```

## 权限模型

- `admin` — 系统全部权限。
- 项目 `owner` — 项目内完全控制（创建者自动成为 owner）。
- `maintainer` — 可管理合集与请求，但不能删除项目或管理成员。
- `developer` — 按 owner 授予的 `新增 / 修改 / 删除` 权限操作。

## 实现路线图

Apiforge 按协议成熟度与协作能力分阶段推进。下表列出各阶段目标与当前状态。

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

## 许可证

Apiforge 采用 [GNU Affero 通用公共许可证 v3.0](LICENSE) 开源。如果你将 Apiforge 作为
网络服务提供，AGPL 要求你向用户开放修改后的源代码。参见 [CONTRIBUTING.md](CONTRIBUTING.md)
开始贡献。
