# Lazy Auto Ops

Lazy Auto Ops 是一个插件化运维平台，提供资产管理、监控告警、自动化执行、变更流程和容器平台管理能力。

## 项目状态

- 后端：Go
- 前端：Vue 3 + Element Plus
- 默认监听端口：`8080`
- 默认数据库：SQLite（`data/lazy-auto-ops.db`）
- 默认账号：`admin`
- 初始化密码说明：
  - 设置 `LAO_ALLOW_INSECURE_BOOTSTRAP=true` 时，初始密码为 `admin123`
  - 设置 `LAO_BOOTSTRAP_ADMIN_PASSWORD=<你的密码>` 时，初始密码为指定值
  - 都不设置时，系统会自动生成随机临时密码并打印到启动日志
  - 以上逻辑只在首次初始化 `admin` 账号时生效；如果数据目录里已有库文件，不会覆盖已有密码

## 功能介绍

### 资产与环境管理

- CMDB：主机、分组、凭据、数据库与云资源统一管理
- 容器平台：Docker 主机与容器管理、Kubernetes 多集群与工作负载管理
- 服务拓扑：自动发现服务关系，展示依赖链路与影响范围

### 监控、告警与通知

- 指标采集：主机、容器、Pod、服务资源指标展示
- 告警中心：规则、聚合、静默、历史事件追踪
- 通知与值班：多通道通知、值班排班、升级策略

### 自动化与变更

- 任务调度：Cron 定时任务与执行记录
- 批量执行：多主机命令并发执行与结果汇总
- 工作流编排：可视化流程设计、步骤编排
- Ansible：Inventory、Playbook 管理与执行
- 变更管理：工单、SQL 审核、GitOps、CI/CD 发布流程

### 平台治理与审计

- RBAC 权限体系：用户、角色、权限点控制
- 系统管理：菜单、组织、日志、审计相关页面
- Web 终端：浏览器内终端能力（含连接预检与错误提示）
- 堡垒机融合：资产接入、授权策略、命令规则、会话审计、在线 SQL 审计执行（MySQL / PostgreSQL）

### AI 相关能力

- AI 运维助手：上下文问答与 Runbook 模板辅助
- 知识库：文档沉淀、检索与问答
- 智能分析：告警/日志场景下的辅助分析能力

## 模块清单（当前代码）

`ai`, `alert`, `ansible`, `application`, `cicd`, `cmdb`, `cost`, `docker`, `domain`, `executor`, `firewall`, `gitops`, `jump`, `k8s`, `knowledge`, `monitor`, `nacos`, `notify`, `oncall`, `rbac`, `remediation`, `sqlaudit`, `system`, `task`, `terminal`, `topology`, `workflow`, `workorder`

## 部署方式

支持三种主部署方式：Docker、Kubernetes、系统服务（systemd）。

### 1) Docker 镜像部署（ACR 匿名拉取）

适用于直接拉取已发布镜像快速部署，不需要本地构建。当前仓库已开放匿名拉取。

```bash
REGISTRY=crpi-iihofxt94xlrdrvd.cn-shanghai.personal.cr.aliyuncs.com
IMAGE=$REGISTRY/lazyops/lazyops
VERSION=v1.0.15

# 拉取镜像
docker pull $IMAGE:$VERSION
```

#### Docker Run 直接部署

```bash
REGISTRY=crpi-iihofxt94xlrdrvd.cn-shanghai.personal.cr.aliyuncs.com
IMAGE=$REGISTRY/lazyops/lazyops
VERSION=v1.0.15

mkdir -p $(pwd)/lazy-aiops/{data,configs}

docker run -d --name lazy-aiops \
  --restart unless-stopped \
  -p 8080:8080 \
  -e TZ=Asia/Shanghai \
  -e LAO_ALLOW_INSECURE_BOOTSTRAP=true \
  -v $(pwd)/lazy-aiops/data:/app/data \
  $IMAGE:$VERSION
```

如需覆盖默认配置，可挂载配置文件：

```bash
docker run -d --name lazy-aiops \
  --restart unless-stopped \
  -p 8080:8080 \
  -e TZ=Asia/Shanghai \
  -e LAO_ALLOW_INSECURE_BOOTSTRAP=true \
  -v $(pwd)/lazy-aiops/data:/app/data \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml:ro \
  $IMAGE:$VERSION
```

#### Docker Swarm 发布

```bash
export LAZY_AIOPS_IMAGE=$IMAGE:$VERSION
export LAO_ALLOW_INSECURE_BOOTSTRAP=true

LAZY_AIOPS_IMAGE=$IMAGE:$VERSION \
docker stack deploy -c deploy/swarm/stack.yml lazy-aiops
```

