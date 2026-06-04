# 可通过 --build-arg 覆盖镜像源（例如私有镜像仓库）
ARG NODE_IMAGE=node:20-bookworm
ARG GO_IMAGE=golang:1.21-bookworm
ARG RUNTIME_IMAGE=debian:bookworm-slim

# === Stage 1: Build Frontend (Vue3) ===
FROM ${NODE_IMAGE} AS frontend-builder
WORKDIR /app
RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources 2>/dev/null || sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list
COPY frontend/package.json ./
RUN npm config set registry https://registry.npmmirror.com
RUN npm install
COPY frontend/ .
RUN npm run build

# === Stage 2: Build Backend (Go) ===
FROM ${GO_IMAGE} AS builder

ARG GOPROXY=https://goproxy.cn,direct
ARG GO111MODULE=on

ENV GOPROXY=${GOPROXY}
ENV GO111MODULE=${GO111MODULE}
ENV CGO_ENABLED=1

RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources 2>/dev/null || sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -tags "sqlite_omit_load_extension" -ldflags="-s -w" -o app_server ./cmd/server && ls -l app_server

# === Stage 3: Runtime Image ===
FROM ${RUNTIME_IMAGE}

WORKDIR /app

RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources 2>/dev/null || sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list

RUN apt-get update && \
    apt-get install -y ca-certificates tzdata ansible sshpass curl docker.io && \
    rm -rf /var/lib/apt/lists/*

ENV TZ=Asia/Shanghai

COPY --from=builder /app/app_server ./lazy-auto-ops
COPY --from=builder /app/configs ./configs
COPY --from=frontend-builder /app/dist ./web/static

RUN mkdir -p data

EXPOSE 8080

CMD ["./lazy-auto-ops"]
