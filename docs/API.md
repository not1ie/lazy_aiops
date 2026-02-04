# API 文档

## 认证

所有API（除登录外）都需要JWT认证。

### 登录
```bash
POST /api/v1/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}

# 响应
{
  "code": 0,
  "data": {
    "token": "eyJhbGc...",
    "expire": 1234567890,
    "user_info": {...}
  }
}
```

### 使用Token
```bash
Authorization: Bearer <token>
```

## 通用响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

错误码：
- 0: 成功
- 400: 参数错误
- 401: 未认证
- 403: 无权限
- 404: 资源不存在
- 500: 服务器错误

## CI/CD插件 API

### 流水线管理

#### 创建流水线
```bash
POST /api/v1/cicd/pipelines
{
  "name": "deploy-prod",
  "provider": "jenkins",  # jenkins/gitlab/argocd/github
  "jenkins_url": "http://jenkins:8080",
  "jenkins_job": "my-job",
  "jenkins_user": "admin",
  "jenkins_token": "xxx"
}
```

#### 获取流水线列表
```bash
GET /api/v1/cicd/pipelines?provider=jenkins
```

#### 触发构建
```bash
POST /api/v1/cicd/pipelines/{id}/trigger
{
  "parameters": {
    "branch": "main",
    "version": "1.0.0"
  }
}
```

#### 获取执行记录
```bash
GET /api/v1/cicd/executions?pipeline_id={id}
```

#### 获取执行日志
```bash
GET /api/v1/cicd/executions/{id}/logs
```

### 定时发布

#### 创建定时任务
```bash
POST /api/v1/cicd/schedules
{
  "name": "每日发布",
  "pipeline_id": "xxx",
  "cron": "0 0 2 * * *",
  "parameters": "{\"branch\":\"main\"}",
  "enabled": true
}
```

#### 启用/禁用定时任务
```bash
POST /api/v1/cicd/schedules/{id}/toggle
```

## Ansible插件 API

### Playbook管理

#### 创建Playbook
```bash
POST /api/v1/ansible/playbooks
{
  "name": "deploy-app",
  "description": "部署应用",
  "content": "---\n- hosts: all\n  tasks:\n    - name: Deploy\n      shell: echo hello"
}
```

#### 执行Playbook
```bash
POST /api/v1/ansible/playbooks/{id}/execute
{
  "inventory_id": "xxx",
  "extra_vars": {
    "version": "1.0.0",
    "env": "prod"
  },
  "tags": "deploy",
  "limit": "web-servers",
  "check": false
}
```

#### 验证语法
```bash
POST /api/v1/ansible/playbooks/{id}/validate
```

#### 解析变量
```bash
GET /api/v1/ansible/playbooks/{id}/variables
```

### Inventory管理

#### 从CMDB同步
```bash
POST /api/v1/ansible/inventories/sync-cmdb
{
  "name": "prod-hosts",
  "group_id": "xxx",
  "host_ids": ["host1", "host2"]
}
```

### 执行记录

#### 实时输出（SSE）
```bash
GET /api/v1/ansible/executions/{id}/stream
```

## Nacos插件 API

### 服务器管理

#### 添加Nacos服务器
```bash
POST /api/v1/nacos/servers
{
  "name": "prod-nacos",
  "address": "http://nacos:8848",
  "namespace": "public",
  "username": "nacos",
  "password": "nacos"
}
```

#### 测试连接
```bash
POST /api/v1/nacos/servers/{id}/test
```

### 配置管理

#### 同步配置
```bash
POST /api/v1/nacos/servers/{id}/sync-configs
```

#### 获取配置列表
```bash
GET /api/v1/nacos/configs?server_id={id}&group=DEFAULT_GROUP
```

#### 更新配置（推送到Nacos）
```bash
PUT /api/v1/nacos/configs/{id}
{
  "content": "server.port=8080\nspring.application.name=demo"
}
```

#### 配置历史
```bash
GET /api/v1/nacos/configs/{id}/history
```

