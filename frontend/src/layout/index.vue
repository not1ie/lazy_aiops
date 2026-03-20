<template>
  <el-container class="layout-container">
    <el-aside width="248px" class="aside">
      <div class="logo">
        <div>
          <div class="logo-title">Lazy Auto Ops</div>
          <div class="logo-subtitle">Ops Control Center</div>
        </div>
      </div>

      <el-scrollbar class="sider-scroll">
        <el-menu
          router
          :default-active="$route.path"
          background-color="transparent"
          text-color="var(--sider-text)"
          active-text-color="var(--sider-active)"
          :class="['el-menu-vertical', { 'is-simple': menuSimpleMode }]"
        >
          <template v-if="!menuSimpleMode">
          <el-menu-item v-if="can('dashboard')" index="/dashboard">
            <el-icon><Odometer /></el-icon>
            <span>仪表盘</span>
          </el-menu-item>

          <el-menu-item v-if="can('ai')" index="/ai">
            <el-icon><MagicStick /></el-icon>
            <span>AI运维助手</span>
          </el-menu-item>

          <el-sub-menu v-if="canAny(['cmdb','firewall','jump','jump:asset','jump:policy','jump:rule','jump:session','terminal'])" index="/cmdb">
            <template #title>
              <el-icon><Monitor /></el-icon>
              <span>资产管理</span>
            </template>
            <el-menu-item v-if="can('cmdb')" index="/asset/overview">资产总览</el-menu-item>
            <el-menu-item v-if="canAny(['cmdb','firewall','jump'])" index="/asset/ops">资产作战台</el-menu-item>
            <el-menu-item v-if="can('cmdb')" index="/host">主机管理</el-menu-item>
            <el-menu-item v-if="can('cmdb')" index="/cmdb/group">主机分组</el-menu-item>
            <el-menu-item v-if="can('cmdb')" index="/cmdb/credential">凭据管理</el-menu-item>
            <el-menu-item v-if="can('cmdb')" index="/cmdb/database">数据库资产</el-menu-item>
            <el-menu-item v-if="can('cmdb')" index="/cmdb/cloud">云资源</el-menu-item>
            <el-menu-item v-if="can('cmdb')" index="/cmdb/network-devices">网络设备</el-menu-item>
            <el-menu-item v-if="canAny(['cmdb','firewall'])" index="/firewall">防火墙管理</el-menu-item>
            <el-menu-item v-if="can('terminal')" index="/terminal">WebTerminal</el-menu-item>
            <el-menu-item v-if="can('jump:asset')" index="/jump/assets">堡垒机资产</el-menu-item>
            <el-menu-item v-if="can('jump:policy')" index="/jump/policies">授权策略</el-menu-item>
            <el-menu-item v-if="can('jump:rule')" index="/jump/command-rules">命令风控</el-menu-item>
            <el-menu-item v-if="can('jump:session')" index="/jump/sessions">会话审计</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAny(['docker','k8s'])" index="/k8s">
            <template #title>
              <el-icon><Platform /></el-icon>
              <span>容器与K8s</span>
            </template>
            <el-menu-item v-if="can('k8s')" index="/k8s/overview">平台总览</el-menu-item>
            <el-menu-item v-if="can('docker')" index="/docker">Docker管理</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/clusters">K8s集群</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/namespaces">命名空间</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/workloads">工作负载</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/deployments">Deployments</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/pods">Pods</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/services">服务与Ingress</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/configs">Config/Secret</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/storage">存储管理</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/nodes">节点管理</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/events">事件与诊断</el-menu-item>
            <el-menu-item v-if="can('k8s')" index="/k8s/terminal">K8s WebShell</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAny(['monitor','alert','notify','domain'])" index="/monitor">
            <template #title>
              <el-icon><Histogram /></el-icon>
              <span>监控告警</span>
            </template>
            <el-menu-item index="/monitor/center">监控告警中心</el-menu-item>
            <el-menu-item v-if="can('domain')" index="/domain/center">域名监控中心</el-menu-item>
            <el-menu-item v-if="can('monitor')" index="/monitor/overview">监控概览</el-menu-item>
            <el-menu-item v-if="can('monitor')" index="/monitor/hosts">主机监控</el-menu-item>
            <el-menu-item v-if="can('monitor')" index="/monitor/metrics">指标采集</el-menu-item>
            <el-menu-item v-if="can('monitor')" index="/monitor/containers">容器监控</el-menu-item>
            <el-menu-item v-if="can('monitor')" index="/monitor/pods">Pod监控</el-menu-item>
            <el-menu-item v-if="can('monitor')" index="/monitor/agents">Agent心跳</el-menu-item>
            <el-menu-item v-if="can('alert')" index="/alert/rules">告警规则</el-menu-item>
            <el-menu-item v-if="can('alert')" index="/alert/events">告警事件</el-menu-item>
            <el-menu-item v-if="can('alert')" index="/alert/silences">告警静默</el-menu-item>
            <el-menu-item v-if="can('alert')" index="/alert/aggregation">告警聚合</el-menu-item>
            <el-menu-item v-if="can('alert')" index="/alert/history">告警复盘</el-menu-item>
            <el-menu-item v-if="can('notify')" index="/notify/channels">通知渠道</el-menu-item>
            <el-menu-item v-if="can('notify')" index="/notify/groups">通知组</el-menu-item>
            <el-menu-item v-if="can('notify')" index="/notify/templates">通知模板</el-menu-item>
            <el-menu-item v-if="can('domain')" index="/domain/ssl">域名与证书</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAny(['workflow','executor','task','ansible','oncall'])" index="/automation">
            <template #title>
              <el-icon><Operation /></el-icon>
              <span>自动化</span>
            </template>
            <el-menu-item v-if="can('workflow')" index="/workflow/designer">工作流编排</el-menu-item>
            <el-menu-item v-if="can('executor')" index="/executor">批量执行</el-menu-item>
            <el-menu-item v-if="can('task')" index="/task/schedules">任务调度</el-menu-item>
            <el-menu-item v-if="can('ansible')" index="/ansible/playbooks">Ansible Playbook</el-menu-item>
            <el-menu-item v-if="can('ansible')" index="/ansible/inventories">Ansible Inventory</el-menu-item>
            <el-menu-item v-if="can('oncall')" index="/oncall/schedule">值班排班</el-menu-item>
            <el-menu-item v-if="can('oncall')" index="/oncall/escalation">升级策略</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAny(['cicd','application','workorder'])" index="/cicd">
            <template #title>
              <el-icon><Connection /></el-icon>
              <span>CI/CD</span>
            </template>
            <el-menu-item v-if="canAny(['cicd','workorder'])" index="/delivery/center">交付中心</el-menu-item>
            <el-menu-item v-if="can('cicd')" index="/cicd/pipelines">流水线管理</el-menu-item>
            <el-menu-item v-if="can('cicd')" index="/cicd/executions">执行记录</el-menu-item>
            <el-menu-item v-if="can('cicd')" index="/cicd/schedules">定时发布</el-menu-item>
            <el-menu-item v-if="can('cicd')" index="/cicd/releases">发布管理</el-menu-item>
            <el-menu-item v-if="can('application')" index="/application">应用中心</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="can('nacos')" index="/nacos">
            <template #title>
              <el-icon><Setting /></el-icon>
              <span>配置中心</span>
            </template>
            <el-menu-item index="/nacos/servers">Nacos服务器</el-menu-item>
            <el-menu-item index="/nacos/configs">配置管理</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAny(['workorder','sqlaudit','gitops'])" index="/change">
            <template #title>
              <el-icon><Tickets /></el-icon>
              <span>变更管理</span>
            </template>
            <el-menu-item v-if="can('workorder')" index="/workorder/tickets">工单管理</el-menu-item>
            <el-menu-item v-if="can('workorder')" index="/workorder/types">工单类型</el-menu-item>
            <el-menu-item v-if="can('sqlaudit')" index="/sqlaudit/requests">SQL工单</el-menu-item>
            <el-menu-item v-if="can('sqlaudit')" index="/sqlaudit/rules">SQL审核规则</el-menu-item>
            <el-menu-item v-if="can('gitops')" index="/gitops/repos">GitOps仓库</el-menu-item>
            <el-menu-item v-if="can('gitops')" index="/gitops/sync">同步记录</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="can('topology')" index="/visual">
            <template #title>
              <el-icon><Share /></el-icon>
              <span>可视化</span>
            </template>
            <el-menu-item index="/topology">服务拓扑</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="can('cost')" index="/cost">
            <template #title>
              <el-icon><Coin /></el-icon>
              <span>成本</span>
            </template>
            <el-menu-item index="/cost/overview">成本概览</el-menu-item>
            <el-menu-item index="/cost/budget">预算与告警</el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="canAny(['system','system:user','system:role','system:permission','system:dept','system:post','system:loginlog','system:captcha','system:log'])" index="/system">
            <template #title>
              <el-icon><Setting /></el-icon>
              <span>系统管理</span>
            </template>
            <el-menu-item v-if="can('system:user')" index="/system/users">用户管理</el-menu-item>
            <el-menu-item v-if="can('system:role')" index="/system/roles">角色管理</el-menu-item>
            <el-menu-item v-if="can('system:permission')" index="/system/menus">权限管理</el-menu-item>
            <el-menu-item v-if="can('system:dept')" index="/system/dept">部门管理</el-menu-item>
            <el-menu-item v-if="can('system:post')" index="/system/posts">岗位管理</el-menu-item>
            <el-menu-item v-if="can('system:loginlog')" index="/system/login-logs">登录日志</el-menu-item>
            <el-menu-item v-if="can('system:log')" index="/system/audit-logs">操作日志</el-menu-item>
            <el-menu-item v-if="can('system:captcha')" index="/system/captcha">验证码配置</el-menu-item>
          </el-sub-menu>
          </template>

          <template v-else>
            <el-menu-item v-if="can('dashboard')" index="/dashboard">
              <el-icon><Odometer /></el-icon>
              <span>仪表盘</span>
            </el-menu-item>

            <el-menu-item v-if="can('ai')" index="/ai">
              <el-icon><MagicStick /></el-icon>
              <span>AI运维助手</span>
            </el-menu-item>

            <el-sub-menu v-if="canAny(['cmdb','firewall','jump','jump:session','terminal'])" index="/cmdb">
              <template #title>
                <el-icon><Monitor /></el-icon>
                <span>资产管理</span>
              </template>
              <el-menu-item v-if="canAny(['cmdb','firewall','jump'])" index="/asset/ops">资产作战台</el-menu-item>
              <el-menu-item v-if="can('cmdb')" index="/asset/overview">资产总览</el-menu-item>
              <el-menu-item v-if="can('terminal')" index="/terminal">WebTerminal</el-menu-item>
            </el-sub-menu>

            <el-sub-menu v-if="canAny(['docker','k8s'])" index="/k8s">
              <template #title>
                <el-icon><Platform /></el-icon>
                <span>容器与K8s</span>
              </template>
              <el-menu-item v-if="can('k8s')" index="/k8s/overview">平台总览</el-menu-item>
              <el-menu-item v-if="can('k8s')" index="/k8s/deployments">Deployments</el-menu-item>
              <el-menu-item v-if="can('k8s')" index="/k8s/pods">Pods</el-menu-item>
            </el-sub-menu>

            <el-sub-menu v-if="canAny(['monitor','alert','domain'])" index="/monitor">
              <template #title>
                <el-icon><Histogram /></el-icon>
                <span>监控告警</span>
              </template>
              <el-menu-item index="/monitor/center">监控告警中心</el-menu-item>
              <el-menu-item v-if="can('domain')" index="/domain/center">域名监控中心</el-menu-item>
            </el-sub-menu>

            <el-sub-menu v-if="canAny(['workflow','executor','task','oncall'])" index="/automation">
              <template #title>
                <el-icon><Operation /></el-icon>
                <span>自动化</span>
              </template>
              <el-menu-item v-if="can('workflow')" index="/workflow/designer">工作流编排</el-menu-item>
              <el-menu-item v-if="can('executor')" index="/executor">批量执行</el-menu-item>
            </el-sub-menu>

            <el-sub-menu v-if="canAny(['cicd','workorder'])" index="/cicd">
              <template #title>
                <el-icon><Connection /></el-icon>
                <span>CI/CD</span>
              </template>
              <el-menu-item index="/delivery/center">交付中心</el-menu-item>
            </el-sub-menu>

            <el-sub-menu v-if="canAny(['system:user','system:role','system:permission'])" index="/system">
              <template #title>
                <el-icon><Setting /></el-icon>
                <span>系统管理</span>
              </template>
              <el-menu-item v-if="can('system:user')" index="/system/users">用户管理</el-menu-item>
              <el-menu-item v-if="can('system:role')" index="/system/roles">角色管理</el-menu-item>
            </el-sub-menu>
          </template>
        </el-menu>
      </el-scrollbar>
    </el-aside>

    <el-container>
      <el-header class="header">
        <div class="header-left">
          <div class="breadcrumb-stack">
            <div class="header-eyebrow">控制台</div>
            <el-breadcrumb separator="/">
              <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
              <el-breadcrumb-item>{{ $route.meta.title }}</el-breadcrumb-item>
            </el-breadcrumb>
          </div>
        </div>
        <div class="header-right">
          <el-button class="theme-toggle" @click="toggleTheme">
            <el-icon><component :is="isDark ? 'Sunny' : 'Moon'" /></el-icon>
            <span>{{ isDark ? '浅色' : '深色' }}</span>
          </el-button>
          <div class="user-chip">
            <div class="user-meta">
              <strong>{{ username }}</strong>
              <span>{{ roleCode || 'user' }}</span>
            </div>
            <el-dropdown>
              <span class="el-dropdown-link">
                操作
                <el-icon class="el-icon--right"><arrow-down /></el-icon>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </el-header>

      <el-main class="main">
        <div class="view-tabs-wrap">
          <el-scrollbar>
            <div class="view-tabs">
              <el-tag
                v-for="tab in viewTabs"
                :key="tab.path"
                :type="tab.path === $route.path ? 'primary' : 'info'"
                :effect="tab.path === $route.path ? 'dark' : 'plain'"
                :closable="tab.closable"
                class="view-tab"
                draggable="true"
                @click="openTab(tab.path)"
                @close="closeTab(tab.path)"
                @dragstart="onViewTabDragStart(tab.path)"
                @dragover.prevent
                @drop.prevent="onViewTabDrop(tab.path)"
                @dragend="onViewTabDragEnd"
                @auxclick="onViewTabAuxClick($event, tab)"
              >
                <span class="view-tab-label">
                  <el-icon v-if="tab.pinned" class="view-tab-pin"><StarFilled /></el-icon>
                  <span>{{ tab.title }}</span>
                </span>
              </el-tag>
            </div>
          </el-scrollbar>
          <el-dropdown trigger="click" @command="handleTabCommand">
            <el-button class="tab-action-btn" link>页签操作</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item :command="activeTab?.pinned ? 'unpinCurrent' : 'pinCurrent'">
                  {{ activeTab?.pinned ? '取消固定当前' : '固定当前页签' }}
                </el-dropdown-item>
                <el-dropdown-item command="closeOthers">关闭其他</el-dropdown-item>
                <el-dropdown-item command="closeAll">关闭全部</el-dropdown-item>
                <el-dropdown-item command="closeUnpinned">关闭未固定页签</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-dropdown trigger="click" @command="handleWorkspacePresetCommand">
            <el-button class="tab-action-btn" link>工作台模板</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  command="openLast"
                  :disabled="!hasLastWorkspacePreset"
                >
                  恢复上次模板
                </el-dropdown-item>
                <el-dropdown-item command="saveCurrent">保存当前页签为模板</el-dropdown-item>
                <el-dropdown-item divided disabled>系统模板</el-dropdown-item>
                <el-dropdown-item
                  v-for="preset in availableBuiltinWorkspacePresets"
                  :key="preset.key"
                  :command="`open:${preset.key}`"
                >
                  {{ preset.label }}
                </el-dropdown-item>
                <el-dropdown-item
                  v-if="availableCustomWorkspacePresets.length"
                  divided
                  disabled
                >
                  自定义模板
                </el-dropdown-item>
                <el-dropdown-item
                  v-for="preset in availableCustomWorkspacePresets"
                  :key="`custom-${preset.key}`"
                  :command="`open:${preset.key}`"
                >
                  {{ preset.label }}
                </el-dropdown-item>
                <template v-for="group in availableTeamWorkspacePresetGroups" :key="`group-${group.key}`">
                  <el-dropdown-item divided disabled>
                    团队模板 · {{ group.label }}
                  </el-dropdown-item>
                  <el-dropdown-item
                    v-for="preset in group.items"
                    :key="`team-${preset.id}`"
                    :command="`openTeam:${preset.id}`"
                  >
                    {{ preset.name }} · {{ preset.owner_name || 'team' }}
                  </el-dropdown-item>
                </template>
                <el-dropdown-item
                  v-if="!availableBuiltinWorkspacePresets.length && !availableCustomWorkspacePresets.length && !availableTeamWorkspacePresets.length"
                  disabled
                >
                  暂无可用模板
                </el-dropdown-item>
                <el-dropdown-item v-if="isAdmin" divided command="saveCurrentTeam">保存为团队模板</el-dropdown-item>
                <el-dropdown-item command="refreshTeam">刷新团队模板</el-dropdown-item>
                <el-dropdown-item command="manageTeam">管理团队模板</el-dropdown-item>
                <el-dropdown-item
                  v-if="customWorkspacePresets.length"
                  command="clearCustom"
                >
                  清空自定义模板
                </el-dropdown-item>
                <el-dropdown-item command="copyShareLink">复制当前工作台链接</el-dropdown-item>
                <el-dropdown-item
                  v-if="customWorkspacePresets.length"
                  command="exportCustom"
                >
                  导出自定义模板
                </el-dropdown-item>
                <el-dropdown-item command="importCustom">导入自定义模板</el-dropdown-item>
                <el-dropdown-item
                  v-if="customWorkspacePresets.length"
                  divided
                  disabled
                >
                  删除单个模板
                </el-dropdown-item>
                <el-dropdown-item
                  v-for="preset in customWorkspacePresets"
                  :key="`delete-${preset.key}`"
                  :command="`delete:${preset.key}`"
                >
                  删除 {{ preset.label }}
                </el-dropdown-item>
                <el-dropdown-item
                  v-if="manageableTeamWorkspacePresets.length"
                  divided
                  disabled
                >
                  删除团队模板
                </el-dropdown-item>
                <el-dropdown-item
                  v-for="preset in manageableTeamWorkspacePresets"
                  :key="`delete-team-${preset.id}`"
                  :command="`deleteTeam:${preset.id}`"
                >
                  删除 {{ preset.name }}
                </el-dropdown-item>
                <el-dropdown-item
                  v-if="manageableTeamWorkspacePresets.length"
                  divided
                  disabled
                >
                  编辑团队模板
                </el-dropdown-item>
                <el-dropdown-item
                  v-for="preset in manageableTeamWorkspacePresets"
                  :key="`rename-team-${preset.id}`"
                  :command="`renameTeam:${preset.id}`"
                >
                  重命名 {{ preset.name }}
                </el-dropdown-item>
                <el-dropdown-item
                  v-for="preset in manageableTeamWorkspacePresets"
                  :key="`overwrite-team-${preset.id}`"
                  :command="`overwriteTeam:${preset.id}`"
                >
                  用当前工作台更新 {{ preset.name }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
        <div v-if="workspaceSceneActions.length" class="workspace-scene-wrap">
          <div class="workspace-scene-label">场景工作台</div>
          <div class="workspace-scene-actions">
            <el-button
              v-for="item in workspaceSceneActions"
              :key="item.key"
              size="small"
              plain
              class="workspace-scene-btn"
              @click="openWorkspaceByCategory(item.key)"
            >
              <el-icon><component :is="item.icon" /></el-icon>
              <span>{{ item.label }}</span>
            </el-button>
          </div>
        </div>
        <div
          v-if="activeModuleGroup"
          :class="[
            'module-context-wrap',
            `module-context-${activeModuleGroup.key}`,
            { 'is-compact': moduleLinksCompact }
          ]"
        >
          <div class="module-context-label">模块页签</div>
          <el-scrollbar>
            <div class="module-context-links">
              <template v-if="activeModuleLinks.length">
                <el-tag
                  v-for="link in activeModuleLinks"
                  :key="link.path"
                  draggable="true"
                  effect="plain"
                  :class="[
                    'module-context-tag',
                    { 'is-active': isContextLinkActive(link.path), 'is-pinned': link.pinned }
                  ]"
                  @click="openTab(link.path)"
                  @dragstart="onModuleLinkDragStart(link.path)"
                  @dragover.prevent
                  @drop.prevent="onModuleLinkDrop(link.path)"
                  @dragend="onModuleLinkDragEnd"
                  @auxclick="onModuleLinkAuxClick($event, link.path)"
                  @contextmenu.prevent="openModuleLinkContextMenu($event, link.path)"
                >
                  <span class="module-context-tag-inner">
                    <span class="module-context-tag-title">{{ link.label }}</span>
                    <span class="module-context-tag-actions">
                      <el-icon class="module-pin-icon" @click.stop="toggleModuleLinkPin(link.path)">
                        <component :is="link.pinned ? 'StarFilled' : 'Star'" />
                      </el-icon>
                      <el-icon
                        v-if="!link.pinned && activeModuleLinks.length > 1"
                        class="module-close-icon"
                        @click.stop="hideModuleLink(link.path)"
                      >
                        <Close />
                      </el-icon>
                    </span>
                  </span>
                </el-tag>
              </template>
              <span v-else class="module-context-empty">当前模块无可见联动页签</span>
            </div>
          </el-scrollbar>
          <el-dropdown trigger="click" @command="handleModuleLinksCommand">
            <el-button class="module-context-action" link>页签管理</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="openAll">打开当前模块全部</el-dropdown-item>
                <el-dropdown-item command="toggleCompact">
                  {{ moduleLinksCompact ? '关闭紧凑模式' : '开启紧凑模式' }}
                </el-dropdown-item>
                <el-dropdown-item command="reset">重置当前模块</el-dropdown-item>
                <el-dropdown-item
                  v-for="link in hiddenModuleLinks"
                  :key="`hidden-${link.path}`"
                  :command="`show:${link.path}`"
                >
                  恢复 {{ link.label }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
        <teleport to="body">
          <transition name="module-context-menu-fade">
            <div
              v-if="moduleContextMenu.visible"
              class="module-context-menu"
              :style="{ left: `${moduleContextMenu.x}px`, top: `${moduleContextMenu.y}px` }"
              @click.stop
            >
              <button class="module-context-menu-item" @click="handleModuleContextMenuCommand('open')">
                打开页面
              </button>
              <button class="module-context-menu-item" @click="handleModuleContextMenuCommand('togglePin')">
                {{ moduleContextMenuLink?.pinned ? '取消固定' : '固定页签' }}
              </button>
              <button
                class="module-context-menu-item"
                :disabled="activeModuleLinks.length <= 1"
                @click="handleModuleContextMenuCommand('closeCurrent')"
              >
                关闭当前
              </button>
              <button
                class="module-context-menu-item"
                :disabled="activeModuleLinks.length <= 1"
                @click="handleModuleContextMenuCommand('closeOthers')"
              >
                关闭其他
              </button>
              <button class="module-context-menu-item" @click="handleModuleContextMenuCommand('toggleCompact')">
                {{ moduleLinksCompact ? '关闭紧凑模式' : '开启紧凑模式' }}
              </button>
              <button class="module-context-menu-item danger" @click="handleModuleContextMenuCommand('reset')">
                重置当前模块
              </button>
            </div>
          </transition>
        </teleport>
        <el-dialog
          v-model="teamWorkspacePanelVisible"
          title="团队模板管理"
          width="920px"
          class="team-workspace-dialog"
        >
          <div class="team-workspace-toolbar">
            <el-input
              v-model="teamWorkspacePanelKeyword"
              placeholder="搜索模板名/创建人/路径"
              clearable
              class="team-workspace-search"
            />
            <el-select v-model="teamWorkspacePanelCategory" class="team-workspace-category">
              <el-option label="全部分类" value="all" />
              <el-option
                v-for="item in teamWorkspaceCategoryOptions"
                :key="item.key"
                :label="item.label"
                :value="item.key"
              />
            </el-select>
            <el-switch
              v-model="teamWorkspacePanelEditableOnly"
              active-text="仅显示可编辑"
              inactive-text="全部模板"
            />
          </div>
          <el-table
            :fit="true"
            :data="filteredTeamWorkspacePanelRows"
            size="small"
            max-height="460"
            empty-text="暂无匹配模板"
          >
            <el-table-column prop="name" label="模板名" min-width="180" />
            <el-table-column prop="categoryLabel" label="分类" width="110" />
            <el-table-column label="推荐" width="90">
              <template #default="{ row }">
                <el-tag v-if="row.recommended" type="success" size="small">推荐</el-tag>
                <span v-else class="muted-text">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="owner_name" label="创建人" width="120" />
            <el-table-column label="页签数" width="86">
              <template #default="{ row }">{{ row.tabs.length }}</template>
            </el-table-column>
            <el-table-column label="使用次数" width="92">
              <template #default="{ row }">{{ row.use_count || 0 }}</template>
            </el-table-column>
            <el-table-column label="最近使用人" width="120">
              <template #default="{ row }">{{ row.last_used_by_name || '-' }}</template>
            </el-table-column>
            <el-table-column label="最近使用" min-width="170">
              <template #default="{ row }">{{ formatDateTime(row.last_used_at) }}</template>
            </el-table-column>
            <el-table-column label="更新时间" min-width="170">
              <template #default="{ row }">{{ formatDateTime(row.updated_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" min-width="280" fixed="right">
              <template #default="{ row }">
                <div class="team-workspace-actions">
                  <el-button size="small" link type="primary" @click="openTeamWorkspacePreset(row.id)">打开</el-button>
                  <el-button
                    size="small"
                    link
                    :disabled="!row.editable"
                    @click="runTeamWorkspaceAction('rename', row.id)"
                  >
                    重命名
                  </el-button>
                  <el-button
                    size="small"
                    link
                    :disabled="!row.editable"
                    @click="runTeamWorkspaceAction('overwrite', row.id)"
                  >
                    覆盖
                  </el-button>
                  <el-button
                    size="small"
                    link
                    :disabled="!row.editable"
                    @click="runTeamWorkspaceAction('toggleRecommend', row.id)"
                  >
                    {{ row.recommended ? '取消推荐' : '设为推荐' }}
                  </el-button>
                  <el-button
                    size="small"
                    link
                    type="danger"
                    :disabled="!row.editable"
                    @click="runTeamWorkspaceAction('delete', row.id)"
                  >
                    删除
                  </el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <div v-if="draggableTeamWorkspaceRows.length > 1" class="team-sort-board">
            <div class="team-sort-title">拖拽排序（仅可编辑模板）</div>
            <div class="team-sort-list">
              <div
                v-for="item in draggableTeamWorkspaceRows"
                :key="`sort-${item.id}`"
                class="team-sort-item"
                draggable="true"
                @dragstart="onTeamPresetDragStart(item.id)"
                @dragover.prevent
                @drop.prevent="onTeamPresetDrop(item.id)"
                @dragend="onTeamPresetDragEnd"
              >
                <div class="team-sort-handle">⋮⋮</div>
                <div class="team-sort-name">{{ item.name }}</div>
                <div class="team-sort-meta">
                  <el-tag v-if="item.recommended" size="small" type="success">推荐</el-tag>
                  <span class="muted-text">使用 {{ item.use_count || 0 }}</span>
                </div>
              </div>
            </div>
          </div>
          <template #footer>
            <div class="team-workspace-footer">
              <el-button @click="teamWorkspacePanelVisible = false">关闭</el-button>
              <el-button @click="listTeamWorkspacePresets({ silent: false })">刷新</el-button>
              <el-button v-if="isAdmin" type="primary" @click="saveCurrentAsTeamWorkspacePreset">保存当前为团队模板</el-button>
            </div>
          </template>
        </el-dialog>
        <router-view v-slot="{ Component, route }">
          <transition name="app-route-fade" mode="out-in">
            <div class="page-view app-fade-in" :key="route.fullPath">
              <component :is="Component" />
            </div>
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useTheme } from '@/utils/theme'

