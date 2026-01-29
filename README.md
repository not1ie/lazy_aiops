# Lazy Auto Ops

**AI 驱动的轻量级运维自动化平台**

## 核心特性

- 🤖 **AI 原生** - 告警智能降噪、故障诊断、自然语言运维
- 🔌 **插件化架构** - 21个功能模块，按需启用
- 🚀 **极简部署** - 单二进制，SQLite，5分钟上线
- 🔄 **低代码编排** - 可视化工作流，拖拽式自动化
- 🔗 **CI/CD集成** - Jenkins/GitLab/ArgoCD/GitHub Actions

## 插件列表 (21个)

| 分类 | 插件 | 功能 |
|------|------|------|
| **AI能力** | ai | 日志分析、故障诊断、智能问答 |
| | alert | AI智能降噪、告警聚合、静默 |
| **资产管理** | cmdb | 主机、凭据、分组管理 |
| | k8s | 多集群、节点、Pod、Deployment |
| | firewall | 防火墙SNMP监控、规则管理 |
| **监控告警** | monitor | 域名监控、主机监控 |
| | domain | 云域名到期、SSL证书监控 |
| | notify | 飞书/钉钉/企微/邮件/Webhook |
| **自动化** | workflow | 可视化工作流编排 |
| | executor | 批量命令执行、实时输出 |
| | task | 定时任务、脚本执行 |
| | ansible | Playbook/Inventory/Role管理 |
| **CI/CD** | cicd | Jenkins/GitLab/ArgoCD集成、定时发布 |
| **配置中心** | nacos | Nacos配置管理、服务发现 |
| **变更管理** | workorder | 运维工单、审批流程 |
| | sqlaudit | SQL工单、审核规则 |
| | gitops | Git配置同步、变更追踪 |
| **协作** | oncall | 值班排班、换班、升级策略 |
| | terminal | WebSocket SSH终端 |
| **可视化** | topology | 服务拓扑、依赖分析 |
| **成本** | cost | 云费用统计、预算管理、优化建议 |

## 快速开始

### 使用 Docker 部署（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/your-org/lazy-auto-ops.git
cd lazy-auto-ops

# 2. 启动服务
cd deploy/docker
docker-compose up -d

# 3. 访问 Web 界面
# 浏览器打开: http://your-server-ip:8080
# 默认账号: admin / admin123
```

### 本地开发

```bash
# 编译
go mod tidy
go build -o bin/lazy-auto-ops ./cmd/server

# 运行
./bin/lazy-auto-ops

# 访问
# Web界面: http://localhost:8080
# API文档: http://localhost:8080/api/v1/system/info
```

## 🌐 Web 用户界面

现在支持通过浏览器访问！

**功能特性：**
- 🎨 美观的登录界面
- 📊 系统仪表板（版本、插件、状态）
- 🔌 插件管理（已加载/可用插件）
- 📱 响应式设计（支持手机、平板、桌面）
- 🔐 JWT Token 自动管理

**访问地址：** `http://your-server-ip:8080`  
**默认账号：** `admin` / `admin123`

详细使用说明请查看 [WEB访问指南.md](./WEB访问指南.md)

## 新增插件 API

### CI/CD 集成
```bash
# 创建Jenkins流水线
curl -X POST http://localhost:8080/api/v1/cicd/pipelines \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "deploy-prod",
    "provider": "jenkins",
    "jenkins_url": "http://jenkins:8080",
    "jenkins_job": "my-job",
    "jenkins_user": "admin",
    "jenkins_token": "xxx"
  }'

# 触发构建
curl -X POST http://localhost:8080/api/v1/cicd/pipelines/{id}/trigger \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"parameters": {"branch": "main"}}'

# 创建定时发布
curl -X POST http://localhost:8080/api/v1/cicd/schedules \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "每日发布",
    "pipeline_id": "xxx",
    "cron": "0 0 2 * * *",
    "enabled": true
  }'
```

### Ansible 管理
```bash
# 创建Playbook
curl -X POST http://localhost:8080/api/v1/ansible/playbooks \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "deploy-app",
    "content": "- hosts: all\n  tasks:\n    - name: Deploy\n      shell: echo hello"
  }'

# 执行Playbook
curl -X POST http://localhost:8080/api/v1/ansible/playbooks/{id}/execute \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"inventory_id": "xxx", "extra_vars": {"version": "1.0"}}'

# 从CMDB同步Inventory
curl -X POST http://localhost:8080/api/v1/ansible/inventories/sync-cmdb \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "prod-hosts", "group_id": "xxx"}'
```

