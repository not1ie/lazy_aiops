<template>
  <div class="log-query-container" v-loading="loading">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-title">
        <div class="icon-box"><el-icon><Search /></el-icon></div>
        <div>
          <h2>日志查询</h2>
          <p>统一日志检索与多维分析中心</p>
        </div>
      </div>
      
      <!-- Metrics row matching top-right of screenshot -->
      <div class="metrics-row">
        <div class="metric-card">
          <div class="label">命中日志</div>
          <div class="value text-primary">{{ formatNumber(totalLogs) }}</div>
        </div>
        <div class="metric-card">
          <div class="label">本次返回</div>
          <div class="value text-success">{{ logs.length }}</div>
        </div>
        <div class="metric-card">
          <div class="label">耗时</div>
          <div class="value text-warning">{{ elapsedMs }}ms</div>
        </div>
      </div>
    </div>

    <!-- Warning banner if no cluster/group connected -->
    <el-alert
      v-if="selectedProject === 'local-simulation'"
      title="提示：系统当前尚未检测到您已接入的 Kubernetes 集群或 CMDB 主机，已自动启用本地默认环境进行日志查询仿真。"
      type="info"
      show-icon
      class="warning-banner"
      style="margin-bottom: 16px;"
    >
      <template #default>
        您可以前往 <el-link type="primary" href="/k8s/clusters">接入 K8s 集群</el-link> 或 <el-link type="primary" href="/host">添加主机</el-link> 后配置真实日志数据源。
      </template>
    </el-alert>

    <!-- Filter & Query Control Card -->
    <el-card class="query-card">
      <div class="filter-row">
        <div class="filter-item">
          <span class="filter-label">项目/环境</span>
          <el-select v-model="selectedProject" placeholder="选择项目/环境" style="width: 280px" @change="onProjectChange">
            <el-option v-for="p in projectOptions" :key="p.value" :label="p.label" :value="p.value" />
          </el-select>
        </div>

        <!-- Dynamic K8s Sub-Selectors -->
        <template v-if="projectType === 'k8s'">
          <div class="filter-item" style="margin-left: 16px">
            <span class="filter-label">空间</span>
            <el-select v-model="k8sNamespace" placeholder="选择空间" style="width: 150px" @change="onNamespaceChange">
              <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
            </el-select>
          </div>
          <div class="filter-item" style="margin-left: 16px">
            <span class="filter-label">Pod容器</span>
            <el-select v-model="k8sPod" placeholder="选择Pod" style="width: 240px" filterable @change="handleSearch">
              <el-option v-for="p in pods" :key="p.name" :label="p.name" :value="p.name" />
            </el-select>
          </div>
        </template>

        <!-- Dynamic CMDB Host Sub-Selectors -->
        <template v-else-if="projectType === 'host'">
          <div class="filter-item" style="margin-left: 16px">
            <span class="filter-label">日志路径</span>
            <el-select v-model="hostFilePath" placeholder="文件路径" style="width: 320px" filterable allow-create default-first-option @change="handleSearch">
              <el-option label="/var/log/messages" value="/var/log/messages" />
              <el-option label="/var/log/syslog" value="/var/log/syslog" />
              <el-option label="/var/log/nginx/access.log" value="/var/log/nginx/access.log" />
              <el-option label="/var/log/nginx/error.log" value="/var/log/nginx/error.log" />
            </el-select>
          </div>
        </template>

        <!-- Standard Log Database Libraries -->
        <template v-else>
          <div class="filter-item" style="margin-left: 16px">
            <span class="filter-label">日志库</span>
            <el-select v-model="datasource" placeholder="选择日志库" style="width: 240px" @change="handleSearch">
              <el-option v-for="lib in libraries" :key="lib.id" :label="`${lib.name} (${lib.type.toUpperCase()})`" :value="lib.id" />
            </el-select>
          </div>
        </template>
      </div>

      <div class="query-bar">
        <span class="filter-label">查询语句</span>
        <el-input 
          v-model="queryStr" 
          placeholder='请输入关键字或查询语句' 
          class="query-input"
          clearable
          @keyup.enter="handleSearch"
        />
      </div>

      <div class="time-limit-row">
        <div class="time-picker-box">
          <span class="filter-label">时间范围</span>
          <el-date-picker
            v-model="timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 380px"
          />
        </div>
        
        <div class="granularity-box" style="margin-left: 16px">
          <span class="filter-label">粒度</span>
          <el-select v-model="granularity" style="width: 100px">
            <el-option label="30秒" value="30s" />
            <el-option label="1分钟" value="1m" />
            <el-option label="5分钟" value="5m" />
            <el-option label="1小时" value="1h" />
          </el-select>
        </div>

        <div class="limit-box" style="margin-left: 16px">
          <span class="filter-label">条数</span>
          <el-input-number v-model="limit" :min="1" :max="10000" controls-position="right" style="width: 110px" />
        </div>

        <div class="button-box" style="margin-left: auto">
          <el-button type="primary" icon="Search" @click="handleSearch" :loading="searchLoading">查询 / 分析</el-button>
          <el-button type="warning" @click="clearForm">清空条件</el-button>
        </div>
      </div>

      <!-- Presets below date picker matching screenshot -->
      <div class="presets-row">
        <el-button-group size="small">
          <el-button v-for="p in timePresets" :key="p.label" @click="applyPreset(p)">{{ p.label }}</el-button>
        </el-button-group>
        <el-button size="small" icon="Refresh" @click="handleSearch" style="margin-left: 12px">刷新</el-button>
      </div>
    </el-card>

    <!-- Chart & Actions Section -->
    <div class="chart-section" v-if="hasSearched">
      <div class="chart-header">
        <span class="chart-title">日志条数：<b>{{ totalLogs }}</b></span>
        <div class="chart-actions">
          <el-button link type="primary" size="small"><el-icon><Plus /></el-icon> 加入仪表盘</el-button>
          <el-button link type="primary" size="small"><el-icon><Bell /></el-icon> 添加告警</el-button>
          <el-button link type="primary" size="small"><el-icon><Download /></el-icon> 下载</el-button>
        </div>
      </div>
      <!-- Real ECharts timeline bar chart -->
      <div class="chart-box">
        <div ref="chartRef" style="width: 100%; height: 180px;"></div>
      </div>
    </div>

    <!-- Main Workspace Layout -->
    <el-row :gutter="20" class="workspace-layout" v-if="hasSearched">
      <!-- Left Sidebar: Available Fields -->
      <el-col :span="5">
        <el-card class="field-sidebar-card" shadow="never">
          <div class="sidebar-title">可用字段 <el-button link type="primary" size="small" icon="Refresh" @click="handleSearch" style="float: right">刷新</el-button></div>
          <el-input v-model="fieldSearch" placeholder="搜索字段" prefix-icon="Search" size="small" clearable @input="filterFields" style="margin-bottom: 12px;" />
          
          <div class="field-group">
            <div class="group-header">显示字段 <span class="badge">5</span></div>
            <el-checkbox-group v-model="selectedFields" class="field-list">
              <el-checkbox label="_time" class="field-item">
                <span class="field-name">_time</span>
                <span class="percent">100%</span>
              </el-checkbox>
              <el-checkbox label="_msg" class="field-item">
                <span class="field-name">_msg</span>
                <span class="percent">100%</span>
              </el-checkbox>
              <el-checkbox label="message" class="field-item">
                <span class="field-name">message</span>
                <span class="percent">100%</span>
              </el-checkbox>
              <el-checkbox label="host_name" class="field-item">
                <span class="field-name">host_name</span>
                <span class="percent">100%</span>
              </el-checkbox>
              <el-checkbox label="file_path" class="field-item">
                <span class="field-name">file_path</span>
                <span class="percent">100%</span>
              </el-checkbox>
            </el-checkbox-group>
          </div>

          <div class="field-group">
            <div class="group-header">索引字段 <span class="badge">3</span></div>
            <el-checkbox-group v-model="selectedFields" class="field-list">
              <el-checkbox label="env" class="field-item">
                <span class="field-name">env</span>
                <span class="percent">100%</span>
              </el-checkbox>
              <el-checkbox label="app" class="field-item">
                <span class="field-name">app</span>
                <span class="percent">100%</span>
              </el-checkbox>
              <el-checkbox label="log_type" class="field-item">
                <span class="field-name">log_type</span>
                <span class="percent">100%</span>
              </el-checkbox>
            </el-checkbox-group>
          </div>

          <div class="field-group">
            <el-collapse>
              <el-collapse-item title="其它字段 (1)" name="other">
                <el-checkbox-group v-model="selectedFields" class="field-list">
                  <el-checkbox label="task" class="field-item">
                    <span class="field-name">task</span>
                    <span class="percent">80%</span>
                  </el-checkbox>
                </el-checkbox-group>
              </el-collapse-item>
            </el-collapse>
          </div>
        </el-card>
      </el-col>

      <!-- Right Area: Document Records -->
      <el-col :span="19">
        <el-card class="records-card" shadow="never">
          <div class="records-header">
            <span class="title">原始日志</span>
            <div class="display-mode-selector">
              <el-radio-group v-model="displayMode" size="small">
                <el-radio-button label="raw">原始日志</el-radio-button>
                <el-radio-button label="table">表格</el-radio-button>
                <el-radio-button label="json">JSON</el-radio-button>
              </el-radio-group>
            </div>
            <span class="page-indicator">第 1 页</span>
          </div>

          <!-- Raw Log Line Listing -->
          <div class="log-lines-wrapper" v-if="displayMode === 'raw'">
            <div class="log-line-item" v-for="(log, index) in logs" :key="log.id">
              <div class="line-header">
                <span class="line-num">{{ index + 1 }}</span>
                <el-button link type="info" size="small" class="expand-btn">
                  <el-icon><ArrowRight /></el-icon>
                </el-button>
                <span class="log-time">{{ formatTime(log.timestamp) }}</span>
                
                <!-- Action links on the right -->
                <div class="line-actions">
                  <el-button link type="primary" size="small">详情</el-button>
                  <el-button link type="primary" size="small" @click="copyLog(log.content)">复制</el-button>
                  <el-button link type="success" size="small" @click="showLogContext(log)">上下文</el-button>
                  <el-button link type="primary" size="small">筛选</el-button>
                  <el-button link type="primary" size="small">排除</el-button>
                  <el-button link type="danger" size="small">告警</el-button>
                </div>
              </div>

              <!-- Metadata tag list block directly below timestamp -->
              <div class="log-tags-row">
                <span class="tag-chip" v-for="(val, key) in log.labels" :key="key">
                  <span class="key">{{ key }}</span>
                  <span class="val">{{ val }}</span>
                </span>
              </div>

              <!-- Actual message body -->
              <div class="log-body-content">{{ log.content }}</div>
            </div>
            <div v-if="logs.length === 0" class="no-logs">未搜索到符合条件的日志数据</div>
          </div>
          <div class="log-table-wrapper" v-else-if="displayMode === 'table'">
            <el-table :data="logs" style="width: 100%" size="small" stripe>
              <el-table-column type="index" label="#" width="50" />
              <el-table-column label="时间" width="180">
                <template #default="{ row }">
                  {{ formatTime(row.timestamp) }}
                </template>
              </el-table-column>
              <el-table-column prop="level" label="级别" width="80">
                <template #default="{ row }">
                  <el-tag :type="getLevelType(row.level)" size="small">{{ row.level }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="content" label="日志内容" min-width="400" show-overflow-tooltip />
              <el-table-column label="元数据" min-width="200">
                <template #default="{ row }">
                  <span style="font-size: 11px; color: #8f959e">{{ JSON.stringify(row.labels) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="100">
                <template #default="{ row }">
                  <el-button link type="success" size="small" @click="showLogContext(row)">上下文</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>

          <!-- JSON View Mode -->
          <div class="log-json-wrapper" v-else>
            <pre>{{ JSON.stringify(logs, null, 2) }}</pre>
          </div>

          <!-- Bottom Pagination matching user screenshot -->
          <div class="pagination-container">
            <span class="total-text">共 {{ totalLogs }} 条</span>
            <el-pagination 
              v-model:current-page="currentPage"
              v-model:page-size="pageSize"
              layout="sizes, prev, pager, next, jumper" 
              :total="totalLogs" 
              :page-sizes="[10, 20, 50, 100]"
              @size-change="handleSearch"
              @current-change="handleSearch"
            />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 日志上下文对话框 -->
    <el-dialog title="日志上下文浏览" v-model="contextVisible" width="900px" append-to-body destroy-on-close>
      <div class="context-dialog-body" v-loading="contextLoading">
        <div class="context-load-more" style="text-align: center; margin-bottom: 12px;">
          <el-button size="small" @click="loadMoreContext('before')" :loading="contextBeforeLoading">
            向上加载更多 (前50行)
          </el-button>
        </div>
        
        <div class="context-logs-list" style="max-height: 500px; overflow-y: auto; background-color: #1e1e1e; color: #d4d4d4; font-family: monospace; padding: 12px; border-radius: 4px;">
          <div 
            v-for="log in contextLogs" 
            :key="log.id" 
            class="context-log-item"
            :class="{ 'is-anchor': log.id === selectedLogForContext?.id || log.labels?.anchor === 'true' }"
            style="padding: 4px 8px; line-height: 1.4; white-space: pre-wrap; font-size: 12px;"
          >
            <span class="c-time" style="color: #858585; margin-right: 12px;">{{ formatTime(log.timestamp) }}</span>
            <span class="c-level" :class="log.level.toLowerCase()" style="font-weight: bold; margin-right: 12px; display: inline-block; width: 50px;">{{ log.level }}</span>
            <span class="c-content">{{ log.content }}</span>
          </div>
        </div>
        
        <div class="context-load-more" style="text-align: center; margin-top: 12px;">
          <el-button size="small" @click="loadMoreContext('after')" :loading="contextAfterLoading">
            向下加载更多 (后50行)
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { Search, Refresh, Plus, Bell, Download, ArrowRight } from '@element-plus/icons-vue'
import * as echarts from 'echarts'

const route = useRoute()

const loading = ref(false)
const searchLoading = ref(false)
const hasSearched = ref(false)

const contextVisible = ref(false)
const contextLoading = ref(false)
const contextBeforeLoading = ref(false)
const contextAfterLoading = ref(false)
const contextLogs = ref([])
const selectedLogForContext = ref(null)

const selectedProject = ref('local-simulation')
const projectOptions = ref([
  { label: '本地默认环境 (未接入集群/主机)', value: 'local-simulation' }
])

const projectType = ref('library') // 'k8s', 'host', or 'library'
const k8sNamespace = ref('default')
const namespaces = ref([])
const k8sPod = ref('')
const pods = ref([])
const hostFilePath = ref('/var/log/messages')

const datasource = ref('')
const queryStr = ref('')
const timeRange = ref([])
const granularity = ref('30s')
const limit = ref(50)

const totalLogs = ref(20894)
const elapsedMs = ref(39)
const logs = ref([])
const displayMode = ref('raw')
const currentPage = ref(1)
const pageSize = ref(50)

const libraries = ref([])
const fieldSearch = ref('')
const selectedFields = ref(['_time', '_msg', 'message', 'host_name', 'file_path', 'env', 'app', 'log_type'])

const chartRef = ref(null)
let myChart = null

const timePresets = [
  { label: '近1m', val: 1 },
  { label: '近5m', val: 5 },
  { label: '近15m', val: 15 },
  { label: '近1h', val: 60 },
  { label: '近4h', val: 240 },
  { label: '近24h', val: 1440 },
  { label: '近7d', val: 10080 }
]

const authHeaders = () => ({
  Authorization: `Bearer ${localStorage.getItem('token')}`
})

const formatNumber = (num) => {
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",")
}

const formatTime = (timeStr) => {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  return date.toISOString().replace('T', ' ').substring(0, 23)
}

const getLevelType = (lvl) => {
  if (lvl === 'ERROR') return 'danger'
  if (lvl === 'WARN') return 'warning'
  return 'info'
}

const applyPreset = (preset) => {
  const end = new Date()
  const start = new Date(end.getTime() - preset.val * 60 * 1000)
  timeRange.value = [formatDateTime(start), formatDateTime(end)]
  handleSearch()
}

const formatDateTime = (date) => {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const h = String(date.getHours()).padStart(2, '0')
  const min = String(date.getMinutes()).padStart(2, '0')
  const s = String(date.getSeconds()).padStart(2, '0')
  return `${y}-${m}-${d} ${h}:${min}:${s}`
}

const clearForm = () => {
  queryStr.value = ''
  timeRange.value = []
  limit.value = 50
  logs.value = []
  hasSearched.value = false
}

const copyLog = (content) => {
  navigator.clipboard.writeText(content).then(() => {
    ElMessage.success('复制成功')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

const filterFields = () => {
  // Simple layout matching
}

const onProjectChange = async (val) => {
  if (val.startsWith('k8s-')) {
    projectType.value = 'k8s'
    const clusterId = val.replace('k8s-', '')
    namespaces.value = []
    pods.value = []
    k8sPod.value = ''
    try {
      const res = await axios.get(`/api/v1/k8s/clusters/${clusterId}/namespaces`, { headers: authHeaders() })
      if (res.data.code === 0) {
        namespaces.value = res.data.data
        if (namespaces.value.length > 0) {
          k8sNamespace.value = namespaces.value.find(n => n.name === 'default')?.name || namespaces.value[0].name
          onNamespaceChange(k8sNamespace.value)
        }
      }
    } catch (err) {
      console.error('获取Namespace失败', err)
    }
  } else if (val.startsWith('host-')) {
    projectType.value = 'host'
    handleSearch()
  } else {
    projectType.value = 'library'
    handleSearch()
  }
}

const onNamespaceChange = async (nsVal) => {
  const clusterId = selectedProject.value.replace('k8s-', '')
  pods.value = []
  k8sPod.value = ''
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${clusterId}/namespaces/${nsVal}/pods`, { headers: authHeaders() })
    if (res.data.code === 0) {
      pods.value = res.data.data
      if (pods.value.length > 0) {
        k8sPod.value = pods.value[0].name
        handleSearch()
      }
    }
  } catch (err) {
    console.error('获取Pods失败', err)
  }
}

const fetchProjects = async () => {
  try {
    const [clustersRes, groupsRes] = await Promise.all([
      axios.get('/api/v1/k8s/clusters', { headers: authHeaders() }).catch(() => null),
      axios.get('/api/v1/cmdb/hosts', { headers: authHeaders() }).catch(() => null)
    ])

    const options = []
    if (clustersRes && clustersRes.data.code === 0) {
      clustersRes.data.data.forEach(c => {
        options.push({ label: `${c.name} (K8s集群)`, value: `k8s-${c.id}` })
      })
    }
    if (groupsRes && groupsRes.data.code === 0) {
      groupsRes.data.data.forEach(h => {
        options.push({ label: `${h.name} (${h.ip}) (物理主机)`, value: `host-${h.id}` })
      })
    }

    if (options.length > 0) {
      projectOptions.value = options
      selectedProject.value = options[0].value
      onProjectChange(selectedProject.value)
    } else {
      projectOptions.value = [
        { label: '本地默认环境 (未接入集群/主机)', value: 'local-simulation' }
      ]
      selectedProject.value = 'local-simulation'
    }
  } catch (err) {
    console.error('获取环境失败', err)
  }
}

const fetchLibraries = async () => {
  try {
    const res = await axios.get('/api/v1/log/libraries', { headers: authHeaders() })
    if (res.data.code === 0) {
      libraries.value = res.data.data
      if (libraries.value.length > 0) {
        if (route.query.library_id) {
          datasource.value = route.query.library_id
        } else {
          datasource.value = libraries.value[0].id
        }
      }
    }
  } catch (err) {
    console.error('获取日志库失败', err)
  }
}

const initChart = (chartData) => {
  if (!chartRef.value) return
  if (!myChart) {
    myChart = echarts.init(chartRef.value)
  }

  const times = chartData.map(p => p.time)
  const counts = chartData.map(p => p.count)

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: function(params) {
        const item = params[0]
        const val = item.value
        const nextTime = new Date('2026-07-08T' + item.name)
        nextTime.setSeconds(nextTime.getSeconds() + 30)
        const nextTimeStr = nextTime.toTimeString().split(' ')[0]
        return `2026-07-08 ${item.name} ~ ${nextTimeStr}<br/>日志数: <b>${val}</b>`
      }
    },
    grid: { left: '3%', right: '3%', bottom: '5%', top: '10%', containLabel: true },
    xAxis: {
      type: 'category',
      data: times,
      axisLabel: { color: '#8f959e', fontSize: 11 },
      axisLine: { lineStyle: { color: '#e4e7ed' } }
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#8f959e', fontSize: 11 },
      splitLine: { lineStyle: { type: 'dashed', color: '#f0f2f5' } }
    },
    series: [
      {
        name: '日志数',
        type: 'bar',
        barWidth: '35%',
        data: counts,
        itemStyle: {
          color: '#52c41a', // Green bars matching the user's screenshot
          borderRadius: [2, 2, 0, 0]
        }
      }
    ]
  }
  myChart.setOption(option)
}

const handleSearch = async () => {
  let projectID = ''
  if (projectType.value === 'k8s') {
    projectID = selectedProject.value.replace('k8s-', '')
    if (!k8sPod.value) {
      return
    }
  } else if (projectType.value === 'host') {
    projectID = selectedProject.value.replace('host-', '')
  }

  searchLoading.value = true
  try {
    const res = await axios.get('/api/v1/log/query', {
      params: {
        library_id: projectType.value === 'library' ? datasource.value : '',
        project_type: projectType.value,
        project_id: projectID,
        namespace: k8sNamespace.value,
        pod: k8sPod.value,
        file_path: hostFilePath.value,
        query: queryStr.value,
        limit: limit.value
      },
      headers: authHeaders()
    })
    if (res.data.code === 0) {
      hasSearched.value = true
      logs.value = res.data.data.logs
      totalLogs.value = res.data.data.total
      elapsedMs.value = res.data.data.elapsed_ms
      nextTick(() => {
        initChart(res.data.data.chart_data)
      })
    } else {
      ElMessage.error(res.data.message || '查询失败')
    }
  } catch (err) {
    ElMessage.error('检索API调用出错')
  } finally {
    searchLoading.value = false
  }
}

const handleResize = () => {
  if (myChart) {
    myChart.resize()
  }
}

onMounted(async () => {
  loading.value = true
  // Set default time presets: last 15 minutes
  const end = new Date()
  const start = new Date(end.getTime() - 15 * 60 * 1000)
  timeRange.value = [formatDateTime(start), formatDateTime(end)]

  await fetchProjects()
  await fetchLibraries()
  loading.value = false
  if (datasource.value) {
    handleSearch()
  }
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  if (myChart) {
    myChart.dispose()
    myChart = null
  }
})

const showLogContext = async (log) => {
  selectedLogForContext.value = log
  contextVisible.value = true
  contextLoading.value = true
  try {
    const res = await axios.get('/api/v1/log/query/context', {
      headers: authHeaders(),
      params: {
        library_id: datasource.value || undefined,
        project_type: projectType.value || undefined,
        project_id: selectedProject.value || undefined,
        namespace: k8sNamespace.value || undefined,
        pod: k8sPod.value || undefined,
        file_path: hostFilePath.value || undefined,
        timestamp: log.timestamp,
        id: log.id,
        limit: 50,
        direction: 'both'
      }
    })
    if (res.data?.code === 0) {
      contextLogs.value = res.data.data || []
    }
  } catch (err) {
    ElMessage.error('加载日志上下文失败')
  } finally {
    contextLoading.value = false
  }
}

const loadMoreContext = async (direction) => {
  if (contextLogs.value.length === 0) return
  
  if (direction === 'before') {
    contextBeforeLoading.value = true
    const anchor = contextLogs.value[0]
    try {
      const res = await axios.get('/api/v1/log/query/context', {
        headers: authHeaders(),
        params: {
          library_id: datasource.value || undefined,
          project_type: projectType.value || undefined,
          project_id: selectedProject.value || undefined,
          namespace: k8sNamespace.value || undefined,
          pod: k8sPod.value || undefined,
          file_path: hostFilePath.value || undefined,
          timestamp: anchor.timestamp,
          limit: 50,
          direction: 'before'
        }
      })
      if (res.data?.code === 0) {
        contextLogs.value = [...res.data.data, ...contextLogs.value]
      }
    } catch (err) {
      ElMessage.error('加载更多日志失败')
    } finally {
      contextBeforeLoading.value = false
    }
  } else {
    contextAfterLoading.value = true
    const anchor = contextLogs.value[contextLogs.value.length - 1]
    try {
      const res = await axios.get('/api/v1/log/query/context', {
        headers: authHeaders(),
        params: {
          library_id: datasource.value || undefined,
          project_type: projectType.value || undefined,
          project_id: selectedProject.value || undefined,
          namespace: k8sNamespace.value || undefined,
          pod: k8sPod.value || undefined,
          file_path: hostFilePath.value || undefined,
          timestamp: anchor.timestamp,
          limit: 50,
          direction: 'after'
        }
      })
      if (res.data?.code === 0) {
        contextLogs.value = [...contextLogs.value, ...res.data.data]
      }
    } catch (err) {
      ElMessage.error('加载更多日志失败')
    } finally {
      contextAfterLoading.value = false
    }
  }
}
</script>

<style scoped>
.log-query-container {
  padding-bottom: 40px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.header-title {
  display: flex;
  align-items: center;
  gap: 16px;
}
.icon-box {
  width: 48px;
  height: 48px;
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}
.header-title h2 {
  margin: 0 0 4px 0;
  font-size: 20px;
  color: #1f2329;
}
.header-title p {
  margin: 0;
  font-size: 13px;
  color: #8f959e;
}

.metrics-row {
  display: flex;
  gap: 16px;
}
.metric-card {
  background: #fff;
  border: 1px solid #f0f2f5;
  border-radius: 8px;
  padding: 8px 16px;
  min-width: 90px;
  text-align: center;
  box-shadow: 0 1px 2px rgba(0,0,0,0.02);
}
.metric-card .label {
  font-size: 11px;
  color: #8f959e;
  margin-bottom: 4px;
}
.metric-card .value {
  font-size: 18px;
  font-weight: bold;
}
.text-primary { color: var(--el-color-primary); }
.text-success { color: #52c41a; }
.text-warning { color: var(--el-color-warning); }

.warning-banner :deep(.el-alert__title) {
  font-weight: 500;
}

.query-card {
  margin-bottom: 24px;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.02) !important;
}

.filter-row {
  display: flex;
  margin-bottom: 12px;
}
.filter-item {
  display: flex;
  align-items: center;
  gap: 8px;
}
.filter-label {
  font-size: 12px;
  font-weight: 500;
  color: #646a73;
  width: 65px;
  text-align: right;
  margin-right: 8px;
}

.query-bar {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}
.query-input {
  flex: 1;
}

.time-limit-row {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}
.time-picker-box {
  display: flex;
  align-items: center;
}

.presets-row {
  display: flex;
  align-items: center;
  padding-left: 68px;
}

.chart-section {
  background: #fff;
  border-radius: 12px;
  padding: 16px 20px;
  margin-bottom: 24px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.02);
}
.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.chart-title {
  font-size: 13px;
  color: #1f2329;
}
.chart-actions .el-button {
  font-size: 12px;
  margin-left: 12px;
}
.chart-box {
  height: 180px;
}

.workspace-layout {
  margin-top: 16px;
}

.field-sidebar-card {
  border-radius: 12px;
  border: 1px solid #f0f2f5;
  background: #fafbfc;
}
.sidebar-title {
  font-size: 13px;
  font-weight: 600;
  color: #1f2329;
  margin-bottom: 12px;
}
.field-group {
  margin-bottom: 16px;
}
.field-group .group-header {
  font-size: 12px;
  color: #8f959e;
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.field-group .group-header .badge {
  background: #e4e7ed;
  color: #646a73;
  padding: 1px 5px;
  border-radius: 10px;
  font-size: 10px;
}
.field-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  margin-right: 0 !important;
}
.field-item :deep(.el-checkbox__label) {
  display: flex;
  justify-content: space-between;
  width: 100%;
  padding-left: 8px;
  font-family: monospace;
  font-size: 11px;
}
.field-name {
  color: #1f2329;
}
.percent {
  color: #8f959e;
}

.records-card {
  border-radius: 12px;
  border: 1px solid #f0f2f5;
}
.records-header {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
  border-bottom: 1px solid #f0f2f5;
  padding-bottom: 12px;
}
.records-header .title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2329;
}
.display-mode-selector {
  margin-left: 20px;
}
.page-indicator {
  margin-left: auto;
  font-size: 12px;
  color: #8f959e;
}

.log-lines-wrapper {
  max-height: 650px;
  overflow-y: auto;
  background: #fff;
  border: 1px solid #f0f2f5;
  border-radius: 6px;
}
.log-line-item {
  padding: 10px 16px;
  border-bottom: 1px solid #f0f2f5;
  font-family: monospace;
  font-size: 12px;
}
.log-line-item:last-child {
  border-bottom: none;
}
.line-header {
  display: flex;
  align-items: center;
  margin-bottom: 6px;
}
.line-num {
  color: #8f959e;
  width: 24px;
  display: inline-block;
}
.expand-btn {
  padding: 0 !important;
  height: auto !important;
  color: #8f959e;
  margin-right: 6px;
}
.log-time {
  color: #646a73;
  font-weight: bold;
}
.line-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
}
.line-actions .el-button {
  font-size: 11px;
}

.log-tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 6px;
  padding-left: 30px;
}
.tag-chip {
  background: #f4f5f7;
  border-radius: 4px;
  padding: 1px 6px;
  font-size: 11px;
}
.tag-chip .key {
  color: #8f959e;
  margin-right: 4px;
}
.tag-chip .val {
  color: #1f2329;
  font-weight: 500;
}

.log-body-content {
  color: #2c3e50;
  white-space: pre-wrap;
  word-break: break-all;
  padding-left: 30px;
  line-height: 1.6;
}

.no-logs {
  padding: 40px;
  text-align: center;
  color: #8f959e;
}

.log-json-wrapper {
  background: #272822;
  color: #f8f8f2;
  padding: 16px;
  border-radius: 6px;
  max-height: 600px;
  overflow-y: auto;
  font-family: monospace;
  font-size: 12px;
}
.log-json-wrapper pre {
  margin: 0;
}

.pagination-container {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  margin-top: 20px;
  border-top: 1px solid #f0f2f5;
  padding-top: 12px;
}
.total-text {
  font-size: 13px;
  color: #606266;
  margin-right: 12px;
}
</style>
