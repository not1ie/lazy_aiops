# AI 助手升级路线

## 目标

把当前的 AI 运维助手从“问答页面”升级成“可感知场景、可调用平台能力、可审计执行”的企业级运维 Copilot，并逐步向 Agent 体系演进。

## 对标与超越方向

- 对标 OpenOcta 的能力层：Agent 平台化、工具调用、MCP 接入、轨迹与记忆。
- 超越 OpenOcta 的抓手：直接利用 Lazy Auto Ops 已有的 CMDB、K8s、监控、工单、工作流与审批能力，形成真正的运维闭环。

## 当前阶段

### 阶段 1：场景感知 Copilot

- 自动感知最近浏览的业务页面
- 自动拼接资产、K8s、监控、交付四大场景的上下文包
- 输出更偏运维诊断的结构化回答

已落地：

- 最近业务页面上下文持久化
- `/api/v1/ai/context-pack` 自动上下文预览接口
- AI 聊天请求自动携带上下文包
- AI 页面展示当前附着的上下文摘要

### 阶段 2：工具调用层

建议新增统一 Tool Registry：

- `get_host_summary`
- `get_host_detail`
- `get_k8s_cluster_summary`
- `get_pod_logs`
- `get_pod_events`
- `get_open_alerts`
- `get_recent_workorders`
- `get_workflow_status`
- `get_release_risk`

分级：

- `read`
- `propose`
- `execute_requires_approval`

要求：

- 每次工具调用必须审计
- 每次回答必须能说明使用了哪些工具与证据

### 阶段 3：AI + Workflow 闭环

- AI 生成 runbook 草案
- 自动映射到现有工作流节点：`shell/http/ai/approval/notify`
- 高风险步骤进入审批
- 执行结果回写 AI 会话与审计日志

### 阶段 4：组织级记忆

- 资产级记忆：主机/服务常见问题、负责人、限制条件
- 事件级记忆：历史故障、处理动作、恢复时长
- 组织级记忆：审批规则、发布规范、应急 SOP
- 会话级记忆：当前问题调查链路

## 推荐优先级

1. 工具调用层
2. 可解释计划输出
3. 审批型执行
4. 组织级记忆

## 成功标准

- 用户不必重复描述环境
- AI 能自动拉取当前场景摘要
- AI 能先给只读排查计划，再给变更建议
- 高风险动作有审批、有审计、有回滚说明
- 同类故障能复用历史经验