const router = useRouter()
const route = useRoute()
const { isDark, toggleTheme } = useTheme()
const username = ref('Admin')
const roleCode = ref(localStorage.getItem('role_code') || '')
const permissions = ref(new Set(JSON.parse(localStorage.getItem('permissions') || '[]')))
const TABS_STORAGE_KEY = 'layout_view_tabs'
const MODULE_LINKS_STORAGE_KEY = 'layout_module_links_state_v1'
const MODULE_LINKS_COMPACT_STORAGE_KEY = 'layout_module_links_compact_v1'
const CUSTOM_WORKSPACE_PRESETS_KEY = 'layout_custom_workspace_presets_v1'
const LAST_WORKSPACE_PRESET_KEY = 'layout_last_workspace_preset_v1'
const WORKSPACE_SHARE_QUERY_KEY = 'workspace'
const TAB_LIMIT = 18
const viewTabs = ref([{ path: '/dashboard', title: '仪表盘', pinned: false, closable: false }])
const draggingViewTabPath = ref('')
const moduleLinkState = ref({})
const draggingModuleLinkPath = ref('')
const moduleLinksCompact = ref(false)
const menuSimpleMode = true
const moduleContextMenu = ref({ visible: false, x: 0, y: 0, path: '' })
const customWorkspacePresets = ref([])
const teamWorkspacePresets = ref([])
const teamWorkspacePanelVisible = ref(false)
const teamWorkspacePanelKeyword = ref('')
const teamWorkspacePanelCategory = ref('all')
const teamWorkspacePanelEditableOnly = ref(false)
const draggingTeamPresetId = ref('')
const lastWorkspacePresetKey = ref('')
const workspaceAppliedFromQuery = ref(false)
const currentUserID = ref('')
const isAdmin = computed(() => roleCode.value === 'admin')

