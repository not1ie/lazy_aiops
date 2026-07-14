# Lazy Auto Ops AI Developer Handoff

更新时间：2026-06-03  
仓库：`/Users/hayakawaaki/workspace/lazy_aiops`  
远端：`github.com:not1ie/lazy_aiops.git`

## 1. 接手前必须先读

这是一个插件化运维平台，不是单纯的 Vue 页面项目。后续 AI 接手时，先按下面顺序读：

1. 本文档：确认产品方向、当前状态、风险点。
2. `README.md`：了解整体功能、部署方式和默认账号策略。
3. `docs/DEVIOPS_ALIGNMENT.md`：了解与 AutoOps / DeviOps 类项目的差异化方向。
4. `docs/jumpserver-fusion-plan.md`：了解堡垒机融合预期。
5. `docs/regression-checklist.md`：改完 UI 或核心接口后按清单回归。
6. `git status --short --branch`：当前工作区存在大量未提交改动，不能直接回滚。

## 2. 产品目标

Lazy Auto Ops 的目标是做一个面向中小企业和多客户生产环境的统一运维平台，核心价值不是“页面多”，而是把资产、访问、监控、告警、交付、容器、堡垒机、AI 运维能力收在同一个工作流里。

用户明确表达过的产品要求：

1. 参考 `opsre/AutoOps`、`openocta/openocta` 等项目，但不要复制，要在功能完整度、状态真实性和 AI 辅助上超越。
2. UI 要保持一种统一风格，不要一会儿 Google、一会儿 Apple、一会儿传统后台。
3. 页面要面向小白也能理解，模块不要过度拆散。
4. 高度耦合的模块要融合，例如网络设备与防火墙、用户与角色权限、告警事件与规则静默聚合。
5. 状态必须真实，不能机器和 Docker 已离线还显示在线。
6. 堡垒机不是装饰页面，要能对接 JumpServer，支持资产同步、授权、会话和命令风控。
7. AI 助手要从问答工具升级为可执行运维技能引擎，能辅助排障、生成 Runbook、做风险预检。

## 3. 稳定发布点与线上状态

当前 `main` 最新已发布稳定点：

- Commit：`b538a28`
- Tag：`v1.0.28`
- 镜像：`crpi-iihofxt94xlrdrvd.cn-shanghai.personal.cr.aliyuncs.com/lazyops/lazyops:v1.0.28`
- Digest：`sha256:9f844b6efbeefa2554794eaf8b8d0af8a6cff90092f33e0f993f5952f3912080`
- 服务器：`192.168.10.101`
- Swarm 服务：`lazy-aiops_lazy-auto-ops`
- 健康检查：`http://127.0.0.1:8080/health`

注意：当前本地工作区已经超过 `v1.0.28`，存在大量未提交 WIP。不要把当前工作区等同于线上版本。

## 4. 当前工作区状态

截至本文档生成时，`main` 分支与 `origin/main` 对齐在 `v1.0.28`，但本地存在大量未提交变更。

核心修改文件：

- `Dockerfile`
- `cmd/server/main.go`
- `configs/config.yaml`
- `deploy/swarm/stack.yml`
- `frontend/src/layout/index.vue`
- `frontend/src/main.js`
- `frontend/src/router/index.js`
- `frontend/src/style.css`
- `frontend/src/views/dashboard/index.vue`
- `frontend/src/views/hub/asset.vue`
- `frontend/src/views/hub/delivery.vue`
- `frontend/src/views/hub/k8s.vue`
- `frontend/src/views/hub/monitor.vue`
- `frontend/src/views/hub/system.vue`
- `internal/api/middleware.go`
- `internal/api/server.go`
- `plugins/ai/*`
- `plugins/cicd/*`
- `plugins/cmdb/handler.go`

删除或待确认删除：

- `frontend/src/views/hub/asset-ops.vue`
- `frontend/src/views/hub/collab.vue`
- `frontend/src/views/hub/domain-center.vue`

新增或未跟踪：