#### 配置对比
```bash
GET /api/v1/nacos/configs/{id}/compare
```

#### 回滚配置
```bash
POST /api/v1/nacos/configs/history/{history_id}/rollback
```

### 服务发现

#### 同步服务
```bash
POST /api/v1/nacos/servers/{id}/sync-services
```

#### 获取服务实例
```bash
GET /api/v1/nacos/services/instances?server_id={id}&service_name=demo
```

## K8s 插件 API

### 集群管理

```bash
GET /api/v1/k8s/clusters
POST /api/v1/k8s/clusters
GET /api/v1/k8s/clusters/{id}
PUT /api/v1/k8s/clusters/{id}
DELETE /api/v1/k8s/clusters/{id}
POST /api/v1/k8s/clusters/{id}/test
```

### 资源列表

```bash
GET /api/v1/k8s/clusters/{id}/nodes
GET /api/v1/k8s/clusters/{id}/namespaces
GET /api/v1/k8s/clusters/{id}/workloads?namespace=default
GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/workloads/{kind}/{name}
GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/workloads/{kind}/{name}/manifest?format=yaml
GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/workloads/{kind}/{name}/manifest?format=json
POST /api/v1/k8s/clusters/{id}/namespaces/{ns}/workloads/{kind}/{name}/manifest/apply
PUT /api/v1/k8s/clusters/{id}/namespaces/{ns}/workloads/{kind}/{name}/scale
POST /api/v1/k8s/clusters/{id}/namespaces/{ns}/workloads/{kind}/{name}/restart

GET /api/v1/k8s/clusters/{id}/services?namespace=default
GET /api/v1/k8s/clusters/{id}/ingresses?namespace=default
GET /api/v1/k8s/clusters/{id}/configmaps?namespace=default
GET /api/v1/k8s/clusters/{id}/secrets?namespace=default
GET /api/v1/k8s/clusters/{id}/storageclasses
GET /api/v1/k8s/clusters/{id}/persistentvolumes
GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/persistentvolumeclaims
GET /api/v1/k8s/clusters/{id}/events?namespace=default
```

### Deployment / Pod 操作

```bash
GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments
PUT /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments/{name}/scale
POST /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments/{name}/restart

GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods
GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods?selector=app%3Ddemo
GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}
GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}/logs?container=xxx&tail=100
GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}/logs/stream?container=xxx&tail=100&token=JWT
DELETE /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}
POST /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}/restart
POST /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{name}/restart-workload
GET /api/v1/k8s/clusters/{id}/namespaces/{ns}/pods/{pod}/exec?container=xxx&token=JWT
```

## 服务拓扑插件 API

### 节点管理

#### 获取拓扑数据
```bash
GET /api/v1/topology/data?namespace=prod
```

#### 创建节点
```bash
POST /api/v1/topology/nodes
{
  "name": "user-service",
  "type": "service",
  "namespace": "prod",
  "cluster": "k8s-prod",
  "description": "用户服务"
}
```

#### 更新节点位置
```bash
PUT /api/v1/topology/nodes/{id}/position
{
  "x": 100,
  "y": 200
}
```

### 关系管理

#### 创建依赖关系
```bash
POST /api/v1/topology/edges
{
  "source_id": "service-a",
  "target_id": "service-b",
  "type": "http",
  "protocol": "HTTP/1.1",
  "port": 8080
}
```

### 分析

#### 依赖分析
```bash
GET /api/v1/topology/analyze
```

#### 节点详情
```bash
GET /api/v1/topology/nodes/{id}/detail
```

### 布局

#### 自动布局
```bash
POST /api/v1/topology/layout/auto
```

#### 保存布局
```bash
POST /api/v1/topology/layout/save
{
  "nodes": [
    {"id": "xxx", "x": 100, "y": 200}
  ]
}
```

### 导入导出

#### 导出拓扑
```bash
GET /api/v1/topology/export
```