const readTabsFromStorage = () => {
  try {
    const raw = localStorage.getItem(TABS_STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return null
    return parsed
      .filter((item) => item?.path && item?.title)
      .map((item) => ({
        path: item.path,
        title: item.title,
        pinned: Boolean(item.pinned),
        closable: item.path !== '/dashboard' && !item.pinned
      }))
  } catch {
    return null
  }
}

const persistTabs = () => {
  localStorage.setItem(TABS_STORAGE_KEY, JSON.stringify(viewTabs.value))
}

const readModuleLinkState = () => {
  try {
    const raw = localStorage.getItem(MODULE_LINKS_STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

const persistModuleLinkState = () => {
  localStorage.setItem(MODULE_LINKS_STORAGE_KEY, JSON.stringify(moduleLinkState.value))
}

const readModuleLinksCompact = () => {
  try {
    return localStorage.getItem(MODULE_LINKS_COMPACT_STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

const persistModuleLinksCompact = () => {
  localStorage.setItem(MODULE_LINKS_COMPACT_STORAGE_KEY, moduleLinksCompact.value ? '1' : '0')
}

const ensureTab = (targetRoute) => {
  if (!targetRoute?.path || targetRoute.path === '/login') return
  const title = targetRoute.meta?.title || '未命名页面'
  const exists = viewTabs.value.some((item) => item.path === targetRoute.path)
  if (exists) return
  viewTabs.value.push({
    path: targetRoute.path,
    title,
    pinned: false,
    closable: targetRoute.path !== '/dashboard'
  })
  if (viewTabs.value.length > TAB_LIMIT) {
    const removableIdx = viewTabs.value.findIndex((item) => item.closable)
    if (removableIdx >= 0) viewTabs.value.splice(removableIdx, 1)
  }
  persistTabs()
}

const openTab = (path) => {
  if (!path || path === route.path) return
  router.push(path)
}

const closeTab = (path) => {
  const idx = viewTabs.value.findIndex((item) => item.path === path)
  if (idx < 0) return
  if (viewTabs.value[idx]?.pinned) return
  const activePath = route.path
  viewTabs.value.splice(idx, 1)
  if (!viewTabs.value.length) {
    viewTabs.value = [{ path: '/dashboard', title: '仪表盘', pinned: false, closable: false }]
  }
  persistTabs()
  if (activePath === path) {
    const fallback = viewTabs.value[Math.max(0, idx - 1)] || viewTabs.value[0]
    if (fallback?.path) router.push(fallback.path)
  }
}

const closeOtherTabs = () => {
  const current = viewTabs.value.find((item) => item.path === route.path)
  const keepMap = new Map()
  keepMap.set('/dashboard', { path: '/dashboard', title: '仪表盘', pinned: false, closable: false })
  viewTabs.value
    .filter((item) => item.pinned && item.path !== '/dashboard')
    .forEach((item) => keepMap.set(item.path, { ...item, closable: false }))
  if (current && current.path !== '/dashboard') {
    keepMap.set(current.path, {
      ...current,
      closable: !current.pinned
    })
  }
  viewTabs.value = [...keepMap.values()]
  persistTabs()
}

const closeAllTabs = () => {
  viewTabs.value = [{ path: '/dashboard', title: '仪表盘', pinned: false, closable: false }]
  persistTabs()
  if (route.path !== '/dashboard') router.push('/dashboard')
}

const closeUnpinnedTabs = () => {
  const keep = viewTabs.value.filter((item) => item.path === '/dashboard' || item.pinned)
  viewTabs.value = keep.length ? keep : [{ path: '/dashboard', title: '仪表盘', pinned: false, closable: false }]
  persistTabs()
  if (!viewTabs.value.some((item) => item.path === route.path)) {
    const fallback = viewTabs.value[0]
    if (fallback?.path && fallback.path !== route.path) router.push(fallback.path)
  }
}

const onViewTabDragStart = (path) => {
  if (!path) return
  draggingViewTabPath.value = path
}

const onViewTabDrop = (targetPath) => {
  const sourcePath = draggingViewTabPath.value
  if (!sourcePath || !targetPath || sourcePath === targetPath) return
  const tabs = [...viewTabs.value]
  const from = tabs.findIndex((item) => item.path === sourcePath)
  const to = tabs.findIndex((item) => item.path === targetPath)
  if (from < 0 || to < 0) return
  const [moved] = tabs.splice(from, 1)
  tabs.splice(to, 0, moved)
  const dashboardIdx = tabs.findIndex((item) => item.path === '/dashboard')
  if (dashboardIdx > 0) {
    const [dashboard] = tabs.splice(dashboardIdx, 1)
    tabs.unshift(dashboard)
  }
  viewTabs.value = tabs
  persistTabs()
}

const onViewTabDragEnd = () => {
  draggingViewTabPath.value = ''
}

const onViewTabAuxClick = (event, tab) => {
  if (event.button !== 1 || !tab?.closable || tab?.pinned) return
  closeTab(tab.path)
}

const activeTab = computed(() => viewTabs.value.find((item) => item.path === route.path) || null)

const setTabPinned = (path, pinned) => {
  const idx = viewTabs.value.findIndex((item) => item.path === path)
  if (idx < 0 || path === '/dashboard') return
  const current = viewTabs.value[idx]
  viewTabs.value[idx] = {
    ...current,
    pinned: Boolean(pinned),
    closable: !pinned
  }
  persistTabs()
}

const handleTabCommand = (command) => {
  if (command === 'pinCurrent' && activeTab.value?.path) {
    setTabPinned(activeTab.value.path, true)
    return
  }
  if (command === 'unpinCurrent' && activeTab.value?.path) {
    setTabPinned(activeTab.value.path, false)
    return
  }
  if (command === 'closeOthers') closeOtherTabs()
  if (command === 'closeAll') closeAllTabs()
  if (command === 'closeUnpinned') closeUnpinnedTabs()
}

const setPermissions = (list = []) => {
  permissions.value = new Set(list)
  localStorage.setItem('permissions', JSON.stringify(list))
}

const can = (code) => {
  if (!code) return true
  if (roleCode.value === 'admin') return true
  if (permissions.value.has(code)) return true
  const parts = code.split(':')
  while (parts.length > 1) {
    parts.pop()
    if (permissions.value.has(parts.join(':'))) return true
  }
  return false
}

const canAny = (codes = []) => codes.some((c) => can(c))
const hasAnyPerm = (codes = []) => {
  if (!codes?.length) return true
  return codes.some((code) => can(code))
}
const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })
const parseAxiosErrorMessage = (error, fallback = '请求失败') =>
  error?.response?.data?.message || error?.response?.data?.msg || error?.message || fallback
const WORKSPACE_CATEGORY_LABELS = {
  asset: '资产',
  k8s: '容器与K8s',
  monitor: '监控告警',
  delivery: '交付发布',
  automation: '自动化协同',
  other: '其他'
}
const WORKSPACE_CATEGORY_ORDER = ['asset', 'k8s', 'monitor', 'delivery', 'automation']
const WORKSPACE_CATEGORY_ICONS = {
  asset: 'Monitor',
  k8s: 'Platform',
  monitor: 'Histogram',
  delivery: 'Connection',
  automation: 'Operation',
  other: 'Grid'
}
const WORKSPACE_CATEGORY_PREFIXES = {
  asset: ['/asset', '/host', '/cmdb', '/jump', '/firewall', '/terminal'],
  k8s: ['/k8s', '/docker'],
  monitor: ['/monitor', '/alert', '/notify', '/domain'],
  delivery: ['/delivery', '/cicd', '/workorder', '/application', '/sqlaudit', '/gitops'],
  automation: ['/workflow', '/executor', '/task', '/oncall', '/ai', '/collab']
}
const formatDateTime = (value) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN', { hour12: false })
}

const workspacePresets = [
  {
    key: 'asset-ops',
    label: '资产排障视图',
    category: 'asset',
    paths: ['/asset/ops', '/host', '/jump/sessions', '/firewall', '/terminal']
  },
  {
    key: 'k8s-oncall',
    label: 'K8s值班视图',
    category: 'k8s',
    paths: ['/k8s/overview', '/k8s/workloads', '/k8s/deployments', '/k8s/events']
  },
  {
    key: 'monitor-duty',
    label: '监控值班视图',
    category: 'monitor',
    paths: ['/monitor/center', '/alert/events', '/domain/center', '/notify/channels']
  },
  {
    key: 'delivery-release',
    label: '交付发布视图',
    category: 'delivery',
    paths: ['/delivery/center', '/cicd/pipelines', '/cicd/executions', '/workorder/tickets']
  },
  {
    key: 'automation-warroom',
    label: '自动化协同视图',
    category: 'automation',
    paths: ['/ai', '/workflow/designer', '/task/schedules', '/oncall/schedule']
  }
]

const normalizeCustomWorkspacePresets = (list = []) =>
  list
    .filter((item) => item?.key && item?.label && Array.isArray(item.paths))
    .map((item) => ({
      key: String(item.key),
      label: String(item.label).trim().slice(0, 24),
      paths: item.paths
        .filter((path) => typeof path === 'string' && path.trim())
        .map((path) => path.trim())
    }))
    .filter((item) => item.label && item.paths.length)
    .slice(0, 12)

const readCustomWorkspacePresets = () => {
  try {
    const raw = localStorage.getItem(CUSTOM_WORKSPACE_PRESETS_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return normalizeCustomWorkspacePresets(parsed)
  } catch {
    return []
  }
}

const persistCustomWorkspacePresets = () => {
  localStorage.setItem(CUSTOM_WORKSPACE_PRESETS_KEY, JSON.stringify(customWorkspacePresets.value))
}

const readLastWorkspacePresetKey = () => {
  try {
    return localStorage.getItem(LAST_WORKSPACE_PRESET_KEY) || ''
  } catch {
    return ''
  }
}

const persistLastWorkspacePresetKey = (key) => {
  lastWorkspacePresetKey.value = key || ''
  localStorage.setItem(LAST_WORKSPACE_PRESET_KEY, lastWorkspacePresetKey.value)
}

const canAccessPath = (path) => {
  const resolved = router.resolve(path)
  const matched = resolved?.matched || []
  if (!matched.length) return false
  const record = matched[matched.length - 1]
  const perm = record?.meta?.perm
  return !perm || can(perm)
}

const availableBuiltinWorkspacePresets = computed(() =>
  workspacePresets
    .map((preset) => ({
      ...preset,
      paths: preset.paths.filter((path) => canAccessPath(path))
    }))
    .filter((preset) => preset.paths.length > 0)
)

const builtinPresetCategoryMap = computed(() => {
  const map = {}
  availableBuiltinWorkspacePresets.value.forEach((preset) => {
    if (!preset.category) return
    if (!map[preset.category]) map[preset.category] = preset.key
  })
  return map
})

const workspaceSceneActions = computed(() =>
  ['asset', 'k8s', 'monitor', 'delivery', 'automation']
    .filter((key) =>
      availableTeamWorkspacePresets.value.some((item) => classifyWorkspaceTabs(item.tabs).key === key) ||
      Boolean(builtinPresetCategoryMap.value[key])
    )
    .map((key) => ({
      key,
      label: WORKSPACE_CATEGORY_LABELS[key] || key,
      icon: WORKSPACE_CATEGORY_ICONS[key] || 'Grid'
    }))
)

const availableCustomWorkspacePresets = computed(() =>
  customWorkspacePresets.value
    .map((preset) => ({
      ...preset,
      paths: preset.paths.filter((path) => canAccessPath(path))
    }))
    .filter((preset) => preset.paths.length > 0)
)

const availableTeamWorkspacePresets = computed(() =>
  teamWorkspacePresets.value
    .map((preset) => {
      const normalizedTabs = normalizeWorkspaceTabs(preset.tabs)
      return {
        ...preset,
        tabs: normalizedTabs,
        paths: normalizedTabs.map((item) => item.path)
      }
    })
    .filter((preset) => preset.paths.length > 0)
    .sort((a, b) => {
      if (a.recommended !== b.recommended) return a.recommended ? -1 : 1
      const aSort = Number.isFinite(Number(a.sort_order)) ? Number(a.sort_order) : 0
      const bSort = Number.isFinite(Number(b.sort_order)) ? Number(b.sort_order) : 0
      if (aSort !== bSort) return aSort - bSort
      return new Date(b.updated_at || 0).getTime() - new Date(a.updated_at || 0).getTime()
    })
)

const manageableTeamWorkspacePresets = computed(() =>
  teamWorkspacePresets.value.filter((preset) => preset.editable)
)

const draggableTeamWorkspaceRows = computed(() =>
  manageableTeamWorkspacePresets.value
    .filter((item) => item.scope === 'team')
    .slice()
    .sort((a, b) => {
      const aSort = Number.isFinite(Number(a.sort_order)) ? Number(a.sort_order) : 0
      const bSort = Number.isFinite(Number(b.sort_order)) ? Number(b.sort_order) : 0
      if (aSort !== bSort) return aSort - bSort
      return new Date(b.updated_at || 0).getTime() - new Date(a.updated_at || 0).getTime()
    })
)

const teamWorkspaceCategoryOptions = computed(() => {
  const map = new Map()
  availableTeamWorkspacePresets.value.forEach((preset) => {
    const category = classifyWorkspaceTabs(preset.tabs)
    if (!map.has(category.key)) map.set(category.key, category.label)
  })
  return Array.from(map.entries()).map(([key, label]) => ({ key, label }))
})

const filteredTeamWorkspacePanelRows = computed(() => {
  const keyword = teamWorkspacePanelKeyword.value.trim().toLowerCase()
  return availableTeamWorkspacePresets.value
    .map((preset) => {
      const category = classifyWorkspaceTabs(preset.tabs)
      return {
        ...preset,
        categoryKey: category.key,
        categoryLabel: category.label,
        searchText: [
          preset.name,
          preset.owner_name,
          preset.paths.join(' '),
          category.label
        ]
          .filter(Boolean)
          .join(' ')
          .toLowerCase()
      }
    })
    .filter((preset) => teamWorkspacePanelCategory.value === 'all' || preset.categoryKey === teamWorkspacePanelCategory.value)
    .filter((preset) => (teamWorkspacePanelEditableOnly.value ? preset.editable : true))
    .filter((preset) => (!keyword ? true : preset.searchText.includes(keyword)))
})

const classifyWorkspaceTabs = (tabs = []) => {
  const scores = {}
  tabs.forEach((item) => {
    const path = item?.path || ''
    Object.entries(WORKSPACE_CATEGORY_PREFIXES).forEach(([category, prefixes]) => {
      if (prefixes.some((prefix) => path.startsWith(prefix))) {
        scores[category] = (scores[category] || 0) + 1
      }
    })
  })
  const categories = Object.keys(scores)
  if (!categories.length) return { key: 'other', label: '其他' }
  categories.sort((a, b) => {
    const diff = (scores[b] || 0) - (scores[a] || 0)
    if (diff !== 0) return diff
    return WORKSPACE_CATEGORY_ORDER.indexOf(a) - WORKSPACE_CATEGORY_ORDER.indexOf(b)
  })
  const key = categories[0]
  return { key, label: WORKSPACE_CATEGORY_LABELS[key] || '其他' }
}

const availableTeamWorkspacePresetGroups = computed(() => {
  const buckets = {}
  availableTeamWorkspacePresets.value.forEach((preset) => {
    const category = classifyWorkspaceTabs(preset.tabs)
    if (!buckets[category.key]) {
      buckets[category.key] = { key: category.key, label: category.label, items: [] }
    }
    buckets[category.key].items.push(preset)
  })
  return Object.values(buckets)
})

const workspacePresetMap = computed(() => {
  const map = {}
  ;[...availableBuiltinWorkspacePresets.value, ...availableCustomWorkspacePresets.value].forEach((preset) => {
    map[preset.key] = preset
  })
  return map
})

const teamWorkspacePresetMap = computed(() => {
  const map = {}
  availableTeamWorkspacePresets.value.forEach((preset) => {
    map[preset.id] = preset
  })
  return map
})

const hasLastWorkspacePreset = computed(() => {
  const key = String(lastWorkspacePresetKey.value || '').trim()
  if (!key) return false
  if (key.startsWith('team:')) {
    const teamID = key.slice(5)
    return Boolean(teamWorkspacePresetMap.value[teamID])
  }
  return Boolean(workspacePresetMap.value[key])
})

const resolveTabInfo = (path) => {
  if (!path || path === '/login') return null
  const resolved = router.resolve(path)
  const matched = resolved?.matched || []
  if (!matched.length) return null
  const record = matched[matched.length - 1]
  return {
    path,
    title: record?.meta?.title || '未命名页面',
    pinned: false,
    closable: path !== '/dashboard'
  }
}

const normalizeWorkspaceTabs = (tabs = []) => {
  const rawRows = Array.isArray(tabs) ? tabs : []
  return rawRows
    .map((item) => {
      if (typeof item === 'string') return { path: item, pinned: false }
      if (item && typeof item === 'object' && item.path) {
        return { path: item.path, pinned: Boolean(item.pinned) }
      }
      return null
    })
    .filter(Boolean)
    .filter((item) => item.path !== '/login' && canAccessPath(item.path))
    .filter((item, idx, arr) => arr.findIndex((one) => one.path === item.path) === idx)
    .slice(0, TAB_LIMIT - 1)
}

const applyWorkspaceTabs = async (tabs = [], noticeLabel = '') => {
  const normalized = normalizeWorkspaceTabs(tabs)
  const dashboard = { path: '/dashboard', title: '仪表盘', pinned: false, closable: false }
  const resolved = normalized
    .map((item) => {
      const info = resolveTabInfo(item.path)
      if (!info) return null
      const pinned = Boolean(item.pinned)
      return { ...info, pinned, closable: !pinned && info.path !== '/dashboard' }
    })
    .filter(Boolean)
  viewTabs.value = [dashboard, ...resolved].slice(0, TAB_LIMIT)
  persistTabs()
  const firstPath = resolved[0]?.path || '/dashboard'
  if (firstPath !== route.path) await router.push(firstPath)
  if (noticeLabel) ElMessage.success(noticeLabel)
}

const encodeWorkspaceSnapshot = () => {
  const payload = {
    version: 1,
    tabs: viewTabs.value
      .filter((item) => item.path !== '/dashboard')
      .map((item) => ({ path: item.path, pinned: Boolean(item.pinned) }))
  }
  return btoa(unescape(encodeURIComponent(JSON.stringify(payload))))
}

const decodeWorkspaceSnapshot = (raw = '') => {
  try {
    const decoded = decodeURIComponent(escape(atob(String(raw))))
    const parsed = JSON.parse(decoded)
    if (!parsed || typeof parsed !== 'object') return null
    return parsed
  } catch {
    return null
  }
}

const applyWorkspaceFromQuery = async () => {
  if (workspaceAppliedFromQuery.value) return
  const token = route.query?.[WORKSPACE_SHARE_QUERY_KEY]
  if (!token || typeof token !== 'string') return
  workspaceAppliedFromQuery.value = true
  const snapshot = decodeWorkspaceSnapshot(token)
  if (!snapshot?.tabs) {
    ElMessage.warning('工作台分享链接无效')
  } else {
    await applyWorkspaceTabs(snapshot.tabs, '已恢复分享工作台')
  }
  const nextQuery = { ...route.query }
  delete nextQuery[WORKSPACE_SHARE_QUERY_KEY]
  await router.replace({ path: route.path, query: nextQuery })
}

const openWorkspacePreset = async (key) => {
  const preset = workspacePresetMap.value[key]
  if (!preset || !preset.paths.length) {
    ElMessage.warning('当前账号暂无可用模板页')
    return
  }
  const nextTabs = [resolveTabInfo('/dashboard'), ...preset.paths.map((path) => resolveTabInfo(path))]
    .filter(Boolean)
    .filter((item, idx, arr) => arr.findIndex((one) => one.path === item.path) === idx)
  const pinnedKeep = viewTabs.value
    .filter((item) => item.pinned && item.path !== '/dashboard')
    .filter((item) => !nextTabs.some((tab) => tab.path === item.path))
    .map((item) => ({ ...item, closable: false }))
  viewTabs.value = [nextTabs[0], ...pinnedKeep, ...nextTabs.slice(1)]
    .filter(Boolean)
    .slice(0, TAB_LIMIT)
  if (!viewTabs.value.length) {
    viewTabs.value = [{ path: '/dashboard', title: '仪表盘', pinned: false, closable: false }]
  }
  persistTabs()
  persistLastWorkspacePresetKey(preset.key)
  const firstPath = preset.paths[0]
  if (firstPath && firstPath !== route.path) {
    await router.push(firstPath)
  } else if (!firstPath && route.path !== '/dashboard') {
    await router.push('/dashboard')
  }
  ElMessage.success(`已打开 ${preset.label}`)
}

const ensureTabByPath = (path) => {
  const info = resolveTabInfo(path)
  if (!info) return false
  if (viewTabs.value.some((item) => item.path === info.path)) return true
  viewTabs.value.push(info)
  if (viewTabs.value.length > TAB_LIMIT) {
    const removableIdx = viewTabs.value.findIndex((item) => item.closable)
    if (removableIdx >= 0) viewTabs.value.splice(removableIdx, 1)
  }
  persistTabs()
  return true
}

const openModuleLinksAsTabs = async () => {
  const links = activeModuleLinks.value
  if (!links.length) {
    ElMessage.warning('当前模块没有可打开的联动页')
    return
  }
  links.forEach((item) => ensureTabByPath(item.path))
  const first = links[0]?.path
  if (first && first !== route.path) await router.push(first)
  ElMessage.success(`已打开 ${links.length} 个模块联动页`)
}

const saveCurrentWorkspacePreset = async () => {
  const currentPaths = viewTabs.value
    .map((item) => item.path)
    .filter((path) => path && path !== '/dashboard' && canAccessPath(path))
  if (!currentPaths.length) {
    ElMessage.warning('当前没有可保存的业务页签')
    return
  }
  const { value } = await ElMessageBox.prompt('请输入模板名称', '保存工作台模板', {
    confirmButtonText: '保存',
    cancelButtonText: '取消',
    inputPattern: /^.{2,24}$/,
    inputErrorMessage: '模板名称长度需为 2-24 个字符',
    closeOnClickModal: false
  })
  const label = String(value || '').trim()
  if (!label) return
  const key = `custom-${Date.now()}`
  const next = customWorkspacePresets.value.filter((item) => item.label !== label)
  next.unshift({ key, label, paths: currentPaths })
  customWorkspacePresets.value = next.slice(0, 12)
  persistCustomWorkspacePresets()
  persistLastWorkspacePresetKey(key)
  ElMessage.success(`模板「${label}」已保存`)
}

const clearCustomWorkspacePresets = async () => {
  if (!customWorkspacePresets.value.length) return
  await ElMessageBox.confirm('确认清空所有自定义模板吗？', '提示', {
    type: 'warning',
    confirmButtonText: '清空',
    cancelButtonText: '取消'
  })
  customWorkspacePresets.value = []
  persistCustomWorkspacePresets()
  if (lastWorkspacePresetKey.value.startsWith('custom-')) {
    persistLastWorkspacePresetKey('')
  }
  ElMessage.success('已清空自定义模板')
}

const deleteCustomWorkspacePreset = async (key) => {
  const target = customWorkspacePresets.value.find((item) => item.key === key)
  if (!target) return
  await ElMessageBox.confirm(`确认删除模板「${target.label}」吗？`, '提示', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  customWorkspacePresets.value = customWorkspacePresets.value.filter((item) => item.key !== key)
  persistCustomWorkspacePresets()
  if (lastWorkspacePresetKey.value === key) persistLastWorkspacePresetKey('')
  ElMessage.success(`模板「${target.label}」已删除`)
}

const exportCustomWorkspacePresets = async () => {
  if (!customWorkspacePresets.value.length) {
    ElMessage.warning('暂无可导出的自定义模板')
    return
  }
  const payload = {
    version: 1,
    exportedAt: new Date().toISOString(),
    presets: customWorkspacePresets.value
  }
  const text = JSON.stringify(payload, null, 2)
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制模板JSON到剪贴板')
  } catch {
    await ElMessageBox.alert(
      '<p>浏览器未授权剪贴板，请手动复制下方 JSON：</p><pre style="max-height:280px;overflow:auto;background:#0f172a;color:#e2e8f0;padding:10px;border-radius:8px;">' +
        text
          .replaceAll('&', '&amp;')
          .replaceAll('<', '&lt;')
          .replaceAll('>', '&gt;') +
        '</pre>',
      '导出模板',
      {
        dangerouslyUseHTMLString: true,
        confirmButtonText: '我知道了'
      }
    )
  }
}

const copyWorkspaceShareLink = async () => {
  const token = encodeWorkspaceSnapshot()
  const href = router.resolve({
    path: '/dashboard',
    query: { [WORKSPACE_SHARE_QUERY_KEY]: token }
  }).href
  const link = new URL(href, window.location.origin).toString()
  try {
    await navigator.clipboard.writeText(link)
    ElMessage.success('工作台分享链接已复制')
  } catch {
    await ElMessageBox.alert(link, '复制失败，请手动复制链接', {
      confirmButtonText: '我知道了'
    })
  }
}

const importCustomWorkspacePresets = async () => {
  const { value } = await ElMessageBox.prompt('粘贴模板 JSON（支持 {presets:[...]} 或直接数组）', '导入自定义模板', {
    confirmButtonText: '导入',
    cancelButtonText: '取消',
    inputType: 'textarea',
    inputPlaceholder: '[{"key":"custom-1","label":"值班模板","paths":["/monitor/center","/alert/events"]}]',
    closeOnClickModal: false
  })
  const raw = String(value || '').trim()
  if (!raw) return
  let parsed
  try {
    parsed = JSON.parse(raw)
  } catch {
    ElMessage.error('JSON 格式无效，请检查后重试')
    return
  }
  const source = Array.isArray(parsed) ? parsed : parsed?.presets
  if (!Array.isArray(source)) {
    ElMessage.error('导入内容必须是数组，或包含 presets 数组')
    return
  }
  const normalized = normalizeCustomWorkspacePresets(source).map((item) => ({
    ...item,
    key: item.key.startsWith('custom-') ? item.key : `custom-${Date.now()}-${Math.random().toString(16).slice(2, 6)}`
  }))
  if (!normalized.length) {
    ElMessage.warning('未识别到可导入模板')
    return
  }
  const existedLabelSet = new Set(customWorkspacePresets.value.map((item) => item.label))
  const incoming = normalized.filter((item) => !existedLabelSet.has(item.label))
  const merged = [...incoming, ...customWorkspacePresets.value].slice(0, 12)
  customWorkspacePresets.value = merged
  persistCustomWorkspacePresets()
  ElMessage.success(`已导入 ${incoming.length} 个模板`)
}

const listTeamWorkspacePresets = async ({ silent = true } = {}) => {
  try {
    const res = await axios.get('/api/v1/user/workspace-presets', {
      headers: authHeaders(),
      params: { scope: 'all' }
    })
    if (res.data?.code === 0) {
      const rows = Array.isArray(res.data.data) ? res.data.data : []
      teamWorkspacePresets.value = rows
        .filter((item) => item?.scope === 'team')
        .map((item) => ({
          id: item.id,
          name: item.name,
          scope: item.scope,
          owner_id: item.owner_id,
          owner_name: item.owner_name,
          editable: Boolean(item.editable),
          recommended: Boolean(item.recommended),
          sort_order: Number(item.sort_order || 0),
          use_count: Number(item.use_count || 0),
          last_used_by_id: item.last_used_by_id || '',
          last_used_by_name: item.last_used_by_name || '',
          last_used_at: item.last_used_at || '',
          updated_at: item.updated_at || '',
          tabs: Array.isArray(item.tabs) ? item.tabs : []
        }))
    } else {
      teamWorkspacePresets.value = []
      if (!silent) {
        ElMessage.error(parseAxiosErrorMessage({ response: { data: res.data } }, '刷新团队模板失败'))
      }
    }
  } catch (error) {
    teamWorkspacePresets.value = []
    if (!silent) {
      ElMessage.error(parseAxiosErrorMessage(error, '刷新团队模板失败'))
    }
  }
}

const openTeamWorkspacePreset = async (id) => {
  const preset = teamWorkspacePresets.value.find((item) => item.id === id)
  if (!preset) {
    ElMessage.warning('团队模板不存在或已删除')
    return
  }
  const tabs = normalizeWorkspaceTabs(preset.tabs)
  if (!tabs.length) {
    ElMessage.warning('该模板无可用页面')
    return
  }
  await applyWorkspaceTabs(tabs, `已打开团队模板：${preset.name}`)
  persistLastWorkspacePresetKey(`team:${id}`)
  await markTeamWorkspacePresetUsed(id)
}

const openRecommendedTeamWorkspaceByCategory = async (categoryKey, silent = false) => {
  const key = String(categoryKey || '').trim()
  if (!key) return false
  const list = availableTeamWorkspacePresets.value.filter(
    (item) => classifyWorkspaceTabs(item.tabs).key === key
  )
  if (!list.length) {
    if (!silent) ElMessage.warning(`暂无「${WORKSPACE_CATEGORY_LABELS[key] || key}」团队模板`)
    return false
  }
  const target = list.find((item) => item.recommended) || list[0]
  await openTeamWorkspacePreset(target.id)
  return true
}

const openWorkspaceByCategory = async (categoryKey, silent = false) => {
  const key = String(categoryKey || '').trim()
  if (!key) return false
  const openedTeam = await openRecommendedTeamWorkspaceByCategory(key, true)
  if (openedTeam) return true
  const builtinKey = builtinPresetCategoryMap.value[key]
  if (builtinKey) {
    await openWorkspacePreset(builtinKey)
    return true
  }
  if (!silent) {
    ElMessage.warning(`暂无「${WORKSPACE_CATEGORY_LABELS[key] || key}」可用模板`)
  }
  return false
}

const saveCurrentAsTeamWorkspacePreset = async () => {
  if (!isAdmin.value) {
    ElMessage.warning('仅管理员可保存团队模板')
    return
  }
  const tabs = normalizeWorkspaceTabs(
    viewTabs.value
      .filter((item) => item.path !== '/dashboard')
      .map((item) => ({ path: item.path, pinned: Boolean(item.pinned) }))
  )
  if (!tabs.length) {
    ElMessage.warning('当前没有可保存的业务页签')
    return
  }
  const { value } = await ElMessageBox.prompt('请输入团队模板名称', '保存团队模板', {
    confirmButtonText: '保存',
    cancelButtonText: '取消',
    inputPattern: /^.{2,24}$/,
    inputErrorMessage: '模板名称长度需为 2-24 个字符',
    closeOnClickModal: false
  })
  const name = String(value || '').trim()
  if (!name) return
  await axios.post('/api/v1/user/workspace-presets', { name, scope: 'team', tabs }, { headers: authHeaders() })
  await listTeamWorkspacePresets()
  ElMessage.success(`团队模板「${name}」已保存`)
}

const deleteTeamWorkspacePreset = async (id) => {
  const preset = manageableTeamWorkspacePresets.value.find((item) => item.id === id)
  if (!preset) return
  await ElMessageBox.confirm(`确认删除团队模板「${preset.name}」吗？`, '提示', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消'
  })
  await axios.delete(`/api/v1/user/workspace-presets/${id}`, { headers: authHeaders() })
  await listTeamWorkspacePresets()
  ElMessage.success(`已删除团队模板：${preset.name}`)
}

const renameTeamWorkspacePreset = async (id) => {
  const preset = manageableTeamWorkspacePresets.value.find((item) => item.id === id)
  if (!preset) return
  const { value } = await ElMessageBox.prompt('请输入新的模板名称', '重命名团队模板', {
    inputValue: preset.name,
    confirmButtonText: '保存',
    cancelButtonText: '取消',
    inputPattern: /^.{2,24}$/,
    inputErrorMessage: '模板名称长度需为 2-24 个字符',
    closeOnClickModal: false
  })
  const name = String(value || '').trim()
  if (!name || name === preset.name) return
  await axios.put(
    `/api/v1/user/workspace-presets/${id}`,
    { name, scope: 'team', tabs: normalizeWorkspaceTabs(preset.tabs) },
    { headers: authHeaders() }
  )
  await listTeamWorkspacePresets()
  ElMessage.success(`模板已重命名为：${name}`)
}

const overwriteTeamWorkspacePresetWithCurrent = async (id) => {
  const preset = manageableTeamWorkspacePresets.value.find((item) => item.id === id)
  if (!preset) return
  const tabs = normalizeWorkspaceTabs(
    viewTabs.value
      .filter((item) => item.path !== '/dashboard')
      .map((item) => ({ path: item.path, pinned: Boolean(item.pinned) }))
  )
  if (!tabs.length) {
    ElMessage.warning('当前没有可用于覆盖的业务页签')
    return
  }
  await ElMessageBox.confirm(`确认用当前工作台覆盖团队模板「${preset.name}」吗？`, '提示', {
    type: 'warning',
    confirmButtonText: '覆盖',
    cancelButtonText: '取消'
  })
  await axios.put(
    `/api/v1/user/workspace-presets/${id}`,
    { name: preset.name, scope: 'team', tabs },
    { headers: authHeaders() }
  )
  await listTeamWorkspacePresets()
  ElMessage.success(`已更新团队模板：${preset.name}`)
}

const markTeamWorkspacePresetUsed = async (id) => {
  try {
    await axios.post(`/api/v1/user/workspace-presets/${id}/use`, {}, { headers: authHeaders() })
    const idx = teamWorkspacePresets.value.findIndex((item) => item.id === id)
    if (idx >= 0) {
      const row = teamWorkspacePresets.value[idx]
      teamWorkspacePresets.value[idx] = {
        ...row,
        use_count: Number(row.use_count || 0) + 1,
        last_used_by_name: username.value || row.last_used_by_name,
        last_used_at: new Date().toISOString()
      }
    }
  } catch {
    // ignore usage record failures
  }
}

const toggleTeamWorkspaceRecommended = async (id) => {
  const preset = manageableTeamWorkspacePresets.value.find((item) => item.id === id)
  if (!preset) return
  await axios.put(
    `/api/v1/user/workspace-presets/${id}`,
    {
      name: preset.name,
      scope: 'team',
      tabs: normalizeWorkspaceTabs(preset.tabs),
      recommended: !preset.recommended,
      sort_order: preset.sort_order
    },
    { headers: authHeaders() }
  )
  await listTeamWorkspacePresets()
  ElMessage.success(!preset.recommended ? `已设为推荐模板：${preset.name}` : `已取消推荐：${preset.name}`)
}

const saveTeamWorkspaceOrder = async (orderedIDs) => {
  if (!orderedIDs.length) return
  await axios.post(
    '/api/v1/user/workspace-presets/reorder',
    { items: orderedIDs.map((id) => ({ id })) },
    { headers: authHeaders() }
  )
  await listTeamWorkspacePresets()
}

const onTeamPresetDragStart = (id) => {
  draggingTeamPresetId.value = id
}

const onTeamPresetDrop = async (targetID) => {
  const sourceID = draggingTeamPresetId.value
  draggingTeamPresetId.value = ''
  if (!sourceID || !targetID || sourceID === targetID) return
  const ids = draggableTeamWorkspaceRows.value.map((item) => item.id)
  const from = ids.indexOf(sourceID)
  const to = ids.indexOf(targetID)
  if (from < 0 || to < 0) return
  const next = ids.slice()
  const [moved] = next.splice(from, 1)
  next.splice(to, 0, moved)
  try {
    await saveTeamWorkspaceOrder(next)
    ElMessage.success('团队模板排序已更新')
  } catch (error) {
    ElMessage.error(parseAxiosErrorMessage(error, '保存排序失败'))
  }
}

const onTeamPresetDragEnd = () => {
  draggingTeamPresetId.value = ''
}

const runTeamWorkspaceAction = async (action, id) => {
  try {
    if (action === 'rename') {
      await renameTeamWorkspacePreset(id)
      return
    }
    if (action === 'overwrite') {
      await overwriteTeamWorkspacePresetWithCurrent(id)
      return
    }
    if (action === 'delete') {
      await deleteTeamWorkspacePreset(id)
      return
    }
    if (action === 'toggleRecommend') {
      await toggleTeamWorkspaceRecommended(id)
    }
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(parseAxiosErrorMessage(error, '团队模板操作失败'))
  }
}

const handleWorkspacePresetCommand = async (command) => {
  try {
    if (command === 'saveCurrent') {
      await saveCurrentWorkspacePreset()
      return
    }
    if (command === 'openLast') {
      const lastKey = String(lastWorkspacePresetKey.value || '').trim()
      if (!lastKey) {
        ElMessage.warning('暂无可恢复的模板')
        return
      }
      if (lastKey.startsWith('team:')) {
        const teamID = lastKey.slice(5)
        if (!teamWorkspacePresetMap.value[teamID]) {
          ElMessage.warning('上次使用的团队模板已不存在或无权限')
          return
        }
        await openTeamWorkspacePreset(teamID)
        return
      }
      if (!workspacePresetMap.value[lastKey]) {
        ElMessage.warning('上次使用的模板已不存在')
        return
      }
      await openWorkspacePreset(lastKey)
      return
    }
    if (command === 'clearCustom') {
      await clearCustomWorkspacePresets()
      return
    }
    if (command === 'exportCustom') {
      await exportCustomWorkspacePresets()
      return
    }
    if (command === 'importCustom') {
      await importCustomWorkspacePresets()
      return
    }
    if (command === 'copyShareLink') {
      await copyWorkspaceShareLink()
      return
    }
    if (command === 'saveCurrentTeam') {
      await saveCurrentAsTeamWorkspacePreset()
      return
    }
    if (command === 'refreshTeam') {
      await listTeamWorkspacePresets({ silent: false })
      ElMessage.success('团队模板已刷新')
      return
    }
    if (command === 'manageTeam') {
      teamWorkspacePanelVisible.value = true
      await listTeamWorkspacePresets({ silent: false })
      return
    }
    if (typeof command === 'string' && command.startsWith('delete:')) {
      await deleteCustomWorkspacePreset(command.slice(7))
      return
    }
    if (typeof command === 'string' && command.startsWith('deleteTeam:')) {
      await deleteTeamWorkspacePreset(command.slice(11))
      return
    }
    if (typeof command === 'string' && command.startsWith('renameTeam:')) {
      await renameTeamWorkspacePreset(command.slice(11))
      return
    }
    if (typeof command === 'string' && command.startsWith('overwriteTeam:')) {
      await overwriteTeamWorkspacePresetWithCurrent(command.slice(14))
      return
    }
    if (typeof command === 'string' && command.startsWith('openTeam:')) {
      await openTeamWorkspacePreset(command.slice(9))
      return
    }
    if (typeof command === 'string' && command.startsWith('open:')) {
      await openWorkspacePreset(command.slice(5))
    }
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(parseAxiosErrorMessage(error, '模板操作失败'))
  }
}

const handleApplyWorkspaceCategoryEvent = async (event) => {
  try {
    const detail = event?.detail || {}
    const category = String(detail.category || '').trim()
    if (!category) return
    await openWorkspaceByCategory(category, Boolean(detail.silent))
  } catch (error) {
    ElMessage.error(parseAxiosErrorMessage(error, '应用推荐工作台失败'))
  }
}

const moduleQuickLinks = [
  {
    key: 'asset',
    prefixes: ['/asset', '/host', '/cmdb', '/firewall', '/jump', '/terminal'],
    links: [
      { label: '资产总览', path: '/asset/overview', permAny: ['cmdb'] },
      { label: '资产作战台', path: '/asset/ops', permAny: ['cmdb', 'firewall', 'jump'] },
      { label: '主机管理', path: '/host', permAny: ['cmdb'] },
      { label: '网络设备', path: '/cmdb/network-devices', permAny: ['cmdb'] },
      { label: '防火墙管理', path: '/firewall', permAny: ['cmdb', 'firewall'] },
      { label: 'WebTerminal', path: '/terminal', permAny: ['terminal'] },
      { label: '堡垒机会话', path: '/jump/sessions', permAny: ['jump', 'jump:session'] }
    ]
  },
  {
    key: 'k8s',
    prefixes: ['/k8s', '/docker'],
    links: [
      { label: '平台总览', path: '/k8s/overview', permAny: ['k8s'] },
      { label: 'Docker管理', path: '/docker', permAny: ['docker'] },
      { label: 'K8s集群', path: '/k8s/clusters', permAny: ['k8s'] },
      { label: '工作负载', path: '/k8s/workloads', permAny: ['k8s'] },
      { label: 'Deployments', path: '/k8s/deployments', permAny: ['k8s'] },
      { label: 'Pods', path: '/k8s/pods', permAny: ['k8s'] },
      { label: '服务与Ingress', path: '/k8s/services', permAny: ['k8s'] }
    ]
  },
  {
    key: 'monitor',
    prefixes: ['/monitor', '/alert', '/notify', '/domain'],
    links: [
      { label: '监控告警中心', path: '/monitor/center', permAny: ['monitor', 'alert', 'notify', 'domain'] },
      { label: '域名监控中心', path: '/domain/center', permAny: ['domain'] },
      { label: '告警事件', path: '/alert/events', permAny: ['alert'], hiddenByDefault: true },
      { label: '通知渠道', path: '/notify/channels', permAny: ['notify'], hiddenByDefault: true },
      { label: '域名与证书', path: '/domain/ssl', permAny: ['domain'], hiddenByDefault: true }
    ]
  },
  {
    key: 'delivery',
    prefixes: ['/delivery', '/cicd', '/workorder', '/sqlaudit', '/gitops', '/application'],
    links: [
      { label: '交付中心', path: '/delivery/center', permAny: ['cicd', 'workorder'] },
      { label: '流水线管理', path: '/cicd/pipelines', permAny: ['cicd'], hiddenByDefault: true },
      { label: '执行记录', path: '/cicd/executions', permAny: ['cicd'], hiddenByDefault: true },
      { label: '发布管理', path: '/cicd/releases', permAny: ['cicd'], hiddenByDefault: true },
      { label: '工单管理', path: '/workorder/tickets', permAny: ['workorder'], hiddenByDefault: true },
      { label: '应用中心', path: '/application', permAny: ['application'], hiddenByDefault: true }
    ]
  },
  {
    key: 'automation',
    prefixes: ['/ai', '/workflow', '/executor', '/task', '/oncall', '/collab'],
    links: [
      { label: 'AI运维助手', path: '/ai', permAny: ['ai'] },
      { label: '工作流编排', path: '/workflow/designer', permAny: ['workflow'] },
      { label: '批量执行', path: '/executor', permAny: ['executor'] },
      { label: '任务调度', path: '/task/schedules', permAny: ['task'] },
      { label: '值班排班', path: '/oncall/schedule', permAny: ['oncall'] },
      { label: '升级策略', path: '/oncall/escalation', permAny: ['oncall'] }
    ]
  }
]

const activeModuleGroup = computed(() => {
  const path = route.path || ''
  return moduleQuickLinks.find((item) => item.prefixes.some((prefix) => path.startsWith(prefix))) || null
})

const ensureModuleState = (group) => {
  if (!group?.key) return
  const defaults = group.links.map((item) => item.path)
  const defaultsMap = new Map(group.links.map((item) => [item.path, item]))
  const existing = Array.isArray(moduleLinkState.value[group.key]) ? moduleLinkState.value[group.key] : []
  const hasCustomVisibility = existing.some((item) => Boolean(item?.hidden || item?.pinned))
  const kept = existing
    .filter((item) => defaults.includes(item?.path))
    .map((item) => {
      if (hasCustomVisibility) return item
      const meta = defaultsMap.get(item.path)
      return {
        ...item,
        hidden: Boolean(meta?.hiddenByDefault)
      }
    })
  const existingPathSet = new Set(kept.map((item) => item.path))
  defaults.forEach((path) => {
    const meta = defaultsMap.get(path) || {}
    if (!existingPathSet.has(path)) {
      kept.push({ path, pinned: false, hidden: Boolean(meta.hiddenByDefault) })
      return
    }
    const idx = kept.findIndex((item) => item.path === path)
    if (idx >= 0 && (kept[idx].hidden === undefined || kept[idx].hidden === null)) {
      kept[idx] = { ...kept[idx], hidden: Boolean(meta.hiddenByDefault) }
    }
  })
  moduleLinkState.value = { ...moduleLinkState.value, [group.key]: kept }
}

const activeModuleLinks = computed(() => {
  const group = activeModuleGroup.value
  if (!group) return []
  const allowed = group.links.filter((link) => hasAnyPerm(link.permAny))
  if (!allowed.length) return []
  const stateRows = Array.isArray(moduleLinkState.value[group.key]) ? moduleLinkState.value[group.key] : []
  const stateMap = new Map(stateRows.map((item) => [item.path, item]))
  const ordered = stateRows
    .map((item) => allowed.find((one) => one.path === item.path))
    .filter(Boolean)
  allowed.forEach((link) => {
    if (!ordered.some((one) => one.path === link.path)) ordered.push(link)
  })
  const visible = ordered
    .map((item) => ({
      ...item,
      pinned: Boolean(stateMap.get(item.path)?.pinned),
      hidden: Boolean(stateMap.get(item.path)?.hidden)
    }))
    .filter((item) => !item.hidden)
  const pinned = visible.filter((item) => item.pinned)
  const normal = visible.filter((item) => !item.pinned)
  return [...pinned, ...normal]
})

const hiddenModuleLinks = computed(() => {
  const group = activeModuleGroup.value
  if (!group) return []
  const allowed = group.links.filter((link) => hasAnyPerm(link.permAny))
  const stateRows = Array.isArray(moduleLinkState.value[group.key]) ? moduleLinkState.value[group.key] : []
  const hiddenSet = new Set(stateRows.filter((item) => item?.hidden).map((item) => item.path))
  return allowed.filter((item) => hiddenSet.has(item.path))
})

const moduleContextMenuLink = computed(() => {
  if (!moduleContextMenu.value.path) return null
  return activeModuleLinks.value.find((item) => item.path === moduleContextMenu.value.path) || null
})

const updateModuleLinkState = (mutator) => {
  const group = activeModuleGroup.value
  if (!group) return
  ensureModuleState(group)
  const base = Array.isArray(moduleLinkState.value[group.key]) ? [...moduleLinkState.value[group.key]] : []
  const next = mutator(base)
  moduleLinkState.value = { ...moduleLinkState.value, [group.key]: next }
  persistModuleLinkState()
}

const toggleModuleLinkPin = (path) => {
  updateModuleLinkState((rows) => {
    const idx = rows.findIndex((item) => item.path === path)
    if (idx < 0) return rows
    const current = rows[idx]
    rows[idx] = { ...current, pinned: !current.pinned, hidden: false }
    return rows
  })
}

const hideModuleLink = (path) => {
  if (activeModuleLinks.value.length <= 1) return
  updateModuleLinkState((rows) => {
    const idx = rows.findIndex((item) => item.path === path)
    if (idx < 0) return rows
    rows[idx] = { ...rows[idx], hidden: true, pinned: false }
    return rows
  })
}

const handleModuleLinksCommand = (command) => {
  const group = activeModuleGroup.value
  if (!group) return
  if (command === 'openAll') {
    openModuleLinksAsTabs()
    return
  }
  if (command === 'toggleCompact') {
    moduleLinksCompact.value = !moduleLinksCompact.value
    persistModuleLinksCompact()
    return
  }
  if (command === 'reset') {
    moduleLinkState.value = {
      ...moduleLinkState.value,
      [group.key]: group.links.map((item) => ({ path: item.path, pinned: false, hidden: Boolean(item.hiddenByDefault) }))
    }
    persistModuleLinkState()
    return
  }
  if (typeof command === 'string' && command.startsWith('show:')) {
    const path = command.slice(5)
    updateModuleLinkState((rows) => {
      const idx = rows.findIndex((item) => item.path === path)
      if (idx < 0) return rows
      rows[idx] = { ...rows[idx], hidden: false }
      return rows
    })
  }
}

const closeModuleLinkOthers = (path) => {
  if (!path) return
  updateModuleLinkState((rows) =>
    rows.map((item) => ({
      ...item,
      hidden: item.path === path ? false : true,
      pinned: item.path === path ? item.pinned : false
    }))
  )
}

const closeModuleContextMenu = () => {
  moduleContextMenu.value = { visible: false, x: 0, y: 0, path: '' }
}

const openModuleLinkContextMenu = (event, path) => {
  if (!path) return
  const menuWidth = 182
  const menuHeight = 260
  const x = Math.max(8, Math.min(event.clientX, window.innerWidth - menuWidth - 8))
  const y = Math.max(8, Math.min(event.clientY, window.innerHeight - menuHeight - 8))
  moduleContextMenu.value = { visible: true, x, y, path }
}

const onModuleLinkAuxClick = (event, path) => {
  if (event.button !== 1) return
  hideModuleLink(path)
}

const handleModuleContextMenuCommand = (command) => {
  const path = moduleContextMenu.value.path
  if (!path) return
  if (command === 'open') openTab(path)
  if (command === 'togglePin') toggleModuleLinkPin(path)
  if (command === 'closeCurrent') hideModuleLink(path)
  if (command === 'closeOthers') closeModuleLinkOthers(path)
  if (command === 'toggleCompact') {
    moduleLinksCompact.value = !moduleLinksCompact.value
    persistModuleLinksCompact()
  }
  if (command === 'reset') handleModuleLinksCommand('reset')
  closeModuleContextMenu()
}

const onModuleLinkDragStart = (path) => {
  draggingModuleLinkPath.value = path
}

const onModuleLinkDrop = (targetPath) => {
  const sourcePath = draggingModuleLinkPath.value
  if (!sourcePath || sourcePath === targetPath) return
  updateModuleLinkState((rows) => {
    const from = rows.findIndex((item) => item.path === sourcePath)
    const to = rows.findIndex((item) => item.path === targetPath)
    if (from < 0 || to < 0) return rows
    const [moved] = rows.splice(from, 1)
    rows.splice(to, 0, moved)
    return rows
  })
}

const onModuleLinkDragEnd = () => {
  draggingModuleLinkPath.value = ''
}

const isContextLinkActive = (path) => {
  if (route.path === path) return true
  return route.path.startsWith(path + '/')
}

const fetchUserInfo = async () => {
  const token = localStorage.getItem('token')
  if (!token) return
  try {
    const res = await axios.get('/api/v1/user/info', { headers: authHeaders() })
    if (res.data.code === 0) {
      const user = res.data.data
      currentUserID.value = user.id || ''
      username.value = user.nickname || user.username || 'Admin'
      roleCode.value = user.role?.code || ''
      localStorage.setItem('role_code', roleCode.value)
      setPermissions(user.role?.permissions?.map((p) => p.code) || [])
      localStorage.setItem('user_info', JSON.stringify(user))
    }
  } catch {
    // keep local cache
  }
}

const logout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('permissions')
  localStorage.removeItem('user_info')
  localStorage.removeItem('role_code')
  router.push('/login')
}

onMounted(async () => {
  await fetchUserInfo()
  await listTeamWorkspacePresets()
  moduleLinkState.value = readModuleLinkState()
  moduleLinksCompact.value = readModuleLinksCompact()
  customWorkspacePresets.value = readCustomWorkspacePresets()
  lastWorkspacePresetKey.value = readLastWorkspacePresetKey()
  const cached = readTabsFromStorage()
  if (cached && cached.length) {
    const hasDashboard = cached.some((item) => item.path === '/dashboard')
    viewTabs.value = hasDashboard
      ? cached
      : [{ path: '/dashboard', title: '仪表盘', pinned: false, closable: false }, ...cached]
  }
  if (activeModuleGroup.value) ensureModuleState(activeModuleGroup.value)
  ensureTab(route)
  window.addEventListener('click', closeModuleContextMenu)
  window.addEventListener('resize', closeModuleContextMenu)
  window.addEventListener('scroll', closeModuleContextMenu, true)
  window.addEventListener('lao:apply-workspace-category', handleApplyWorkspaceCategoryEvent)
  await applyWorkspaceFromQuery()
})

watch(
  () => route.path,
  () => {
    closeModuleContextMenu()
    ensureTab(route)
    if (activeModuleGroup.value) {
      ensureModuleState(activeModuleGroup.value)
      persistModuleLinkState()
    }
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  window.removeEventListener('click', closeModuleContextMenu)
  window.removeEventListener('resize', closeModuleContextMenu)
  window.removeEventListener('scroll', closeModuleContextMenu, true)
  window.removeEventListener('lao:apply-workspace-category', handleApplyWorkspaceCategoryEvent)
})
</script>

<style scoped>
.layout-container {
  height: 100vh;
  background: transparent;
}

.aside {
  margin: 14px 0 14px 14px;
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 28px;
  background:
    linear-gradient(180deg, rgba(11, 22, 39, 0.94) 0%, rgba(15, 23, 42, 0.92) 100%),
    radial-gradient(circle at top, rgba(37, 99, 235, 0.18) 0%, transparent 48%);
  color: #fff;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 24px 56px rgba(2, 6, 23, 0.24);
}

:global(html[data-theme='dark'] .aside) {
  border-color: rgba(148, 163, 184, 0.08);
  box-shadow: 0 28px 60px rgba(2, 6, 23, 0.48);
}

.logo {
  padding: 24px 20px 18px;
}

.logo-title {
  color: #f8fafc;
  font-size: 26px;
  font-weight: 700;
  letter-spacing: -0.04em;
}

.logo-subtitle {
  margin-top: 6px;
  color: rgba(226, 232, 240, 0.62);
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sider-scroll {
  flex: 1;
  min-height: 0;
  padding-bottom: 12px;
}

.el-menu-vertical {
  border-right: none;
  padding: 0 10px 12px;
}

.el-menu-vertical.is-simple {
  padding-bottom: 24px;
}

.header {
  margin: 14px 14px 0 14px;
  padding: 16px 20px;
  min-height: 74px;
  border: 1px solid var(--glass-border);
  border-radius: 24px;
  background: var(--header-bg);
  backdrop-filter: blur(22px);
  box-shadow: var(--surface-shadow);
}

.breadcrumb-stack {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.header-eyebrow {
  color: var(--muted-text);
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.user-chip {
  display: inline-flex;
  align-items: center;
  gap: 14px;
  padding: 8px 12px;
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(148, 163, 184, 0.18);
}

:global(html[data-theme='dark'] .user-chip) {
  background: rgba(15, 23, 42, 0.62);
  border-color: rgba(148, 163, 184, 0.14);
}

.user-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-meta strong {
  color: var(--el-text-color-primary);
  font-size: 14px;
}

.user-meta span {
  color: var(--muted-text);
  font-size: 12px;
  text-transform: uppercase;
}

.theme-toggle {
  border-radius: 14px;
}

.main {
  background: transparent;
  padding: 20px 14px 14px;
  overflow: auto;
}

.view-tabs-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 10px;
  padding: 8px 10px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(148, 163, 184, 0.18);
  backdrop-filter: blur(8px);
}

:global(html[data-theme='dark'] .view-tabs-wrap) {
  background: rgba(15, 23, 42, 0.52);
  border-color: rgba(148, 163, 184, 0.14);
}

.view-tabs {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 0;
}

.view-tab {
  cursor: pointer;
  user-select: none;
  border-radius: 10px;
  transition: transform 0.15s ease, box-shadow 0.2s ease;
}

.view-tab-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.view-tab-pin {
  color: #f59e0b;
  font-size: 12px;
}

.view-tab:hover {
  transform: translateY(-1px);
  box-shadow: 0 8px 16px rgba(15, 23, 42, 0.08);
}

:global(html[data-theme='dark'] .view-tab:hover) {
  box-shadow: 0 10px 18px rgba(15, 23, 42, 0.35);
}

.tab-action-btn {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
}

.workspace-scene-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 0 10px;
  padding: 8px 10px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid rgba(148, 163, 184, 0.16);
  backdrop-filter: blur(8px);
}

:global(html[data-theme='dark'] .workspace-scene-wrap) {
  background: rgba(15, 23, 42, 0.46);
  border-color: rgba(148, 163, 184, 0.14);
}

.workspace-scene-label {
  flex-shrink: 0;
  color: var(--muted-text);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.workspace-scene-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 2px;
}

.workspace-scene-btn {
  border-radius: 999px;
  border-color: rgba(59, 130, 246, 0.22);
  color: #1d4ed8;
  background: rgba(59, 130, 246, 0.06);
}

:global(html[data-theme='dark'] .workspace-scene-btn) {
  border-color: rgba(96, 165, 250, 0.28);
  color: #93c5fd;
  background: rgba(37, 99, 235, 0.15);
}

.module-context-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 0 12px;
  padding: 8px 10px;
  border-radius: 14px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.66) 0%, rgba(248, 250, 252, 0.58) 100%);
  border: 1px solid rgba(148, 163, 184, 0.24);
  backdrop-filter: blur(12px);
  --module-accent-rgb: 37, 99, 235;
}

:global(html[data-theme='dark'] .module-context-wrap) {
  background: rgba(15, 23, 42, 0.5);
  border-color: rgba(148, 163, 184, 0.14);
}

.module-context-asset {
  --module-accent-rgb: 14, 116, 244;
}

.module-context-k8s {
  --module-accent-rgb: 5, 150, 105;
}

.module-context-monitor {
  --module-accent-rgb: 219, 39, 119;
}

.module-context-delivery {
  --module-accent-rgb: 245, 158, 11;
}

.module-context-automation {
  --module-accent-rgb: 99, 102, 241;
}

.module-context-wrap.is-compact {
  padding: 6px 8px;
  gap: 8px;
}

.module-context-wrap.is-compact .module-context-label {
  display: none;
}

.module-context-wrap :deep(.el-scrollbar) {
  flex: 1;
  min-width: 0;
}

.module-context-wrap :deep(.el-scrollbar__view) {
  min-width: max-content;
}

.module-context-label {
  flex-shrink: 0;
  color: rgba(var(--module-accent-rgb), 0.78);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  padding: 0 6px;
}

.module-context-links {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 1px 0;
}

.module-context-tag {
  position: relative;
  cursor: pointer;
  user-select: none;
  border-radius: 10px 10px 0 0;
  border: 1px solid rgba(148, 163, 184, 0.26);
  border-bottom-color: transparent;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.78) 0%, rgba(241, 245, 249, 0.6) 100%);
  --el-tag-bg-color: transparent;
  --el-tag-border-color: transparent;
  --el-tag-text-color: var(--el-text-color-secondary);
  transition: border-color 0.18s ease, background-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;
}

