# === Stage 1: Build Frontend (Vue3) ===
FROM node:18 AS frontend-builder
WORKDIR /app
# 换源加速
RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources 2>/dev/null || sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list
COPY frontend/package.json ./
RUN npm config set registry https://registry.npmmirror.com
RUN npm install
COPY frontend/ .
RUN npm run build

# === Stage 2: Build Backend (Go) ===
FROM golang:1.21-bookworm AS builder

# 接收构建参数
ARG GOPROXY=https://goproxy.cn,direct
ARG GO111MODULE=on

# 配置 Go 代理
ENV GOPROXY=${GOPROXY}
ENV GO111MODULE=${GO111MODULE}
ENV CGO_ENABLED=1

# 换源加速
RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources 2>/dev/null || sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建
RUN go build -tags "sqlite_omit_load_extension" -ldflags="-s -w" -o app_server ./cmd/server && ls -l app_server

# === Stage 3: Runtime Image ===
FROM debian:bookworm-slim

WORKDIR /app

# 换源加速
RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources 2>/dev/null || sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list

# 安装运行时依赖
# 尝试直接安装 docker.io (Debian源包含)
RUN apt-get update && \
    apt-get install -y ca-certificates tzdata ansible sshpass curl docker.io && \
    rm -rf /var/lib/apt/lists/*

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