#### 导入拓扑
```bash
POST /api/v1/topology/import
{
  "nodes": [...],
  "edges": [...]
}
```

## 成本分析插件 API

### 账号管理

#### 添加云账号
```bash
POST /api/v1/cost/accounts
{
  "name": "aliyun-prod",
  "provider": "aliyun",  # aliyun/tencent/aws/huawei
  "access_key": "xxx",
  "secret_key": "xxx",
  "region": "cn-hangzhou"
}
```

#### 同步费用
```bash
POST /api/v1/cost/accounts/{id}/sync
{
  "start_date": "2024-01-01",
  "end_date": "2024-01-31"
}
```

### 费用查询

#### 费用汇总
```bash
GET /api/v1/cost/summary?start_date=2024-01-01&end_date=2024-01-31&account_id=xxx
```

#### 费用趋势
```bash
GET /api/v1/cost/trend
```

#### TOP资源
```bash
GET /api/v1/cost/top-resources
```

#### 费用记录
```bash
GET /api/v1/cost/records?account_id=xxx&product_code=ecs
```

### 预算管理

#### 创建预算
```bash
POST /api/v1/cost/budgets
{
  "name": "月度预算",
  "account_id": "xxx",
  "product_code": "ecs",
  "budget_type": "monthly",
  "amount": 10000,
  "alert_at": 80,
  "start_date": "2024-01-01",
  "end_date": "2024-12-31"
}
```

#### 预算执行状态
```bash
GET /api/v1/cost/budgets/status
```

### 优化建议

#### 获取优化建议
```bash
GET /api/v1/cost/optimizations?status=0
```

#### 更新建议状态
```bash
PUT /api/v1/cost/optimizations/{id}
{
  "status": 1  # 0待处理 1已采纳 2已忽略
}
```

#### 分析优化建议
```bash
POST /api/v1/cost/optimizations/analyze
```

## 其他插件 API

### 工作流
```bash
# 创建工作流
POST /api/v1/workflow/workflows

# 执行工作流
POST /api/v1/workflow/workflows/{id}/execute

# 取消执行
POST /api/v1/workflow/executions/{id}/cancel
```

### 批量执行
```bash
# 批量执行命令
POST /api/v1/executor/execute

# 实时输出
GET /api/v1/executor/executions/{id}/stream
```

### 告警中心
```bash
# 告警列表
GET /api/v1/alert/alerts

# 确认告警
POST /api/v1/alert/alerts/{id}/ack

# 解决告警
POST /api/v1/alert/alerts/{id}/resolve

# 创建静默
POST /api/v1/alert/silences
```

### 值班排班
```bash
# 谁在值班
GET /api/v1/oncall/whoisoncall

# 创建排班
POST /api/v1/oncall/schedules

# 换班
POST /api/v1/oncall/shifts/{id}/swap
```

## Webhook

### CI/CD Webhook
```bash
POST /api/v1/cicd/webhook/{provider}
```

支持的provider:
- jenkins
- gitlab
- github
- argocd

### 工作流Webhook
```bash
POST /api/v1/workflow/webhook/{id}
```

## SSE实时推送

### Ansible执行输出
```bash
GET /api/v1/ansible/executions/{id}/stream
Accept: text/event-stream
```

### 批量执行输出
```bash
GET /api/v1/executor/executions/{id}/stream
Accept: text/event-stream
```

## 分页

支持分页的API使用以下参数：
- `page`: 页码（从1开始）
- `page_size`: 每页数量（默认20）

```bash
GET /api/v1/xxx?page=1&page_size=20
```

## 过滤和排序

支持过滤：
```bash
GET /api/v1/xxx?status=1&type=shell
```

支持排序：
```bash
GET /api/v1/xxx?sort=created_at&order=desc
```

## 监控插件 API

### Prometheus 查询

```bash
GET /api/v1/monitor/prometheus/query?query=up
GET /api/v1/monitor/prometheus/query_range?query=up&start=1700000000&end=1700003600&step=30
GET /api/v1/monitor/prometheus/history
POST /api/v1/monitor/prometheus/history
PUT /api/v1/monitor/prometheus/history/{id}
```

