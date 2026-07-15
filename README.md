# Apiforge

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)
[![CI](https://github.com/your-org/apiforge/actions/workflows/ci.yml/badge.svg)](.github/workflows/ci.yml)

A web-based API client. The backend is Go (GORM / JWT) and serves the built
frontend; the frontend is Vue 3 + Vite. It supports several protocols,
per-project member permissions, saved request collections, dark/light themes
and a Chinese/English UI.

## Features

- Protocol clients: HTTP / HTTP2, WebSocket, MQTT, GraphQL, Socket.IO, gRPC,
  and TCP / UDP relay.
- Collaboration: system `admin`, project `owner` and `developer` roles; project
  members get fine-grained `add / edit / delete` permissions.
- Request collections: nestable folders and saved requests, organized per project.
- Themes and language: dark / light and Chinese / English, persisted to
  `localStorage`.
- Databases: SQLite by default, switchable to PostgreSQL / MySQL.

Planned: MCP and AI endpoint debugging.

## Tech stack

| Layer    | Choice                                            |
| -------- | ------------------------------------------------- |
| Backend  | Go · GORM · JWT · static file hosting            |
| Frontend | Vue 3 · Vite · Pinia · Tailwind CSS · vue-i18n    |
| Database | SQLite (default) · PostgreSQL · MySQL            |

## Quick start

### Backend

```bash
cd backend
go run ./cmd/server
```

Listens on `:8080` by default. On first launch an admin is seeded with
`admin / admin123`. In production it serves the built frontend from
`frontend/dist`.

### Frontend (development)

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173, proxies /api to :8080
```

### Production build

```bash
cd frontend
npm run build      # outputs to ../backend/frontend/dist
cd ../backend
go run ./cmd/server # visit http://localhost:8080
```

## Configuration

`backend/config.yaml` controls the port, JWT expiry, proxy response size limit
and CORS allow-list. Sensitive values can be overridden by environment
variables:

| Env var               | Description                                  |
| --------------------- | -------------------------------------------- |
| `APIFORGE_JWT_SECRET`| JWT signing secret (change in production)    |
| `DB_DRIVER`           | `sqlite` (default) / `pg` / `mysql`          |
| `DB_DSN`              | Database connection string                   |
| `SERVER_PORT`         | Listen port                                  |

Example (switch to PostgreSQL):

```bash
DB_DRIVER=pg DB_DSN="host=localhost user=app dbname=apiforge sslmode=disable" ./apiforge
```

## Permission model

- `admin` — full system access.
- Project `owner` — full control within the project (the creator becomes owner
  automatically).
- `maintainer` — can manage collections and requests, but cannot delete the
  project or manage members.
- `developer` — operates according to the `add / edit / delete` permissions
  granted by the owner.

## Roadmap

Apiforge advances in phases by protocol maturity and collaboration capability.
The table below lists each phase's goals and current status.

| Phase | Goals | Status |
| --- | --- | --- |
| Phase 1 · Core request protocols | HTTP / HTTP2, WebSocket, MQTT | Done |
| Phase 2 · Realtime & messaging protocols | Socket.IO, TCP / UDP relay | Done |
| Phase 3 · Structured & RPC protocols | GraphQL, gRPC | Done |
| Phase 4 · Collaboration & engineering | Member roles & permissions, request collections, themes & i18n, multi-database | Done |
| Phase 5 · Planned | MCP debugging client, AI / LLM endpoint debugging | Planned |

### Phase 1 · Core request protocols (Done)

Covers the most common request-response and long-lived connection scenarios:

- **HTTP / HTTP2**: request building, params, headers, body, environments and
  auth.
- **WebSocket**: connection management, message exchange and frame inspection.
- **MQTT**: publish / subscribe, QoS and topic management.

### Phase 2 · Realtime & messaging protocols (Done)

- **Socket.IO**: event exchange, namespaces and rooms.
- **TCP / UDP relay**: binary send/receive and encoding over `/ws/relay`.

### Phase 3 · Structured & RPC protocols (Done)

- **GraphQL**: queries, variables and Schema exploration.
- **gRPC**: proto loading, unary and streaming calls.

### Phase 4 · Collaboration & engineering (Done)

- **Member roles & permissions**: system `admin`, project `owner` /
  `maintainer` / `developer` with fine-grained `add / edit / delete`
  permissions.
- **Request collections**: nestable folders and saved requests, organized per
  project.
- **Themes & i18n**: dark / light, Chinese / English, persisted to
  `localStorage`.
- **Multi-database**: SQLite by default, switchable to PostgreSQL / MySQL.

### Phase 5 · Planned

- **MCP debugging**: connect to an MCP Server, invoke tools and resources and
  inspect results.
- **AI / LLM endpoint debugging**: request building, streaming responses and
  debugging for large language models.

### Future plans

The following capabilities are planned but not yet implemented:

- **Import / export format expansion**: beyond the current Postman v2.1, add
  OpenAPI / Swagger and Bruno (.bru) import and export to ease migration from
  other tools.
- **GraphQL introspection and docs**: Schema introspection, field
  autocompletion and a documentation browser.
- **More auth methods**: add Digest, AWS Signature V4, OAuth 1.0a and NTLM;
  OAuth 2.0 gains the authorization-code grant with PKCE.
- **WebSocket UX**: auto-reconnect and message history import / export.
- **form-data file upload**: reliable multipart binary file upload.
- **Environment scope expansion**: a global environment scope and
  collection-level auth inheritance.
- **Response comparison**: response diff and "save as example".

## License

Apiforge is licensed under the [GNU Affero General Public License v3.0](LICENSE).
If you plan to offer Apiforge as a network service, AGPL requires you to make your
modified source available to users. See [CONTRIBUTING.md](CONTRIBUTING.md) to get
started.