### Nacos 配置中心
```bash
# 添加Nacos服务器
curl -X POST http://localhost:8080/api/v1/nacos/servers \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "prod-nacos",
    "address": "http://nacos:8848",
    "username": "nacos",
    "password": "nacos"
  }'

# 同步配置
curl -X POST http://localhost:8080/api/v1/nacos/servers/{id}/sync-configs \
  -H "Authorization: Bearer $TOKEN"

# 更新配置(推送到Nacos)
curl -X PUT http://localhost:8080/api/v1/nacos/configs/{id} \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"content": "server.port=8080"}'
```

### 服务拓扑
```bash
# 获取拓扑数据
curl http://localhost:8080/api/v1/topology/data \
  -H "Authorization: Bearer $TOKEN"

# 创建服务节点
curl -X POST http://localhost:8080/api/v1/topology/nodes \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "user-service", "type": "service", "namespace": "prod"}'

# 创建依赖关系
curl -X POST http://localhost:8080/api/v1/topology/edges \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"source_id": "xxx", "target_id": "yyy", "type": "http"}'

# 分析依赖
curl http://localhost:8080/api/v1/topology/analyze \
  -H "Authorization: Bearer $TOKEN"
```

### 成本分析
```bash
# 添加云账号
curl -X POST http://localhost:8080/api/v1/cost/accounts \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "aliyun-prod",
    "provider": "aliyun",
    "access_key": "xxx",
    "secret_key": "xxx"
  }'

# 费用汇总
curl "http://localhost:8080/api/v1/cost/summary?start_date=2024-01-01&end_date=2024-01-31" \
  -H "Authorization: Bearer $TOKEN"

# 创建预算
curl -X POST http://localhost:8080/api/v1/cost/budgets \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "月度预算",
    "amount": 10000,
    "budget_type": "monthly",
    "alert_at": 80
  }'

# 优化建议
curl http://localhost:8080/api/v1/cost/optimizations \
  -H "Authorization: Bearer $TOKEN"
```

## 配置 AI

```yaml
plugins:
  ai:
    enabled: true
    config:
      provider: openai  # 或 ollama
      api_key: your-key
      model: gpt-3.5-turbo
```

## 目录结构

```
lazy-auto-ops/
├── cmd/server/          # 入口
├── internal/            # 核心模块
├── plugins/             # 21个可插拔模块
│   ├── ai/              # AI运维助手
│   ├── alert/           # 告警中心
│   ├── ansible/         # Ansible管理 [NEW]
│   ├── cicd/            # CI/CD集成 [NEW]
│   ├── cmdb/            # 资产管理
│   ├── cost/            # 成本分析 [NEW]
│   ├── domain/          # 域名监控
│   ├── executor/        # 批量执行
│   ├── firewall/        # 防火墙管理
│   ├── gitops/          # GitOps
│   ├── k8s/             # K8s管理
│   ├── monitor/         # 监控
│   ├── nacos/           # Nacos配置中心 [NEW]
│   ├── notify/          # 通知中心
│   ├── oncall/          # 值班排班
│   ├── sqlaudit/        # SQL审计
│   ├── task/            # 任务调度
│   ├── terminal/        # WebTerminal
│   ├── topology/        # 服务拓扑 [NEW]
│   ├── workflow/        # 运维编排
│   └── workorder/       # 运维工单
├── deploy/              # 部署配置
└── configs/             # 配置文件
```

## 差异化优势

1. **AI深度融合** - 不是附加功能，而是核心能力
2. **告警智能降噪** - 同类告警聚合，减少骚扰
3. **低代码编排** - 可视化拖拽，比Ansible更直观
4. **极简部署** - 单文件运行，零依赖
5. **CI/CD原生集成** - Jenkins/GitLab/ArgoCD一键对接
6. **配置中心** - Nacos配置管理、版本对比、回滚
7. **成本可视化** - 云费用分析、预算告警、优化建议
8. **服务拓扑** - 可视化依赖关系、影响分析

## 建议后续功能

- [x] **Web 用户界面** (纯 JavaScript 实现) ✨ 已完成
- [ ] 前端功能增强 (更多插件管理界面)
- [ ] Prometheus/Grafana集成
- [ ] 日志聚合 (Loki/ES)
- [ ] 审计日志增强
- [ ] API限流
- [ ] 多租户支持

## 📚 文档

- [API 文档](./API.md)
- [功能特性](./FEATURES.md)
- [部署指南](./DEPLOYMENT.md)
- [Web 访问指南](./WEB访问指南.md) ✨ NEW
