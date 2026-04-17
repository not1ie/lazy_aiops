<template>
  <el-card class="asset-host-hub">
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
    label: '网络与防火墙',
    permAny: ['cmdb'],
    component: defineAsyncComponent(() => import('@/views/cmdb/network-device.vue'))
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
  const normalizedTabMap = {
    firewall: 'network',
    group: 'host'
  }
  const normalizedTab = normalizedTabMap[queryTab] || queryTab
  const matched = available.find((item) => item.key === normalizedTab)
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

const activeComponent = computed(() => {
  const tab = visibleTabs.value.find((item) => item.key === activeTab.value)
  return tab?.component || null
})
</script>

<style scoped>
.asset-host-hub {
  border-radius: 16px;
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
