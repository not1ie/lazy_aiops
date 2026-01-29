# Lazy Auto Ops - Docker 部署指南

## 📋 目录

1. [前提条件](#前提条件)
2. [快速开始](#快速开始)
3. [方式1：Docker直接运行](#方式1docker直接运行)
4. [方式2：Docker Compose运行](#方式2docker-compose运行)
5. [配置说明](#配置说明)
6. [数据持久化](#数据持久化)
7. [多架构支持](#多架构支持)
8. [生产环境部署](#生产环境部署)
9. [常见问题](#常见问题)

---

## 前提条件

### 必须完成的准备工作

⚠️ **重要**: 在Docker部署之前，必须先修复 `plugins/topology/handler.go` 文件！

```bash
# 1. 用文本编辑器打开 topology_handler_code.txt
# 2. 复制全部内容
# 3. 粘贴到 plugins/topology/handler.go
# 4. 保存文件

# 验证文件是否正确
ls -lh plugins/topology/handler.go  # 应该 > 5KB
file plugins/topology/handler.go     # 应该显示 "Go source"
```

### 系统要求

- Docker >= 20.10
- Docker Compose >= 2.0 (如果使用docker-compose)
- 至少 2GB 可用内存
- 至少 5GB 可用磁盘空间

### 安装Docker

**macOS:**
```bash
brew install docker docker-compose
```

**Ubuntu/Debian:**
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
```

**CentOS/RHEL:**
```bash
sudo yum install -y docker docker-compose
sudo systemctl start docker
sudo systemctl enable docker
```

---

## 快速开始

最快的部署方式（3步）：

```bash
# 1. 进入项目目录
cd lazy-auto-ops

# 2. 修复 topology/handler.go（必须！）
# 用文本编辑器复制 topology_handler_code.txt 到 plugins/topology/handler.go

# 3. 使用 Docker Compose 启动
cd deploy/docker
docker-compose up -d
```

等待1-2分钟后访问：http://localhost:8080

---

## 方式1：Docker直接运行

### 1.1 构建镜像

```bash
cd lazy-auto-ops

# 构建镜像
docker build -t lazy-auto-ops:latest .

# 查看镜像
docker images | grep lazy-auto-ops
```

### 1.2 运行容器

**基础运行（使用SQLite）:**

```bash
docker run -d \
  --name lazy-auto-ops \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -e TZ=Asia/Shanghai \
  lazy-auto-ops:latest
```

**完整运行（带配置文件）:**

```bash
docker run -d \
  --name lazy-auto-ops \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml:ro \
  -e TZ=Asia/Shanghai \
  -e LAO_SERVER_MODE=release \
  --restart unless-stopped \
  lazy-auto-ops:latest
```

### 1.3 查看日志

```bash
# 查看实时日志
docker logs -f lazy-auto-ops

# 查看最近100行日志
docker logs --tail 100 lazy-auto-ops
```

### 1.4 停止和删除

```bash
# 停止容器
docker stop lazy-auto-ops

# 删除容器
docker rm lazy-auto-ops

# 删除镜像
docker rmi lazy-auto-ops:latest
```

---

## 方式2：Docker Compose运行

### 2.1 准备配置文件

Docker Compose配置文件位于 `deploy/docker/docker-compose.yml`

**查看配置:**
```bash
cd lazy-auto-ops/deploy/docker
cat docker-compose.yml
```

**自定义配置（可选）:**
```bash
# 复制并编辑配置文件
cp config.yaml config.yaml.bak
vim config.yaml
```

### 2.2 启动服务

```bash
cd lazy-auto-ops/deploy/docker

# 启动服务（后台运行）
docker-compose up -d

# 启动服务（前台运行，查看日志）
docker-compose up

# 查看服务状态
docker-compose ps
```

### 2.3 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f lazy-auto-ops

# 查看最近100行
docker-compose logs --tail 100
```

### 2.4 管理服务

```bash
# 停止服务
docker-compose stop

# 启动服务
docker-compose start

# 重启服务
docker-compose restart

# 停止并删除容器
docker-compose down

# 停止并删除容器、网络、卷
docker-compose down -v
```

### 2.5 更新服务

```bash
# 重新构建镜像
docker-compose build

# 重新构建并启动
docker-compose up -d --build

# 拉取最新镜像（如果使用远程镜像）
docker-compose pull
docker-compose up -d
```

---

## 配置说明

### 环境变量

可以通过环境变量覆盖配置：

```bash
docker run -d \
  --name lazy-auto-ops \
  -p 8080:8080 \
  -e LAO_SERVER_PORT=8080 \
  -e LAO_SERVER_MODE=release \
  -e LAO_DB_DRIVER=sqlite \
  -e LAO_DB_DSN=data/lazy-auto-ops.db \
  -e LAO_JWT_SECRET=your-secret-key \
  -e TZ=Asia/Shanghai \
  lazy-auto-ops:latest
```

### 配置文件

**编辑 `deploy/docker/config.yaml`:**

```yaml
server:
  port: "8080"
  mode: "release"  # debug, release, test

database:
  driver: "sqlite"  # sqlite, mysql, postgres
  dsn: "data/lazy-auto-ops.db"

jwt:
  secret: "your-production-secret-key-change-me"
  expire: 24

plugins:
  # 启用所有插件
  cmdb:
    enabled: true
  monitor:
    enabled: true
  alert:
    enabled: true
  notify:
    enabled: true
  task:
    enabled: true
  k8s:
    enabled: true
  ai:
    enabled: true
    config:
      provider: "openai"
      api_key: "your-api-key"
      model: "gpt-3.5-turbo"
  gitops:
    enabled: true
  terminal:
    enabled: true
  firewall:
    enabled: true
  domain:
    enabled: true
  sqlaudit:
    enabled: true
  workorder:
    enabled: true
  oncall:
    enabled: true
  workflow:
    enabled: true
  executor:
    enabled: true
  cicd:
    enabled: true
  ansible:
    enabled: true
  nacos:
    enabled: true
  topology:
    enabled: true
  cost:
    enabled: true
```

---

## 数据持久化

### SQLite（默认）

```bash
# 数据目录映射
-v $(pwd)/data:/app/data
```

数据文件位置：`data/lazy-auto-ops.db`

### MySQL

**1. 启动MySQL容器:**

```yaml
# docker-compose.yml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: lazy-auto-ops-mysql
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: lazy_auto_ops
      MYSQL_USER: lazyops
      MYSQL_PASSWORD: lazyops123
    volumes:
      - mysql-data:/var/lib/mysql
    ports:
      - "3306:3306"

  lazy-auto-ops:
    build:
      context: ../..
      dockerfile: Dockerfile
    container_name: lazy-auto-ops
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/configs/config.yaml:ro
    environment:
      - TZ=Asia/Shanghai
    depends_on:
      - mysql

volumes:
  mysql-data:
```

**2. 修改配置文件:**

```yaml
# config.yaml
database:
  driver: "mysql"
  dsn: "lazyops:lazyops123@tcp(mysql:3306)/lazy_auto_ops?charset=utf8mb4&parseTime=True&loc=Local"
```

### PostgreSQL

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:15-alpine
    container_name: lazy-auto-ops-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: lazy_auto_ops
      POSTGRES_USER: lazyops
      POSTGRES_PASSWORD: lazyops123
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

volumes:
  postgres-data:
```

```yaml
# config.yaml
database:
  driver: "postgres"
  dsn: "host=postgres user=lazyops password=lazyops123 dbname=lazy_auto_ops port=5432 sslmode=disable"
```

---

## 多架构支持

### 构建多架构镜像

**在ARM Mac上构建x86镜像（用于Linux服务器）:**

```bash
# 使用 buildx
docker buildx create --use
docker buildx build --platform linux/amd64 -t lazy-auto-ops:amd64 --load .

# 或使用 Makefile
make docker-build-amd64
```

**构建多架构镜像并推送:**

```bash
# 构建并推送到Docker Hub
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t yourusername/lazy-auto-ops:latest \
  --push .
```

### 使用预构建镜像

```bash
# 拉取镜像
docker pull yourusername/lazy-auto-ops:latest

# 运行
docker run -d \
  --name lazy-auto-ops \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  yourusername/lazy-auto-ops:latest
```

---

## 生产环境部署

### 完整的生产环境配置

**docker-compose.prod.yml:**

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: lazy-auto-ops-mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: lazy_auto_ops
      MYSQL_USER: lazyops
      MYSQL_PASSWORD: ${MYSQL_PASSWORD}
    volumes:
      - mysql-data:/var/lib/mysql
      - ./mysql-init:/docker-entrypoint-initdb.d
    networks:
      - lazy-auto-ops-net
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: lazy-auto-ops-redis
    restart: always
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis-data:/data
    networks:
      - lazy-auto-ops-net
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  lazy-auto-ops:
    image: lazy-auto-ops:latest
    container_name: lazy-auto-ops
    restart: always
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./config.yaml:/app/configs/config.yaml:ro
      - ./logs:/app/logs
    environment:
      - TZ=Asia/Shanghai
      - LAO_SERVER_MODE=release
      - LAO_JWT_SECRET=${JWT_SECRET}
    networks:
      - lazy-auto-ops-net
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  nginx:
    image: nginx:alpine
    container_name: lazy-auto-ops-nginx
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    networks:
      - lazy-auto-ops-net
    depends_on:
      - lazy-auto-ops

networks:
  lazy-auto-ops-net:
    driver: bridge

volumes:
  mysql-data:
  redis-data:
```

**.env 文件:**

```bash
# .env
MYSQL_ROOT_PASSWORD=your-strong-root-password
MYSQL_PASSWORD=your-strong-password
REDIS_PASSWORD=your-redis-password
JWT_SECRET=your-jwt-secret-key-at-least-32-chars
```

**启动生产环境:**

```bash
# 创建 .env 文件
cat > .env << EOF
MYSQL_ROOT_PASSWORD=$(openssl rand -base64 32)
MYSQL_PASSWORD=$(openssl rand -base64 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 32)
EOF

# 启动
docker-compose -f docker-compose.prod.yml up -d

# 查看状态
docker-compose -f docker-compose.prod.yml ps
```

### Nginx反向代理配置

**nginx.conf:**

```nginx
events {
    worker_connections 1024;
}

http {
    upstream lazy-auto-ops {
        server lazy-auto-ops:8080;
    }

    server {
        listen 80;
        server_name your-domain.com;

        # 重定向到HTTPS
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name your-domain.com;

        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;

        location / {
            proxy_pass http://lazy-auto-ops;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # WebSocket支持
        location /ws {
            proxy_pass http://lazy-auto-ops;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}
```

---

## 常见问题

### Q1: 容器启动失败

**检查日志:**
```bash
docker logs lazy-auto-ops
```

**常见原因:**
- topology/handler.go 文件为空（必须先修复）
- 端口被占用
- 配置文件错误
- 权限问题

### Q2: 无法访问服务

**检查容器状态:**
```bash
docker ps | grep lazy-auto-ops
```

**检查端口映射:**
```bash
docker port lazy-auto-ops
```

**测试连接:**
```bash
curl http://localhost:8080/health
```

### Q3: 数据丢失

确保使用了数据卷：
```bash
docker run -v $(pwd)/data:/app/data ...
```

### Q4: 性能问题

**增加资源限制:**
```bash
docker run \
  --memory="2g" \
  --cpus="2" \
  ...
```

### Q5: 如何备份数据

**SQLite备份:**
```bash
# 停止容器
docker-compose stop

# 备份数据库
cp data/lazy-auto-ops.db data/lazy-auto-ops.db.backup

# 启动容器
docker-compose start
```

**MySQL备份:**
```bash
docker exec lazy-auto-ops-mysql \
  mysqldump -u lazyops -p lazy_auto_ops > backup.sql
```

### Q6: 如何更新版本

```bash
# 停止服务
docker-compose down

# 拉取最新代码
git pull

# 重新构建
docker-compose build

# 启动服务
docker-compose up -d
```

---

## 验证部署

### 1. 健康检查

```bash
curl http://localhost:8080/health
# 预期: {"status":"ok"}
```

### 2. 登录测试

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

### 3. 获取插件列表

```bash
TOKEN="your-token"
curl http://localhost:8080/api/v1/plugins \
  -H "Authorization: Bearer $TOKEN"
```

### 4. 测试新增插件

```bash
# CI/CD
curl http://localhost:8080/api/v1/cicd/pipelines \
  -H "Authorization: Bearer $TOKEN"

# Ansible
curl http://localhost:8080/api/v1/ansible/playbooks \
  -H "Authorization: Bearer $TOKEN"

# Nacos
curl http://localhost:8080/api/v1/nacos/servers \
  -H "Authorization: Bearer $TOKEN"

# Topology
curl http://localhost:8080/api/v1/topology/nodes \
  -H "Authorization: Bearer $TOKEN"

# Cost
curl http://localhost:8080/api/v1/cost/accounts \
  -H "Authorization: Bearer $TOKEN"
```

---

## 下一步

- 配置AI功能（OpenAI API Key）
- 配置通知渠道（钉钉、企微、飞书）
- 添加K8s集群
- 配置CI/CD集成
- 开发前端界面

---

**祝你部署成功！🎉**

如有问题，请查看其他文档或提交Issue。