- `Dockerfile.binary`
- `Dockerfile.simple`
- `deploy/supervisord.conf`
- `frontend/src/views/hub/ai-skills.vue`
- `frontend/src/views/hub/hub-common.css`
- `frontend/src/views/hub/registries.vue`
- `plugins/ai/skill_engine.go`
- `plugins/ai/skill_handler.go`
- `plugins/cicd/registry_client.go`
- `plugins/cicd/registry_handler.go`
- `plugins/sre/`
- `scripts/check_skills.go`
- `scripts/dump_skills.go`
- `scripts/init_skills.go`
- `scripts/run_check.sh`
- `scripts/test_api.sh`
- `vendor/`
- 若干本地二进制：`app_server_host`、`app_server_linux`、`check_skills_linux`

处理原则：

1. 不要执行 `git reset --hard`。
2. 不要删除未跟踪文件，除非用户明确确认这些是构建垃圾。
3. 后续 AI 应先用 `git diff --stat` 和 `git diff -- <file>` 审核每类改动，再决定是否拆 commit。
4. 当前 WIP 里可能包含另一次大重构：前端路由被大幅收敛，部分旧 hub 页被删除，SRE sidecar 插件被加入。

## 5. 当前验证结果

前端构建通过：

```bash
cd frontend
npm run build
```

后端核心包测试通过：

```bash
go test ./cmd/... ./internal/... ./plugins/... ./pkg/...
```

全量测试当前失败：

```bash
go test ./...
```

失败原因是 `scripts/` 目录下存在多个 `package main` 文件，作为同一个包测试时出现 `main redeclared`。这不是核心服务或插件测试失败。建议后续处理方式：

1. 把 `scripts/*.go` 移到各自子目录。
2. 或给脚本文件加 build tag。
3. 或 CI 改为测试 `./cmd/... ./internal/... ./plugins/... ./pkg/...`。

## 6. 技术架构

后端：

- Go 1.21
- Gin
- Gorm
- SQLite 默认存储
- Viper 配置
- JWT 鉴权
- 插件化模块注册

前端：

- Vue 3
- Vite
- Element Plus
- Vue Router
- Pinia
- Axios
- ECharts
- xterm

部署：

- Docker
- Docker Swarm
- Kubernetes manifests
- systemd / supervisord 辅助文件

默认端口：

- 应用：`8080`
- SQLite：`data/lazy-auto-ops.db`

默认账号策略：

- 默认用户：`admin`
- 首次初始化时可设置 `LAO_ALLOW_INSECURE_BOOTSTRAP=true` 使用 `admin123`
- 生产更推荐 `LAO_BOOTSTRAP_ADMIN_PASSWORD=<强密码>`
- 若数据目录已有数据库，不会覆盖现有密码

## 7. 后端插件机制

插件接口在 `pkg/plugin/plugin.go`。

每个插件需要实现：

- `Name()`
- `Version()`
- `Description()`
- `Init(core, config)`
- `Start()`
- `Stop()`
- `RegisterRoutes(group)`
- `Migrate()`

插件通过 `init()` 注册：

```go
plugin.Register("cmdb", func() plugin.Plugin {
    return &Plugin{}
})
```

服务启动入口是 `cmd/server/main.go`：

- 空白导入所有插件。
- `config.Load()` 读取配置。
- `core.New(cfg)` 初始化 DB、Auth、AI 等核心对象。
- `plugin.GetManager().LoadEnabledPlugins(cfg.Plugins)` 根据 `configs/config.yaml` 启用插件。
- `internal/api/server.go` 注册公共路由、鉴权路由和插件路由。

插件路由挂载方式：

```text
/api/v1/<plugin-name>/*
```

例如：

- `/api/v1/cmdb/*`
- `/api/v1/k8s/*`
- `/api/v1/ai/*`
- `/api/v1/sre/*`

## 8. 前端导航约束

当前产品方向是 5 个主中心，而不是几十个左侧菜单：

1. `资产与安全中心`：`/asset`
2. `容器编排平台`：`/k8s`
3. `统一观测中心`：`/monitor`
4. `交付与生命周期`：`/delivery`
5. `系统治理中心`：`/system`

旧功能页可以作为隐藏直达页或中心内 tab，不要重新铺回主侧边栏。

当前路由文件：

- `frontend/src/router/index.js`

当前布局文件：

- `frontend/src/layout/index.vue`

当前全局样式：

- `frontend/src/style.css`

Hub 公共样式：

- `frontend/src/views/hub/hub-common.css`

后续 UI 开发原则：

