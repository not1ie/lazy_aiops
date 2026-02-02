# 构建阶段 - 使用 Debian 基础镜像避免 musl libc 问题
FROM golang:1.21-bullseye AS builder

# 接收构建参数
ARG GOPROXY=https://goproxy.cn,direct
ARG GO111MODULE=on

# 配置 Go 代理
ENV GOPROXY=${GOPROXY}
ENV GO111MODULE=${GO111MODULE}
ENV CGO_ENABLED=1

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建
RUN go build -tags "sqlite_omit_load_extension" -ldflags="-s -w" -o app_server ./cmd/server && ls -l app_server

# 运行阶段 - 使用 Debian slim
FROM debian:bullseye-slim

WORKDIR /app

# 安装运行时依赖
RUN apt-get update && \
    apt-get install -y ca-certificates tzdata ansible sshpass curl && \
    curl -fsSL https://download.docker.com/linux/static/stable/x86_64/docker-24.0.7.tgz -o docker.tgz && \
    tar xzvf docker.tgz --strip 1 -C /usr/local/bin docker/docker && \
    rm docker.tgz && \
    rm -rf /var/lib/apt/lists/*

# 设置时区
ENV TZ=Asia/Shanghai

# 复制二进制文件和配置
COPY --from=builder /app/app_server ./lazy-auto-ops
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/web ./web

# 创建数据目录
RUN mkdir -p data

EXPOSE 8080

CMD ["./lazy-auto-ops"]
