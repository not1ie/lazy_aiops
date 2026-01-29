.PHONY: build run dev docker-build docker-run clean test

# 变量
APP_NAME := lazy-auto-ops
VERSION := 1.0.0

# 本地开发
dev:
	go run ./cmd/server

# 构建本地二进制
build:
	CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/$(APP_NAME) ./cmd/server

# 构建 Linux amd64 (用于部署到 x86 服务器)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(APP_NAME)-linux-amd64 ./cmd/server

# Docker 构建 (多架构)
docker-build:
	docker build -t $(APP_NAME):$(VERSION) .

# Docker 构建 x86 镜像 (在 ARM Mac 上构建 x86 镜像)
docker-build-amd64:
	docker buildx build --platform linux/amd64 -t $(APP_NAME):$(VERSION)-amd64 --load .

# Docker Compose 启动
docker-up:
	cd deploy/docker && docker-compose up -d

# Docker Compose 停止
docker-down:
	cd deploy/docker && docker-compose down

# 清理
clean:
	rm -rf bin/
	rm -rf data/*.db

# 测试
test:
	go test -v ./...

# 依赖下载
deps:
	go mod download
	go mod tidy

# 帮助
help:
	@echo "可用命令:"
	@echo "  make dev           - 本地开发运行"
	@echo "  make build         - 构建本地二进制"
	@echo "  make build-linux   - 构建 Linux x86 二进制"
	@echo "  make docker-build  - 构建 Docker 镜像"
	@echo "  make docker-build-amd64 - 构建 x86 Docker 镜像"
	@echo "  make docker-up     - Docker Compose 启动"
	@echo "  make docker-down   - Docker Compose 停止"
	@echo "  make clean         - 清理构建产物"
	@echo "  make test          - 运行测试"
