<template>
  <el-card class="dashboard-card" v-loading="loading">
    <div class="page-header motion-up delay-1">
      <div>
        <h2>全局仪表盘</h2>
        <p class="page-desc">主机、容器、K8s、告警与任务的统一视图。</p>
      </div>
      <div class="page-actions">
        <span class="updated-at" v-if="lastUpdated">刷新时间：{{ lastUpdated }}</span>
        <el-button icon="Refresh" :loading="refreshing" @click="refreshDashboard">刷新</el-button>
      </div>
    </div>

    <div class="scope-bar motion-up delay-2">
      <span class="scope-label">范围</span>
      <el-select v-model="scope.environment" class="scope-item" @change="handleScopeEnvironmentChange">
        <el-option v-for="item in environmentOptions" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
      <el-select v-model="scope.clusterId" class="scope-item" clearable placeholder="全部集群" @change="handleScopeClusterChange">
        <el-option v-for="item in scopeClusters" :key="item.id" :label="item.display_name || item.name" :value="item.id" />
      </el-select>
      <el-select v-model="scope.namespace" class="scope-item" clearable placeholder="全部命名空间" @change="refreshDashboard">
        <el-option v-for="item in scopeNamespaces" :key="item" :label="item" :value="item" />
      </el-select>
      <el-select v-model="scope.timeWindowHours" class="scope-item narrow" @change="refreshDashboard">
        <el-option v-for="item in timeWindowOptions" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
    </div>

    <el-row :gutter="16" class="motion-up delay-3">
      <el-col :span="3">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">主机总数</div>
          <div class="kpi-value">{{ stats.hostTotal }}</div>
          <div class="kpi-sub">在线 {{ stats.hostOnline }} / 离线 {{ stats.hostOffline }} / 过期 {{ stats.hostStale }}</div>
        </el-card>
      </el-col>
      <el-col :span="3">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">Docker 环境</div>
          <div class="kpi-value">{{ stats.dockerTotal }}</div>
          <div class="kpi-sub">在线 {{ stats.dockerOnline }} / 离线 {{ stats.dockerOffline }} / 过期 {{ stats.dockerStale }}</div>
        </el-card>
      </el-col>
      <el-col :span="3">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">K8s 集群</div>
          <div class="kpi-value">{{ stats.k8sTotal }}</div>
          <div class="kpi-sub">正常 {{ stats.k8sHealthy }} / 异常 {{ stats.k8sUnhealthy }} / 过期 {{ stats.k8sStale }}</div>
        </el-card>
      </el-col>
      <el-col :span="3">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">防火墙设备</div>
          <div class="kpi-value">{{ stats.firewallTotal }}</div>
          <div class="kpi-sub">在线 {{ stats.firewallOnline }} / 离线 {{ stats.firewallOffline }} / 过期 {{ stats.firewallStale }}</div>
        </el-card>
      </el-col>
      <el-col :span="3">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">域名健康</div>
          <div class="kpi-value">{{ stats.domainTotal }}</div>
          <div class="kpi-sub">健康 {{ stats.domainHealthy }} / 预警 {{ stats.domainWarning }} / 故障 {{ stats.domainCritical }}</div>
        </el-card>
      </el-col>
      <el-col :span="3">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">活跃告警</div>
          <div class="kpi-value danger">{{ stats.alertOpen }}</div>
          <div class="kpi-sub">总告警 {{ stats.alertTotal }}</div>
        </el-card>
      </el-col>
      <el-col :span="3">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">启用任务</div>
          <div class="kpi-value">{{ stats.taskEnabled }}</div>
          <div class="kpi-sub">总任务 {{ stats.taskTotal }}</div>
        </el-card>
      </el-col>
      <el-col :span="3">
        <el-card class="kpi-card" shadow="never">
          <div class="kpi-title">在线 Agent</div>
          <div class="kpi-value">{{ stats.agentOnline }}</div>
          <div class="kpi-sub">总 Agent {{ stats.agentTotal }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-4">
      <el-col :span="24">
        <el-card shadow="never" class="backlog-overview-card">
          <div class="panel-header backlog-header">
            <div>
              <h3>待处置积压总览</h3>
              <p class="panel-desc">按中心聚合超时与未闭环事项，可直接跳转处置。</p>
            </div>
            <div class="backlog-totals">
              <el-tag type="warning" effect="light">总积压 {{ backlog.total }}</el-tag>
              <el-tag type="danger" effect="light">超时 {{ backlog.overdue }}</el-tag>
            </div>
          </div>
          <div class="backlog-grid">
            <div v-for="item in backlogCards" :key="item.key" class="backlog-item">
              <div class="backlog-item-top">
                <span class="backlog-label">{{ item.label }}</span>
                <el-tag size="small" :type="item.overdue > 0 ? 'warning' : 'success'">
                  {{ item.overdue > 0 ? `超时 ${item.overdue}` : '正常' }}
                </el-tag>
              </div>
              <div class="backlog-value">{{ item.value }}</div>
              <div class="backlog-desc">{{ item.desc }}</div>
              <div class="backlog-actions">
                <el-button
                  link
                  type="warning"
                  :disabled="item.value <= 0"
                  :loading="backlogActionLoading[item.key]"
                  @click="runBacklogAction(item.key)"
                >
                  一键处置
                </el-button>
                <el-button link type="primary" @click="go(item.path)">进入处置</el-button>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-5">
      <el-col :span="24">
        <el-card shadow="never" class="selfcheck-card">
          <div class="panel-header">
            <div>
              <h3>功能完整性自检</h3>
              <p class="panel-desc">自动巡检核心数据源与模块链路，减少手工逐页验证。</p>
            </div>
            <div class="selfcheck-header-actions">
              <el-progress type="circle" :percentage="completenessScore" :width="56" :stroke-width="8" />
              <el-button size="small" type="primary" :loading="deepChecking || refreshing" @click="runSelfCheck">运行深度自检</el-button>
            </div>
          </div>
          <div class="selfcheck-meta">
            <el-tag type="success" effect="light">正常 {{ dataSourceSummary.ok }}</el-tag>
            <el-tag type="danger" effect="light">失败 {{ dataSourceSummary.error }}</el-tag>
            <el-tag type="warning" effect="light">降级 {{ dataSourceSummary.warning }}</el-tag>
            <el-tag v-if="overviewSourceErrorCount > 0" type="danger" effect="plain">概览源失败 {{ overviewSourceErrorCount }}</el-tag>
            <el-tag type="info" effect="plain">契约 {{ overviewContractVersion || 'legacy' }}</el-tag>
          </div>
          <div class="selfcheck-contract">状态契约：{{ statusContractSummary }}</div>
          <el-table :fit="true" :data="dataSourceDiagnostics" size="small" style="width: 100%" empty-text="暂无自检数据">
            <el-table-column prop="label" label="检查项" min-width="180" />
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-tag :type="row.status === 'ok' ? 'success' : row.status === 'warning' ? 'warning' : 'danger'">
                  {{ row.status === 'ok' ? '正常' : row.status === 'warning' ? '降级' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="countText" label="数据量" width="110" />
            <el-table-column prop="message" label="说明" min-width="260" show-overflow-tooltip />
            <el-table-column label="处置" width="130" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">进入模块</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-5">
      <el-col :span="14" :lg="14" :xs="24">
        <el-card shadow="never" class="integrity-card">
          <div class="panel-header">
            <div>
              <h3>差异清单（对比强化）</h3>
              <p class="panel-desc">按“现状-目标-影响”列出优先修复项，减少逐页人工验证。</p>
            </div>
            <div class="integrity-summary-tags">
              <el-tag type="danger" effect="light">P0 {{ gapChecklistSummary.p0 }}</el-tag>
              <el-tag type="warning" effect="light">P1 {{ gapChecklistSummary.p1 }}</el-tag>
              <el-tag type="info" effect="light">P2 {{ gapChecklistSummary.p2 }}</el-tag>
            </div>
          </div>
          <el-table :fit="true" :data="gapChecklistRows" size="small" style="width: 100%" empty-text="当前无高优先级差异">
            <el-table-column label="优先级" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.priority >= 90 ? 'danger' : row.priority >= 70 ? 'warning' : 'info'">
                  {{ row.priorityLabel }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="gap" label="差异项" min-width="150" show-overflow-tooltip />
            <el-table-column prop="current" label="现状" min-width="220" show-overflow-tooltip />
            <el-table-column prop="target" label="目标" min-width="200" show-overflow-tooltip />
            <el-table-column prop="impact" label="影响" min-width="190" show-overflow-tooltip />
            <el-table-column label="操作" width="130" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">进入修复</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="10" :lg="10" :xs="24">
        <el-card shadow="never" class="integrity-card">
          <div class="panel-header">
            <div>
              <h3>状态波动与恢复</h3>
              <p class="panel-desc">基于近 24 小时时间线采样，统计在线率、恢复时长与失败链路。</p>
            </div>
            <el-tag :type="statusFlappingSummary.chainAvailabilityRate >= 95 ? 'success' : statusFlappingSummary.chainAvailabilityRate >= 85 ? 'warning' : 'danger'" effect="light">
              可用率 {{ statusFlappingSummary.chainAvailabilityRate }}%
            </el-tag>
          </div>
          <div class="flap-kpi-grid">
            <div class="flap-kpi-item">
              <div class="flap-kpi-label">主机在线率</div>
              <div class="flap-kpi-value">{{ statusFlappingSummary.hostOnlineRate }}%</div>
            </div>
            <div class="flap-kpi-item">
              <div class="flap-kpi-label">Docker在线率</div>
              <div class="flap-kpi-value">{{ statusFlappingSummary.dockerOnlineRate }}%</div>
            </div>
            <div class="flap-kpi-item">
              <div class="flap-kpi-label">K8s健康率</div>
              <div class="flap-kpi-value">{{ statusFlappingSummary.k8sHealthyRate }}%</div>
            </div>
            <div class="flap-kpi-item">
              <div class="flap-kpi-label">恢复中位时长</div>
              <div class="flap-kpi-value">{{ formatDurationMinutes(statusFlappingSummary.medianRecoveryMinutes) }}</div>
            </div>
          </div>
          <div class="flap-meta">
            <span>采样数 {{ statusFlappingSummary.samples }}</span>
            <span>恢复事件 {{ statusFlappingSummary.recoveryCount }}</span>
            <span>平均恢复 {{ formatDurationMinutes(statusFlappingSummary.avgRecoveryMinutes) }}</span>
          </div>
          <el-table :fit="true" :data="failedLinkTopRows" size="small" style="width: 100%" empty-text="近 24h 无失败链路采样">
            <el-table-column prop="label" label="链路" min-width="130" show-overflow-tooltip />
            <el-table-column prop="count" label="失败采样" width="95" />
            <el-table-column prop="errorCount" label="错误" width="80" />
            <el-table-column prop="warningCount" label="预警" width="80" />
            <el-table-column label="最近发生" width="160">
              <template #default="{ row }">{{ formatTime(row.lastAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="110" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">进入</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-5">
      <el-col :span="10" :lg="10" :xs="24">
        <el-card shadow="never" class="integrity-card">
          <div class="panel-header">
            <div>
              <h3>状态可信度指数</h3>
              <p class="panel-desc">基于链路可用性、状态时效、字段完整性与一致性综合评估。</p>
            </div>
            <div class="trust-score-wrap">
              <el-progress type="circle" :percentage="statusTrust.score" :width="62" :stroke-width="8" />
              <el-tag :type="statusTrust.tagType" effect="light">{{ statusTrust.grade }}</el-tag>
            </div>
          </div>
          <div class="trust-dimension-list">
            <div class="trust-dimension-item" v-for="item in trustDimensions" :key="item.key">
              <div class="trust-dimension-top">
                <span>{{ item.label }}</span>
                <strong>{{ item.score }}%</strong>
              </div>
              <el-progress :percentage="item.score" :stroke-width="10" :show-text="false" />
              <div class="trust-dimension-desc">{{ item.detail }}</div>
            </div>
          </div>
          <div class="trust-summary">{{ statusTrust.summary }}</div>
        </el-card>
      </el-col>
      <el-col :span="14" :lg="14" :xs="24">
        <el-card shadow="never" class="integrity-card">
          <div class="panel-header">
            <div>
              <h3>智能处置建议</h3>
              <p class="panel-desc">按影响范围与风险等级排序，优先处理最影响稳定性的事项。</p>
            </div>
            <el-tag :type="intelligentActions.length > 0 ? 'warning' : 'success'" effect="light">
              建议 {{ intelligentActions.length }}
            </el-tag>
          </div>
          <el-table :fit="true" :data="intelligentActions.slice(0, 8)" size="small" style="width: 100%" empty-text="当前无高优先级建议">
            <el-table-column prop="priorityLabel" label="优先级" width="92">
              <template #default="{ row }">
                <el-tag size="small" :type="row.priority >= 90 ? 'danger' : row.priority >= 70 ? 'warning' : 'info'">
                  {{ row.priorityLabel }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="module" label="模块" width="120" />
            <el-table-column prop="title" label="建议" min-width="170" show-overflow-tooltip />
            <el-table-column prop="reason" label="判断依据" min-width="230" show-overflow-tooltip />
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">{{ row.action }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-5">
      <el-col :span="12" :lg="12" :xs="24">
        <el-card shadow="never" class="integrity-card">
          <div class="panel-header">
            <div>
              <h3>状态字段完整性</h3>
              <p class="panel-desc">检查状态、时间戳、异常原因等关键字段是否完整。</p>
            </div>
            <el-tag :type="fieldCompletenessSummary.problem > 0 ? 'warning' : 'success'" effect="light">
              缺口 {{ fieldCompletenessSummary.problem }} / {{ fieldCompletenessSummary.total }}
            </el-tag>
          </div>
          <el-table :fit="true" :data="fieldCompletenessRows" size="small" style="width: 100%" empty-text="暂无数据">
            <el-table-column prop="module" label="模块" width="120" />
            <el-table-column label="完整率" width="120">
              <template #default="{ row }">
                <el-progress :percentage="row.rate" :stroke-width="12" :show-text="false" />
                <div class="mini-percent">{{ row.rate }}%</div>
              </template>
            </el-table-column>
            <el-table-column prop="issue" label="缺口" min-width="220" show-overflow-tooltip />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">进入模块</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12" :lg="12" :xs="24">
        <el-card shadow="never" class="integrity-card">
          <div class="panel-header">
            <div>
              <h3>跨模块一致性</h3>
              <p class="panel-desc">识别 CMDB、Docker、K8s、堡垒机资产之间的状态/映射冲突。</p>
            </div>
            <el-tag :type="consistencySummary.total > 0 ? 'warning' : 'success'" effect="light">
              冲突 {{ consistencySummary.total }}
            </el-tag>
          </div>
          <el-table :fit="true" :data="consistencyIssueRows" size="small" style="width: 100%" empty-text="跨模块状态一致">
            <el-table-column prop="module" label="链路" width="140" />
            <el-table-column prop="name" label="对象" min-width="160" show-overflow-tooltip />
            <el-table-column prop="reason" label="冲突说明" min-width="220" show-overflow-tooltip />
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">进入模块</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-5">
      <el-col :span="16">
        <el-card shadow="never">
          <div class="panel-header">
            <div>
              <h3>系统趋势（24h）</h3>
              <p class="panel-desc">CPU / 内存 / 磁盘历史变化。</p>
            </div>
          </div>
          <div ref="trendRef" class="chart-box"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="never" class="stack-card">
          <div class="panel-header">
            <div>
              <h3>实时资源</h3>
              <p class="panel-desc">当前主机资源占用。</p>
            </div>
          </div>
          <div class="resource-row">
            <span>CPU</span>
            <el-progress :percentage="safePercent(realtime.cpu)" :show-text="false" />
            <b>{{ formatPercent(realtime.cpu) }}</b>
          </div>
          <div class="resource-row">
            <span>内存</span>
            <el-progress :percentage="safePercent(realtime.memory)" :show-text="false" />
            <b>{{ formatPercent(realtime.memory) }}</b>
          </div>
          <div class="resource-row">
            <span>磁盘</span>
            <el-progress :percentage="safePercent(realtime.disk)" :show-text="false" />
            <b>{{ formatPercent(realtime.disk) }}</b>
          </div>
          <div class="resource-row">
            <span>网络</span>
            <div class="network-value">{{ formatNumber(realtime.network) }} MB/s</div>
          </div>
        </el-card>

        <el-card shadow="never" class="stack-card">
          <div class="panel-header">
            <div>
              <h3>模块状态</h3>
              <p class="panel-desc">核心模块健康与快捷入口。</p>
            </div>
          </div>
          <div class="module-row">
            <span>CMDB</span>
            <el-tag :type="moduleTagType(moduleStatus.cmdb)">{{ moduleTagText(moduleStatus.cmdb) }}</el-tag>
            <el-button link @click="go('/host')">进入</el-button>
          </div>
          <div class="module-row">
            <span>Docker</span>
            <el-tag :type="moduleTagType(moduleStatus.docker)">{{ moduleTagText(moduleStatus.docker) }}</el-tag>
            <el-button link @click="go('/docker')">进入</el-button>
          </div>
          <div class="module-row">
            <span>K8s</span>
            <el-tag :type="moduleTagType(moduleStatus.k8s)">{{ moduleTagText(moduleStatus.k8s) }}</el-tag>
            <el-button link @click="go('/k8s/clusters')">进入</el-button>
          </div>
          <div class="module-row">
            <span>监控</span>
            <el-tag :type="moduleTagType(moduleStatus.monitor)">{{ moduleTagText(moduleStatus.monitor) }}</el-tag>
            <el-button link @click="go('/monitor/overview')">进入</el-button>
          </div>
          <div class="module-row">
            <span>任务调度</span>
            <el-tag :type="moduleTagType(moduleStatus.task)">{{ moduleTagText(moduleStatus.task) }}</el-tag>
            <el-button link @click="go('/task/schedules')">进入</el-button>
          </div>
          <div class="module-row">
            <span>防火墙</span>
            <el-tag :type="moduleTagType(moduleStatus.firewall)">{{ moduleTagText(moduleStatus.firewall) }}</el-tag>
            <el-button link @click="go('/firewall')">进入</el-button>
          </div>
          <div class="module-row">
            <span>域名证书</span>
            <el-tag :type="moduleTagType(moduleStatus.domain)">{{ moduleTagText(moduleStatus.domain) }}</el-tag>
            <el-button link @click="go('/domain/ssl')">进入</el-button>
          </div>
          <div class="module-row">
            <span>堡垒机会话</span>
            <el-tag :type="moduleTagType(moduleStatus.jump)">{{ moduleTagText(moduleStatus.jump) }}</el-tag>
            <el-button link @click="go('/jump/sessions')">进入</el-button>
          </div>
          <div class="module-row">
            <span>主机状态时效</span>
            <el-tag :type="stats.hostStale > 0 ? 'warning' : 'success'">
              {{ stats.hostStale > 0 ? `过期 ${stats.hostStale}` : '实时' }}
            </el-tag>
            <el-button link @click="go('/host')">巡检</el-button>
          </div>
          <div class="module-row">
            <span>Docker状态时效</span>
            <el-tag :type="stats.dockerStale > 0 ? 'warning' : 'success'">
              {{ stats.dockerStale > 0 ? `过期 ${stats.dockerStale}` : '实时' }}
            </el-tag>
            <el-button link @click="go('/docker')">巡检</el-button>
          </div>
          <div class="module-row">
            <span>K8s状态时效</span>
            <el-tag :type="stats.k8sStale > 0 ? 'warning' : 'success'">
              {{ stats.k8sStale > 0 ? `过期 ${stats.k8sStale}` : '实时' }}
            </el-tag>
            <el-button link @click="go('/k8s/clusters')">巡检</el-button>
          </div>
        </el-card>

        <el-card shadow="never" class="stack-card integrity-card capability-card">
          <div class="panel-header">
            <div>
              <h3>功能完整度矩阵</h3>
              <p class="panel-desc">按功能链路维度量化完整度，定位待补齐能力。</p>
            </div>
            <div class="integrity-summary-tags">
              <el-tag type="success" effect="light">完整 {{ capabilitySummary.complete }}</el-tag>
              <el-tag type="warning" effect="light">待补齐 {{ capabilitySummary.partial }}</el-tag>
              <el-tag type="danger" effect="light">缺口 {{ capabilitySummary.gap }}</el-tag>
            </div>
          </div>
          <el-table :fit="true" :data="moduleCapabilityRows" size="small" style="width: 100%" empty-text="暂无模块能力数据">
            <el-table-column prop="label" label="模块" min-width="130" />
            <el-table-column label="链路" width="84">
              <template #default="{ row }">
                <el-tag size="small" :type="row.linkStatus === 'ok' ? 'success' : row.linkStatus === 'warning' ? 'warning' : 'danger'">
                  {{ row.linkStatus === 'ok' ? '正常' : row.linkStatus === 'warning' ? '降级' : '异常' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态判定" width="84">
              <template #default="{ row }">
                <el-tag size="small" :type="row.status === 'ok' ? 'success' : row.status === 'warning' ? 'warning' : row.status === 'error' ? 'danger' : 'info'">
                  {{ moduleTagText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="自动处置" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.automationScore >= 90 ? 'success' : row.automationScore >= 70 ? 'warning' : 'info'">
                  {{ row.automationScore >= 90 ? '完善' : row.automationScore >= 70 ? '可用' : '较弱' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="目标值" width="78">
              <template #default="{ row }">{{ row.targetScore }}%</template>
            </el-table-column>
            <el-table-column label="当前值" width="118">
              <template #default="{ row }">
                <el-progress :percentage="row.currentScore" :stroke-width="10" :show-text="false" />
                <div class="mini-percent">{{ row.currentScore }}%</div>
              </template>
            </el-table-column>
            <el-table-column label="趋势" width="96">
              <template #default="{ row }">
                <el-tag size="small" :type="capabilityTrendTag(row.trendDirection)">
                  {{ capabilityTrendArrow(row.trendDirection) }} {{ formatTrendDelta(row.trendDelta) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="suggestion" label="建议" min-width="180" show-overflow-tooltip />
            <el-table-column label="操作" width="90" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">进入</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <el-card shadow="never" class="stack-card integrity-card capability-gap-card">
          <div class="panel-header">
            <div>
              <h3>能力缺口追踪</h3>
              <p class="panel-desc">优先处理高影响缺口，快速提升功能完整度。</p>
            </div>
            <el-tag :type="capabilityGapRows.length > 0 ? 'warning' : 'success'" effect="light">
              缺口 {{ capabilityGapRows.length }}
            </el-tag>
          </div>
          <el-table :fit="true" :data="capabilityGapRows" size="small" style="width: 100%" empty-text="当前无高优先级能力缺口">
            <el-table-column prop="module" label="模块" width="110" />
            <el-table-column prop="gap" label="缺口" min-width="160" show-overflow-tooltip />
            <el-table-column prop="impact" label="影响" min-width="180" show-overflow-tooltip />
            <el-table-column label="操作" width="110" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">去完善</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <el-card shadow="never" class="stack-card risk-card">
          <div class="panel-header">
            <div>
              <h3>Deployment 风险</h3>
              <p class="panel-desc">{{ deploymentRisk.clusterName }} / {{ deploymentRisk.namespaceLabel }}</p>
            </div>
            <el-button link type="primary" @click="goDeploymentCenter">进入</el-button>
          </div>
          <div class="risk-kpi-grid">
            <div class="risk-kpi-item">
              <div class="risk-kpi-label">总量</div>
              <div class="risk-kpi-value">{{ deploymentRisk.total }}</div>
            </div>
            <div class="risk-kpi-item">
              <div class="risk-kpi-label">健康</div>
              <div class="risk-kpi-value success">{{ deploymentRisk.healthy }}</div>
            </div>
            <div class="risk-kpi-item">
              <div class="risk-kpi-label">发布中</div>
              <div class="risk-kpi-value warning">{{ deploymentRisk.progressing }}</div>
            </div>
            <div class="risk-kpi-item">
              <div class="risk-kpi-label">异常</div>
              <div class="risk-kpi-value danger">{{ deploymentRisk.degraded }}</div>
            </div>
          </div>
          <div class="risk-extra">
            <span>副本缺口 {{ deploymentRisk.gapReplicas }}</span>
            <span>异常 Pod {{ deploymentRisk.podAbnormal }}</span>
          </div>
          <div class="risk-list" v-if="deploymentRisk.topRisk.length">
            <div class="risk-list-item" v-for="item in deploymentRisk.topRisk" :key="item.key">
              <span class="risk-name">{{ item.name }}</span>
              <el-tag size="small" :type="item.type">{{ item.label }}</el-tag>
            </div>
          </div>
          <div class="panel-desc" v-else>当前范围内无明显风险。</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-6">
      <el-col :span="16">
        <el-card shadow="never">
          <div class="panel-header">
            <div>
              <h3>最近告警</h3>
              <p class="panel-desc">最新 8 条告警事件。</p>
            </div>
            <el-button size="small" @click="go('/alert/events')">查看全部</el-button>
          </div>
          <el-table :fit="true" :data="recentAlerts" size="small" style="width: 100%">
            <el-table-column prop="rule_name" label="规则" min-width="140" />
            <el-table-column prop="target" label="目标" min-width="160" />
            <el-table-column prop="severity" label="级别" width="110" />
            <el-table-column prop="status" label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="row.status === 0 ? 'danger' : row.status === 1 ? 'success' : 'info'">
                  {{ row.status === 0 ? '未处理' : row.status === 1 ? '已处理' : '已忽略' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="never">
          <div class="panel-header">
            <div>
              <h3>主机 CPU Top</h3>
              <p class="panel-desc">按当前 CPU 使用率排序。</p>
            </div>
            <el-button size="small" @click="refreshTopHosts">刷新</el-button>
          </div>
          <el-table :fit="true" :data="topHosts" size="small" style="width: 100%">
            <el-table-column prop="instance" label="主机" min-width="160" />
            <el-table-column prop="cpu" label="CPU%" width="90">
              <template #default="{ row }">
                {{ formatNumber(row.cpu, 1) }}
              </template>
            </el-table-column>
            <el-table-column prop="memory" label="内存%" width="90">
              <template #default="{ row }">
                {{ formatNumber(row.memory, 1) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-6">
      <el-col :span="24">
        <el-card shadow="never" class="integrity-card">
          <div class="panel-header">
            <div>
              <h3>自检历史</h3>
              <p class="panel-desc">记录每次自检结果，方便快速判断回归与波动。</p>
            </div>
            <div class="integrity-summary-tags">
              <el-tag type="success" effect="light">通过 {{ selfCheckHistorySummary.ok }}</el-tag>
              <el-tag type="warning" effect="light">降级 {{ selfCheckHistorySummary.warning }}</el-tag>
              <el-tag type="danger" effect="light">失败 {{ selfCheckHistorySummary.error }}</el-tag>
              <el-button size="small" text @click="clearSelfCheckHistory">清空历史</el-button>
            </div>
          </div>
          <el-table :fit="true" :data="selfCheckHistory.slice(0, 12)" size="small" style="width: 100%" empty-text="暂无自检历史">
            <el-table-column label="时间" width="180">
              <template #default="{ row }">{{ formatTime(row.at) }}</template>
            </el-table-column>
            <el-table-column label="触发方式" width="120">
              <template #default="{ row }">{{ row.source === 'manual' ? '手动' : row.source === 'probe' ? '主动探测' : '自动' }}</template>
            </el-table-column>
            <el-table-column prop="score" label="完整性分" width="120" />
            <el-table-column label="结果" width="120">
              <template #default="{ row }">
                <el-tag :type="row.status === 'ok' ? 'success' : row.status === 'warning' ? 'warning' : 'danger'">
                  {{ row.status === 'ok' ? '通过' : row.status === 'warning' ? '降级' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="summary" label="摘要" min-width="260" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="panel-row motion-up delay-6">
      <el-col :span="24">
        <el-card shadow="never" class="integrity-card">
          <div class="panel-header">
            <div>
              <h3>状态完整性总览</h3>
              <p class="panel-desc">离线、过期与异常原因统一清单，支持快速跳转处置。</p>
            </div>
            <div class="integrity-summary-tags">
              <el-tag type="danger" effect="light">高风险 {{ integritySummary.critical }}</el-tag>
              <el-tag type="warning" effect="light">过期待复检 {{ integritySummary.stale }}</el-tag>
              <el-tag type="info" effect="light">离线 {{ integritySummary.offline }}</el-tag>
              <el-tag type="success" effect="light">总问题 {{ integritySummary.total }}</el-tag>
            </div>
          </div>
          <el-table :fit="true" :data="integrityIssueRows" size="small" style="width: 100%" empty-text="当前状态完整性良好">
            <el-table-column prop="module" label="模块" width="120" />
            <el-table-column prop="name" label="对象" min-width="170" show-overflow-tooltip />
            <el-table-column label="状态" width="120">
              <template #default="{ row }">
                <el-tag :type="row.level === 'danger' ? 'danger' : 'warning'">{{ row.statusText }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="异常原因" min-width="260" show-overflow-tooltip />
            <el-table-column label="最后检查" width="180">
              <template #default="{ row }">{{ formatTime(row.checkedAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="150" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="go(row.path)">进入处置</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </el-card>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import * as echarts from 'echarts'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getErrorMessage, isCancelError } from '@/utils/error'

const router = useRouter()
const loading = ref(false)
const refreshing = ref(false)
const deepChecking = ref(false)
const lastUpdated = ref('')
const realtimeRefreshing = ref(false)

const trendRef = ref(null)
let trendChart = null
let realtimeTimer = null
let overviewTimer = null
const SELF_CHECK_HISTORY_KEY = 'lao:dashboard:selfcheck-history'
const STATUS_TIMELINE_HISTORY_KEY = 'lao:dashboard:timeline-history'

const realtime = reactive({
  cpu: 0,
  memory: 0,
  disk: 0,
  network: 0
})

const stats = reactive({
  hostTotal: 0,
  hostOnline: 0,
  hostOffline: 0,
  hostStale: 0,
  dockerTotal: 0,
  dockerOnline: 0,
  dockerOffline: 0,
  dockerStale: 0,
  k8sTotal: 0,
  k8sHealthy: 0,
  k8sUnhealthy: 0,
  k8sMaintenance: 0,
  k8sStale: 0,
  firewallTotal: 0,
  firewallOnline: 0,
  firewallOffline: 0,
  firewallAlert: 0,
  firewallStale: 0,
  domainTotal: 0,
  domainHealthy: 0,
  domainWarning: 0,
  domainCritical: 0,
  domainStale: 0,
  alertTotal: 0,
  alertOpen: 0,
  taskTotal: 0,
  taskEnabled: 0,
  agentTotal: 0,
  agentOnline: 0
})

const backlog = reactive({
  total: 0,
  overdue: 0,
  asset: 0,
  monitor: 0,
  k8s: 0,
  delivery: 0,
  assetOverdue: 0,
  monitorOverdue: 0,
  k8sOverdue: 0,
  deliveryOverdue: 0
})
const backlogActionLoading = reactive({
  asset: false,
  monitor: false,
  k8s: false,
  delivery: false
})
const backlogSource = reactive({
  offlineHostIds: [],
  offlineNetworkDeviceIds: [],
  staleDomainNames: [],
  staleCertIds: [],
  degradedClusterIds: [],
  staleClusterIds: [],
  longRunningExecutionIds: [],
  longRunningWorkflowIds: [],
  failedTerminalSessionIds: []
})

const moduleStatus = reactive({
  cmdb: 'unknown',
  docker: 'unknown',
  k8s: 'unknown',
  firewall: 'unknown',
  domain: 'unknown',
  jump: 'unknown',
  monitor: 'unknown',
  task: 'unknown'
})
const jumpIntegrationState = reactive({
  enabled: false,
  lastSyncStatus: '',
  lastSyncMsg: '',
  lastSyncAt: ''
})
const overviewContractVersion = ref('')
const overviewSummarySnapshot = ref(null)
const overviewQualitySnapshot = ref(null)
const overviewStatusContractSnapshot = ref(null)
const overviewSourceErrorsSnapshot = ref({})

const environmentOptions = [
  { label: '全部环境', value: 'all' },
  { label: '生产', value: 'prod' },
  { label: '预发/测试', value: 'staging' },
  { label: '开发', value: 'dev' }
]

const timeWindowOptions = [
  { label: '最近 6 小时', value: 6 },
  { label: '最近 12 小时', value: 12 },
  { label: '最近 24 小时', value: 24 },
  { label: '最近 72 小时', value: 72 }
]

const scope = reactive({
  environment: 'all',
  clusterId: '',
  namespace: '',
  timeWindowHours: 24
})

const scopeClusters = ref([])
const scopeNamespaces = ref([])
const scopeNamespaceLoadedFor = ref('')
const partialFailureNotice = reactive({ key: '', at: 0 })

const deploymentRisk = reactive({
  clusterId: '',
  clusterName: '未选择集群',
  namespaceLabel: '全部命名空间',
  total: 0,
  healthy: 0,
  progressing: 0,
  degraded: 0,
  gapReplicas: 0,
  podAbnormal: 0,
  topRisk: []
})

const recentAlerts = ref([])
const topHosts = ref([])
const trendRecords = ref([])
const dataSourceDiagnostics = ref([])
const selfCheckHistory = ref([])
const statusTimelineHistory = ref([])
const cmdbHostsSnapshot = ref([])
const dockerHostsSnapshot = ref([])
const k8sClustersSnapshot = ref([])
const networkDevicesSnapshot = ref([])
const firewallsSnapshot = ref([])
const jumpAssetsSnapshot = ref([])

const readHistory = () => {
  try {
    const raw = localStorage.getItem(SELF_CHECK_HISTORY_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter((item) => item && typeof item === 'object')
      .slice(0, 30)
  } catch (err) {
    return []
  }
}
const writeHistory = () => {
  try {
    localStorage.setItem(SELF_CHECK_HISTORY_KEY, JSON.stringify(selfCheckHistory.value.slice(0, 30)))
  } catch (err) {}
}
const clearSelfCheckHistory = () => {
  selfCheckHistory.value = []
  try {
    localStorage.removeItem(SELF_CHECK_HISTORY_KEY)
  } catch (err) {}
}
selfCheckHistory.value = readHistory()

const readTimelineHistory = () => {
  try {
    const raw = localStorage.getItem(STATUS_TIMELINE_HISTORY_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter((item) => item && typeof item === 'object' && item.at)
      .slice(0, 2000)
  } catch (err) {
    return []
  }
}
const writeTimelineHistory = () => {
  try {
    localStorage.setItem(STATUS_TIMELINE_HISTORY_KEY, JSON.stringify(statusTimelineHistory.value.slice(0, 2000)))
  } catch (err) {}
}
statusTimelineHistory.value = readTimelineHistory()

const backlogCards = computed(() => [
  {
    key: 'asset',
    label: '资产管理中心',
    value: backlog.asset,
    overdue: backlog.assetOverdue,
    desc: '主机/网络/防火墙/堡垒机',
    path: '/host'
  },
  {
    key: 'monitor',
    label: '监控告警中心',
    value: backlog.monitor,
    overdue: backlog.monitorOverdue,
    desc: '告警超时、证书与域名风险',
    path: '/monitor/center'
  },
  {
    key: 'k8s',
    label: '容器平台总览',
    value: backlog.k8s,
    overdue: backlog.k8sOverdue,
    desc: '集群异常与状态时效',
    path: '/k8s/overview'
  },
  {
    key: 'delivery',
    label: '服务管理',
    value: backlog.delivery,
    overdue: backlog.deliveryOverdue,
    desc: '交付、自动化、终端会话',
    path: '/delivery/center'
  }
])

const integrityIssueRows = computed(() => {
  const rows = []
  const pushIssue = (item) => {
    rows.push({
      module: item.module || '-',
      name: item.name || '-',
      statusText: item.statusText || '异常',
      reason: item.reason || '状态异常',
      checkedAt: item.checkedAt || '',
      level: item.level === 'danger' ? 'danger' : 'warning',
      stale: item.stale === true,
      offline: item.offline === true,
      path: item.path || '/dashboard'
    })
  }

  toArray(cmdbHostsSnapshot.value).forEach((item) => {
    const online = isOnlineStatus(item.status)
    const stale = isStatusStale(item, 3)
    if (!online) {
      pushIssue({
        module: 'CMDB主机',
        name: item.name || item.ip || item.id,
        statusText: '离线',
        reason: item.status_reason || '主机连通失败',
        checkedAt: statusFreshnessTs(item),
        level: 'danger',
        offline: true,
        path: '/host'
      })
    } else if (stale) {
      pushIssue({
        module: 'CMDB主机',
        name: item.name || item.ip || item.id,
        statusText: '状态过期',
        reason: item.status_reason || '超过 3 分钟未巡检',
        checkedAt: statusFreshnessTs(item),
        level: 'warning',
        stale: true,
        path: '/host'
      })
    }
  })

  toArray(dockerHostsSnapshot.value).forEach((item) => {
    const online = normalizeText(item.status) === 'online'
    const stale = isStatusStale(item, 3)
    if (!online) {
      pushIssue({
        module: 'Docker',
        name: item.name || item.host_id || item.id,
        statusText: '离线',
        reason: item.last_error || item.status_reason || 'Docker 连接失败',
        checkedAt: statusFreshnessTs(item),
        level: 'danger',
        offline: true,
        path: '/docker'
      })
    } else if (stale) {
      pushIssue({
        module: 'Docker',
        name: item.name || item.host_id || item.id,
        statusText: '状态过期',
        reason: item.last_error || '超过 3 分钟未巡检',
        checkedAt: statusFreshnessTs(item),
        level: 'warning',
        stale: true,
        path: '/docker'
      })
    }
  })

  toArray(k8sClustersSnapshot.value).forEach((item) => {
    const online = isOnlineStatus(item.status)
    const maintenance = isMaintenanceStatus(item.status)
    const stale = online && elapsedMinutes(clusterFreshnessTs(item)) >= 15
    if (!online && !maintenance) {
      pushIssue({
        module: 'K8s集群',
        name: item.display_name || item.name || item.id,
        statusText: '异常',
        reason: item.status_reason || `状态 ${item.status || '-'}`,
        checkedAt: clusterFreshnessTs(item),
        level: 'danger',
        path: '/k8s/clusters'
      })
    } else if (stale) {
      pushIssue({
        module: 'K8s集群',
        name: item.display_name || item.name || item.id,
        statusText: '状态过期',
        reason: item.status_reason || '超过 15 分钟未巡检',
        checkedAt: clusterFreshnessTs(item),
        level: 'warning',
        stale: true,
        path: '/k8s/clusters'
      })
    }
  })

  toArray(networkDevicesSnapshot.value).forEach((item) => {
    const status = toNumber(item.status, -1)
    const online = isOnlineStatus(status)
    const stale = online && isStatusStale(item, 5)
    if (status === 2) {
      pushIssue({
        module: '网络设备',
        name: item.name || item.ip || item.id,
        statusText: '告警',
        reason: item.status_reason || '连通异常',
        checkedAt: statusFreshnessTs(item),
        level: 'danger',
        path: '/cmdb/network-devices'
      })
    } else if (!online) {
      pushIssue({
        module: '网络设备',
        name: item.name || item.ip || item.id,
        statusText: '离线',
        reason: item.status_reason || '管理口不可达',
        checkedAt: statusFreshnessTs(item),
        level: 'warning',
        offline: true,
        path: '/cmdb/network-devices'
      })
    } else if (stale) {
      pushIssue({
        module: '网络设备',
        name: item.name || item.ip || item.id,
        statusText: '状态过期',
        reason: item.status_reason || '超过 5 分钟未巡检',
        checkedAt: statusFreshnessTs(item),
        level: 'warning',
        stale: true,
        path: '/cmdb/network-devices'
      })
    }
  })

  toArray(firewallsSnapshot.value).forEach((item) => {
    const status = toNumber(item.status, -1)
    const stale = status === 1 && isStatusStale(item, 5)
    if (status === 2) {
      pushIssue({
        module: '防火墙',
        name: item.name || item.ip || item.id,
        statusText: '告警',
        reason: item.status_reason || 'SNMP 指标异常',
        checkedAt: statusFreshnessTs(item),
        level: 'danger',
        path: '/firewall'
      })
    } else if (status === 0) {
      pushIssue({
        module: '防火墙',
        name: item.name || item.ip || item.id,
        statusText: '离线',
        reason: item.status_reason || '设备不可达',
        checkedAt: statusFreshnessTs(item),
        level: 'warning',
        offline: true,
        path: '/firewall'
      })
    } else if (stale) {
      pushIssue({
        module: '防火墙',
        name: item.name || item.ip || item.id,
        statusText: '状态过期',
        reason: item.status_reason || '超过 5 分钟未采集',
        checkedAt: statusFreshnessTs(item),
        level: 'warning',
        stale: true,
        path: '/firewall'
      })
    }
  })

  const severityRank = (level) => (level === 'danger' ? 2 : 1)
  rows.sort((a, b) => {
    const diff = severityRank(b.level) - severityRank(a.level)
    if (diff !== 0) return diff
    return parseTimestamp(a.checkedAt || 0) - parseTimestamp(b.checkedAt || 0)
  })
  return rows.slice(0, 18)
})

const integritySummary = computed(() => {
  const rows = integrityIssueRows.value
  return {
    total: rows.length,
    critical: rows.filter((item) => item.level === 'danger').length,
    stale: rows.filter((item) => item.stale).length,
    offline: rows.filter((item) => item.offline).length
  }
})

const dataSourceSummary = computed(() => {
  const rows = dataSourceDiagnostics.value
  return {
    ok: rows.filter((item) => item.status === 'ok').length,
    warning: rows.filter((item) => item.status === 'warning').length,
    error: rows.filter((item) => item.status === 'error').length
  }
})

const overviewSourceErrorCount = computed(() => Object.keys(overviewSourceErrorsSnapshot.value || {}).length)

const statusContractSummary = computed(() => {
  const contract = overviewStatusContractSnapshot.value || {}
  const hostMinutes = toNumber(contract.host_stale_minutes, 3)
  const dockerMinutes = toNumber(contract.docker_stale_minutes, 3)
  const k8sMinutes = toNumber(contract.k8s_stale_minutes, 15)
  const networkMinutes = toNumber(contract.network_stale_minutes, 5)
  const firewallMinutes = toNumber(contract.firewall_stale_minutes, 5)
  const domainHours = toNumber(contract.domain_stale_hours, 24)
  const agentSeconds = toNumber(contract.agent_offline_seconds, 90)
  return `主机 ${hostMinutes}m / Docker ${dockerMinutes}m / K8s ${k8sMinutes}m / 网络 ${networkMinutes}m / 防火墙 ${firewallMinutes}m / 域名 ${domainHours}h / Agent ${agentSeconds}s`
})

const statusSeverity = Object.freeze({
  unknown: 0,
  ok: 1,
  warning: 2,
  error: 3
})

const normalizeModuleStatus = (value) => {
  const normalized = normalizeText(value)
  if (normalized === 'ok') return 'ok'
  if (normalized === 'warning') return 'warning'
  if (normalized === 'error') return 'error'
  return 'unknown'
}

const clampPercent = (value) => Math.max(0, Math.min(100, Math.round(toNumber(value, 0))))

const statusTrust = computed(() => {
  const quality = overviewQualitySnapshot.value
  if (quality && typeof quality === 'object') {
    const dimensions = toArray(quality.dimensions)
    const dimScoreByKey = dimensions.reduce((acc, item) => {
      const key = normalizeText(item?.key)
      if (key) acc[key] = clampPercent(item?.score)
      return acc
    }, {})
    const score = clampPercent(quality.trust_score)
    const gradeRaw = String(quality.trust_grade || '').trim().toUpperCase()
    const grade = gradeRaw || (score < 60 ? 'D' : score < 75 ? 'C' : score < 90 ? 'B' : 'A')
    let tagType = 'success'
    if (grade === 'D' || score < 60) {
      tagType = 'danger'
    } else if (grade === 'C' || score < 75) {
      tagType = 'warning'
    } else if (grade === 'B' || score < 90) {
      tagType = ''
    }
    return {
      score,
      grade,
      tagType,
      availabilityScore: clampPercent(dimScoreByKey.availability ?? 100),
      freshnessScore: clampPercent(dimScoreByKey.freshness ?? 100),
      completenessScore: clampPercent(dimScoreByKey.completeness ?? 100),
      consistencyScore: clampPercent(dimScoreByKey.consistency ?? 100),
      summary: quality.summary || '状态链路已生成综合评估，可按建议项优先处置。',
      moduleError: Object.values(moduleStatus).filter((item) => item === 'error').length,
      moduleWarning: Object.values(moduleStatus).filter((item) => item === 'warning').length,
      freshnessIssues: stats.hostStale + stats.dockerStale + stats.k8sStale + stats.firewallStale + stats.domainStale
    }
  }

  const moduleStates = Object.values(moduleStatus)
  const moduleTotal = moduleStates.length || 1
  const moduleError = moduleStates.filter((item) => item === 'error').length
  const moduleWarning = moduleStates.filter((item) => item === 'warning').length
  const availabilityScore = clampPercent(((moduleTotal - moduleError - moduleWarning * 0.5) / moduleTotal) * 100)

  const freshnessTotal = stats.hostTotal + stats.dockerTotal + stats.k8sTotal + stats.firewallTotal + stats.domainTotal
  const freshnessIssues = stats.hostStale + stats.dockerStale + stats.k8sStale + stats.firewallStale + stats.domainStale
  const freshnessScore = freshnessTotal > 0 ? clampPercent((1 - freshnessIssues / freshnessTotal) * 100) : 100

  const completenessRows = fieldCompletenessRows.value
  const completenessScore = completenessRows.length
    ? clampPercent(completenessRows.reduce((sum, item) => sum + toNumber(item.rate, 0), 0) / completenessRows.length)
    : 100

  const consistencyScore = consistencySummary.value.total > 0
    ? clampPercent(100 - consistencySummary.value.total * 8)
    : 100

  const weighted = availabilityScore * 0.32 + freshnessScore * 0.28 + completenessScore * 0.22 + consistencyScore * 0.18
  const riskPenalty = Math.min(25, integritySummary.value.critical * 2 + Math.ceil(integritySummary.value.stale / 3))
  const score = clampPercent(weighted - riskPenalty)

  let grade = 'A'
  let tagType = 'success'
  if (score < 60) {
    grade = 'D'
    tagType = 'danger'
  } else if (score < 75) {
    grade = 'C'
    tagType = 'warning'
  } else if (score < 90) {
    grade = 'B'
    tagType = ''
  }
  const summary = score >= 90
    ? '状态链路整体稳定，可重点关注增长性优化。'
    : score >= 75
      ? '核心链路可用，但存在局部时效或字段缺口，建议按建议清单优先处理。'
      : score >= 60
        ? '状态可信度一般，建议先修复失败链路与跨模块冲突，再做体验优化。'
        : '状态可信度偏低，应优先恢复核心链路可用性并补齐状态字段。'

  return {
    score,
    grade,
    tagType,
    availabilityScore,
    freshnessScore,
    completenessScore,
    consistencyScore,
    summary,
    moduleError,
    moduleWarning,
    freshnessIssues
  }
})

const trustDimensions = computed(() => {
  const backendDims = toArray(overviewQualitySnapshot.value?.dimensions)
    .map((item) => ({
      key: item?.key || '',
      label: item?.label || item?.key || '-',
      score: clampPercent(item?.score),
      detail: item?.detail || ''
    }))
    .filter((item) => item.key)
  if (backendDims.length) {
    return backendDims
  }
  return [
    {
      key: 'availability',
      label: '链路可用性',
      score: statusTrust.value.availabilityScore,
      detail: `失败 ${statusTrust.value.moduleError} 项，降级 ${statusTrust.value.moduleWarning} 项`
    },
    {
      key: 'freshness',
      label: '状态时效',
      score: statusTrust.value.freshnessScore,
      detail: `过期状态 ${statusTrust.value.freshnessIssues} 个`
    },
    {
      key: 'completeness',
      label: '字段完整性',
      score: statusTrust.value.completenessScore,
      detail: `缺口模块 ${fieldCompletenessSummary.value.problem} / ${fieldCompletenessSummary.value.total}`
    },
    {
      key: 'consistency',
      label: '跨模块一致性',
      score: statusTrust.value.consistencyScore,
      detail: `冲突项 ${consistencySummary.value.total}`
    }
  ]
})

const intelligentActions = computed(() => {
  const backendHints = toArray(overviewQualitySnapshot.value?.action_hints)
  if (backendHints.length) {
    return backendHints
      .map((item) => {
        const priority = Math.max(0, Math.min(100, toNumber(item?.priority, 0)))
        const label = String(item?.priority_label || '').trim()
        return {
          key: item?.key || `hint-${priority}-${item?.module || ''}`,
          priority,
          priorityLabel: label || (priority >= 90 ? 'P0' : priority >= 70 ? 'P1' : 'P2'),
          module: item?.module || '全局',
          title: item?.title || '处理建议',
          reason: item?.reason || '',
          action: item?.action || '进入处置',
          path: item?.path || '/dashboard'
        }
      })
      .sort((a, b) => b.priority - a.priority)
  }

  const rows = []
  const add = (item) => {
    rows.push({
      key: item.key,
      priority: Math.max(0, Math.min(100, toNumber(item.priority, 0))),
      priorityLabel: item.priority >= 90 ? 'P0' : item.priority >= 70 ? 'P1' : 'P2',
      module: item.module,
      title: item.title,
      reason: item.reason,
      action: item.action || '进入处置',
      path: item.path || '/dashboard'
    })
  }

  if (dataSourceSummary.value.error > 0) {
    add({
      key: 'pipeline-errors',
      priority: 95,
      module: '全局链路',
      title: '先恢复失败数据链路',
      reason: `当前有 ${dataSourceSummary.value.error} 条链路失败，建议先恢复采集与鉴权。`,
      action: '查看自检',
      path: '/dashboard'
    })
  }

  if (stats.hostOffline > 0 || stats.hostStale > 0) {
    add({
      key: 'cmdb-health',
      priority: stats.hostOffline > 0 ? 90 : 74,
      module: '资产管理',
      title: '修复主机状态可信度',
      reason: `主机离线 ${stats.hostOffline} 台，状态过期 ${stats.hostStale} 台。`,
      action: '进入主机管理',
      path: '/host'
    })
  }

  if (stats.dockerOffline > 0 || stats.dockerStale > 0) {
    add({
      key: 'docker-health',
      priority: stats.dockerOffline > 0 ? 86 : 72,
      module: '容器管理',
      title: '修复 Docker 环境状态',
      reason: `Docker 离线 ${stats.dockerOffline} 台，状态过期 ${stats.dockerStale} 台。`,
      action: '进入 Docker 管理',
      path: '/docker'
    })
  }

  if (stats.k8sUnhealthy > 0 || stats.k8sStale > 0) {
    add({
      key: 'k8s-health',
      priority: stats.k8sUnhealthy > 0 ? 88 : 76,
      module: 'K8s',
      title: '处理异常或过期集群',
      reason: `异常集群 ${stats.k8sUnhealthy} 个，状态过期 ${stats.k8sStale} 个。`,
      action: '进入集群列表',
      path: '/k8s/clusters'
    })
  }

  if (moduleStatus.jump !== 'ok' || normalizeText(jumpIntegrationState.lastSyncStatus) === 'failed') {
    const syncMsg = truncateText(jumpIntegrationState.lastSyncMsg || 'Jump 同步链路异常', 72)
    add({
      key: 'jump-sync',
      priority: 85,
      module: '堡垒机',
      title: '修复 Jump 资产同步',
      reason: `最近同步状态异常：${syncMsg}`,
      action: '进入堡垒机资产',
      path: '/jump/assets'
    })
  }

  if (consistencySummary.value.total > 0) {
    add({
      key: 'cross-consistency',
      priority: 80,
      module: '跨模块',
      title: '消除资产映射冲突',
      reason: `发现 ${consistencySummary.value.total} 条跨模块一致性冲突，可能导致状态误判。`,
      action: '进入完整性总览',
      path: '/dashboard'
    })
  }

  if (fieldCompletenessSummary.value.problem > 0) {
    add({
      key: 'field-completeness',
      priority: 68,
      module: '数据质量',
      title: '补齐状态字段',
      reason: `共有 ${fieldCompletenessSummary.value.problem} 个模块存在字段缺口，建议补齐状态/时间戳/原因。`,
      action: '进入字段完整性',
      path: '/dashboard'
    })
  }

  if (backlog.overdue > 0) {
    add({
      key: 'overdue-backlog',
      priority: 82,
      module: '待处置',
      title: '优先清理超时积压',
      reason: `当前超时积压 ${backlog.overdue} 项，优先处理可显著降低故障放大风险。`,
      action: '进入积压总览',
      path: '/dashboard'
    })
  }

  rows.sort((a, b) => b.priority - a.priority)
  return rows
})

const completenessScore = computed(() => {
  const moduleStates = Object.values(moduleStatus)
  const moduleErrors = moduleStates.filter((item) => item === 'error').length
  const moduleWarnings = moduleStates.filter((item) => item === 'warning').length
  const dataErrors = dataSourceSummary.value.error
  const dataWarnings = dataSourceSummary.value.warning
  const raw = 100 - moduleErrors * 7 - moduleWarnings * 3 - dataErrors * 5 - dataWarnings * 2
  return Math.max(0, Math.min(100, raw))
})

const fieldCompletenessRows = computed(() => {
  const summarize = (module, path, rows, options = {}) => {
    const list = toArray(rows)
    const total = list.length
    if (!total) {
      return {
        module,
        path,
        rate: 0,
        missing: 1,
        issue: '暂无数据'
      }
    }
    let missingStatus = 0
    let missingFreshness = 0
    let missingReason = 0
    let missingIdentity = 0

    list.forEach((row) => {
      const statusRaw = row?.status
      if (!hasValue(statusRaw)) missingStatus += 1
      const checkedAt = options.freshness ? options.freshness(row) : statusFreshnessTs(row)
      if (!hasValue(checkedAt)) missingFreshness += 1
      if (options.identity) {
        const identity = options.identity(row)
        if (!hasValue(identity)) missingIdentity += 1
      }
      if (typeof options.reasonRequired === 'function' && options.reasonRequired(row)) {
        const reason = options.reason(row)
        if (!hasValue(reason)) missingReason += 1
      }
    })

    const missing = missingStatus + missingFreshness + missingReason + missingIdentity
    const denominator = total * 2 + (options.reason ? total : 0) + (options.identity ? total : 0)
    const rate = Math.max(0, Math.min(100, Math.round((1 - missing / Math.max(1, denominator)) * 100)))
    const parts = []
    if (missingStatus > 0) parts.push(`缺状态 ${missingStatus}`)
    if (missingFreshness > 0) parts.push(`缺检测时间 ${missingFreshness}`)
    if (missingReason > 0) parts.push(`缺异常原因 ${missingReason}`)
    if (missingIdentity > 0) parts.push(`缺关联标识 ${missingIdentity}`)
    if (!parts.length) parts.push('字段完整')
    return { module, path, rate, missing, issue: parts.join(' / ') }
  }

  return [
    summarize('CMDB主机', '/host', cmdbHostsSnapshot.value, {
      freshness: statusFreshnessTs,
      identity: (row) => row?.id || row?.ip || row?.name,
      reason: (row) => row?.status_reason,
      reasonRequired: (row) => !isOnlineStatus(row?.status)
    }),
    summarize('Docker', '/docker', dockerHostsSnapshot.value, {
      freshness: statusFreshnessTs,
      identity: (row) => row?.host_id || row?.id,
      reason: (row) => row?.last_error || row?.status_reason,
      reasonRequired: (row) => normalizeText(row?.status) !== 'online'
    }),
    summarize('K8s集群', '/k8s/clusters', k8sClustersSnapshot.value, {
      freshness: clusterFreshnessTs,
      identity: (row) => row?.id || row?.name
    }),
    summarize('网络设备', '/cmdb/network-devices', networkDevicesSnapshot.value, {
      freshness: statusFreshnessTs,
      identity: (row) => row?.id || row?.ip || row?.name,
      reason: (row) => row?.status_reason,
      reasonRequired: (row) => !isOnlineStatus(row?.status)
    }),
    summarize('防火墙', '/firewall', firewallsSnapshot.value, {
      freshness: statusFreshnessTs,
      identity: (row) => row?.id || row?.ip || row?.name,
      reason: (row) => row?.status_reason,
      reasonRequired: (row) => toNumber(row?.status, -1) !== 1
    }),
    summarize('堡垒机资产', '/jump/assets', jumpAssetsSnapshot.value, {
      freshness: (row) => row?.updated_at || row?.created_at,
      identity: (row) => row?.source_ref || row?.id
    })
  ]
})

const fieldCompletenessSummary = computed(() => {
  const rows = fieldCompletenessRows.value
  return {
    total: rows.length,
    problem: rows.filter((item) => item.missing > 0).length
  }
})

const consistencyIssueRows = computed(() => {
  const rows = []
  const cmdbById = new Map()
  const dockerByHostID = new Map()
  const k8sByID = new Map()

  toArray(cmdbHostsSnapshot.value).forEach((item) => {
    if (item?.id) cmdbById.set(item.id, item)
  })
  toArray(dockerHostsSnapshot.value).forEach((item) => {
    if (item?.host_id) dockerByHostID.set(item.host_id, item)
  })
  toArray(k8sClustersSnapshot.value).forEach((item) => {
    if (item?.id) k8sByID.set(item.id, item)
  })

  toArray(dockerHostsSnapshot.value).forEach((docker) => {
    const cmdb = cmdbById.get(docker?.host_id)
    if (!cmdb) {
      rows.push({
        module: 'CMDB ↔ Docker',
        name: docker?.name || docker?.id || '-',
        reason: 'Docker 主机未关联到 CMDB 主机',
        path: '/docker'
      })
      return
    }
    const dockerOnline = normalizeText(docker?.status) === 'online'
    const cmdbOnline = isOnlineStatus(cmdb?.status)
    if (dockerOnline !== cmdbOnline) {
      rows.push({
        module: 'CMDB ↔ Docker',
        name: docker?.name || cmdb?.name || docker?.id || '-',
        reason: `状态冲突：CMDB=${cmdbOnline ? '在线' : '离线'}，Docker=${dockerOnline ? '在线' : '离线'}`,
        path: '/docker'
      })
    }
  })

  toArray(jumpAssetsSnapshot.value).forEach((asset) => {
    const source = normalizeText(asset?.source)
    const ref = asset?.source_ref
    if (!hasValue(ref)) {
      rows.push({
        module: '资产 ↔ 堡垒机',
        name: asset?.name || asset?.id || '-',
        reason: '缺少 source_ref，无法追踪来源资产',
        path: '/jump/assets'
      })
      return
    }
    if (source === 'cmdb_host' && !cmdbById.has(ref)) {
      rows.push({
        module: '资产 ↔ 堡垒机',
        name: asset?.name || ref,
        reason: '来源标记为 CMDB 主机，但在 CMDB 不存在',
        path: '/jump/assets'
      })
    } else if (source === 'docker_host' && !dockerByHostID.has(ref)) {
      rows.push({
        module: 'Docker ↔ 堡垒机',
        name: asset?.name || ref,
        reason: '来源标记为 Docker 主机，但 Docker 侧不存在关联主机',
        path: '/jump/assets'
      })
    } else if (source === 'k8s_cluster' && !k8sByID.has(ref)) {
      rows.push({
        module: 'K8s ↔ 堡垒机',
        name: asset?.name || ref,
        reason: '来源标记为 K8s 集群，但集群列表未找到对应对象',
        path: '/jump/assets'
      })
    }
  })

  return rows.slice(0, 16)
})

const consistencySummary = computed(() => ({
  total: consistencyIssueRows.value.length
}))

const selfCheckHistorySummary = computed(() => {
  const rows = toArray(selfCheckHistory.value)
  return {
    ok: rows.filter((item) => item.status === 'ok').length,
    warning: rows.filter((item) => item.status === 'warning').length,
    error: rows.filter((item) => item.status === 'error').length
  }
})

const moduleCapabilityRows = computed(() => {
  const statusScoreByStatus = {
    ok: 100,
    warning: 72,
    error: 38,
    unknown: 50
  }
  const diagRows = toArray(dataSourceDiagnostics.value)
  const actionRows = toArray(intelligentActions.value)
  const pickWorstDiagnosticStatus = (keywords = []) => {
    const related = diagRows.filter((item) => keywords.some((word) => String(item?.label || '').includes(word)))
    if (!related.length) return 'unknown'
    return related.reduce((worst, item) => {
      const next = normalizeModuleStatus(item?.status)
      return statusSeverity[next] > statusSeverity[worst] ? next : worst
    }, 'ok')
  }
  const hasActionCoverage = (keywords = [], pathPrefixes = []) =>
    actionRows.some((item) => {
      const moduleText = `${item?.module || ''} ${item?.title || ''} ${item?.reason || ''}`
      if (keywords.some((word) => moduleText.includes(word))) return true
      return pathPrefixes.some((prefix) => String(item?.path || '').startsWith(prefix))
    })
  const statusScore = (value) => statusScoreByStatus[normalizeModuleStatus(value)] ?? 50
  const freshnessScore = (staleCount, total) => {
    const t = Math.max(0, toNumber(total, 0))
    const s = Math.max(0, toNumber(staleCount, 0))
    if (t <= 0) return 70
    return clampPercent((1 - Math.min(1, s / t)) * 100)
  }

  const rowDefs = [
    {
      key: 'cmdb',
      label: '资产管理',
      path: '/host',
      linkKeywords: ['CMDB主机状态链路', '网络设备链路'],
      actionKeywords: ['资产管理', '主机', 'CMDB'],
      actionPaths: ['/host', '/cmdb'],
      status: moduleStatus.cmdb,
      targetScore: 90,
      freshScore: freshnessScore(stats.hostStale, stats.hostTotal + Math.max(0, toNumber(networkDevicesSnapshot.value?.length, 0)))
    },
    {
      key: 'docker',
      label: 'Docker',
      path: '/docker',
      linkKeywords: ['Docker状态链路'],
      actionKeywords: ['Docker', '容器管理'],
      actionPaths: ['/docker'],
      status: moduleStatus.docker,
      targetScore: 88,
      freshScore: freshnessScore(stats.dockerStale, stats.dockerTotal)
    },
    {
      key: 'k8s',
      label: 'K8s',
      path: '/k8s/clusters',
      linkKeywords: ['K8s集群状态链路'],
      actionKeywords: ['K8s', '集群'],
      actionPaths: ['/k8s'],
      status: moduleStatus.k8s,
      targetScore: 90,
      freshScore: freshnessScore(stats.k8sStale, stats.k8sTotal)
    },
    {
      key: 'monitor',
      label: '监控告警',
      path: '/monitor/center',
      linkKeywords: ['告警事件链路', '监控指标链路', 'Agent在线链路'],
      actionKeywords: ['监控', '告警'],
      actionPaths: ['/monitor', '/alert'],
      status: moduleStatus.monitor,
      targetScore: 92,
      freshScore: freshnessScore(stats.alertOpen > 0 ? 1 : 0, Math.max(1, stats.alertTotal))
    },
    {
      key: 'task',
      label: '任务交付',
      path: '/delivery/center',
      linkKeywords: ['任务与工单链路'],
      actionKeywords: ['任务', '交付', '工单'],
      actionPaths: ['/delivery', '/task', '/workflow'],
      status: moduleStatus.task,
      targetScore: 90,
      freshScore: freshnessScore(backlog.deliveryOverdue, Math.max(1, backlog.delivery))
    },
    {
      key: 'firewall',
      label: '防火墙',
      path: '/firewall',
      linkKeywords: ['防火墙链路'],
      actionKeywords: ['防火墙'],
      actionPaths: ['/firewall'],
      status: moduleStatus.firewall,
      targetScore: 88,
      freshScore: freshnessScore(stats.firewallStale, stats.firewallTotal)
    },
    {
      key: 'domain',
      label: '域名证书',
      path: '/domain/ssl',
      linkKeywords: ['域名与证书链路'],
      actionKeywords: ['域名', '证书'],
      actionPaths: ['/domain'],
      status: moduleStatus.domain,
      targetScore: 89,
      freshScore: freshnessScore(stats.domainStale, stats.domainTotal)
    },
    {
      key: 'jump',
      label: '堡垒机',
      path: '/jump/assets',
      linkKeywords: ['JumpServer会话链路', '堡垒机资产映射链路'],
      actionKeywords: ['堡垒机', 'Jump'],
      actionPaths: ['/jump'],
      status: moduleStatus.jump,
      targetScore: 90,
      freshScore: freshnessScore(normalizeText(jumpIntegrationState.lastSyncStatus) === 'failed' ? 1 : 0, 1)
    }
  ]

  return rowDefs.map((item) => {
    const linkStatus = pickWorstDiagnosticStatus(item.linkKeywords)
    const linkScore = statusScore(linkStatus)
    const stateScore = statusScore(item.status)
    const automationHit = hasActionCoverage(item.actionKeywords, item.actionPaths)
    const automationScore = automationHit ? 100 : (normalizeModuleStatus(item.status) === 'ok' ? 82 : 68)
    const score = clampPercent(Math.round(stateScore * 0.35 + linkScore * 0.3 + automationScore * 0.2 + item.freshScore * 0.15))
    const targetScore = clampPercent(item.targetScore || 88)
    const trend = capabilityTrendByTarget(score, targetScore)
    const level = score >= 85 ? 'complete' : score >= 65 ? 'partial' : 'gap'
    const suggestion = level === 'complete'
      ? '保持巡检频率，关注趋势波动'
      : level === 'partial'
        ? '优先补齐降级链路与字段缺口'
        : '需要优先修复核心链路并补全自动处置'
    return {
      key: item.key,
      label: item.label,
      path: item.path,
      status: normalizeModuleStatus(item.status),
      linkStatus,
      automationScore,
      targetScore,
      currentScore: score,
      trendDirection: trend.direction,
      trendDelta: trend.delta,
      score,
      level,
      suggestion
    }
  }).sort((a, b) => a.score - b.score)
})

const capabilitySummary = computed(() => {
  const rows = moduleCapabilityRows.value
  return {
    complete: rows.filter((item) => item.level === 'complete').length,
    partial: rows.filter((item) => item.level === 'partial').length,
    gap: rows.filter((item) => item.level === 'gap').length
  }
})

const capabilityGapRows = computed(() => {
  const rows = moduleCapabilityRows.value
    .filter((item) => item.level !== 'complete')
    .map((item) => ({
      module: item.label,
      gap: item.level === 'gap' ? '核心链路/状态判定能力不足' : '链路存在降级，自动处置能力待加强',
      impact: item.level === 'gap' ? '状态失真风险较高，影响故障响应' : '存在误判与处置时延风险',
      path: item.path,
      score: item.score
    }))
    .sort((a, b) => a.score - b.score)
  return rows.slice(0, 8)
})

const gapChecklistSummary = computed(() => ({
  p0: gapChecklistRows.value.filter((item) => item.priority >= 90).length,
  p1: gapChecklistRows.value.filter((item) => item.priority >= 70 && item.priority < 90).length,
  p2: gapChecklistRows.value.filter((item) => item.priority < 70).length
}))

const gapTargetByLabel = (label) => {
  const text = String(label || '')
  if (text.includes('CMDB')) return '主机状态可达且检测时间不超阈值'
  if (text.includes('Docker')) return '环境在线且与CMDB状态一致'
  if (text.includes('K8s')) return '集群健康状态实时更新'
  if (text.includes('Jump')) return '同步成功且授权数据可拉取'
  if (text.includes('防火墙')) return '设备采集成功且无告警阻塞'
  if (text.includes('域名') || text.includes('证书')) return '健康检查与证书检测按计划更新'
  if (text.includes('任务') || text.includes('工单')) return '调度链路成功，超时积压清零'
  if (text.includes('监控')) return '实时指标与趋势链路均可用'
  if (text.includes('字段完整')) return '状态、时间戳、异常原因字段齐全'
  if (text.includes('一致性')) return '跨模块映射无冲突'
  return '链路状态稳定且可持续自动恢复'
}

const gapImpactByLabel = (label) => {
  const text = String(label || '')
  if (text.includes('Jump')) return '会话授权与资产同步失真，影响堡垒机可用性'
  if (text.includes('CMDB') || text.includes('Docker') || text.includes('K8s')) return '资产在线判断偏差，影响容量与故障定位'
  if (text.includes('监控') || text.includes('告警')) return '告警可信度下降，可能漏报或误报'
  if (text.includes('任务') || text.includes('工单')) return '交付链路阻塞，处理时延增加'
  if (text.includes('字段完整') || text.includes('一致性')) return '状态推断偏差放大，跨模块联动失效'
  return '影响系统状态可信度与处置效率'
}

const gapChecklistRows = computed(() => {
  const rows = []
  const add = (item) => {
    const priority = Math.max(0, Math.min(100, toNumber(item.priority, 0)))
    rows.push({
      key: item.key || `${item.gap}-${rows.length}`,
      priority,
      priorityLabel: priority >= 90 ? 'P0' : priority >= 70 ? 'P1' : 'P2',
      gap: item.gap || '-',
      current: item.current || '-',
      target: item.target || '-',
      impact: item.impact || '影响排障与稳定性',
      path: item.path || '/dashboard'
    })
  }

  if (overviewSourceErrorCount.value > 0) {
    add({
      key: 'overview-source-errors',
      priority: 96,
      gap: '聚合源存在失败',
      current: `概览源失败 ${overviewSourceErrorCount.value} 项`,
      target: '核心链路聚合可用率 >= 99%',
      impact: '聚合视图降级为多接口拼接，增加状态分叉风险',
      path: '/dashboard'
    })
  }

  toArray(dataSourceDiagnostics.value)
    .filter((item) => item?.status === 'warning' || item?.status === 'error')
    .forEach((item) => {
      add({
        key: `diagnostic-${item.label}`,
        priority: item.status === 'error' ? 90 : 74,
        gap: item.label || '链路异常',
        current: truncateText(item.message || '状态异常', 72),
        target: gapTargetByLabel(item.label),
        impact: gapImpactByLabel(item.label),
        path: item.path || '/dashboard'
      })
    })

  toArray(intelligentActions.value).slice(0, 6).forEach((item) => {
    add({
      key: `action-${item.key}`,
      priority: Math.max(66, toNumber(item.priority, 66)),
      gap: item.title || '待处置项',
      current: truncateText(item.reason || '', 72),
      target: '告警与状态链路恢复到正常并维持稳定',
      impact: `${item.module || '全局'}模块存在稳定性风险`,
      path: item.path || '/dashboard'
    })
  })

  const dedup = new Map()
  rows.forEach((item) => {
    const key = `${item.gap}|${item.path}`
    const old = dedup.get(key)
    if (!old || item.priority > old.priority) {
      dedup.set(key, item)
    }
  })
  return Array.from(dedup.values())
    .sort((a, b) => b.priority - a.priority)
    .slice(0, 14)
})

const timelineSamples24h = computed(() => {
  const from = nowMs() - 24 * 60 * 60 * 1000
  return toArray(statusTimelineHistory.value)
    .filter((item) => {
      const ts = parseTimestamp(item?.at)
      return ts && ts >= from
    })
    .sort((a, b) => parseTimestamp(a?.at || 0) - parseTimestamp(b?.at || 0))
})

const statusFlappingSummary = computed(() => {
  const samples = timelineSamples24h.value
  if (!samples.length) {
    return {
      samples: 0,
      hostOnlineRate: 0,
      dockerOnlineRate: 0,
      k8sHealthyRate: 0,
      chainAvailabilityRate: 100,
      avgRecoveryMinutes: 0,
      medianRecoveryMinutes: 0,
      recoveryCount: 0
    }
  }

  const ratioAvg = (numeratorKey, denominatorKey) => {
    const rates = samples
      .map((item) => {
        const n = toNumber(item?.stats?.[numeratorKey], 0)
        const d = toNumber(item?.stats?.[denominatorKey], 0)
        if (d <= 0) return null
        return n / d
      })
      .filter((item) => item !== null)
    if (!rates.length) return 0
    return clampPercent((rates.reduce((sum, item) => sum + item, 0) / rates.length) * 100)
  }

  const chainPassCount = samples.filter((item) => {
    const modules = item?.modules || {}
    const hasModuleError = Object.values(modules).some((status) => status === 'error')
    const hasDiagnosticError = toArray(item?.diagnostics).some((row) => row?.status === 'error')
    return !hasModuleError && !hasDiagnosticError
  }).length

  const recoveryDurations = []
  const moduleKeys = ['cmdb', 'docker', 'k8s', 'firewall', 'domain', 'jump', 'monitor', 'task']
  moduleKeys.forEach((key) => {
    let unstableSince = null
    samples.forEach((sample) => {
      const current = normalizeModuleStatus(sample?.modules?.[key])
      const at = parseTimestamp(sample?.at)
      if (!at) return
      if (current !== 'ok') {
        if (!unstableSince) unstableSince = at
        return
      }
      if (unstableSince) {
        recoveryDurations.push(Math.max(1, Math.round((at - unstableSince) / 60000)))
        unstableSince = null
      }
    })
  })

  const avgRecoveryMinutes = recoveryDurations.length
    ? Math.round(recoveryDurations.reduce((sum, item) => sum + item, 0) / recoveryDurations.length)
    : 0
  const sortedRecovery = [...recoveryDurations].sort((a, b) => a - b)
  const medianRecoveryMinutes = sortedRecovery.length
    ? sortedRecovery[Math.floor(sortedRecovery.length / 2)]
    : 0

  return {
    samples: samples.length,
    hostOnlineRate: ratioAvg('hostOnline', 'hostTotal'),
    dockerOnlineRate: ratioAvg('dockerOnline', 'dockerTotal'),
    k8sHealthyRate: ratioAvg('k8sHealthy', 'k8sTotal'),
    chainAvailabilityRate: clampPercent((chainPassCount / samples.length) * 100),
    avgRecoveryMinutes,
    medianRecoveryMinutes,
    recoveryCount: recoveryDurations.length
  }
})

const failedLinkTopRows = computed(() => {
  const counter = new Map()
  timelineSamples24h.value.forEach((sample) => {
    const diagnostics = toArray(sample?.diagnostics).filter((item) => item?.status === 'error' || item?.status === 'warning')
    diagnostics.forEach((item) => {
      const key = item?.label || '未命名链路'
      const prev = counter.get(key) || {
        label: key,
        path: item?.path || '/dashboard',
        count: 0,
        errorCount: 0,
        warningCount: 0,
        lastAt: ''
      }
      prev.count += 1
      if (item.status === 'error') prev.errorCount += 1
      if (item.status === 'warning') prev.warningCount += 1
      prev.lastAt = sample?.at || prev.lastAt
      counter.set(key, prev)
    })
  })
  return Array.from(counter.values())
    .sort((a, b) => (b.errorCount * 2 + b.warningCount) - (a.errorCount * 2 + a.warningCount))
    .slice(0, 8)
})

const authHeaders = () => ({ Authorization: `Bearer ${localStorage.getItem('token')}` })

const go = (path) => router.push(path)

const toArray = (v) => (Array.isArray(v) ? v : [])
const toNumber = (v, fallback = 0) => {
  const n = Number(v)
  return Number.isFinite(n) ? n : fallback
}
const normalizeText = (v) => String(v ?? '').trim().toLowerCase()
const nowMs = () => Date.now()

const inferClusterEnvironment = (cluster) => {
  const raw = `${cluster?.display_name || ''} ${cluster?.name || ''} ${cluster?.description || ''}`
  const text = normalizeText(raw)
  if (text.includes('prod') || text.includes('生产')) return 'prod'
  if (text.includes('staging') || text.includes('stage') || text.includes('测试') || text.includes('预发')) return 'staging'
  if (text.includes('dev') || text.includes('开发')) return 'dev'
  return 'other'
}

const clusterInEnvironment = (cluster) => {
  if (!cluster) return false
  if (scope.environment === 'all') return true
  return inferClusterEnvironment(cluster) === scope.environment
}

const clusterInScope = (cluster) => {
  if (!clusterInEnvironment(cluster)) return false
  if (!scope.clusterId) return true
  return cluster.id === scope.clusterId
}

const parseTimestamp = (value) => {
  if (!value) return null
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? null : ts
}

const elapsedMinutes = (value) => {
  const ts = parseTimestamp(value)
  if (!ts) return 0
  const diff = Math.floor((nowMs() - ts) / 60000)
  return diff > 0 ? diff : 0
}

const statusFreshnessTs = (row) =>
  row?.last_check_at ||
  row?.last_seen_at ||
  row?.last_heartbeat_at ||
  row?.updated_at

const isStatusStale = (row, minutes = 3) => {
  const ts = parseTimestamp(statusFreshnessTs(row))
  if (!ts) return true
  return nowMs() - ts > minutes * 60 * 1000
}

const isOnlineStatus = (status) => {
  const normalized = normalizeText(status)
  return normalized === 'online' || normalized === 'ready' || normalized === 'running' || Number(status) === 1
}

const isMaintenanceStatus = (status) => {
  const normalized = normalizeText(status)
  return normalized === 'maintenance' || normalized === 'maintain' || Number(status) === 2
}

const isAlertOpen = (status) => {
  const v = Number(status)
  return v === 0 || v === 1
}

const isWorkorderApprovalPending = (status) => {
  const v = Number(status)
  return v === 0 || v === 1
}

const isScheduleEnabled = (row) => row?.enabled === true || Number(row?.enabled) === 1

const clusterFreshnessTs = (row) =>
  row?.last_check_at ||
  row?.last_sync_at ||
  row?.last_seen_at ||
  row?.last_heartbeat_at ||
  row?.last_checked_at ||
  row?.updated_at

const inTimeWindow = (value) => {
  const ts = parseTimestamp(value)
  if (!ts) return true
  const hours = toNumber(scope.timeWindowHours, 24)
  return ts >= nowMs() - hours * 60 * 60 * 1000
}

const hasValue = (value) => {
  if (value === null || value === undefined) return false
  if (typeof value === 'string') return value.trim() !== ''
  return true
}

const truncateText = (value, max = 120) => {
  const text = String(value || '').trim()
  if (text.length <= max) return text
  return `${text.slice(0, Math.max(0, max - 1))}…`
}

const pushSelfCheckHistory = (source, probeSummary = null) => {
  const errors = dataSourceSummary.value.error
  const warnings = dataSourceSummary.value.warning
  const status = errors > 0 ? 'error' : warnings > 0 ? 'warning' : 'ok'
  const summaryParts = [
    `失败 ${errors}`,
    `降级 ${warnings}`,
    `高风险 ${integritySummary.value.critical}`
  ]
  if (probeSummary && probeSummary.total > 0) {
    summaryParts.push(`主动探测 ${probeSummary.ok}/${probeSummary.total}`)
  }
  const entry = {
    at: new Date().toISOString(),
    source,
    score: completenessScore.value,
    status,
    summary: summaryParts.join(' / ')
  }
  const prev = selfCheckHistory.value[0]
  const recentGapMs = prev ? nowMs() - new Date(prev.at).getTime() : Number.MAX_SAFE_INTEGER
  const sameSignature =
    prev &&
    prev.status === entry.status &&
    prev.score === entry.score &&
    prev.summary === entry.summary
  if (source !== 'manual' && sameSignature && recentGapMs < 5 * 60 * 1000) return

  selfCheckHistory.value = [entry, ...toArray(selfCheckHistory.value)].slice(0, 30)
  writeHistory()
}

const pushStatusTimelineSnapshot = (source) => {
  const entry = {
    at: new Date().toISOString(),
    source,
    modules: {
      cmdb: moduleStatus.cmdb,
      docker: moduleStatus.docker,
      k8s: moduleStatus.k8s,
      firewall: moduleStatus.firewall,
      domain: moduleStatus.domain,
      jump: moduleStatus.jump,
      monitor: moduleStatus.monitor,
      task: moduleStatus.task
    },
    stats: {
      hostOnline: stats.hostOnline,
      hostTotal: stats.hostTotal,
      dockerOnline: stats.dockerOnline,
      dockerTotal: stats.dockerTotal,
      k8sHealthy: stats.k8sHealthy,
      k8sTotal: stats.k8sTotal
    },
    diagnostics: toArray(dataSourceDiagnostics.value).map((item) => ({
      label: item?.label || '-',
      status: normalizeModuleStatus(item?.status),
      path: item?.path || '/dashboard'
    }))
  }

  const signature = JSON.stringify({
    modules: entry.modules,
    stats: entry.stats,
    diagnostics: entry.diagnostics
  })
  const prev = statusTimelineHistory.value[0]
  if (prev) {
    const prevAt = parseTimestamp(prev.at)
    const nowAt = parseTimestamp(entry.at)
    if (prevAt && nowAt && nowAt-prevAt < 45 * 1000) {
      const prevSignature = JSON.stringify({
        modules: prev.modules || {},
        stats: prev.stats || {},
        diagnostics: toArray(prev.diagnostics)
      })
      if (prevSignature === signature) {
        return
      }
    }
  }

  statusTimelineHistory.value = [entry, ...toArray(statusTimelineHistory.value)].slice(0, 2000)
  writeTimelineHistory()
}

const runDeepStatusProbe = async () => {
  const jobs = []

  const hostTargets = toArray(cmdbHostsSnapshot.value)
    .filter((item) => !isOnlineStatus(item.status) || isStatusStale(item, 3))
    .map((item) => item.id)
    .filter(Boolean)
    .slice(0, 3)
  hostTargets.forEach((id) => {
    jobs.push(axios.post(`/api/v1/cmdb/hosts/${id}/test`, {}, { headers: authHeaders() }))
  })

  const dockerTargets = toArray(dockerHostsSnapshot.value)
    .filter((item) => normalizeText(item.status) !== 'online' || isStatusStale(item, 3))
    .map((item) => item.id)
    .filter(Boolean)
    .slice(0, 3)
  dockerTargets.forEach((id) => {
    jobs.push(axios.post(`/api/v1/docker/hosts/${id}/test`, {}, { headers: authHeaders() }))
  })

  const networkTargets = toArray(networkDevicesSnapshot.value)
    .filter((item) => !isOnlineStatus(item.status) || isStatusStale(item, 5))
    .map((item) => item.id)
    .filter(Boolean)
    .slice(0, 3)
  networkTargets.forEach((id) => {
    jobs.push(axios.post(`/api/v1/cmdb/network-devices/${id}/test`, {}, { headers: authHeaders() }))
  })

  const k8sTargets = toArray(k8sClustersSnapshot.value)
    .filter((item) => (!isOnlineStatus(item.status) && !isMaintenanceStatus(item.status)) || elapsedMinutes(clusterFreshnessTs(item)) >= 15)
    .map((item) => item.id)
    .filter(Boolean)
    .slice(0, 3)
  k8sTargets.forEach((id) => {
    jobs.push(axios.post(`/api/v1/k8s/clusters/${id}/test`, {}, { headers: authHeaders() }))
  })

  if (!jobs.length) return { total: 0, ok: 0, fail: 0 }
  const result = summarizeSettled(await Promise.allSettled(jobs))
  return { total: jobs.length, ok: result.ok, fail: result.fail }
}

const summarizeSettled = (results) => {
  const ok = results.filter((item) => item.status === 'fulfilled').length
  return { ok, fail: results.length - ok }
}

const throttledPartialFailureMessage = (failures) => {
  if (!failures.length) return
  const key = failures.join('|')
  const now = nowMs()
  if (partialFailureNotice.key === key && now - partialFailureNotice.at < 15000) return
  partialFailureNotice.key = key
  partialFailureNotice.at = now
  ElMessage.warning(`部分数据加载失败：${failures.join('、')}`)
}

const ensureScopeNamespaces = async (force = false) => {
  if (!scope.clusterId) {
    scopeNamespaces.value = []
    scope.namespace = ''
    scopeNamespaceLoadedFor.value = ''
    return
  }
  if (!force && scopeNamespaceLoadedFor.value === scope.clusterId && scopeNamespaces.value.length) return
  try {
    const res = await axios.get(`/api/v1/k8s/clusters/${scope.clusterId}/namespaces`, { headers: authHeaders() })
    const list = toArray(res.data?.data).map((item) => item.name).filter(Boolean)
    scopeNamespaces.value = list
    scopeNamespaceLoadedFor.value = scope.clusterId
    if (scope.namespace && !scopeNamespaces.value.includes(scope.namespace)) {
      scope.namespace = ''
    }
  } catch (err) {
    scopeNamespaces.value = []
    scope.namespace = ''
  }
}

const handleScopeEnvironmentChange = async () => {
  if (scope.clusterId) {
    scope.clusterId = ''
    scope.namespace = ''
    scopeNamespaceLoadedFor.value = ''
  }
  await refreshDashboard()
}

const handleScopeClusterChange = async () => {
  scope.namespace = ''
  await ensureScopeNamespaces(true)
  await refreshDashboard()
}

const goDeploymentCenter = () => {
  router.push({
    path: '/k8s/deployments',
    query: {
      clusterId: scope.clusterId || undefined,
      namespace: scope.namespace || undefined
    }
  })
}

const refreshDeploymentRisk = async (scopedClusters, failures) => {
  const targetCluster = scope.clusterId
    ? scopedClusters.find((item) => item.id === scope.clusterId)
    : scopedClusters[0]

  if (!targetCluster) {
    deploymentRisk.clusterId = ''
    deploymentRisk.clusterName = '未选择集群'
    deploymentRisk.namespaceLabel = '全部命名空间'
    deploymentRisk.total = 0
    deploymentRisk.healthy = 0
    deploymentRisk.progressing = 0
    deploymentRisk.degraded = 0
    deploymentRisk.gapReplicas = 0
    deploymentRisk.podAbnormal = 0
    deploymentRisk.topRisk = []
    return
  }

  const namespace = scope.namespace || ''
  deploymentRisk.clusterId = targetCluster.id
  deploymentRisk.clusterName = targetCluster.display_name || targetCluster.name
  deploymentRisk.namespaceLabel = namespace || '全部命名空间'

  try {
    const workloadsRes = await axios.get(`/api/v1/k8s/clusters/${targetCluster.id}/workloads`, {
      headers: authHeaders(),
      params: { namespace }
    })
    const all = toArray(workloadsRes.data?.data)
    const deployments = all.filter((item) => item.kind === 'Deployment')
    deploymentRisk.total = deployments.length
    deploymentRisk.healthy = 0
    deploymentRisk.progressing = 0
    deploymentRisk.degraded = 0
    deploymentRisk.gapReplicas = 0
    const topRisk = []

    deployments.forEach((item) => {
      const replicas = toNumber(item.replicas, 0)
      const ready = toNumber(item.ready, 0)
      const available = toNumber(item.available, 0)
      const gap = Math.max(0, replicas - ready)
      deploymentRisk.gapReplicas += gap
      if (replicas <= 0) {
        topRisk.push({ key: `${item.namespace}/${item.name}`, name: `${item.namespace}/${item.name}`, label: 'Scaled 0', type: 'info' })
        return
      }
      if (ready >= replicas && available >= replicas) {
        deploymentRisk.healthy += 1
        return
      }
      if (ready > 0 || available > 0) {
        deploymentRisk.progressing += 1
        topRisk.push({ key: `${item.namespace}/${item.name}`, name: `${item.namespace}/${item.name}`, label: `${ready}/${replicas}`, type: 'warning' })
        return
      }
      deploymentRisk.degraded += 1
      topRisk.push({ key: `${item.namespace}/${item.name}`, name: `${item.namespace}/${item.name}`, label: '0 Ready', type: 'danger' })
    })

    const candidateNamespaces = namespace
      ? [namespace]
      : Array.from(new Set(deployments.map((item) => item.namespace).filter(Boolean))).slice(0, 6)
    const podCalls = await Promise.allSettled(
      candidateNamespaces.map((ns) => axios.get(`/api/v1/k8s/clusters/${targetCluster.id}/namespaces/${ns}/pods`, { headers: authHeaders() }))
    )
    const abnormal = []
    podCalls.forEach((item) => {
      if (item.status !== 'fulfilled') return
      const rows = toArray(item.value.data?.data)
      rows.forEach((pod) => {
        const status = normalizeText(pod.status)
        if (pod.owner_kind !== 'Deployment') return
        if (status === 'running' || status === 'succeeded' || status === 'completed') return
        abnormal.push(pod)
      })
    })
    deploymentRisk.podAbnormal = abnormal.length
    deploymentRisk.topRisk = topRisk.slice(0, 6)
  } catch (err) {
    deploymentRisk.total = 0
    deploymentRisk.healthy = 0
    deploymentRisk.progressing = 0
    deploymentRisk.degraded = 0
    deploymentRisk.gapReplicas = 0
    deploymentRisk.podAbnormal = 0
    deploymentRisk.topRisk = []
    failures.push(getErrorMessage(err, 'Deployment 风险'))
  }
}

const runBacklogAction = async (key) => {
  if (!Object.prototype.hasOwnProperty.call(backlogActionLoading, key)) return
  if (backlogActionLoading[key]) return
  backlogActionLoading[key] = true
  try {
    if (key === 'asset') {
      const hostIds = backlogSource.offlineHostIds.slice(0, 3)
      const networkIds = backlogSource.offlineNetworkDeviceIds.slice(0, 3)
      await ElMessageBox.confirm(
        `将同步防火墙资产，并巡检离线主机(${hostIds.length})/网络设备(${networkIds.length})，确认执行吗？`,
        '资产一键处置',
        { type: 'warning' }
      )
      const jobs = [
        axios.post('/api/v1/cmdb/network-devices/sync/firewalls', {}, { headers: authHeaders() }),
        ...hostIds.map((id) => axios.post(`/api/v1/cmdb/hosts/${id}/test`, {}, { headers: authHeaders() })),
        ...networkIds.map((id) => axios.post(`/api/v1/cmdb/network-devices/${id}/test`, {}, { headers: authHeaders() }))
      ]
      const summary = summarizeSettled(await Promise.allSettled(jobs))
      ElMessage.success(`资产处置完成：成功 ${summary.ok}，失败 ${summary.fail}`)
    } else if (key === 'monitor') {
      const domains = backlogSource.staleDomainNames.slice(0, 5)
      const certs = backlogSource.staleCertIds.slice(0, 5)
      if (!domains.length && !certs.length) {
        ElMessage.info('当前没有待复检的域名/证书项')
        return
      }
      await ElMessageBox.confirm(
        `将复检域名(${domains.length})和证书(${certs.length})，确认执行吗？`,
        '监控中心一键处置',
        { type: 'warning' }
      )
      const jobs = [
        ...domains.map((domain) => axios.post('/api/v1/domain/domains/check', { domain }, { headers: authHeaders() })),
        ...certs.map((id) => axios.post(`/api/v1/domain/certs/${id}/check`, {}, { headers: authHeaders() }))
      ]
      const summary = summarizeSettled(await Promise.allSettled(jobs))
      ElMessage.success(`监控复检完成：成功 ${summary.ok}，失败 ${summary.fail}`)
    } else if (key === 'k8s') {
      const clusterIds = [...new Set([...backlogSource.degradedClusterIds, ...backlogSource.staleClusterIds])].slice(0, 5)
      if (!clusterIds.length) {
        ElMessage.info('当前没有需要巡检的集群')
        return
      }
      await ElMessageBox.confirm(
        `将对 ${clusterIds.length} 个异常/超时集群执行连接测试，确认执行吗？`,
        'K8s一键处置',
        { type: 'warning' }
      )
      const jobs = clusterIds.map((id) => axios.post(`/api/v1/k8s/clusters/${id}/test`, {}, { headers: authHeaders() }))
      const summary = summarizeSettled(await Promise.allSettled(jobs))
      ElMessage.success(`K8s巡检完成：成功 ${summary.ok}，失败 ${summary.fail}`)
    } else if (key === 'delivery') {
      const executionIds = backlogSource.longRunningExecutionIds.slice(0, 3)
      const workflowIds = backlogSource.longRunningWorkflowIds.slice(0, 3)
      const failedSessions = backlogSource.failedTerminalSessionIds.slice(0, 10)
      if (!executionIds.length && !workflowIds.length && !failedSessions.length) {
        ElMessage.info('当前没有可自动处置的服务管理积压')
        return
      }
      await ElMessageBox.confirm(
        `将取消超时执行(${executionIds.length})、超时流程(${workflowIds.length})，并清理失败会话(${failedSessions.length})，确认执行吗？`,
        '服务管理一键处置',
        { type: 'warning' }
      )
      const jobs = [
        ...executionIds.map((id) => axios.post(`/api/v1/cicd/executions/${id}/cancel`, {}, { headers: authHeaders() })),
        ...workflowIds.map((id) => axios.post(`/api/v1/workflow/executions/${id}/cancel`, {}, { headers: authHeaders() })),
        ...failedSessions.map((id) => axios.delete(`/api/v1/terminal/sessions/${id}/purge`, { headers: authHeaders() }))
      ]
      const summary = summarizeSettled(await Promise.allSettled(jobs))
      ElMessage.success(`服务管理处置完成：成功 ${summary.ok}，失败 ${summary.fail}`)
    }
    await refreshDashboard()
  } catch (err) {
    if (isCancelError(err)) return
    ElMessage.error(getErrorMessage(err, '执行一键处置失败'))
  } finally {
    backlogActionLoading[key] = false
  }
}

const safePercent = (v) => Math.max(0, Math.min(100, toNumber(v)))
const formatPercent = (v) => `${toNumber(v).toFixed(1)}%`
const formatNumber = (v, digits = 2) => toNumber(v).toFixed(digits)
const formatDurationMinutes = (minutes) => {
  const value = Math.max(0, toNumber(minutes, 0))
  if (value <= 0) return '-'
  if (value < 1) return '<1 分钟'
  return `${Math.round(value)} 分钟`
}
const formatTime = (val) => {
  if (!val) return '-'
  const t = new Date(val)
  if (Number.isNaN(t.getTime())) return '-'
  return t.toLocaleString()
}

const moduleTagType = (status) => {
  if (status === 'ok') return 'success'
  if (status === 'warning') return 'warning'
  if (status === 'error') return 'danger'
  return 'info'
}

const moduleTagText = (status) => {
  if (status === 'ok') return '正常'
  if (status === 'warning') return '预警'
  if (status === 'error') return '异常'
  return '未知'
}

const capabilityTrendByTarget = (currentScore, targetScore) => {
  const delta = Math.round((Number(currentScore) || 0) - (Number(targetScore) || 0))
  if (delta >= 3) return { direction: 'up', delta }
  if (delta <= -3) return { direction: 'down', delta }
  return { direction: 'flat', delta }
}

const capabilityTrendArrow = (direction) => {
  if (direction === 'up') return '↑'
  if (direction === 'down') return '↓'
  return '→'
}

const capabilityTrendTag = (direction) => {
  if (direction === 'up') return 'success'
  if (direction === 'down') return 'danger'
  return 'info'
}

const formatTrendDelta = (value) => {
  const delta = Math.round(Number(value) || 0)
  if (delta > 0) return `+${delta}%`
  return `${delta}%`
}

const extractData = (result) => {
  if (result.status !== 'fulfilled') return null
  const body = result.value?.data
  if (!body || typeof body !== 'object') return null
  if (Object.prototype.hasOwnProperty.call(body, 'code')) {
    return body.code === 0 ? body.data : null
  }
  return body.data ?? null
}

const extractFailure = (result, label) => {
  if (result.status !== 'rejected') return ''
  return `${label}(${getErrorMessage(result.reason, '请求失败')})`
}

const wrapFulfilled = (data) => ({
  status: 'fulfilled',
  value: { data: { code: 0, data } }
})

const wrapRejected = (message) => ({
  status: 'rejected',
  reason: { message: message || '请求失败' }
})

const buildOverviewSettledCalls = (payload) => {
  const snapshots = payload?.snapshots && typeof payload.snapshots === 'object' ? payload.snapshots : {}
  const sourceErrors = payload?.source_errors && typeof payload.source_errors === 'object' ? payload.source_errors : {}
  const pick = (key) => (Object.prototype.hasOwnProperty.call(sourceErrors, key)
    ? wrapRejected(sourceErrors[key])
    : wrapFulfilled(snapshots[key] ?? null))

  return [
    pick('hosts'),
    pick('docker_hosts'),
    pick('k8s_clusters'),
    pick('alerts'),
    pick('tasks'),
    pick('agents'),
    pick('metrics'),
    pick('metric_history'),
    pick('network_devices'),
    pick('firewalls'),
    pick('jump_sessions'),
    pick('jump_risk_events'),
    pick('domains'),
    pick('certs'),
    pick('cicd_executions'),
    pick('cicd_schedules'),
    pick('workorders'),
    pick('workflow_executions'),
    pick('terminal_sessions'),
    pick('jump_assets'),
    pick('jump_integration')
  ]
}

const resetOverviewSnapshotState = () => {
  overviewContractVersion.value = ''
  overviewSummarySnapshot.value = null
  overviewQualitySnapshot.value = null
  overviewStatusContractSnapshot.value = null
  overviewSourceErrorsSnapshot.value = {}
}

const syncOverviewSnapshotState = (payload) => {
  if (!payload || typeof payload !== 'object') {
    resetOverviewSnapshotState()
    return
  }
  overviewContractVersion.value = String(payload.contract_version || '')
  overviewSummarySnapshot.value = payload.summary && typeof payload.summary === 'object' ? payload.summary : null
  overviewQualitySnapshot.value = payload.quality && typeof payload.quality === 'object' ? payload.quality : null
  overviewStatusContractSnapshot.value = payload.status_contract && typeof payload.status_contract === 'object'
    ? payload.status_contract
    : null
  overviewSourceErrorsSnapshot.value = payload.source_errors && typeof payload.source_errors === 'object'
    ? payload.source_errors
    : {}
}

const loadLegacyDashboardCalls = () => Promise.allSettled([
  axios.get('/api/v1/cmdb/hosts', { headers: authHeaders(), params: { live: 1 } }),
  axios.get('/api/v1/docker/hosts', { headers: authHeaders(), params: { sync: 1 } }),
  axios.get('/api/v1/k8s/clusters', { headers: authHeaders(), params: { live: 1 } }),
  axios.get('/api/v1/monitor/alerts', { headers: authHeaders() }),
  axios.get('/api/v1/task/tasks', { headers: authHeaders() }),
  axios.get('/api/v1/monitor/agents', { headers: authHeaders() }),
  axios.get('/api/v1/monitor/metrics', { headers: authHeaders() }),
  axios.get('/api/v1/monitor/metrics/history', { headers: authHeaders(), params: { hours: Number(scope.timeWindowHours || 24) } }),
  axios.get('/api/v1/cmdb/network-devices', { headers: authHeaders(), params: { live: 1 } }),
  axios.get('/api/v1/firewall/devices', { headers: authHeaders(), params: { live: 1 } }),
  axios.get('/api/v1/jump/sessions', { headers: authHeaders() }),
  axios.get('/api/v1/jump/risk-events', { headers: authHeaders() }),
  axios.get('/api/v1/domain/domains', { headers: authHeaders() }),
  axios.get('/api/v1/domain/certs', { headers: authHeaders() }),
  axios.get('/api/v1/cicd/executions', { headers: authHeaders() }),
  axios.get('/api/v1/cicd/schedules', { headers: authHeaders() }),
  axios.get('/api/v1/workorder/orders', { headers: authHeaders() }),
  axios.get('/api/v1/workflow/executions', { headers: authHeaders() }),
  axios.get('/api/v1/terminal/sessions', { headers: authHeaders() }),
  axios.get('/api/v1/jump/assets', { headers: authHeaders() }),
  axios.get('/api/v1/jump/integration/config', { headers: authHeaders() })
])

const renderTrend = () => {
  const dom = trendRef.value
  if (!(dom instanceof HTMLDivElement)) return
  if (!trendChart) {
    trendChart = echarts.init(dom)
  }

  const records = toArray(trendRecords.value)
  if (!records.length) {
    trendChart.setOption({
      title: {
        text: '暂无历史数据',
        left: 'center',
        top: 'middle',
        textStyle: { color: '#9ca3af', fontSize: 14, fontWeight: 500 }
      },
      xAxis: { show: false, type: 'category', data: [] },
      yAxis: { show: false, type: 'value' },
      series: []
    })
    return
  }

  trendChart.setOption({
    color: ['#3b82f6', '#10b981', '#f59e0b'],
    tooltip: { trigger: 'axis' },
    legend: { top: 0, data: ['CPU', '内存', '磁盘'] },
    grid: { left: 40, right: 20, top: 36, bottom: 24 },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: records.map((item) => formatTime(item.timestamp))
    },
    yAxis: {
      type: 'value',
      axisLabel: { formatter: '{value}%' }
    },
    series: [
      {
        name: 'CPU',
        type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: { opacity: 0.12 },
        data: records.map((item) => toNumber(item.cpu_usage))
      },
      {
        name: '内存',
        type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: { opacity: 0.1 },
        data: records.map((item) => toNumber(item.memory_usage))
      },
      {
        name: '磁盘',
        type: 'line',
        smooth: true,
        showSymbol: false,
        areaStyle: { opacity: 0.08 },
        data: records.map((item) => toNumber(item.disk_usage))
      }
    ]
  })
}

const refreshTopHosts = async () => {
  try {
    const res = await axios.get('/api/v1/monitor/servers', { headers: authHeaders() })
    const list = toArray(res.data?.data)
    topHosts.value = list
      .map((item) => ({
        instance: item.instance || item.ip || item.hostname || '-',
        cpu: toNumber(item.cpu || item.cpu_usage),
        memory: toNumber(item.memory || item.memory_usage)
      }))
      .sort((a, b) => b.cpu - a.cpu)
      .slice(0, 8)
  } catch (e) {
    topHosts.value = []
    ElMessage.error(getErrorMessage(e, '加载主机排行失败'))
  }
}

const refreshRealtimeMetrics = async () => {
  if (realtimeRefreshing.value) return
  realtimeRefreshing.value = true
  try {
    const res = await axios.get('/api/v1/monitor/metrics', { headers: authHeaders() })
    const payload = res.data?.code === 0 ? res.data.data : null
    if (payload && typeof payload === 'object') {
      realtime.cpu = toNumber(payload.cpu)
      realtime.memory = toNumber(payload.memory)
      realtime.disk = toNumber(payload.disk)
      realtime.network = toNumber(payload.network)
      moduleStatus.monitor = 'ok'
    }
  } catch (e) {
    moduleStatus.monitor = 'error'
  } finally {
    realtimeRefreshing.value = false
  }
}

const refreshDashboard = async (options = {}) => {
  if (refreshing.value) return
  refreshing.value = true
  const source = options.source || 'auto'
  try {
    let calls = []
    let overviewPayload = null
    try {
      const overviewRes = await axios.get('/api/v1/dashboard/overview', {
        headers: authHeaders(),
        params: { hours: Number(scope.timeWindowHours || 24) }
      })
      overviewPayload = overviewRes.data?.code === 0 ? overviewRes.data?.data : null
      if (overviewPayload && typeof overviewPayload === 'object') {
        syncOverviewSnapshotState(overviewPayload)
        calls = buildOverviewSettledCalls(overviewPayload)
      } else {
        resetOverviewSnapshotState()
        calls = await loadLegacyDashboardCalls()
      }
    } catch (overviewErr) {
      resetOverviewSnapshotState()
      calls = await loadLegacyDashboardCalls()
    }

    const failures = []

    const hostsPayload = extractData(calls[0])
    const hosts = toArray(hostsPayload)
    cmdbHostsSnapshot.value = hosts
    if (hostsPayload !== null) {
      stats.hostTotal = hosts.length
      stats.hostOnline = hosts.filter((h) => toNumber(h.status, -1) === 1 || normalizeText(h.status) === 'online').length
      stats.hostOffline = hosts.filter((h) => !isOnlineStatus(h.status)).length
      stats.hostStale = hosts.filter((h) => isStatusStale(h, 3)).length
      if (stats.hostOffline > 0) {
        moduleStatus.cmdb = 'error'
      } else if (stats.hostStale > 0) {
        moduleStatus.cmdb = 'warning'
      } else {
        moduleStatus.cmdb = 'ok'
      }
    } else {
      stats.hostOffline = 0
      stats.hostStale = 0
      moduleStatus.cmdb = 'error'
      failures.push(extractFailure(calls[0], 'CMDB') || 'CMDB')
    }

    const dockerPayload = extractData(calls[1])
    const dockerHosts = toArray(dockerPayload)
    dockerHostsSnapshot.value = dockerHosts
    if (dockerPayload !== null) {
      stats.dockerTotal = dockerHosts.length
      stats.dockerOnline = dockerHosts.filter((h) => normalizeText(h.status) === 'online').length
      stats.dockerOffline = dockerHosts.filter((h) => normalizeText(h.status) !== 'online').length
      stats.dockerStale = dockerHosts.filter((h) => isStatusStale(h, 3)).length
      if (stats.dockerOffline > 0) {
        moduleStatus.docker = 'error'
      } else if (stats.dockerStale > 0) {
        moduleStatus.docker = 'warning'
      } else {
        moduleStatus.docker = 'ok'
      }
    } else {
      stats.dockerOffline = 0
      stats.dockerStale = 0
      moduleStatus.docker = 'error'
      failures.push(extractFailure(calls[1], 'Docker') || 'Docker')
    }

    const clustersPayload = extractData(calls[2])
    const allClusters = toArray(clustersPayload)
    const envClusters = allClusters.filter((item) => clusterInEnvironment(item))
    scopeClusters.value = envClusters
    if (scope.clusterId && !envClusters.some((item) => item.id === scope.clusterId)) {
      scope.clusterId = ''
      scope.namespace = ''
      scopeNamespaceLoadedFor.value = ''
    }
    if (!scope.clusterId && envClusters.length === 1) {
      scope.clusterId = envClusters[0].id
    }
    await ensureScopeNamespaces()

    const scopedClusters = allClusters.filter((item) => clusterInScope(item))
    k8sClustersSnapshot.value = scopedClusters
    if (clustersPayload !== null) {
      stats.k8sTotal = scopedClusters.length
      stats.k8sHealthy = scopedClusters.filter((c) => isOnlineStatus(c.status)).length
      stats.k8sMaintenance = scopedClusters.filter((c) => isMaintenanceStatus(c.status)).length
      stats.k8sUnhealthy = scopedClusters.filter((c) => !isOnlineStatus(c.status) && !isMaintenanceStatus(c.status)).length
      stats.k8sStale = scopedClusters.filter((c) => isOnlineStatus(c.status) && elapsedMinutes(clusterFreshnessTs(c)) >= 15).length
      if (stats.k8sUnhealthy > 0) {
        moduleStatus.k8s = 'error'
      } else if (stats.k8sStale > 0 || stats.k8sMaintenance > 0) {
        moduleStatus.k8s = 'warning'
      } else {
        moduleStatus.k8s = 'ok'
      }
    } else {
      stats.k8sMaintenance = 0
      stats.k8sUnhealthy = 0
      stats.k8sStale = 0
      moduleStatus.k8s = 'error'
      failures.push(extractFailure(calls[2], 'K8s') || 'K8s')
    }

    const alertsPayload = extractData(calls[3])
    const alerts = toArray(alertsPayload)
    const scopedAlerts = alerts.filter((item) => inTimeWindow(item.fired_at || item.created_at))
    if (alertsPayload !== null) {
      stats.alertTotal = scopedAlerts.length
      stats.alertOpen = scopedAlerts.filter((a) => toNumber(a.status, -1) === 0).length
      recentAlerts.value = scopedAlerts.slice(0, 8)
    } else {
      failures.push(extractFailure(calls[3], '告警') || '告警')
    }

    const tasksPayload = extractData(calls[4])
    const tasks = toArray(tasksPayload)
    if (tasksPayload !== null) {
      stats.taskTotal = tasks.length
      stats.taskEnabled = tasks.filter((t) => Boolean(t.enabled)).length
      moduleStatus.task = 'ok'
    } else {
      moduleStatus.task = 'error'
      failures.push(extractFailure(calls[4], '任务') || '任务')
    }

    const agentsPayload = extractData(calls[5])
    const agents = toArray(agentsPayload)
    if (agentsPayload !== null) {
      stats.agentTotal = agents.length
      stats.agentOnline = agents.filter((a) => normalizeText(a.status) === 'online').length
    } else {
      failures.push(extractFailure(calls[5], 'Agent') || 'Agent')
    }

    const metricData = extractData(calls[6])
    if (metricData !== null) {
      realtime.cpu = toNumber(metricData.cpu)
      realtime.memory = toNumber(metricData.memory)
      realtime.disk = toNumber(metricData.disk)
      realtime.network = toNumber(metricData.network)
      moduleStatus.monitor = 'ok'
    } else {
      moduleStatus.monitor = 'error'
      failures.push(extractFailure(calls[6], '监控') || '监控')
    }

    const historyPayload = extractData(calls[7])
    const history = toArray(historyPayload).filter((item) => inTimeWindow(item.timestamp))
    if (historyPayload === null) {
      failures.push(extractFailure(calls[7], '趋势') || '趋势')
    }
    trendRecords.value = history
      .map((item) => ({
        timestamp: item.timestamp,
        cpu_usage: toNumber(item.cpu_usage),
        memory_usage: toNumber(item.memory_usage),
        disk_usage: toNumber(item.disk_usage)
      }))
      .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())

    const networkPayload = extractData(calls[8])
    const firewallPayload = extractData(calls[9])
    const jumpSessionsPayload = extractData(calls[10])
    const jumpRiskPayload = extractData(calls[11])
    const domainsPayload = extractData(calls[12])
    const certsPayload = extractData(calls[13])
    const jumpAssetsPayload = extractData(calls[19])
    const jumpIntegrationPayload = extractData(calls[20])

    const networkDevices = toArray(networkPayload)
    const firewalls = toArray(firewallPayload)
    networkDevicesSnapshot.value = networkDevices
    firewallsSnapshot.value = firewalls
    const jumpSessions = toArray(jumpSessionsPayload)
    const jumpRiskEvents = toArray(jumpRiskPayload)
    const domains = toArray(domainsPayload)
    const certs = toArray(certsPayload)
    const jumpAssets = toArray(jumpAssetsPayload)
    jumpAssetsSnapshot.value = jumpAssets
    if (jumpIntegrationPayload && typeof jumpIntegrationPayload === 'object') {
      jumpIntegrationState.enabled = Boolean(jumpIntegrationPayload.enabled)
      jumpIntegrationState.lastSyncStatus = jumpIntegrationPayload.last_sync_status || ''
      jumpIntegrationState.lastSyncMsg = jumpIntegrationPayload.last_sync_msg || ''
      jumpIntegrationState.lastSyncAt = jumpIntegrationPayload.last_sync_at || ''
    } else {
      jumpIntegrationState.enabled = false
      jumpIntegrationState.lastSyncStatus = ''
      jumpIntegrationState.lastSyncMsg = ''
      jumpIntegrationState.lastSyncAt = ''
    }
    const deliveryExecutions = toArray(extractData(calls[14]))
    const deliverySchedules = toArray(extractData(calls[15]))
    const workorders = toArray(extractData(calls[16]))
    const workflowExecutions = toArray(extractData(calls[17]))
    const terminalSessions = toArray(extractData(calls[18]))

    if (firewallPayload !== null) {
      stats.firewallTotal = firewalls.length
      stats.firewallOnline = firewalls.filter((item) => toNumber(item.status, -1) === 1).length
      stats.firewallOffline = firewalls.filter((item) => toNumber(item.status, -1) === 0).length
      stats.firewallAlert = firewalls.filter((item) => toNumber(item.status, -1) === 2).length
      stats.firewallStale = firewalls.filter((item) => isStatusStale(item, 5)).length
      if (stats.firewallAlert > 0) {
        moduleStatus.firewall = 'error'
      } else if (stats.firewallOffline > 0 || stats.firewallStale > 0) {
        moduleStatus.firewall = 'warning'
      } else {
        moduleStatus.firewall = 'ok'
      }
    } else {
      stats.firewallTotal = 0
      stats.firewallOnline = 0
      stats.firewallOffline = 0
      stats.firewallAlert = 0
      stats.firewallStale = 0
      moduleStatus.firewall = 'error'
      failures.push(extractFailure(calls[9], '防火墙') || '防火墙')
    }

    if (domainsPayload !== null) {
      stats.domainTotal = domains.length
      stats.domainHealthy = domains.filter((item) => normalizeText(item.health_status) === 'healthy').length
      stats.domainWarning = domains.filter((item) => normalizeText(item.health_status) === 'warning').length
      stats.domainCritical = domains.filter((item) => normalizeText(item.health_status) === 'critical').length
      stats.domainStale = domains.filter((item) => {
        const checkedAt = item.last_check_at || item.updated_at
        const ts = parseTimestamp(checkedAt)
        return !ts || nowMs() - ts > 24 * 60 * 60 * 1000
      }).length
      if (stats.domainCritical > 0) {
        moduleStatus.domain = 'error'
      } else if (stats.domainWarning > 0 || stats.domainStale > 0) {
        moduleStatus.domain = 'warning'
      } else {
        moduleStatus.domain = 'ok'
      }
    } else {
      stats.domainTotal = 0
      stats.domainHealthy = 0
      stats.domainWarning = 0
      stats.domainCritical = 0
      stats.domainStale = 0
      moduleStatus.domain = 'error'
      failures.push(extractFailure(calls[12], '域名') || '域名')
    }

    const offlineHosts = hosts.filter((item) => !isOnlineStatus(item.status))
    const offlineNetworks = networkDevices.filter((item) => !isOnlineStatus(item.status))
    const staleNetworks = networkDevices.filter((item) => isOnlineStatus(item.status) && isStatusStale(item, 5))
    if (networkPayload !== null && moduleStatus.cmdb !== 'error') {
      if (offlineNetworks.length > 0) {
        moduleStatus.cmdb = 'warning'
      } else if (staleNetworks.length > 0 && moduleStatus.cmdb === 'ok') {
        moduleStatus.cmdb = 'warning'
      }
    }
    const hostOffline = offlineHosts.length
    const networkOffline = offlineNetworks.length
    const firewallAlert = firewalls.filter((item) => Number(item.status) === 2).length
    const jumpPendingTimeout = jumpSessions.filter(
      (item) => normalizeText(item.status) === 'pending_approval' && elapsedMinutes(item.started_at) >= 30
    ).length
    const riskCritical = jumpRiskEvents.filter((item) => normalizeText(item.severity) === 'critical').length
    const jumpSyncFailed = normalizeText(jumpIntegrationPayload?.last_sync_status) === 'failed'
    const jumpSyncMsg = jumpIntegrationPayload?.last_sync_msg || ''
    if (jumpSessionsPayload !== null && jumpRiskPayload !== null) {
      if (riskCritical > 0) {
        moduleStatus.jump = 'error'
      } else if (jumpPendingTimeout > 0 || jumpSyncFailed) {
        moduleStatus.jump = 'warning'
      } else {
        moduleStatus.jump = 'ok'
      }
    } else {
      moduleStatus.jump = 'error'
      failures.push('堡垒机')
    }
    if (jumpAssetsPayload === null) {
      failures.push(extractFailure(calls[19], '堡垒机资产') || '堡垒机资产')
    }
    if (jumpIntegrationPayload === null) {
      failures.push(extractFailure(calls[20], 'Jump集成') || 'Jump集成')
    }

    const overviewModuleStatus = overviewSummarySnapshot.value?.module_status
    if (overviewModuleStatus && typeof overviewModuleStatus === 'object') {
      Object.entries(overviewModuleStatus).forEach(([key, value]) => {
        if (!Object.prototype.hasOwnProperty.call(moduleStatus, key)) return
        const next = normalizeModuleStatus(value)
        if (statusSeverity[next] > statusSeverity[moduleStatus[key] || 'unknown']) {
          moduleStatus[key] = next
        }
      })
    }

    backlog.asset = hostOffline + networkOffline + firewallAlert + jumpPendingTimeout + riskCritical
    backlog.assetOverdue = jumpPendingTimeout

    const monitorAlertTimeout = scopedAlerts.filter(
      (item) => isAlertOpen(item.status) && elapsedMinutes(item.fired_at || item.created_at) >= 60
    ).length
    const monitorCriticalOpen = scopedAlerts.filter(
      (item) => isAlertOpen(item.status) && normalizeText(item.severity) === 'critical'
    ).length
    const domainRiskRows = []
    const staleDomainNames = []
    const staleCertIds = []
    domains.forEach((item) => {
      const health = normalizeText(item.health_status)
      if (health === 'warning' || health === 'critical') {
        const checkedAt = item.last_check_at || item.updated_at
        domainRiskRows.push({ checkedAt })
        const ts = parseTimestamp(checkedAt)
        if (!ts || nowMs() - ts > 24 * 60 * 60 * 1000) {
          if (item.domain) staleDomainNames.push(item.domain)
        }
      }
    })
    certs.forEach((item) => {
      if (toNumber(item.days_to_expire, 0) <= 30) {
        const checkedAt = item.last_check_at || item.updated_at
        domainRiskRows.push({ checkedAt })
        const ts = parseTimestamp(checkedAt)
        if (!ts || nowMs() - ts > 24 * 60 * 60 * 1000) {
          if (item.id) staleCertIds.push(item.id)
        }
      }
    })
    const monitorRiskStale = domainRiskRows.filter((item) => {
      const ts = parseTimestamp(item.checkedAt)
      return !ts || nowMs() - ts > 24 * 60 * 60 * 1000
    }).length
    backlog.monitor = monitorAlertTimeout + monitorCriticalOpen + monitorRiskStale
    backlog.monitorOverdue = monitorAlertTimeout + monitorRiskStale

    const degradedClusters = scopedClusters.filter(
      (item) => !isOnlineStatus(item.status) && !isMaintenanceStatus(item.status)
    )
    const staleClusters = scopedClusters.filter(
      (item) => isOnlineStatus(item.status) && elapsedMinutes(clusterFreshnessTs(item)) >= 15
    )
    const k8sClusterDegraded = degradedClusters.length
    const k8sClusterStale = staleClusters.length
    backlog.k8s = k8sClusterDegraded + k8sClusterStale
    backlog.k8sOverdue = k8sClusterStale

    const deliveryOrderTimeout = workorders.filter(
      (item) => isWorkorderApprovalPending(item.status) && elapsedMinutes(item.created_at) >= 120
    ).length
    const deliveryLongExecutions = deliveryExecutions.filter((item) => Number(item.status) === 0 && elapsedMinutes(item.started_at) >= 30)
    const deliveryExecutionLong = deliveryLongExecutions.length
    const deliveryScheduleStale = deliverySchedules.filter((item) => {
      if (!isScheduleEnabled(item)) return false
      const nextTs = parseTimestamp(item.next_run_at)
      if (!nextTs) return true
      return nextTs < nowMs() - 5 * 60 * 1000
    }).length
    backlog.delivery = deliveryOrderTimeout + deliveryExecutionLong + deliveryScheduleStale

    const collabOrderTimeout = workorders.filter((item) => {
      const status = Number(item.status)
      return (status === 0 || status === 1 || status === 4) && elapsedMinutes(item.created_at) >= 120
    }).length
    const longWorkflowExecutions = workflowExecutions.filter(
      (item) => Number(item.status) === 0 && elapsedMinutes(item.started_at) >= 15
    )
    const collabWorkflowLong = longWorkflowExecutions.length
    const collabTerminalPending = terminalSessions.filter((item) => {
      if (Number(item.status) !== 0) return false
      return elapsedMinutes(item.started_at || item.created_at) >= 10
    }).length
    const failedTerminalSessions = terminalSessions.filter((item) => Number(item.status) === 3)
    const collabTerminalFailed = failedTerminalSessions.length
    backlog.delivery += collabOrderTimeout + collabWorkflowLong + collabTerminalPending + collabTerminalFailed
    backlog.deliveryOverdue = deliveryOrderTimeout + deliveryExecutionLong + deliveryScheduleStale + collabOrderTimeout + collabWorkflowLong + collabTerminalPending
    backlog.total = backlog.asset + backlog.monitor + backlog.k8s + backlog.delivery
    backlog.overdue =
      backlog.assetOverdue +
      backlog.monitorOverdue +
      backlog.k8sOverdue +
      backlog.deliveryOverdue

    backlogSource.offlineHostIds = offlineHosts.map((item) => item.id).filter(Boolean)
    backlogSource.offlineNetworkDeviceIds = offlineNetworks.map((item) => item.id).filter(Boolean)
    backlogSource.staleDomainNames = staleDomainNames
    backlogSource.staleCertIds = staleCertIds
    backlogSource.degradedClusterIds = degradedClusters.map((item) => item.id).filter(Boolean)
    backlogSource.staleClusterIds = staleClusters.map((item) => item.id).filter(Boolean)
    backlogSource.longRunningExecutionIds = deliveryLongExecutions
      .filter((item) => elapsedMinutes(item.started_at) >= 120)
      .map((item) => item.id)
      .filter(Boolean)
    backlogSource.longRunningWorkflowIds = longWorkflowExecutions
      .filter((item) => elapsedMinutes(item.started_at) >= 120)
      .map((item) => item.id)
      .filter(Boolean)
    backlogSource.failedTerminalSessionIds = failedTerminalSessions.map((item) => item.id).filter(Boolean)

    const callErrorMessage = (idx, label) => {
      const item = calls[idx]
      if (!item || item.status !== 'rejected') return ''
      return getErrorMessage(item.reason, `${label}请求失败`)
    }
    const diagnosticRows = [
      {
        label: 'CMDB主机状态链路',
        path: '/host',
        status: hostsPayload === null ? 'error' : (hostOffline > 0 || stats.hostStale > 0 ? 'warning' : 'ok'),
        countText: `${hosts.length}`,
        message: hostsPayload === null ? callErrorMessage(0, 'CMDB') : (hostOffline > 0 ? `离线 ${hostOffline} 台` : (stats.hostStale > 0 ? `状态过期 ${stats.hostStale} 台` : '主机状态实时'))
      },
      {
        label: 'Docker状态链路',
        path: '/docker',
        status: dockerPayload === null ? 'error' : (stats.dockerOffline > 0 || stats.dockerStale > 0 ? 'warning' : 'ok'),
        countText: `${dockerHosts.length}`,
        message: dockerPayload === null ? callErrorMessage(1, 'Docker') : (stats.dockerOffline > 0 ? `离线 ${stats.dockerOffline} 台` : (stats.dockerStale > 0 ? `状态过期 ${stats.dockerStale} 台` : 'Docker采集正常'))
      },
      {
        label: 'K8s集群状态链路',
        path: '/k8s/clusters',
        status: clustersPayload === null ? 'error' : (stats.k8sUnhealthy > 0 || stats.k8sStale > 0 ? 'warning' : 'ok'),
        countText: `${scopedClusters.length}`,
        message: clustersPayload === null ? callErrorMessage(2, 'K8s') : (stats.k8sUnhealthy > 0 ? `异常 ${stats.k8sUnhealthy} 个` : (stats.k8sStale > 0 ? `状态过期 ${stats.k8sStale} 个` : '集群状态实时'))
      },
      {
        label: '告警事件链路',
        path: '/alert/events',
        status: alertsPayload === null ? 'error' : (monitorAlertTimeout > 0 ? 'warning' : 'ok'),
        countText: `${scopedAlerts.length}`,
        message: alertsPayload === null ? callErrorMessage(3, '告警') : (monitorAlertTimeout > 0 ? `超时未恢复 ${monitorAlertTimeout} 条` : '告警事件可用')
      },
      {
        label: '监控指标链路',
        path: '/monitor/overview',
        status: metricData === null || historyPayload === null ? 'error' : 'ok',
        countText: `${trendRecords.value.length}`,
        message: metricData === null ? callErrorMessage(6, '监控指标') : (historyPayload === null ? callErrorMessage(7, '趋势') : '实时+历史指标可用')
      },
      {
        label: '网络设备链路',
        path: '/cmdb/network-devices',
        status: networkPayload === null ? 'error' : (networkOffline > 0 || staleNetworks.length > 0 ? 'warning' : 'ok'),
        countText: `${networkDevices.length}`,
        message: networkPayload === null ? callErrorMessage(8, '网络设备') : (networkOffline > 0 ? `离线 ${networkOffline} 台` : (staleNetworks.length > 0 ? `状态过期 ${staleNetworks.length} 台` : '网络设备状态正常'))
      },
      {
        label: '防火墙链路',
        path: '/firewall',
        status: firewallPayload === null ? 'error' : (stats.firewallAlert > 0 || stats.firewallOffline > 0 || stats.firewallStale > 0 ? 'warning' : 'ok'),
        countText: `${firewalls.length}`,
        message: firewallPayload === null ? callErrorMessage(9, '防火墙') : (stats.firewallAlert > 0 ? `告警 ${stats.firewallAlert} 台` : (stats.firewallOffline > 0 ? `离线 ${stats.firewallOffline} 台` : (stats.firewallStale > 0 ? `状态过期 ${stats.firewallStale} 台` : 'SNMP采集正常')))
      },
      {
        label: 'JumpServer会话链路',
        path: '/jump/sessions',
        status: jumpSessionsPayload === null || jumpRiskPayload === null ? 'error' : (jumpPendingTimeout > 0 || riskCritical > 0 || jumpSyncFailed ? 'warning' : 'ok'),
        countText: `${jumpSessions.length}`,
        message: jumpSessionsPayload === null || jumpRiskPayload === null
          ? (callErrorMessage(10, 'Jump会话') || callErrorMessage(11, 'Jump风控') || '堡垒机链路异常')
          : (riskCritical > 0 ? `高危风控 ${riskCritical} 条` : (jumpPendingTimeout > 0 ? `超时待审批 ${jumpPendingTimeout} 条` : (jumpSyncFailed ? truncateText(jumpSyncMsg || '最近同步失败', 80) : '会话同步正常')))
      },
      {
        label: '堡垒机资产映射链路',
        path: '/jump/assets',
        status: jumpAssetsPayload === null ? 'error' : (consistencyIssueRows.value.some((item) => item.module.includes('堡垒机')) ? 'warning' : 'ok'),
        countText: `${jumpAssets.length}`,
        message: jumpAssetsPayload === null
          ? (callErrorMessage(19, 'Jump资产') || '堡垒机资产读取失败')
          : (consistencyIssueRows.value.some((item) => item.module.includes('堡垒机')) ? '存在来源映射不一致' : '堡垒机资产映射正常')
      },
      {
        label: '域名与证书链路',
        path: '/domain/ssl',
        status: domainsPayload === null || certsPayload === null ? 'error' : (stats.domainCritical > 0 || stats.domainWarning > 0 || stats.domainStale > 0 ? 'warning' : 'ok'),
        countText: `${domains.length}/${certs.length}`,
        message: domainsPayload === null || certsPayload === null
          ? (callErrorMessage(12, '域名') || callErrorMessage(13, '证书') || '域名证书链路异常')
          : (stats.domainCritical > 0 ? `故障域名 ${stats.domainCritical} 个` : (stats.domainWarning > 0 ? `预警域名 ${stats.domainWarning} 个` : (stats.domainStale > 0 ? `检查过期 ${stats.domainStale} 个` : '域名证书状态正常')))
      },
      {
        label: '任务与工单链路',
        path: '/delivery/center',
        status: (extractData(calls[4]) === null || extractData(calls[15]) === null || extractData(calls[16]) === null) ? 'error' : (backlog.deliveryOverdue > 0 ? 'warning' : 'ok'),
        countText: `${tasks.length}/${deliverySchedules.length}/${workorders.length}`,
        message: (extractData(calls[4]) === null || extractData(calls[15]) === null || extractData(calls[16]) === null)
          ? (callErrorMessage(4, '任务') || callErrorMessage(15, '定时发布') || callErrorMessage(16, '工单') || '任务工单链路异常')
          : (backlog.deliveryOverdue > 0 ? `超时待处理 ${backlog.deliveryOverdue} 项` : '任务调度与工单正常')
      },
      {
        label: 'Agent在线链路',
        path: '/monitor/agents',
        status: agentsPayload === null ? 'error' : ((stats.agentTotal > 0 && stats.agentOnline === 0) ? 'warning' : 'ok'),
        countText: `${stats.agentOnline}/${stats.agentTotal}`,
        message: agentsPayload === null ? callErrorMessage(5, 'Agent') : ((stats.agentTotal > 0 && stats.agentOnline === 0) ? '所有 Agent 离线' : 'Agent状态正常')
      },
      {
        label: '状态字段完整性',
        path: '/dashboard',
        status: fieldCompletenessSummary.value.problem > 0 ? 'warning' : 'ok',
        countText: `${fieldCompletenessSummary.value.total}`,
        message: fieldCompletenessSummary.value.problem > 0
          ? `存在 ${fieldCompletenessSummary.value.problem} 个模块字段缺口`
          : '关键状态字段完整'
      },
      {
        label: '跨模块一致性',
        path: '/dashboard',
        status: consistencySummary.value.total > 0 ? 'warning' : 'ok',
        countText: `${consistencySummary.value.total}`,
        message: consistencySummary.value.total > 0
          ? `检测到 ${consistencySummary.value.total} 条跨模块冲突`
          : '跨模块状态一致'
      }
    ]
    dataSourceDiagnostics.value = diagnosticRows

    await refreshDeploymentRisk(scopedClusters, failures)

    lastUpdated.value = new Date().toLocaleString()
    renderTrend()
    await refreshTopHosts()
    throttledPartialFailureMessage(failures)
    pushStatusTimelineSnapshot(source)
    pushSelfCheckHistory(source)
  } catch (err) {
    dataSourceDiagnostics.value = [
      { label: '全局仪表盘刷新', status: 'error', countText: '-', message: getErrorMessage(err, '刷新失败'), path: '/dashboard' }
    ]
    pushStatusTimelineSnapshot(source)
    pushSelfCheckHistory(source)
    ElMessage.error(getErrorMessage(err, '仪表盘刷新失败'))
  } finally {
    refreshing.value = false
  }
}

const runSelfCheck = async () => {
  if (deepChecking.value) return
  deepChecking.value = true
  try {
    await refreshDashboard({ source: 'manual' })
    const probeSummary = await runDeepStatusProbe()
    const probeText = probeSummary.total > 0 ? `${probeSummary.ok}/${probeSummary.total}` : '无目标'
    if (probeSummary.total > 0) {
      await refreshDashboard({ source: 'probe' })
    }
    if (dataSourceSummary.value.error > 0) {
      ElMessage.warning(`深度自检完成：失败 ${dataSourceSummary.value.error} 项，主动探测 ${probeText}`)
      return
    }
    if (dataSourceSummary.value.warning > 0) {
      ElMessage.warning(`深度自检完成：降级 ${dataSourceSummary.value.warning} 项，主动探测 ${probeText}`)
      return
    }
    ElMessage.success(`深度自检完成：核心链路正常，主动探测 ${probeText}`)
  } finally {
    deepChecking.value = false
  }
}

const onResize = () => {
  if (trendChart) trendChart.resize()
}

const startAutoRefresh = () => {
  if (!realtimeTimer) {
    realtimeTimer = setInterval(() => {
      if (document.hidden) return
      refreshRealtimeMetrics()
    }, 10000)
  }
  if (!overviewTimer) {
    overviewTimer = setInterval(() => {
      if (document.hidden) return
      refreshDashboard()
    }, 60000)
  }
}

const stopAutoRefresh = () => {
  if (realtimeTimer) {
    clearInterval(realtimeTimer)
    realtimeTimer = null
  }
  if (overviewTimer) {
    clearInterval(overviewTimer)
    overviewTimer = null
  }
}

onMounted(async () => {
  loading.value = true
  await refreshDashboard()
  loading.value = false
  startAutoRefresh()
  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  stopAutoRefresh()
  window.removeEventListener('resize', onResize)
  if (trendChart) {
    trendChart.dispose()
    trendChart = null
  }
})
</script>

<style scoped>
.dashboard-card {
  max-width: 1280px;
  margin: 0 auto;
  border-radius: 18px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 14px;
}

.page-header h2 {
  margin: 0;
  font-size: 26px;
  font-weight: 600;
  letter-spacing: -0.2px;
}

.page-desc {
  margin: 6px 0 0;
  color: #6b7280;
}

.page-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.updated-at {
  color: #9ca3af;
  font-size: 12px;
}

.scope-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
}

.scope-label {
  color: #4b5563;
  font-size: 12px;
  font-weight: 600;
}

.scope-item {
  width: 180px;
}

.scope-item.narrow {
  width: 150px;
}

.kpi-card {
  border-radius: 14px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
  border: 1px solid #edf2f7;
}

.kpi-title {
  color: #6b7280;
  font-size: 12px;
}

.kpi-value {
  margin-top: 8px;
  font-size: 30px;
  font-weight: 600;
  line-height: 1;
}

.kpi-value.danger {
  color: #ef4444;
}

.kpi-sub {
  margin-top: 8px;
  font-size: 12px;
  color: #9ca3af;
}

.panel-row {
  margin-top: 16px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.panel-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  letter-spacing: -0.2px;
}

.panel-desc {
  margin: 4px 0 0;
  color: #9ca3af;
  font-size: 12px;
}

.chart-box {
  width: 100%;
  height: 320px;
}

.stack-card {
  margin-bottom: 12px;
}

.backlog-overview-card {
  border-radius: 14px;
  border: 1px solid #e5e7eb;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
}

.backlog-header {
  margin-bottom: 14px;
}

.backlog-totals {
  display: flex;
  align-items: center;
  gap: 8px;
}

.backlog-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.backlog-item {
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 12px;
  background: #fff;
}

.backlog-item-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.backlog-label {
  font-size: 13px;
  color: #4b5563;
}

.backlog-value {
  margin-top: 10px;
  font-size: 26px;
  font-weight: 600;
  line-height: 1;
  color: #111827;
}

.backlog-desc {
  margin-top: 6px;
  font-size: 12px;
  color: #9ca3af;
}

.backlog-actions {
  margin-top: 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.selfcheck-card {
  border-radius: 14px;
  border: 1px solid #e5e7eb;
}

.selfcheck-header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.selfcheck-meta {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.selfcheck-contract {
  margin-bottom: 10px;
  font-size: 12px;
  color: #6b7280;
}

.mini-percent {
  margin-top: 2px;
  font-size: 11px;
  color: #6b7280;
}

.resource-row {
  display: grid;
  grid-template-columns: 48px 1fr 58px;
  gap: 8px;
  align-items: center;
  margin-bottom: 10px;
  color: #4b5563;
}

.resource-row b {
  text-align: right;
  color: #111827;
}

.network-value {
  grid-column: 2 / 4;
  color: #111827;
  font-weight: 600;
}

.module-row {
  display: grid;
  grid-template-columns: 1fr auto auto;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.risk-card {
  border: 1px solid #e5e7eb;
}

.capability-card,
.capability-gap-card {
  border: 1px solid #e5e7eb;
}

.risk-kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-top: 4px;
}

.risk-kpi-item {
  padding: 8px 10px;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  background: #fff;
}

.risk-kpi-label {
  font-size: 12px;
  color: #6b7280;
}

.risk-kpi-value {
  margin-top: 4px;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.risk-kpi-value.success { color: #16a34a; }
.risk-kpi-value.warning { color: #d97706; }
.risk-kpi-value.danger { color: #dc2626; }

.risk-extra {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: #6b7280;
}

.risk-list {
  margin-top: 10px;
  border-top: 1px dashed #e5e7eb;
  padding-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.risk-list-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.risk-name {
  font-size: 12px;
  color: #374151;
}

.flap-kpi-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 10px;
}

.flap-kpi-item {
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  padding: 8px 10px;
  background: #fff;
}

.flap-kpi-label {
  font-size: 12px;
  color: #6b7280;
}

.flap-kpi-value {
  margin-top: 4px;
  font-size: 16px;
  font-weight: 600;
  color: #111827;
}

.flap-meta {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  font-size: 12px;
  color: #6b7280;
}

.integrity-card {
  border: 1px solid #e5e7eb;
  border-radius: 14px;
}

.integrity-summary-tags {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.trust-score-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
}

.trust-dimension-list {
  margin-top: 6px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.trust-dimension-item {
  border: 1px solid #eef2f7;
  border-radius: 10px;
  padding: 10px;
  background: #fff;
}

.trust-dimension-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
  font-size: 12px;
  color: #4b5563;
}

.trust-dimension-top strong {
  font-size: 13px;
  color: #111827;
}

.trust-dimension-desc {
  margin-top: 6px;
  font-size: 12px;
  color: #9ca3af;
}

.trust-summary {
  margin-top: 10px;
  font-size: 12px;
  color: #4b5563;
}

.motion-up {
  animation: motion-up 0.38s ease both;
}

.delay-1 { animation-delay: 0.03s; }
.delay-2 { animation-delay: 0.06s; }
.delay-3 { animation-delay: 0.09s; }
.delay-4 { animation-delay: 0.12s; }
.delay-5 { animation-delay: 0.15s; }
.delay-6 { animation-delay: 0.18s; }

@keyframes motion-up {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 1400px) {
  .backlog-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 980px) {
  .backlog-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .scope-item {
    width: 100%;
  }

  .risk-kpi-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .trust-dimension-list {
    grid-template-columns: 1fr;
  }

  .panel-header {
    align-items: flex-start;
    gap: 8px;
  }
}

@media (max-width: 760px) {
  .backlog-grid {
    grid-template-columns: 1fr;
  }

  .flap-kpi-grid {
    grid-template-columns: 1fr;
  }
}
</style>
