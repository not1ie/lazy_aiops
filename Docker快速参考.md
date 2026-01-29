# Docker 快速参考

## 🚀 快速开始（3步）

### 步骤1：修复文件（必须！）
```bash
# 用文本编辑器打开 topology_handler_code.txt
# 复制全部内容到 plugins/topology/handler.go
# 保存
```

### 步骤2：选择部署方式

**方式A：一键部署（推荐）**
```bash
chmod +x docker-deploy.sh
./docker-deploy.sh
```

**方式B：Docker Compose**
```bash
cd deploy/docker
docker-compose up -d
```

**方式C：Docker直接运行**
```bash
docker build -t lazy-auto-ops .
docker run -d -p 8080:8080 -v $(pwd)/data:/app/data lazy-auto-ops
```

### 步骤3：访问服务
```
地址: http://localhost:8080
账号: admin
密码: admin123
```

---

## 📋 常用命令

### Docker Compose

```bash
# 启动服务
docker-compose up -d

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose stop

# 重启服务
docker-compose restart

# 删除服务
docker-compose down

# 重新构建
docker-compose up -d --build
```

### Docker

```bash
# 查看容器
docker ps

# 查看日志
docker logs -f lazy-auto-ops

# 进入容器
docker exec -it lazy-auto-ops sh

# 停止容器
docker stop lazy-auto-ops

# 启动容器
docker start lazy-auto-ops

# 删除容器
docker rm lazy-auto-ops

# 删除镜像
docker rmi lazy-auto-ops
```

---

## 🔧 配置

### 端口映射
```bash
-p 8080:8080  # HTTP端口
```

### 数据持久化
```bash
-v $(pwd)/data:/app/data  # 数据目录
```

### 配置文件
```bash
-v $(pwd)/configs/config.yaml:/app/configs/config.yaml:ro
```

### 环境变量
```bash
-e TZ=Asia/Shanghai
-e LAO_SERVER_MODE=release
```

---

## 🐛 故障排查

### 容器无法启动
```bash
# 查看日志
docker logs lazy-auto-ops

# 检查文件
ls -lh plugins/topology/handler.go  # 应该 > 5KB
```

### 无法访问服务
```bash
# 检查容器状态
docker ps | grep lazy-auto-ops

# 检查端口
docker port lazy-auto-ops

# 测试连接
curl http://localhost:8080/health
```

### 数据丢失
```bash
# 确保使用了数据卷
docker inspect lazy-auto-ops | grep Mounts
```

---

## 📚 详细文档

- **DOCKER部署指南.md** - 完整的Docker部署文档
- **docker-deploy.sh** - 一键部署脚本
- **deploy/docker/** - Docker配置文件目录

---

## ✅ 验证部署

```bash
# 1. 健康检查
curl http://localhost:8080/health

# 2. 登录
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 3. 获取插件列表
TOKEN="your-token"
curl http://localhost:8080/api/v1/plugins \
  -H "Authorization: Bearer $TOKEN"
```

---

## 🎯 生产环境

```bash
# 1. 创建环境变量
cd deploy/docker
cp .env.example .env
vim .env  # 修改密码

# 2. 启动生产环境
docker-compose -f docker-compose.prod.yml up -d

# 3. 配置SSL证书
# 将证书放到 deploy/docker/ssl/ 目录
# 编辑 nginx.conf 取消SSL配置注释
```

---

**快速帮助**: 查看 DOCKER部署指南.md 获取详细说明
