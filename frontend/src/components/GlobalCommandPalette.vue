<template>
  <el-dialog
    v-model="visible"
    title=""
    width="640px"
    class="command-palette-dialog"
    :show-close="false"
    :append-to-body="true"
    @opened="onOpened"
  >
    <div class="palette-container">
      <div class="palette-search-bar">
        <el-icon class="search-icon"><Search /></el-icon>
        <input
          ref="searchInputRef"
          v-model="query"
          type="text"
          class="palette-input"
          placeholder="搜索资产、域名、告警、工单或菜单 (按 Esc 退出)..."
          @keydown.down.prevent="navigateResults(1)"
          @keydown.up.prevent="navigateResults(-1)"
          @keydown.enter.prevent="selectCurrent"
        />
        <span class="shortcut-tag">Cmd + K</span>
      </div>

      <div class="palette-results" v-loading="loading">
        <template v-if="filteredResults.length > 0">
          <div
            v-for="(item, index) in filteredResults"
            :key="item.id || index"
            :class="['palette-item', { active: selectedIndex === index }]"
            @mouseenter="selectedIndex = index"
            @click="executeItem(item)"
          >
            <div class="item-icon">
              <el-icon><component :is="item.icon || 'Document'" /></el-icon>
            </div>
            <div class="item-info">
              <span class="item-title">{{ item.title }}</span>
              <span class="item-sub">{{ item.subtitle }}</span>
            </div>
            <span class="item-category">{{ item.category }}</span>
          </div>
        </template>

        <div v-else class="palette-empty">
          <el-empty description="未找到匹配的资源或指令" :image-size="80" />
        </div>
      </div>

      <div class="palette-footer">
        <span class="footer-tip"><kbd>↑</kbd> <kbd>↓</kbd> 切换选项</span>
        <span class="footer-tip"><kbd>↵</kbd> 跳转执行</span>
        <span class="footer-tip"><kbd>Esc</kbd> 关闭</span>
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { Search } from '@element-plus/icons-vue'

const router = useRouter()
const visible = ref(false)
const query = ref('')
const loading = ref(false)
const selectedIndex = ref(0)
const searchInputRef = ref(null)

const menuItems = [
  { id: 'm-dashboard', title: '仪表盘', subtitle: '监控指标与系统概览', category: '菜单', icon: 'Menu', path: '/dashboard' },
  { id: 'm-host', title: '主机管理', subtitle: 'CMDB 主机与服务器列表', category: '菜单', icon: 'Coin', path: '/host' },
  { id: 'm-cloud', title: '云资源', subtitle: '阿里云/腾讯云资产纳管', category: '菜单', icon: 'Cloudy', path: '/cmdb/cloud' },
  { id: 'm-k8s', title: 'K8s集群', subtitle: 'Kubernetes 集群控制台', category: '菜单', icon: 'Platform', path: '/k8s/clusters' },
  { id: 'm-docker', title: 'Docker容器', subtitle: '容器实例与镜像管理', category: '菜单', icon: 'Box', path: '/docker' },
  { id: 'm-events', title: '告警事件', subtitle: '实时告警通知与认领', category: '菜单', icon: 'Bell', path: '/alert/events' },
  { id: 'm-ssl', title: '域名证书', subtitle: 'SSL 证书与域名到期监控', category: '菜单', icon: 'Lock', path: '/domain/ssl' },
  { id: 'm-tickets', title: '工单管理', subtitle: '运维服务目录与工单审批', category: '菜单', icon: 'Tickets', path: '/workorder/orders' },
  { id: 'm-terminal', title: '堡垒机接入', subtitle: 'Web SSH 终端与审计', category: '菜单', icon: 'Monitor', path: '/jump/assets' },
  { id: 'm-cost', title: '成本分析', subtitle: '多云消费与预算概览', category: '菜单', icon: 'Money', path: '/cost/overview' }
]

