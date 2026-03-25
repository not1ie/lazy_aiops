<template>
  <el-card class="asset-host-hub">
    <div class="hub-header">
      <div>
        <h2>资产管理中心</h2>
        <p>主机、分组、凭据、数据库、云资源、网络设备、防火墙与堡垒机资产统一入口。</p>
      </div>
    </div>

    <div class="hub-tabs-wrap">
      <el-tabs v-model="activeTab" @tab-click="handleTabClick">
        <el-tab-pane
          v-for="tab in visibleTabs"
          :key="tab.key"
          :name="tab.key"
          :label="tab.label"
        />
      </el-tabs>
    </div>

    <div class="hub-panel">
      <component :is="activeComponent" />
    </div>
  </el-card>
</template>

<script setup>
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const panels = [
  {
    key: 'host',
    label: '主机管理',
    permAny: ['cmdb'],
    component: defineAsyncComponent(() => import('@/views/cmdb/host-main.vue'))
  },
  {
    key: 'group',
    label: '主机分组',
    permAny: ['cmdb'],
    component: defineAsyncComponent(() => import('@/views/cmdb/group.vue'))
  },
  {
    key: 'credential',
    label: '凭据管理',
    permAny: ['cmdb'],
    component: defineAsyncComponent(() => import('@/views/cmdb/credential.vue'))
  },
  {
    key: 'database',
    label: '数据库资产',
    permAny: ['cmdb'],
    component: defineAsyncComponent(() => import('@/views/cmdb/database.vue'))
  },
  {
    key: 'cloud',
    label: '云资源',
    permAny: ['cmdb'],
    component: defineAsyncComponent(() => import('@/views/cmdb/cloud.vue'))
  },
  {
    key: 'network',
    label: '网络设备',
    permAny: ['cmdb'],
    component: defineAsyncComponent(() => import('@/views/cmdb/network-device.vue'))
  },
  {
    key: 'firewall',
    label: '防火墙管理',
    permAny: ['cmdb', 'firewall'],
    component: defineAsyncComponent(() => import('@/views/firewall/index.vue'))
  },
  {
    key: 'jump-assets',
    label: '堡垒机资产',
    permAny: ['jump', 'jump:asset'],
    component: defineAsyncComponent(() => import('@/views/jump/assets.vue'))
  }
]

const isAdmin = computed(() => {
  const roleCode = String(localStorage.getItem('role_code') || '').toLowerCase()
  return roleCode === 'admin'
})

const hasAnyPerm = (codes = []) => {
  if (isAdmin.value) return true
  let perms = []
  try {
    const raw = JSON.parse(localStorage.getItem('permissions') || '[]')
    perms = Array.isArray(raw) ? raw : []
  } catch {
    perms = []
  }
  return codes.some((code) => perms.includes(code))
}

const visibleTabs = computed(() => panels.filter((tab) => hasAnyPerm(tab.permAny)))

const activeTab = ref('host')

const syncTabFromRoute = () => {
  const available = visibleTabs.value
  if (!available.length) {
    activeTab.value = ''
    return
  }
  const queryTab = String(route.query?.tab || '').trim()
  const matched = available.find((item) => item.key === queryTab)
  activeTab.value = matched ? matched.key : available[0].key
}

const updateRouteTab = (tab) => {
  if (!tab) return
  if (String(route.query?.tab || '') === tab) return
  router.replace({
    path: '/host',
    query: {
      ...route.query,
      tab
    }
  })
}

watch(
  () => route.query.tab,
  () => syncTabFromRoute(),
  { immediate: true }
)

watch(
  () => visibleTabs.value.map((item) => item.key).join(','),
  () => {
    syncTabFromRoute()
    if (activeTab.value) updateRouteTab(activeTab.value)
  },
  { immediate: true }
)

const handleTabClick = (pane) => {
  const tab = String(pane?.paneName || '').trim()
  if (!tab) return
  activeTab.value = tab
  updateRouteTab(tab)
}

const activeComponent = computed(() => {
  const tab = visibleTabs.value.find((item) => item.key === activeTab.value)
  return tab?.component || null
})
</script>

<style scoped>
.asset-host-hub {
  border-radius: 16px;
}

.hub-header {
  margin-bottom: 8px;
}

.hub-header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
}

.hub-header p {
  margin: 8px 0 0;
  color: var(--el-text-color-secondary);
}

.hub-tabs-wrap {
  margin-bottom: 8px;
}

.hub-panel {
  min-height: 240px;
}

.hub-panel :deep(.page-card) {
  box-shadow: none;
  border: none;
}

.hub-panel :deep(.el-card) {
  border-radius: 12px;
}
</style>