:global(html[data-theme='dark'] .module-context-tag) {
  border-color: rgba(100, 116, 139, 0.4);
  border-bottom-color: transparent;
  background: linear-gradient(180deg, rgba(30, 41, 59, 0.74) 0%, rgba(15, 23, 42, 0.64) 100%);
}

.module-context-tag:hover {
  transform: translateY(-1px);
  border-color: rgba(var(--module-accent-rgb), 0.4);
  border-bottom-color: transparent;
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.1);
}

.module-context-tag.is-active {
  border-color: rgba(var(--module-accent-rgb), 0.48);
  border-bottom-color: rgba(var(--module-accent-rgb), 0.22);
  background: linear-gradient(180deg, rgba(var(--module-accent-rgb), 0.18) 0%, rgba(var(--module-accent-rgb), 0.09) 100%);
  --el-tag-text-color: rgb(var(--module-accent-rgb));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.4);
}

.module-context-tag.is-pinned::before {
  content: '';
  position: absolute;
  left: 10px;
  top: 50%;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  transform: translateY(-50%);
  background: rgb(var(--module-accent-rgb));
}

.module-context-tag-inner {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
}

.module-context-wrap.is-compact .module-context-tag-inner {
  padding: 3px 8px;
}

.module-context-tag-title {
  max-width: 136px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.module-context-tag-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 0;
  opacity: 0;
  overflow: hidden;
  pointer-events: none;
  transition: max-width 0.2s ease, opacity 0.2s ease;
}