### Pushgateway 指标

```bash
GET /api/v1/monitor/pushgateway/metrics
```

### Agent 心跳

```bash
POST /api/v1/monitor/agents/heartbeat
Content-Type: application/json
X-Agent-Token: <agent_secret>

{
  "agent_id": "host-001",
  "hostname": "web-01",
  "ip": "192.168.1.10",
  "version": "1.0.0",
  "os": "linux",
  "labels": {"env": "prod"},
  "meta": {"cpu": 12.3, "memory": 65.2, "disk": 40.1, "net_in": 12345, "net_out": 4567}
}

GET /api/v1/monitor/agents
```

### Agent 详情

```bash
GET /api/v1/monitor/agents/{agent_id}
GET /api/v1/monitor/agents/{agent_id}/history?hours=24
```

### 告警聚合配置

```bash
GET /api/v1/alert/aggregations
POST /api/v1/alert/aggregations
PUT /api/v1/alert/aggregations/{id}
DELETE /api/v1/alert/aggregations/{id}
```

### 告警静默

```bash
GET /api/v1/alert/silences
POST /api/v1/alert/silences
PUT /api/v1/alert/silences/{id}
DELETE /api/v1/alert/silences/{id}
```

### 通知组测试

```bash
POST /api/v1/notify/groups/{id}/test
{
  "title": "测试标题",
  "content": "测试内容",
  "receiver": "可选"
}
```

### 告警复盘历史

```bash
GET /api/v1/alert/history
```

### 告警复盘详情

```bash
GET /api/v1/alert/history/{id}
PUT /api/v1/alert/history/{id}
```

### 告警复盘筛选

```bash
GET /api/v1/alert/history?severity=critical&start=2026-02-01&end=2026-02-04
```

### 告警复盘筛选（目标/规则）

```bash
GET /api/v1/alert/history?severity=critical&target=db&rule_id=cpu
```

### CMDB 分组

```bash
GET /api/v1/cmdb/groups
POST /api/v1/cmdb/groups
PUT /api/v1/cmdb/groups/{id}
DELETE /api/v1/cmdb/groups/{id}
```

### CMDB 凭据

```bash
GET /api/v1/cmdb/credentials
POST /api/v1/cmdb/credentials
PUT /api/v1/cmdb/credentials/{id}
DELETE /api/v1/cmdb/credentials/{id}
```

### 数据库资产

```bash
GET /api/v1/cmdb/databases
POST /api/v1/cmdb/databases
GET /api/v1/cmdb/databases/{id}
PUT /api/v1/cmdb/databases/{id}
DELETE /api/v1/cmdb/databases/{id}
```

### 云账号

```bash
GET /api/v1/cmdb/cloud/accounts
POST /api/v1/cmdb/cloud/accounts
GET /api/v1/cmdb/cloud/accounts/{id}
PUT /api/v1/cmdb/cloud/accounts/{id}
DELETE /api/v1/cmdb/cloud/accounts/{id}
```

### 云资源

```bash
GET /api/v1/cmdb/cloud/resources
POST /api/v1/cmdb/cloud/resources
GET /api/v1/cmdb/cloud/resources/{id}
PUT /api/v1/cmdb/cloud/resources/{id}
DELETE /api/v1/cmdb/cloud/resources/{id}
```

### CI/CD 发布记录

```bash
GET /api/v1/cicd/releases
POST /api/v1/cicd/releases
PUT /api/v1/cicd/releases/{id}
DELETE /api/v1/cicd/releases/{id}
```

### Nacos 同步计划

```bash
GET /api/v1/nacos/schedules
POST /api/v1/nacos/schedules
PUT /api/v1/nacos/schedules/{id}
DELETE /api/v1/nacos/schedules/{id}
POST /api/v1/nacos/schedules/{id}/toggle
```
