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