const dynamicItems = ref([])

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const loadDynamicItems = async () => {
  if (dynamicItems.value.length > 0) return
  loading.value = true
  try {
    const list = []
    const [hostsRes, certsRes] = await Promise.allSettled([
      axios.get('/api/v1/cmdb/hosts?page=1&page_size=20', { headers: authHeaders() }),
      axios.get('/api/v1/domain/certs', { headers: authHeaders() })
    ])

    if (hostsRes.status === 'fulfilled' && hostsRes.value.data?.code === 0) {
      const hosts = hostsRes.value.data.data?.items || []
      hosts.forEach(h => {
        list.push({
          id: `host-${h.id}`,
          title: `${h.name || h.hostname} (${h.ip})`,
          subtitle: `OS: ${h.os || 'Linux'} | CPU: ${h.cpu_cores || 4}核 | 内存: ${h.memory || 8}G`,
          category: '主机',
          icon: 'Monitor',
          path: '/host'
        })
      })
    }

    if (certsRes.status === 'fulfilled' && certsRes.value.data?.code === 0) {
      const certs = certsRes.value.data.data || []
      certs.forEach(c => {
        list.push({
          id: `cert-${c.id}`,
          title: c.domain,
          subtitle: `颁发者: ${c.issuer} | 剩余: ${c.days_to_expire || 0}天`,
          category: '域名',
          icon: 'Lock',
          path: '/domain/ssl'
        })
      })
    }

    dynamicItems.value = list
  } catch (e) {
  } finally {
    loading.value = false
  }
}

const filteredResults = computed(() => {
  const q = query.value.trim().toLowerCase()
  const all = [...menuItems, ...dynamicItems.value]
  if (!q) return all.slice(0, 8)
  return all.filter(item => 
    item.title.toLowerCase().includes(q) ||
    item.subtitle.toLowerCase().includes(q) ||
    item.category.toLowerCase().includes(q)
  ).slice(0, 10)
})

const navigateResults = (direction) => {
  const max = filteredResults.value.length
  if (max === 0) return
  selectedIndex.value = (selectedIndex.value + direction + max) % max
}

const executeItem = (item) => {
  if (!item) return
  visible.value = false
  if (item.path) {
    router.push(item.path)
  }
}

const selectCurrent = () => {
  const item = filteredResults.value[selectedIndex.value]
  if (item) executeItem(item)
}

const onOpened = () => {
  searchInputRef.value?.focus()
  loadDynamicItems()
}

const handleKeydown = (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    visible.value = !visible.value
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})

defineExpose({
  open: () => {
    visible.value = true
  }
})
</script>

<style scoped>
:deep(.command-palette-dialog) {
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.24);
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
}

:deep(.command-palette-dialog .el-dialog__header) {
  display: none;
}

:deep(.command-palette-dialog .el-dialog__body) {
  padding: 0;
}

.palette-container {
  display: flex;
  flex-direction: column;
  color: #1e293b;
}

.palette-search-bar {
  display: flex;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(226, 232, 240, 0.8);
  gap: 12px;
}

.search-icon {
  font-size: 20px;
  color: #64748b;
}

.palette-input {
  flex: 1;
  border: none;
  background: transparent;
  outline: none;
  font-size: 16px;
  color: inherit;
}

.shortcut-tag {
  background: rgba(241, 245, 249, 0.9);
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  padding: 2px 8px;
  font-size: 12px;
  color: #64748b;
  font-family: monospace;
}

.palette-results {
  max-height: 380px;
  overflow-y: auto;
  padding: 8px 12px;
}

.palette-item {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.15s ease;
  gap: 12px;

  &.active {
    background: #3b82f6;
    color: #ffffff;

    .item-sub {
      color: rgba(255, 255, 255, 0.8);
    }
    .item-category {
      background: rgba(255, 255, 255, 0.2);
      color: #ffffff;
    }
  }
}

.item-icon {
  font-size: 18px;
  display: flex;
  align-items: center;
}

.item-info {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.item-title {
  font-size: 14px;
  font-weight: 600;
}

.item-sub {
  font-size: 12px;
  color: #64748b;
}

.item-category {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  background: #f1f5f9;
  color: #475569;
}

.palette-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16px;
  padding: 10px 20px;
  border-top: 1px solid rgba(226, 232, 240, 0.8);
  background: rgba(248, 250, 252, 0.7);
  font-size: 12px;
  color: #64748b;

  kbd {
    background: #ffffff;
    border: 1px solid #cbd5e1;
    border-radius: 4px;
    padding: 1px 5px;
    font-size: 11px;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  }
}

.palette-empty {
  padding: 24px 0;
}
</style>