1. 主中心页面负责聚合业务，不要只做跳转目录。
2. 原子功能页保留，但优先挂在中心内部。
3. 对用户高频场景建立一屏闭环，例如“告警发现 -> 通知 -> 处置 -> 复盘”。
4. 表格操作列必须避免按钮堆叠，优先使用主要按钮 + 更多下拉。
5. 状态字段要展示来源、最后检测时间、降级原因，不能只显示绿色 `在线`。
6. 不要再引入第二套视觉风格。

## 9. 已完成的重要方向

最近已完成或已开始的方向：

1. 资产模块融合：主机、凭据、数据库、云资源、网络与防火墙、堡垒机资产收口。
2. 网络设备与防火墙融合：防火墙作为网络设备类型，不再独立成重复中心。
3. 主机管理增强：资产分组、云厂商、CPU/内存/磁盘、最后检测、状态说明、进程/TCP/监控入口。
4. 顶部模块 tag 去重：保留业务内页签，减少重复导航。
5. 监控、交付、容器、系统中心开始按业务分区融合。
6. JumpServer 接入：已支持连接配置、测试连接、同步入口，但权限不足时需要清晰提示。
7. AI SRE sidecar 初步接入：新增 `plugins/sre`，配置默认指向 `http://lazysre:8000`。
8. CI/CD 镜像仓库能力开始扩展：新增 registry 相关文件。

## 10. 线上部署流程

镜像仓库：

```bash
REGISTRY=crpi-iihofxt94xlrdrvd.cn-shanghai.personal.cr.aliyuncs.com
IMAGE=$REGISTRY/lazyops/lazyops
```

常用版本规则：

- Git tag 使用 `v1.0.xx`
- 镜像 tag 同步使用 `v1.0.xx`
- 同时可推短 commit tag，例如 `b538a28`

服务器手动更新：

```bash
IMAGE=crpi-iihofxt94xlrdrvd.cn-shanghai.personal.cr.aliyuncs.com/lazyops/lazyops
VERSION=v1.0.28

docker pull ${IMAGE}:${VERSION}
docker service update --with-registry-auth --force --image ${IMAGE}:${VERSION} lazy-aiops_lazy-auto-ops
docker service ps lazy-aiops_lazy-auto-ops --no-trunc
curl -sS --max-time 15 http://127.0.0.1:8080/health
```

Swarm 单节点注意事项：

- 服务端口是 host mode `8080`。
- 滚动更新时可能短暂出现 `host-mode port already in use on 1 node`。
- 如果最终 `Service converged` 且健康检查 `{"status":"ok"}`，通常是可接受的。
- `deploy/swarm/stack.yml` 当前包含 `lazy-auto-ops` 和 `lazysre` 两个服务。

建议发布前检查：

```bash
npm run build
go test ./cmd/... ./internal/... ./plugins/... ./pkg/...
git diff --stat
```

## 11. 重点已知问题

优先级 P0：

1. 当前工作区 WIP 很大，需要拆分 commit；不要一次性提交所有未跟踪文件。
2. 路由被大幅收敛后，很多旧路径可能丢失直达能力，需要逐个回归。
3. `go test ./...` 被 `scripts/` 多个 main 文件挡住，需要修。
4. `vendor/` 是否应该入库需要确认；如果不是刻意 vendor 化，不要提交。
5. 本地二进制文件不应提交：`app_server_host`、`app_server_linux`、`check_skills_linux`。

优先级 P1：

1. 主机 CPU/内存/磁盘为 `0` 时，需要区分“未采集”“采集失败”“真实 0”。
2. 主机连接信息需要支持页内编辑，不要跳转。
3. 主机进程、TCP、监控弹窗要和连接凭据、状态检测联动。
4. Docker 环境离线检测要定时刷新，不要只看数据库状态。
5. JumpServer 同步失败时，需要给出权限缺失、接口路径、组织 ID、认证方式等诊断信息。
6. 监控中心需要把事件、规则、静默、聚合、复盘合成真正的告警运营台。
7. 系统管理需要把用户、角色、权限、部门、岗位合成一套身份授权工作流。

优先级 P2：