验证：

```bash
docker ps | grep lazy-aiops
curl -s http://127.0.0.1:8080/health
```

### 2) Docker 源码部署

```bash
git clone https://github.com/not1ie/lazy_aiops.git
cd lazy_aiops

docker compose -f deploy/docker/docker-compose.yml up -d --build
```

如需初始化默认密码为 `admin123`，可在启动前导出：

```bash
export LAO_ALLOW_INSECURE_BOOTSTRAP=true
docker compose -f deploy/docker/docker-compose.yml up -d --build
```

生产配置：

```bash
docker compose -f deploy/docker/docker-compose.prod.yml up -d --build
```

验证：

```bash
docker logs -f lazy-auto-ops
curl -s http://127.0.0.1:8080/health
```

### 3) Kubernetes 部署

```bash
git clone https://github.com/not1ie/lazy_aiops.git
cd lazy_aiops

IMAGE=crpi-iihofxt94xlrdrvd.cn-shanghai.personal.cr.aliyuncs.com/lazyops/lazyops:v1.0.15
docker pull $IMAGE

kubectl apply -k deploy/k8s
kubectl -n lazy-aiops set image deployment/lazy-auto-ops \
  lazy-auto-ops=$IMAGE
kubectl -n lazy-aiops set env deployment/lazy-auto-ops \
  LAO_ALLOW_INSECURE_BOOTSTRAP=true
kubectl -n lazy-aiops rollout status deployment/lazy-auto-ops
```

验证：

```bash
kubectl -n lazy-aiops get pod,svc
kubectl -n lazy-aiops port-forward svc/lazy-auto-ops 8080:80
```

如集群已安装 Ingress Controller，可启用：

```bash
kubectl apply -f deploy/k8s/ingress.yaml
```

### 4) 系统部署（Linux + systemd）

```bash
git clone https://github.com/not1ie/lazy_aiops.git
cd lazy_aiops
go mod tidy
go build -o bin/lazy-auto-ops ./cmd/server

sudo mkdir -p /opt/lazy-aiops/{bin,configs,data}
sudo cp bin/lazy-auto-ops /opt/lazy-aiops/bin/
sudo cp configs/config.yaml /opt/lazy-aiops/configs/
```

创建 `/etc/systemd/system/lazy-aiops.service`：

```ini
[Unit]
Description=Lazy Auto Ops
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/lazy-aiops
ExecStart=/opt/lazy-aiops/bin/lazy-auto-ops
Restart=always
RestartSec=5
Environment=TZ=Asia/Shanghai
Environment=LAO_ALLOW_INSECURE_BOOTSTRAP=true

[Install]
WantedBy=multi-user.target
```

启动：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now lazy-aiops
sudo systemctl status lazy-aiops --no-pager
curl -s http://127.0.0.1:8080/health
```

## 可选：统一脚本（K8s / Swarm）

```bash
# Kubernetes
REGISTRY_IMAGE=registry.example.com/lazy-aiops:v1.0.15 deploy/scripts/deploy.sh k8s

# Docker Swarm
REGISTRY_IMAGE=registry.example.com/lazy-aiops:v1.0.15 deploy/scripts/deploy.sh swarm
```

## 版本信息

- 当前推荐版本：`v1.0.15`
- 对应代码提交：`7e5dc0046964`
- ACR 镜像示例：
  - `crpi-iihofxt94xlrdrvd.cn-shanghai.personal.cr.aliyuncs.com/lazyops/lazyops:v1.0.15`
  - `crpi-iihofxt94xlrdrvd.cn-shanghai.personal.cr.aliyuncs.com/lazyops/lazyops:latest`

## 开发与验证

```bash
# 后端
go build ./...

# 前端
cd frontend && npm run build && cd ..

# 一键检查
bash scripts/verify_all.sh

# 堡垒机 P0 验证（需要可登录管理员密码）
PASSWORD='your-password' BASE_URL='http://127.0.0.1:8080' bash scripts/verify_jump_p0.sh
```

## 文档

- `deploy/docker/README.md`
- `deploy/k8s/README.md`
- `deploy/swarm/README.md`
- `deploy/README.md`
- `docs/DEPLOYMENT.md`
- `docs/regression-checklist.md`
- `docs/jumpserver-fusion-plan.md`

## 联系方式

有工作推荐请联系：`slpsz1774190386@gmail.com`

联系作者：

![微信二维码](docs/wechat-qrcode.png)