.module-context-tag:hover .module-context-tag-actions,
.module-context-tag.is-active .module-context-tag-actions,
.module-context-tag.is-pinned .module-context-tag-actions {
  max-width: 56px;
  opacity: 1;
  pointer-events: auto;
}

.module-pin-icon {
  font-size: 12px;
  color: var(--muted-text);
}

.module-context-tag.is-pinned .module-pin-icon,
.module-context-tag.is-active .module-pin-icon {
  color: rgb(var(--module-accent-rgb));
}

.module-close-icon {
  font-size: 12px;
  color: var(--muted-text);
}

.module-context-action {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  border-radius: 12px;
}

.module-context-empty {
  color: var(--muted-text);
  font-size: 12px;
  padding: 0 4px;
}

.module-context-menu {
  position: fixed;
  z-index: 3999;
  width: 182px;
  padding: 6px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.96);
  border: 1px solid rgba(148, 163, 184, 0.2);
  box-shadow: 0 18px 45px rgba(2, 6, 23, 0.22);
  backdrop-filter: blur(10px);
}

:global(html[data-theme='dark'] .module-context-menu) {
  background: rgba(15, 23, 42, 0.94);
  border-color: rgba(100, 116, 139, 0.36);
  box-shadow: 0 20px 48px rgba(2, 6, 23, 0.52);
}

