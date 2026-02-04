# Deviops 对齐矩阵（更新）

本表用于对齐 `zhang1024fan/deviops` 的功能点与当前 Lazy Auto Ops 的实现状态。
状态说明：
- ✅ 已有实现（后端 + 前端页面）
- ⚠️ 部分存在，需要补齐细节或增强能力
- ❌ 未发现实现（需要新增）

## 1. CMDB / 资产管理
| Deviops 功能点 | Lazy Auto Ops 对应插件/API | 前端页面 | 状态 |
| --- | --- | --- | --- |
| 主机资产管理 | `cmdb` | `/host` | ✅ |
| 主机分组 | `cmdb` | `/cmdb/group` | ✅ |
| 主机 SSH 连接 | `terminal` | `/terminal` | ✅ |
| 凭据管理（密码/SSH Key/API Key） | `cmdb` | `/cmdb/credential` | ✅ |
| 数据库资产 | `cmdb` | `/cmdb/database` | ✅ |
| 云资源管理 | `cmdb` | `/cmdb/cloud` | ✅ |
| SQL 操作审计（deviops 标注未开发） | `sqlaudit` | `/sqlaudit/*` | ✅ |

## 2. 配置中心
| Deviops 功能点 | Lazy Auto Ops 对应插件/API | 前端页面 | 状态 |
| --- | --- | --- | --- |
| SSH 密钥管理 / API 密钥管理 | `cmdb` | `/cmdb/credential` | ✅ |
| ECS 云主机授权（云账号） | `cmdb` | `/cmdb/cloud` | ✅ |
| 账户权限管理 | `rbac/system` | `/system/*` | ✅ |
| 配置同步 / 定时同步 | `nacos` | `/nacos/servers`, `/nacos/configs` | ✅ |

## 3. 任务中心
| Deviops 功能点 | Lazy Auto Ops 对应插件/API | 前端页面 | 状态 |
| --- | --- | --- | --- |
| Ansible 任务编排 | `ansible` | `/ansible/playbooks`, `/ansible/inventories` | ✅ |
| 任务模板 | `executor` | `/executor` | ✅ |
| 定时任务 | `task` | `/task/schedules` | ✅ |
| 执行监控 | `executor`/`task` | `/executor`, `/task/schedules` | ✅ |
| WebSocket/实时输出 | `executor`/`ansible` | `/executor`, `/ansible/playbooks` | ✅ |
| 任务队列 | `executor`/`task` | `/executor`, `/task/schedules` | ✅ |

## 4. K8s 管理
| Deviops 功能点 | Lazy Auto Ops 对应插件/API | 前端页面 | 状态 |
| --- | --- | --- | --- |
| 多集群管理 | `k8s` | `/k8s/clusters` | ✅ |
| Namespace 管理 | `k8s` | `/k8s/namespaces` | ✅ |
| Workload 管理 | `k8s` | `/k8s/workloads` | ✅ |
| Service/Ingress | `k8s` | `/k8s/services` | ✅ |
| ConfigMap/Secret | `k8s` | `/k8s/configs` | ✅ |
| PV/PVC/SC 存储 | `k8s` | `/k8s/storage` | ✅ |
| 节点管理 | `k8s` | `/k8s/nodes` | ✅ |
| 事件查看 | `k8s` | `/k8s/events` | ✅ |
| WebShell | `k8s`/`terminal` | `/k8s/terminal` | ✅ |

## 5. 应用管理 / CI/CD
| Deviops 功能点 | Lazy Auto Ops 对应插件/API | 前端页面 | 状态 |
| --- | --- | --- | --- |
| Jenkins/CI 集成 | `cicd` | `/cicd/pipelines` | ✅ |
| 构建历史 | `cicd` | `/cicd/executions` | ✅ |
| 发布管理 | `cicd` | `/cicd/releases` | ✅ |
| 定时发布 | `cicd` | `/cicd/schedules` | ✅ |

## 6. 监控告警
| Deviops 功能点 | Lazy Auto Ops 对应插件/API | 前端页面 | 状态 |
| --- | --- | --- | --- |
| Prometheus 指标采集 | `monitor` | `/monitor/metrics` | ✅ |
| Pushgateway | `monitor` | `/monitor/metrics` | ✅ |
| Agent 心跳 | `monitor` | `/monitor/agents` | ✅ |
| 告警规则 | `alert` | `/alert/rules` | ✅ |
| 告警事件 | `alert` | `/alert/events` | ✅ |
| 通知渠道/模板/组 | `notify` | `/notify/channels`, `/notify/templates`, `/notify/groups` | ✅ |
| 告警复盘 | `alert` | `/alert/history` | ✅ |

## 7. 系统管理
| Deviops 功能点 | Lazy Auto Ops 对应插件/API | 前端页面 | 状态 |
| --- | --- | --- | --- |
| 用户管理 | `rbac` | `/system/users` | ✅ |
| 角色管理 | `rbac` | `/system/roles` | ✅ |
| 菜单管理 | `system` | `/system/menus` | ✅ |
| 部门管理 | `system` | `/system/dept` | ✅ |
| 岗位管理 | `system` | `/system/posts` | ✅ |
| 登录日志 | `system` | `/system/login-logs` | ✅ |
| 操作日志 | `system` | `/system/audit-logs` | ✅ |
| 验证码 | `system` | `/system/captcha` | ✅ |

## 8. 运维看板
| Deviops 功能点 | Lazy Auto Ops 对应插件/API | 前端页面 | 状态 |
| --- | --- | --- | --- |
| 运维数据看板 | `dashboard` | `/dashboard` | ✅ |

## 9. 其他差异化功能（超出 deviops）
| 功能 | Lazy Auto Ops 插件 | 前端页面 | 状态 |
| --- | --- | --- | --- |
| AI 运维助手 | `ai` | `/ai` | ✅ |
| 服务拓扑 | `topology` | `/topology` | ✅ |
| 成本管理 | `cost` | `/cost/overview`, `/cost/budget` | ✅ |
| 变更管理 | `workorder`, `sqlaudit`, `gitops` | `/workorder/*`, `/sqlaudit/*`, `/gitops/*` | ✅ |
| 值班排班 | `oncall` | `/oncall/*` | ✅ |
