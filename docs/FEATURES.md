# Lazy Auto Ops 功能详解

## 项目概述

Lazy Auto Ops 是一个AI驱动的轻量级运维自动化平台，采用Go语言开发，插件化架构设计。

**核心特点：**
- 单二进制部署，SQLite存储
- 21个可插拔功能模块
- AI原生集成
- 低代码工作流编排

## 完整插件列表 (21个)

### 1. AI能力 (2个)

#### ai - AI运维助手
- 日志分析
- 故障诊断
- 智能问答
- 支持OpenAI/Ollama

#### alert - 智能告警中心
- AI智能降噪
- 告警聚合（按规则+目标）
- 静默规则
- 告警统计

### 2. 资产管理 (3个)

#### cmdb - 资产管理
- 主机管理
- 凭据管理
- 主机分组

#### k8s - Kubernetes管理
- 多集群管理
- 节点管理
- Pod/Deployment/Service管理
- ConfigMap/Secret管理

#### firewall - 防火墙管理
- SNMP监控
- 规则管理
- 指标采集

### 3. 监控告警 (3个)

#### monitor - 监控中心
- 域名监控
- 主机监控
- 告警规则

#### domain - 域名/SSL管理
- 云域名到期监控
- SSL证书到期监控
- 支持阿里云/腾讯云

#### notify - 通知中心
- 飞书/钉钉/企微/邮件/Webhook
- 通知模板
- 通知分组

### 4. 自动化 (4个)

#### workflow - 运维编排
- 可视化工作流
- 支持节点：shell/http/condition/parallel/wait/notify/ai/approval
- 定时触发/手动触发/Webhook触发
- 执行历史

#### executor - 批量执行
- SSH并发执行
- 实时输出（SSE）
- 执行模板
- 取消执行

#### task - 任务调度
- 定时任务（cron）
- Shell/Python脚本
- Ansible Playbook

#### ansible - Ansible管理 [NEW]
- Playbook在线编辑
- Inventory管理
- 从CMDB同步主机
- Role安装（Galaxy）
- 语法校验
- 实时输出

### 5. CI/CD (1个)

#### cicd - CI/CD集成 [NEW]
- Jenkins集成
- GitLab CI集成
- ArgoCD集成
- GitHub Actions集成
- 定时发布（cron）
- Webhook触发
- 构建状态轮询

### 6. 配置中心 (1个)

#### nacos - Nacos配置中心 [NEW]
- 配置同步
- 配置更新（推送到Nacos）
- 配置历史
- 配置回滚
- 配置对比
- 服务发现
- 命名空间管理

### 7. 变更管理 (3个)

#### workorder - 运维工单
- 工单类型管理
- 审批流程
- AI辅助处理
- 工单统计

#### sqlaudit - SQL审计
- SQL工单
- 审核规则
- 执行审计

#### gitops - GitOps
- Git仓库同步
- 配置变更追踪
- 配置管理

### 8. 协作 (2个)

#### oncall - 值班排班
- 轮换排班
- 换班
- 升级策略
- 值班查询

#### terminal - WebTerminal
- WebSocket SSH
- 会话管理
- 操作录像

### 9. 可视化 (1个)

#### topology - 服务拓扑 [NEW]
- 服务节点管理
- 依赖关系管理
- 依赖分析
- 影响评分
- 自动布局
- 导入导出
- 从K8s同步

### 10. 成本 (1个)

#### cost - 成本分析 [NEW]
- 云账号管理（阿里云/腾讯云/AWS/华为云）
- 费用记录同步
- 费用汇总
- 趋势分析
- 预算管理
- 超支告警
- 优化建议

## 核心差异化功能

### 1. AI深度融合
- 告警智能降噪（相似告警聚合）
- 故障诊断
- 工单AI分析
- 工作流AI节点

### 2. 低代码编排
- 可视化工作流设计
- 支持9种节点类型
- 模板变量
- 条件分支
- 并行执行

### 3. CI/CD原生集成
- 4种CI/CD工具支持
- 定时发布
- 参数化构建
- 状态轮询

### 4. 配置中心对接
- Nacos配置管理
- 版本对比
- 一键回滚

### 5. 成本可视化
- 多云费用统计
- 预算管理
- 优化建议

### 6. 服务拓扑
- 可视化依赖关系
- 影响分析
- 关键路径识别

## 技术架构

### 后端
- 语言：Go 1.21+
- 框架：Gin
- 数据库：SQLite（可切换MySQL/PostgreSQL）
- 认证：JWT

### 插件架构
```go
type Plugin interface {
    Name() string
    Version() string
    Description() string
    Init(core *core.Core, cfg map[string]interface{}) error
    Start() error
    Stop() error
    Migrate() error
    RegisterRoutes(r *gin.RouterGroup)
}
```

### 部署方式
1. 单二进制
2. Docker
3. Kubernetes

## API设计

所有API遵循统一格式：
```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

认证方式：
```
Authorization: Bearer <token>
```

## 配置文件

`configs/config.yaml`:
```yaml
server:
  port: "8080"
  mode: "debug"

database:
  driver: "sqlite"
  dsn: "data/lazy-auto-ops.db"

jwt:
  secret: "your-secret"
  expire: 24

plugins:
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
  # ... 其他插件
```

## 使用场景

### 场景1：自动化发布
1. 在cicd插件创建Jenkins流水线
2. 配置定时发布（每天凌晨2点）
3. 发布失败自动告警
4. AI分析失败原因

### 场景2：配置变更
1. 在nacos插件修改配置
2. 自动推送到Nacos
3. 记录变更历史
4. 支持一键回滚

### 场景3：成本优化
1. 同步云账单
2. 分析费用趋势
3. 识别闲置资源
4. 生成优化建议

### 场景4：故障处理
1. 告警触发
2. AI智能降噪
3. 自动执行诊断工作流
4. 通知值班人员
5. 创建工单跟踪

## 后续规划

### 短期
- [ ] 前端界面（Vue3 + Element Plus）
- [ ] 更多云厂商API对接
- [ ] Prometheus集成
- [ ] 审计日志增强

### 中期
- [ ] 多租户支持
- [ ] RBAC权限细化
- [ ] 插件市场
- [ ] 移动端适配

### 长期
- [ ] AIOps能力增强
- [ ] 自动化运维决策
- [ ] 智能容量规划
- [ ] 混沌工程集成
