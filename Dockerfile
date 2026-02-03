# === Stage 1: Build Frontend (Vue3) ===
FROM node:18-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json ./
# 使用淘宝源加速
RUN npm config set registry https://registry.npmmirror.com
RUN npm install
COPY frontend/ .
RUN npm run build

# === Stage 2: Build Backend (Go) ===
FROM golang:1.21-alpine AS builder

# 接收构建参数
ARG GOPROXY=https://goproxy.cn,direct
ARG GO111MODULE=on

# 配置 Go 代理
ENV GOPROXY=${GOPROXY}
ENV GO111MODULE=${GO111MODULE}
ENV CGO_ENABLED=1

# 安装构建依赖 (gcc for cgo)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建
RUN go build -tags "sqlite_omit_load_extension" -ldflags="-s -w" -o app_server ./cmd/server && ls -l app_server

# === Stage 3: Runtime Image ===
FROM alpine:3.19

WORKDIR /app

# 安装运行时依赖
# docker-cli: Docker 客户端
# ansible: 自动化工具
# sshpass: SSH 密码支持
# curl: 网络工具
# tzdata: 时区支持
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    ansible \
    sshpass \
    curl \
    docker-cli

# 设置时区
ENV TZ=Asia/Shanghai

# 复制后端二进制文件
COPY --from=builder /app/app_server ./lazy-auto-ops
# 复制后端配置
COPY --from=builder /app/configs ./configs
# 复制前端构建产物 (dist -> web/static)
COPY --from=frontend-builder /app/dist ./web/static

# 创建数据目录
RUN mkdir -p data

EXPOSE 8080

CMD ["./lazy-auto-ops"]
