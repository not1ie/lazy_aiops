# Jump P0 验收清单

适用版本：包含 `jump` 插件的当前主干版本。  
目标：验证堡垒机 P0（SSH 闭环）能力可用。

## 1. 前置条件

- 服务健康：`GET /health` 返回 `{"status":"ok"}`
- `jump` 与 `rbac` 插件已加载
- 管理员账号可正常登录
- 至少存在 1 个可访问资产与 1 个可用账号

## 2. 功能验收

1. 资产接入
- 能创建、编辑、删除堡垒机资产
- 能从 CMDB/K8s/Docker 执行同步

2. 策略授权
- 能创建用户或角色策略
- 支持审批开关、授权时段、最大会话时长、并发上限

3. 会话发起与连接
- 已授权用户可发起会话
- 未授权用户发起会话返回 403
- 会话状态流转正确：`pending_approval -> active -> closed/rejected/blocked`

4. 命令风控
- 命中阻断规则时，会话被标记为 `blocked`
- 命中风险规则但未阻断时，会写审计和风险事件

5. 审计与管控
- `sessions/:id/commands` 可查询命令审计
- `risk-events` 可按会话筛选
- 管理员可执行强制断开 `sessions/:id/disconnect`

## 3. 自动化脚本验收

```bash
PASSWORD='你的密码' BASE_URL='http://127.0.0.1:8080' bash scripts/verify_jump_p0.sh
```

通过标准：

- 输出 `PASS: jump P0 verification completed`
- 脚本执行中无 `ERROR:` 输出

## 4. 回归关注点

- RBAC 权限与左侧菜单一致（无“有权限无入口”）
- 非管理员无法调用审批与强制断开接口
- 前端会话列表的审批人、风险事件展示正常
- 升级后历史会话与命令数据可继续查询
