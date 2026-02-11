<template>
  <el-container class="layout-container">
    <el-aside width="220px" class="aside">
      <div class="logo">
        <el-icon><ElementPlus /></el-icon>
        <span>Lazy Auto Ops</span>
      </div>
      <el-menu
        router
        :default-active="$route.path"
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
        class="el-menu-vertical"
      >
        <el-menu-item index="/dashboard">
          <el-icon><Odometer /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>

        <el-menu-item index="/ai">
          <el-icon><MagicStick /></el-icon>
          <span>AI运维助手</span>
        </el-menu-item>

        <el-sub-menu index="/cmdb">
          <template #title>
            <el-icon><Monitor /></el-icon>
            <span>资产管理</span>
          </template>
          <el-menu-item index="/host">主机管理</el-menu-item>
          <el-menu-item index="/cmdb/group">主机分组</el-menu-item>
          <el-menu-item index="/cmdb/credential">凭据管理</el-menu-item>
          <el-menu-item index="/cmdb/database">数据库资产</el-menu-item>
          <el-menu-item index="/cmdb/cloud">云资源</el-menu-item>
          <el-menu-item index="/firewall">防火墙管理</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="/k8s">
          <template #title>
            <el-icon><Platform /></el-icon>
            <span>容器与K8s</span>
          </template>
          <el-menu-item index="/docker">Docker管理</el-menu-item>
          <el-menu-item index="/k8s/clusters">K8s集群</el-menu-item>
          <el-menu-item index="/k8s/namespaces">命名空间</el-menu-item>
          <el-menu-item index="/k8s/workloads">工作负载</el-menu-item>
          <el-menu-item index="/k8s/pods">Pods</el-menu-item>
          <el-menu-item index="/k8s/services">服务与Ingress</el-menu-item>
          <el-menu-item index="/k8s/configs">Config/Secret</el-menu-item>
          <el-menu-item index="/k8s/storage">存储管理</el-menu-item>
          <el-menu-item index="/k8s/nodes">节点管理</el-menu-item>
          <el-menu-item index="/k8s/events">事件与诊断</el-menu-item>
          <el-menu-item index="/k8s/terminal">K8s WebShell</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="/monitor">
          <template #title>
            <el-icon><Histogram /></el-icon>
            <span>监控告警</span>
          </template>
          <el-menu-item index="/monitor/overview">监控概览</el-menu-item>
          <el-menu-item index="/monitor/metrics">指标采集</el-menu-item>
          <el-menu-item index="/monitor/containers">容器监控</el-menu-item>
          <el-menu-item index="/monitor/pods">Pod监控</el-menu-item>
          <el-menu-item index="/monitor/agents">Agent心跳</el-menu-item>
          <el-menu-item index="/alert/rules">告警规则</el-menu-item>
          <el-menu-item index="/alert/events">告警事件</el-menu-item>
          <el-menu-item index="/alert/silences">告警静默</el-menu-item>
          <el-menu-item index="/alert/aggregation">告警聚合</el-menu-item>
          <el-menu-item index="/alert/history">告警复盘</el-menu-item>
          <el-menu-item index="/notify/channels">通知渠道</el-menu-item>
          <el-menu-item index="/notify/groups">通知组</el-menu-item>
          <el-menu-item index="/notify/templates">通知模板</el-menu-item>
          <el-menu-item index="/domain/ssl">域名与证书</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="/automation">
          <template #title>
            <el-icon><Operation /></el-icon>
            <span>自动化</span>
          </template>
          <el-menu-item index="/workflow/designer">工作流编排</el-menu-item>
          <el-menu-item index="/executor">批量执行</el-menu-item>
          <el-menu-item index="/task/schedules">任务调度</el-menu-item>
          <el-menu-item index="/ansible/playbooks">Ansible Playbook</el-menu-item>
          <el-menu-item index="/ansible/inventories">Ansible Inventory</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="/cicd">
          <template #title>
            <el-icon><Connection /></el-icon>
            <span>CI/CD</span>
          </template>
          <el-menu-item index="/cicd/pipelines">流水线管理</el-menu-item>
          <el-menu-item index="/cicd/executions">执行记录</el-menu-item>
          <el-menu-item index="/cicd/schedules">定时发布</el-menu-item>
          <el-menu-item index="/cicd/releases">发布管理</el-menu-item>
          <el-menu-item index="/application">应用中心</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="/nacos">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>配置中心</span>
          </template>
          <el-menu-item index="/nacos/servers">Nacos服务器</el-menu-item>
          <el-menu-item index="/nacos/configs">配置管理</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="/change">
          <template #title>
            <el-icon><Tickets /></el-icon>
            <span>变更管理</span>
          </template>
          <el-menu-item index="/workorder/tickets">工单管理</el-menu-item>
          <el-menu-item index="/workorder/types">工单类型</el-menu-item>
          <el-menu-item index="/sqlaudit/requests">SQL工单</el-menu-item>
          <el-menu-item index="/sqlaudit/rules">SQL审核规则</el-menu-item>
          <el-menu-item index="/gitops/repos">GitOps仓库</el-menu-item>
          <el-menu-item index="/gitops/sync">同步记录</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="/collab">
          <template #title>
            <el-icon><User /></el-icon>
            <span>协作</span>
          </template>
          <el-menu-item index="/oncall/schedule">值班排班</el-menu-item>
          <el-menu-item index="/oncall/escalation">升级策略</el-menu-item>
          <el-menu-item index="/terminal">WebTerminal</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="/visual">
          <template #title>
            <el-icon><Share /></el-icon>
            <span>可视化</span>
          </template>
          <el-menu-item index="/topology">服务拓扑</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="/cost">
          <template #title>
            <el-icon><Coin /></el-icon>
            <span>成本</span>
          </template>
          <el-menu-item index="/cost/overview">成本概览</el-menu-item>
          <el-menu-item index="/cost/budget">预算与告警</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="/system">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>系统管理</span>
          </template>
          <el-menu-item index="/system/users">用户管理</el-menu-item>
          <el-menu-item index="/system/roles">角色管理</el-menu-item>
          <el-menu-item index="/system/menus">菜单管理</el-menu-item>
          <el-menu-item index="/system/dept">部门管理</el-menu-item>
          <el-menu-item index="/system/posts">岗位管理</el-menu-item>
          <el-menu-item index="/system/login-logs">登录日志</el-menu-item>
          <el-menu-item index="/system/audit-logs">操作日志</el-menu-item>
          <el-menu-item index="/system/captcha">验证码配置</el-menu-item>
        </el-sub-menu>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item>{{ $route.meta.title }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-dropdown>
            <span class="el-dropdown-link">
              Admin <el-icon class="el-icon--right"><arrow-down /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <el-main class="main">
        <router-view v-slot="{ Component, route }">
          <component :is="Component" :key="route.fullPath" />
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { useRouter } from 'vue-router'

const router = useRouter()

const logout = () => {
  localStorage.removeItem('token')
  router.push('/login')
}
</script>

<style scoped>
.layout-container { height: 100vh; }
.aside { background-color: #304156; color: white; display: flex; flex-direction: column; }
.logo { height: 60px; line-height: 60px; text-align: center; font-size: 20px; font-weight: bold; background-color: #2b2f3a; display: flex; align-items: center; justify-content: center; gap: 10px;}
.el-menu-vertical { border-right: none; }
.header { background: white; border-bottom: 1px solid #dcdfe6; display: flex; align-items: center; justify-content: space-between; padding: 0 20px; box-shadow: 0 1px 4px rgba(0,21,41,.08); }
.main { background-color: #f0f2f5; padding: 20px; }
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