.module-context-menu-item {
  width: 100%;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-regular);
  text-align: left;
  font-size: 13px;
  line-height: 34px;
  border-radius: 8px;
  padding: 0 10px;
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.module-context-menu-item:hover:not(:disabled) {
  background: rgba(59, 130, 246, 0.1);
  color: rgb(37, 99, 235);
}

.module-context-menu-item:disabled {
  opacity: 0.42;
  cursor: not-allowed;
}

.module-context-menu-item.danger:hover {
  background: rgba(239, 68, 68, 0.12);
  color: rgb(220, 38, 38);
}

.module-context-menu-fade-enter-active,
.module-context-menu-fade-leave-active {
  transition: opacity 0.14s ease, transform 0.14s ease;
}

.module-context-menu-fade-enter-from,
.module-context-menu-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}

:deep(.team-workspace-dialog .el-dialog__body) {
  padding-top: 14px;
}

.team-workspace-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.team-workspace-search {
  width: 260px;
}

.team-workspace-category {
  width: 140px;
}

.team-workspace-actions {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  flex-wrap: wrap;
}

.team-workspace-footer {
  width: 100%;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.team-sort-board {
  margin-top: 14px;
  border: 1px dashed rgba(148, 163, 184, 0.5);
  border-radius: 12px;
  padding: 10px 12px;
  background: rgba(148, 163, 184, 0.06);
}

.team-sort-title {
  font-size: 12px;
  color: var(--muted-text);
  margin-bottom: 8px;
}

.team-sort-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 180px;
  overflow: auto;
}

