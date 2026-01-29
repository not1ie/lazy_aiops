# 部署指南

## 前置要求

- Go 1.21+
- SQLite3（或MySQL/PostgreSQL）
- （可选）Docker
- （可选）Kubernetes

## 本地开发部署

### 1. 克隆代码
```bash
git clone <your-repo>
cd lazy-auto-ops
```

### 2. 安装依赖
```bash
go mod tidy
```

### 3. 配置
编辑 `configs/config.yaml`：
```yaml
server:
  port: "8080"
  mode: "debug"

database:
  driver: "sqlite"
  dsn: "data/lazy-auto-ops.db"

jwt:
  secret: "change-me-in-production"
  expire: 24

plugins:
  # 启用需要的插件
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
  ai:
    enabled: true
    config:
      provider: "openai"
      api_key: "your-api-key"
      model: "gpt-3.5-turbo"
```

### 4. 编译
```bash
go build -o bin/lazy-auto-ops ./cmd/server
```

### 5. 运行
```bash
./bin/lazy-auto-ops
```

### 6. 访问
打开浏览器访问：http://localhost:8080

默认账号：`admin` / `admin123`

## Docker部署

### 1. 构建镜像
```bash
docker build -t lazy-auto-ops:latest .
```

### 2. 运行容器
```bash
docker run -d \
  --name lazy-auto-ops \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs \
  lazy-auto-ops:latest
```

### 3. 使用Docker Compose
```bash
cd deploy/docker
docker-compose up -d
```

## Kubernetes部署

### 1. 创建ConfigMap
```bash
kubectl create configmap lazy-auto-ops-config \
  --from-file=configs/config.yaml
```

### 2. 部署
```bash
kubectl apply -f deploy/k8s/deployment.yaml
```

### 3. 暴露服务
```bash
kubectl expose deployment lazy-auto-ops \
  --type=LoadBalancer \
  --port=8080
```

## 生产环境建议

### 1. 数据库
使用MySQL或PostgreSQL替代SQLite：
```yaml
database:
  driver: "mysql"
  dsn: "user:password@tcp(mysql:3306)/lazy_auto_ops?charset=utf8mb4&parseTime=True"
```

### 2. 安全
- 修改默认密码
- 使用强JWT密钥
- 启用HTTPS
- 配置防火墙

### 3. 高可用
- 多实例部署
- 使用外部数据库
- 配置负载均衡

### 4. 监控
- 配置Prometheus监控
- 设置告警规则
- 日志聚合

### 5. 备份
定期备份数据库：
```bash
# SQLite
cp data/lazy-auto-ops.db data/backup/lazy-auto-ops-$(date +%Y%m%d).db

# MySQL
mysqldump -u user -p lazy_auto_ops > backup.sql
```

## 插件配置

### AI插件
```yaml
plugins:
  ai:
    enabled: true
    config:
      provider: "openai"  # 或 "ollama"
      api_key: "sk-xxx"
      model: "gpt-3.5-turbo"
      base_url: ""  # 可选，自定义API地址
```

### CI/CD插件
无需额外配置，在界面中添加Jenkins/GitLab等连接信息。

### Ansible插件
确保服务器已安装ansible：
```bash
pip install ansible
```

### Nacos插件
在界面中配置Nacos服务器地址和认证信息。

### Cost插件
在界面中配置云账号AccessKey/SecretKey。

## 故障排查

### 1. 服务无法启动
检查日志：
```bash
./bin/lazy-auto-ops 2>&1 | tee app.log
```

### 2. 数据库连接失败
- 检查数据库配置
- 确认数据库服务运行
- 检查网络连接

### 3. 插件加载失败
- 检查配置文件语法
- 查看插件依赖
- 检查日志输出

### 4. API调用失败
- 检查JWT token是否有效
- 确认API路径正确
- 查看返回的错误信息

## 性能优化

### 1. 数据库优化
- 添加索引
- 定期清理历史数据
- 使用连接池

### 2. 缓存
- 启用Redis缓存
- 缓存频繁查询的数据

### 3. 并发
- 调整Go并发参数
- 优化数据库连接数

## 升级指南

### 1. 备份数据
```bash
cp -r data data.backup
```

### 2. 停止服务
```bash
pkill lazy-auto-ops
```

### 3. 更新代码
```bash
git pull
go build -o bin/lazy-auto-ops ./cmd/server
```

### 4. 数据库迁移
服务启动时会自动执行数据库迁移。

### 5. 启动服务
```bash
./bin/lazy-auto-ops
```

## 常见问题

**Q: 如何修改默认端口？**
A: 修改 `configs/config.yaml` 中的 `server.port`

**Q: 如何禁用某个插件？**
A: 在配置文件中设置 `plugins.<plugin-name>.enabled: false`

**Q: 如何重置管理员密码？**
A: 直接修改数据库中的users表，或删除数据库重新初始化

**Q: 支持哪些数据库？**
A: SQLite、MySQL、PostgreSQL

**Q: 如何配置HTTPS？**
A: 使用Nginx反向代理，或修改代码支持TLS

## 技术支持

- 文档：查看 README.md 和 FEATURES.md
- 问题：提交 GitHub Issue
- 讨论：加入社区讨论组