1. 仪表盘还需要增强：状态完整度、异常排行、近期变更、风险趋势、AI 建议。
2. 交付中心需要把流水线、发布、工单、GitOps、SQL 审核形成变更闭环。
3. 容器中心需要把 Deployments、Pods、Service、Events、WebShell 形成排障工作台。
4. AI 助手需要对接更多平台内工具，而不是只做聊天回复。

## 12. 后续 AI 推荐工作顺序

第一阶段：稳定当前 WIP

1. 运行 `git diff --stat`，按模块拆分改动。
2. 先处理明显不该提交的本地二进制和临时文件。
3. 验证前端 5 个主中心是否都能打开。
4. 验证旧路径是否需要兼容，例如 `/host?tab=host`、`/monitor/center`、`/delivery/center`。
5. 修复 `scripts/` 导致的 `go test ./...` 失败。

第二阶段：功能真实性

1. 主机状态：真实检测 SSH、Ping、Agent、Docker、K8s 来源。
2. Docker 状态：定时检测 Docker daemon、容器数、镜像数、版本。
3. JumpServer：补全资产同步权限诊断、分页、组织过滤、失败详情。
4. 监控：告警状态与待处置积压从真实数据计算。

第三阶段：内容融合

1. 资产与安全中心：把主机、凭据、网络、防火墙、堡垒机授权真正串起来。
2. 统一观测中心：把告警运营、通知治理、域名证书、监控指标串起来。
3. 交付生命周期：把编排、流水线、发布、工单、SQL 审核串起来。
4. 系统治理中心：把身份、组织、权限、审计串起来。

第四阶段：AI 运维能力

1. 建立 Skill 模型、Skill 执行、工具绑定、运行记录。
2. 对接 `lazysre` sidecar 的健康检查、执行接口、Runbook、SLO。
3. 把 AI 建议落到具体页面操作，例如巡检、诊断、重启、扩缩容、生成工单。
4. 所有 AI 动作必须有审计、预检和确认机制。

## 13. 关键文件地图

后端入口：

- `cmd/server/main.go`
- `internal/api/server.go`
- `internal/api/middleware.go`
- `internal/core/*`
- `pkg/plugin/plugin.go`

配置：

- `configs/config.yaml`
- `deploy/swarm/stack.yml`
- `deploy/docker/*`
- `deploy/k8s/*`

前端入口：

- `frontend/src/main.js`
- `frontend/src/router/index.js`
- `frontend/src/layout/index.vue`
- `frontend/src/style.css`

前端 Hub：

- `frontend/src/views/hub/asset.vue`
- `frontend/src/views/hub/k8s.vue`
- `frontend/src/views/hub/monitor.vue`
- `frontend/src/views/hub/delivery.vue`
- `frontend/src/views/hub/system.vue`
- `frontend/src/views/hub/hub-common.css`

资产与堡垒机：

- `frontend/src/views/cmdb/host.vue`
- `frontend/src/views/cmdb/host-main.vue`
- `frontend/src/views/cmdb/network-device.vue`
- `plugins/cmdb/handler.go`
- `plugins/jump/*`

AI 与 SRE：

- `plugins/ai/*`
- `plugins/sre/*`
- `frontend/src/views/hub/ai-skills.vue`

CI/CD 与镜像仓库：

- `plugins/cicd/*`
- `frontend/src/views/hub/registries.vue`

## 14. 开发注意事项

1. 不要只做前端假状态。涉及在线、离线、风险、健康度的地方，要明确数据来源和最后检测时间。
2. 不要把高度耦合功能拆成多个并列一级模块。
3. 不要为了看起来完整加空页面；功能不完整时要展示“未配置/未采集/权限不足/连接失败”的真实状态。
4. 操作型页面优先保证密度、对齐、可扫描和低跳转成本。
5. 任何删除旧路由或旧页面的操作，都要确认是否有兼容跳转。
6. 发布镜像前必须跑前端构建和后端核心包测试。
7. 当前项目没有完整自动化回归，改 UI 后需要手动访问主中心页面。

## 15. 给下一个 AI 的第一条指令建议

建议用户把下一次任务开头写成：

```text
先读取 docs/AI_DEVELOPER_HANDOFF.md 和当前 git status。
不要回滚未提交改动。
请先拆分当前 WIP，确认哪些文件应该提交、哪些是构建产物，然后修复 go test ./... 的 scripts 冲突，并回归 5 个主中心页面。
```

