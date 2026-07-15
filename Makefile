.PHONY: build run dev test lint docker clean

# 构建前后端：前端产物输出到 backend/frontend/dist，后端二进制输出到 bin/apiforge
build:
	cd backend && go build -o ../bin/apiforge ./cmd/server
	cd frontend && npm install && npm run build

# 仅运行后端（需先 build 前端，或开发时另开 npm run dev）
run:
	cd backend && go run ./cmd/server

# 仅启动前端开发服务器（热更新，代理 /api 到 :8080）
dev:
	cd frontend && npm run dev

# 运行测试
test:
	cd backend && go test ./...
	cd frontend && npm run test --if-present

# 静态检查
lint:
	cd backend && go vet ./...
	cd frontend && npm run lint --if-present

# 构建并打 Docker 镜像
docker:
	docker build -t apiforge:latest .

clean:
	rm -rf bin backend/frontend/dist