.team-sort-item {
  display: flex;
  align-items: center;
  gap: 10px;
  border: 1px solid rgba(148, 163, 184, 0.36);
  background: rgba(255, 255, 255, 0.84);
  border-radius: 10px;
  padding: 8px 10px;
  cursor: grab;
}

.team-sort-item:active {
  cursor: grabbing;
}

.team-sort-handle {
  font-size: 14px;
  line-height: 1;
  color: var(--muted-text);
}

.team-sort-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.team-sort-meta {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.muted-text {
  color: var(--muted-text);
}

.page-view {
  min-height: calc(100vh - 122px);
}

.app-route-fade-enter-active,
.app-route-fade-leave-active {
  transition: opacity 0.24s ease, transform 0.24s ease;
}

.app-route-fade-enter-from,
.app-route-fade-leave-to {
  opacity: 0;
  transform: translateY(6px);
}

:deep(.el-menu) {
  border-right: none;
}

:deep(.el-menu-item),
:deep(.el-sub-menu__title) {
  height: 44px;
  line-height: 44px;
  border-radius: 14px;
  margin: 4px 0;
  transition: background-color 0.18s ease, color 0.18s ease, transform 0.18s ease;
}

:deep(.el-sub-menu .el-menu-item) {
  margin-left: 6px;
}

:deep(.el-menu-item:hover),
:deep(.el-sub-menu__title:hover) {
  background-color: rgba(255, 255, 255, 0.08) !important;
}

:deep(.el-menu-item.is-active) {
  background: linear-gradient(90deg, rgba(36, 146, 255, 0.3) 0%, rgba(36, 146, 255, 0.12) 100%) !important;
  color: #ffffff !important;
  box-shadow: inset 0 0 0 1px rgba(125, 189, 255, 0.12);
}

:deep(.el-dropdown-link) {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  color: var(--el-text-color-primary);
}

@media (max-width: 1200px) {
  .header {
    align-items: flex-start;
  }

  .header-left,
  .header-right {
    width: 100%;
  }

  .header-right {
    justify-content: flex-end;
  }
}

@media (max-width: 768px) {
  .aside {
    margin: 10px 0 10px 10px;
    width: 228px !important;
  }

  .main {
    padding: 14px 10px 10px;
  }

  .header {
    margin: 10px 10px 0 10px;
    padding: 14px;
  }

  .header-right {
    justify-content: flex-start;
  }
}
</style>
