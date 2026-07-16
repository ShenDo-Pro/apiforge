# ---- Frontend build ----
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
# vite 将产物输出到 ../backend/frontend/dist（即 /app/backend/frontend/dist）
RUN mkdir -p /app/backend/frontend && npm run build

# ---- Backend build ----
FROM golang:1.26-alpine AS backend
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -o /apiforge ./cmd/server

# ---- Runtime ----
FROM alpine:3.20
WORKDIR /app/backend
RUN apk add --no-cache ca-certificates
COPY --from=backend /apiforge /app/backend/apiforge
COPY --from=frontend /app/backend/frontend/dist /app/backend/frontend/dist
EXPOSE 8080
CMD ["/app/backend/apiforge"]
