<template>
  <el-card class="host-page-card">
    <template #header>
      <div class="flex justify-between items-center header-wrap">
        <span class="font-bold">CMDB 主机管理</span>
        <div class="header-actions">
          <el-button type="primary" icon="Plus" @click="handleAdd">添加主机</el-button>
          <el-button type="primary" plain icon="Connection" :disabled="selectedRows.length !== 1" @click="openConnectionEditor">
            连接信息
          </el-button>
          <el-button plain icon="FolderOpened" @click="openGroupManager">分组维护</el-button>
          <el-button icon="Upload" @click="openImport">批量导入</el-button>
          <el-button icon="Download" @click="exportCSV">导出</el-button>
          <el-button type="warning" plain icon="Edit" :disabled="selectedRows.length === 0" @click="openBatchStatus">
            批量状态
          </el-button>
          <el-button type="primary" plain icon="Folder" :disabled="selectedRows.length === 0" @click="openBatchGroup">
            批量分组 ({{ selectedRows.length }})
          </el-button>
          <el-button type="success" plain icon="VideoPlay" :disabled="selectedRows.length === 0" @click="openBatchExec">
            批量执行 ({{ selectedRows.length }})
          </el-button>
          <el-button type="danger" plain icon="Delete" :disabled="selectedRows.length === 0" @click="handleBatchDelete">
            批量删除 ({{ selectedRows.length }})
          </el-button>
          <el-button type="success" plain icon="Promotion" :loading="syncingStatus" @click="syncStatuses()">巡检状态</el-button>
          <el-button icon="Refresh" @click="fetchData">刷新列表</el-button>
        </div>
      </div>
    </template>

    <div class="host-layout">
      <div class="host-aside">
        <el-card shadow="never" class="group-card">
          <template #header>
            <div class="group-card-header">
              <span>资产分组</span>
              <div class="group-card-header-actions">
                <el-button link type="primary" @click="openGroupManager">维护</el-button>
                <el-button link type="primary" @click="clearGroupFilter">重置</el-button>
              </div>
            </div>
          </template>
          <el-input
            v-model="groupKeyword"
            placeholder="筛选分组"
            clearable
            class="group-search"
            @input="handleGroupKeywordChange"
          />
          <el-tree
            ref="groupTreeRef"
            :data="groupTreeData"
            :props="{ children: 'children', label: 'label' }"
            node-key="id"
            default-expand-all
            highlight-current
            :current-node-key="activeGroupId || 'all'"
            :filter-node-method="groupNodeFilter"
            :expand-on-click-node="false"
            @node-click="onGroupNodeClick"
          >
            <template #default="{ data }">
              <div class="group-tree-node">
                <span class="group-tree-label">{{ data.label }}</span>
                <el-tag size="small" effect="plain" type="info">{{ data.count || 0 }}</el-tag>
              </div>
            </template>
          </el-tree>
        </el-card>

        <el-card shadow="never" class="provider-card">
          <template #header>
            <div class="group-card-header">
              <span>云厂商分布</span>
              <span class="provider-total">{{ filteredTableData.length }}</span>
            </div>
          </template>
          <div class="provider-grid">
            <div v-for="item in providerSummaryList" :key="item.key" class="provider-item">
              <span class="provider-name">{{ item.label }}</span>
              <el-tag :type="providerTagType(item.key)" effect="plain">{{ item.count }}</el-tag>
            </div>
          </div>
        </el-card>
      </div>

      <div class="host-main">
        <div class="mb-4">
          <div class="flex gap-2 items-center filters-row">
            <el-input
              v-model="searchKeyword"
              placeholder="搜索主机名或IP"
              class="w-64"
              clearable
              @clear="fetchData"
              @keyup.enter="fetchData"
            >
              <template #append>
                <el-button icon="Search" @click="fetchData" />
              </template>
            </el-input>
            <el-select v-model="activeGroupId" placeholder="分组" clearable class="w-64" @change="onGroupSelectChange">
              <el-option label="全部主机" value="" />
              <el-option label="未分组" :value="UNGROUPED_GROUP_ID" />
              <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
            </el-select>
            <div class="status-filter-group flex items-center gap-2">
              <el-tag
                :type="statusFilter === 'all' ? 'primary' : 'info'"
                :effect="statusFilter === 'all' ? 'dark' : 'plain'"
                style="cursor: pointer; user-select: none; transition: all 0.2s;"
                @click="toggleStatusFilter('all')"
              >
                总计 {{ totalCount }}
              </el-tag>
              <el-tag
                type="success"
                :effect="statusFilter === 'online' ? 'dark' : 'plain'"
                style="cursor: pointer; user-select: none; transition: all 0.2s;"
                @click="toggleStatusFilter('online')"
              >
                在线 {{ onlineCount }}
              </el-tag>
              <el-tag
                type="warning"
                :effect="statusFilter === 'offline' ? 'dark' : 'plain'"
                style="cursor: pointer; user-select: none; transition: all 0.2s;"
                @click="toggleStatusFilter('offline')"
              >
                离线 {{ offlineCount }}
              </el-tag>
            </div>
            
            <el-popover placement="bottom" title="列显示设置" :width="200" trigger="click">
              <template #reference>
                <el-button icon="Setting" style="margin-left: auto" circle />
              </template>
              <div class="column-setting-list" style="display: flex; flex-direction: column; gap: 8px;">
                <el-checkbox v-model="showColumns.ip">IP地址</el-checkbox>
                <el-checkbox v-model="showColumns.provider">云厂商</el-checkbox>
                <el-checkbox v-model="showColumns.os">操作系统</el-checkbox>
                <el-checkbox v-model="showColumns.cpu">CPU使用率</el-checkbox>
                <el-checkbox v-model="showColumns.memory">内存使用率</el-checkbox>
                <el-checkbox v-model="showColumns.disk">磁盘使用率</el-checkbox>
                <el-checkbox v-model="showColumns.status">监控状态</el-checkbox>
                <el-checkbox v-model="showColumns.desc">备注</el-checkbox>
              </div>
            </el-popover>
          </div>
        </div>

        <div class="table-scroll">
          <el-table
            class="host-table"
            :fit="true"
            :data="filteredTableData"
            v-loading="loading"
            style="width: 100%"
            @selection-change="selectedRows = $event"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column prop="name" label="主机名" min-width="170" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="flex items-center gap-2">
                  <el-icon class="text-gray-500 text-lg"><Monitor /></el-icon>
                  <span class="font-bold">{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="ip" label="IP地址" min-width="140" v-if="showColumns.ip" />
            <el-table-column label="云厂商" min-width="110" align="center" v-if="showColumns.provider">
              <template #default="{ row }">
                <el-tag size="small" :type="providerTagType(hostProvider(row))" effect="plain">
                  {{ providerLabel(hostProvider(row)) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="os" label="操作系统" min-width="130" show-overflow-tooltip v-if="showColumns.os" />
            <el-table-column label="CPU" min-width="145" align="center" v-if="showColumns.cpu">
              <template #default="{ row }">
                <div class="flex items-center justify-center gap-1.5 whitespace-nowrap" style="white-space: nowrap;">
                  <span class="font-medium" style="font-size: 13px; white-space: nowrap;">{{ hardwareUsageText(row, 'cpu', '核') }}</span>
                  <el-tag v-if="hasMetricValue(row, 'cpu') && hardwareUsageText(row, 'cpu', '核').includes('/')" size="small" effect="light" :type="metricTagType(metricValue(row, 'cpu'))" style="font-size: 10px; padding: 0 4px; height: 18px; line-height: 18px;">
                    {{ metricText(row, 'cpu') }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="内存" min-width="145" align="center" v-if="showColumns.memory">
              <template #default="{ row }">
                <div class="flex items-center justify-center gap-1.5 whitespace-nowrap" style="white-space: nowrap;">
                  <span class="font-medium" style="font-size: 13px; white-space: nowrap;">{{ hardwareUsageText(row, 'memory', 'G') }}</span>
                  <el-tag v-if="hasMetricValue(row, 'memory') && hardwareUsageText(row, 'memory', 'G').includes('/')" size="small" effect="light" :type="metricTagType(metricValue(row, 'memory'))" style="font-size: 10px; padding: 0 4px; height: 18px; line-height: 18px;">
                    {{ metricText(row, 'memory') }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="磁盘" min-width="145" align="center" v-if="showColumns.disk">
              <template #default="{ row }">
                <div class="flex items-center justify-center gap-1.5 whitespace-nowrap" style="white-space: nowrap;">
                  <span class="font-medium" style="font-size: 13px; white-space: nowrap;">{{ hardwareUsageText(row, 'disk', 'G') }}</span>
                  <el-tag v-if="hasMetricValue(row, 'disk') && hardwareUsageText(row, 'disk', 'G').includes('/')" size="small" effect="light" :type="metricTagType(metricValue(row, 'disk'))" style="font-size: 10px; padding: 0 4px; height: 18px; line-height: 18px;">
                    {{ metricText(row, 'disk') }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" min-width="120" v-if="showColumns.status">
              <template #default="{ row }">
                <StatusBadge
                  :text="hostStatusMeta(row).text"
                  :type="hostStatusMeta(row).type"
                  :source="hostStatusMeta(row).source"
                  :check-at="hostStatusMeta(row).checkAt"
                  :is-stale="hostStatusMeta(row).isStale"
                  :stale-text="hostStatusMeta(row).staleText"
                  :reason="hostStatusMeta(row).reason"
                />
              </template>
            </el-table-column>
            <el-table-column prop="last_check_at" label="最后检测" width="160" v-if="showColumns.status">
              <template #default="{ row }">
                {{ formatTime(hostStatusMeta(row).checkAt || row.last_check_at) }}
              </template>
            </el-table-column>
            <el-table-column prop="status_reason" label="状态说明" min-width="200" show-overflow-tooltip v-if="showColumns.status">
              <template #default="{ row }">
                {{ hostStatusMeta(row).reason || '-' }}
              </template>
            </el-table-column>
            <el-table-column prop="group.name" label="分组" min-width="120" show-overflow-tooltip>
              <template #default="{ row }">{{ row?.group?.name || '-' }}</template>
            </el-table-column>
            <el-table-column prop="description" label="备注 (双击编辑)" min-width="180" show-overflow-tooltip v-if="showColumns.desc">
              <template #default="{ row }">
                <div @dblclick.stop="startEditDescription(row)" style="min-height: 24px; display: flex; align-items: center;">
                  <el-input
                    v-if="editingHostId === row.id"
                    v-model="editingDescription"
                    size="small"
                    :ref="(el) => { if(el) descInputRefs[row.id] = el }"
                    @blur="saveDescription(row)"
                    @keyup.enter="saveDescription(row)"
                  />
                  <span v-else style="cursor: pointer; display: inline-block; width: 100%;">
                    {{ row.description || '—' }}
                  </span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="410" min-width="410" fixed="right" class-name="fixed-op-col">
              <template #default="{ row }">
                <div class="op-row">
                  <el-button size="small" type="primary" plain icon="Monitor" @click="openWebTerminal(row)">终端</el-button>
                  <el-button size="small" plain icon="View" @click="openDetail(row)">详情</el-button>
                  <el-button size="small" type="warning" plain icon="FirstAidKit" @click="handleTest(row)">检测</el-button>
                  <el-button size="small" plain icon="Edit" @click="handleEdit(row)">编辑</el-button>
                  <el-dropdown trigger="click" @command="(command) => handleRowCommand(row, command)">
                    <el-button size="small" plain>
                      更多
                      <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                    </el-button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="terminal">Web 终端</el-dropdown-item>
                        <el-dropdown-item command="diagnose">网络诊断</el-dropdown-item>
                        <el-dropdown-item command="process">进程</el-dropdown-item>
                        <el-dropdown-item command="tcp">TCP</el-dropdown-item>
                        <el-dropdown-item command="monitor">监控</el-dropdown-item>
                        <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </template>
            </el-table-column>
          </el-table>

          <div class="pagination-wrapper flex items-center justify-between" style="display: flex; align-items: center; justify-content: space-between; width: 100%;">
            <div class="flex items-center gap-2">
              <el-button
                size="small"
                :type="pageSize >= 1000 ? 'primary' : 'default'"
                style="border-radius: 12px; font-size: 12px;"
                @click="selectAllPage"
              >
                显示全部 (All)
              </el-button>
            </div>
            <el-pagination
              v-model:current-page="currentPage"
              v-model:page-size="pageSize"
              :page-sizes="[10, 20, 50, 100, 500, 1000]"
              layout="total, sizes, prev, pager, next, jumper"
              :total="totalHosts"
              @size-change="fetchData"
              @current-change="fetchData"
            />
          </div>
        </div>
      </div>
    </div>

    <el-drawer
      v-model="detailVisible"
      :title="`主机详情 - ${detailHost?.name || '-'}`"
      size="75%"
      append-to-body
      :destroy-on-close="false"
      class="host-detail-drawer"
    >
      <div class="detail-toolbar">
        <div class="detail-host-info">
          <el-tag :type="hostStatusMeta(detailHost || {}).type" effect="dark">
            {{ hostStatusMeta(detailHost || {}).text || '未知' }}
          </el-tag>
          <span>IP: {{ detailHost?.ip || '-' }}</span>
          <span>系统: {{ detailHost?.os || '-' }}</span>
          <span>分组: {{ detailHost?.group?.name || '-' }}</span>
          <span>实例: {{ detailInstanceLabel || '-' }}</span>
        </div>
        <div class="detail-conn">
          <span class="conn-label">SSH:</span>
          <el-input-number v-model="detailConnForm.port" :min="1" :max="65535" size="small" controls-position="right" style="width:100px" />
          <el-input v-model="detailConnForm.username" placeholder="用户名" size="small" style="width:110px" />
          <el-input v-model="detailConnForm.password" type="password" show-password placeholder="新密码" size="small" style="width:120px" />
          <el-button size="small" type="primary" :loading="detailConnSaving" @click="saveDetailConn">保存连接</el-button>
        </div>

        <div class="detail-actions">
          <el-radio-group v-model="detailRangeHours" size="small" @change="fetchDetailMetrics()">
            <el-radio-button :label="1">1h</el-radio-button>
            <el-radio-button :label="6">6h</el-radio-button>
            <el-radio-button :label="24">24h</el-radio-button>
          </el-radio-group>
          <el-switch v-model="detailAutoRefresh" inline-prompt active-text="自动刷新" inactive-text="手动" />
          <el-button size="small" icon="Refresh" :loading="detailLoading" @click="fetchDetailMetrics()">刷新</el-button>
          <el-button size="small" type="primary" plain icon="Monitor" @click="openWebTerminal(detailHost)">Web 终端</el-button>
          <el-button size="small" type="success" plain @click="openInspect(detailHost, 'process')">进程</el-button>
          <el-button size="small" type="info" plain @click="openInspect(detailHost, 'tcp')">TCP</el-button>
        </div>
      </div>

      <div class="detail-metrics mb-4">
        <div class="metric-card">
          <span class="metric-label">CPU ({{ hardwareUsageText(detailHost, 'cpu', '核') }})</span>
          <strong>{{ metricText(detailHost, 'cpu') }}</strong>
        </div>
        <div class="metric-card">
          <span class="metric-label">内存 ({{ hardwareUsageText(detailHost, 'memory', 'G') }})</span>
          <strong>{{ metricText(detailHost, 'memory') }}</strong>
        </div>
        <div class="metric-card">
          <span class="metric-label">磁盘 ({{ hardwareUsageText(detailHost, 'disk', 'G') }})</span>
          <strong>{{ metricText(detailHost, 'disk') }}</strong>
        </div>
        <div class="metric-card">
          <span class="metric-label">最后检测</span>
          <strong>{{ formatTime(hostStatusMeta(detailHost || {}).checkAt || detailHost?.last_check_at) }}</strong>
        </div>
      </div>

      <!-- Cross-ref: Related Alerts & Docker -->
      <div class="detail-cross-ref" v-if="detailHost">
        <div class="cross-section">
          <div class="cross-title">
            <el-icon><WarningFilled /></el-icon> 关联告警
            <span class="cross-count" v-if="detailAlerts.length > 0">{{ detailAlerts.length }}</span>
          </div>
          <div v-if="detailAlerts.length === 0" class="cross-empty">无关联告警</div>
          <div v-for="a in detailAlerts.slice(0, 3)" :key="a.id" class="cross-item" @click="go('/alert/events/detail?id=' + a.id)">
            <el-tag :type="a.severity === 'critical' ? 'danger' : 'warning'" size="small">{{ a.severity }}</el-tag>
            <span>{{ a.alert_name || a.rule_name }}</span>
            <span class="cross-time">{{ fmtTimeAgo(a.fired_at || a.created_at) }}</span>
          </div>
          <el-button v-if="detailAlerts.length > 3" link type="primary" size="small" @click="go('/alert/events')">查看全部 →</el-button>
        </div>
        <div class="cross-section">
          <div class="cross-title">
            <el-icon><Platform /></el-icon> Docker 容器
            <span class="cross-count" v-if="detailDockerContainers.length > 0">{{ detailDockerContainers.length }}</span>
          </div>
          <div v-if="!detailDockerHost" class="cross-empty">非 Docker 主机或未配置</div>
          <div v-else-if="detailDockerContainers.length === 0" class="cross-empty">无运行中容器</div>
          <div v-for="c in detailDockerContainers.slice(0, 3)" :key="c.id || c.name" class="cross-item">
            <span class="cross-dot" :class="c.status === 'running' ? 'bg-green' : 'bg-red'"></span>
            <span>{{ c.name || c.id?.slice(0, 12) }}</span>
            <span class="cross-time">{{ c.image || '' }}</span>
          </div>
          <el-button v-if="detailDockerHost" link type="primary" size="small" @click="go('/docker')">Docker 管理 →</el-button>
        </div>
      </div>

      <el-empty v-if="!detailInstanceLabel" description="未匹配到监控实例，无法展示趋势图" />
      <div v-else class="detail-chart-grid" v-loading="detailLoading">
        <div class="chart-card"><div class="chart-title">CPU 使用率</div><div ref="detailCpuChartRef" class="detail-chart" /></div>
        <div class="chart-card"><div class="chart-title">内存使用率</div><div ref="detailMemChartRef" class="detail-chart" /></div>
        <div class="chart-card"><div class="chart-title">磁盘使用率</div><div ref="detailDiskChartRef" class="detail-chart" /></div>
        <div class="chart-card"><div class="chart-title">系统负载</div><div ref="detailLoadChartRef" class="detail-chart" /></div>
        <div class="chart-card"><div class="chart-title">网络接收 (KB/s)</div><div ref="detailNetInChartRef" class="detail-chart" /></div>
        <div class="chart-card"><div class="chart-title">网络发送 (KB/s)</div><div ref="detailNetOutChartRef" class="detail-chart" /></div>
      </div>
    </el-drawer>

    <el-dialog append-to-body v-model="dialogVisible" :title="isEdit ? '编辑主机' : '添加主机'" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="主机名" required>
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="IP地址" required>
          <el-input v-model="form.ip" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="form.port" :min="1" :max="65535" />
        </el-form-item>

        <el-form-item label="用户名">
          <el-input v-model="form.username" placeholder="root" />
        </el-form-item>
        <el-form-item label="密码">
          <div class="flex gap-2" style="width: 100%">
            <el-input v-model="form.password" :type="passwordVisible ? 'text' : 'password'" placeholder="如有变更请填写" style="flex: 1" />
            <el-checkbox v-model="passwordVisible">可见</el-checkbox>
          </div>
          <div v-if="isEdit" class="helper-row">已加载当前密码，可直接修改。</div>
        </el-form-item>
        <el-form-item label="分组">
          <el-input v-model="form.group_name" placeholder="默认分组" :disabled="isEdit" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="groupManageVisible" title="分组维护" width="820px">
      <div class="group-manage-toolbar">
        <el-button type="primary" icon="Plus" @click="openCreateGroup">新增分组</el-button>
        <el-button icon="Refresh" @click="fetchGroups">刷新</el-button>
      </div>
      <el-table :fit="true" :data="groups" stripe max-height="420" empty-text="暂无分组">
        <el-table-column prop="name" label="分组名称" min-width="180" />
        <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
        <el-table-column prop="parent_id" label="父级ID" min-width="160" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" plain @click="openEditGroup(row)">编辑</el-button>
            <el-button size="small" type="danger" plain @click="handleGroupDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog append-to-body v-model="groupEditorVisible" :title="groupEditorEdit ? '编辑分组' : '新增分组'" width="480px">
      <el-form :model="groupForm" label-width="90px">
        <el-form-item label="分组名称" required>
          <el-input v-model="groupForm.name" placeholder="如：生产环境" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="groupForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="父级ID">
          <el-input v-model="groupForm.parent_id" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupEditorVisible = false">取消</el-button>
        <el-button type="primary" :loading="groupSubmitting" @click="saveGroup">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="testVisible" title="主机测试" width="640px">
      <el-alert v-if="testError" type="error" :closable="false" show-icon>{{ testError }}</el-alert>
      <el-skeleton v-if="testLoading" :rows="4" animated />
      <div v-else class="test-block">
        <div class="test-title">uname -a</div>
        <pre class="test-pre">{{ testResult?.uname?.out || '-' }}</pre>
        <div class="test-title">/etc/os-release</div>
        <pre class="test-pre">{{ testResult?.os_release?.out || '-' }}</pre>
      </div>
      <template #footer>
        <el-button @click="testVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="inspectVisible" :title="inspectTitle" width="1040px" @closed="clearInspectAutoTimer">
      <div class="inspect-toolbar">
        <div class="inspect-host">
          <strong>{{ inspectHost?.name || '-' }}</strong>
          <span>{{ inspectHost?.ip || '-' }}</span>
          <span>更新时间：{{ inspectData.updatedAt ? formatTime(inspectData.updatedAt) : '-' }}</span>
        </div>
        <div class="inspect-actions">
          <el-radio-group v-model="inspectMode" size="small">
            <el-radio-button label="process">进程监控</el-radio-button>
            <el-radio-button label="tcp">TCP连接</el-radio-button>
          </el-radio-group>
          <el-switch v-model="inspectAutoRefresh" inline-prompt active-text="自动刷新" inactive-text="手动" />
          <el-button size="small" icon="Refresh" :loading="inspectLoading" @click="refreshInspect">手动刷新</el-button>
        </div>
      </div>

      <el-skeleton v-if="inspectLoading" :rows="6" animated />
      <template v-else>
        <div v-if="inspectMode === 'process'" class="inspect-process-grid">
          <el-card class="inspect-card" shadow="never">
            <template #header>Top CPU 进程</template>
            <el-table
              :fit="true"
              :data="inspectData.topCpu"
              size="small"
              max-height="300"
              empty-text="暂无进程数据"
              :row-class-name="processRowClassName"
            >
              <el-table-column prop="pid" label="PID" width="90" />
              <el-table-column prop="command" label="进程" min-width="180" show-overflow-tooltip />
              <el-table-column label="CPU%" width="130">
                <template #default="{ row }">
                  <el-progress
                    :percentage="toPercentNumber(row.cpu)"
                    :stroke-width="8"
                    :show-text="false"
                    :status="toNumber(row.cpu) >= 70 ? 'exception' : undefined"
                  />
                  <div class="mini-percent">{{ formatPercentText(row.cpu) }}</div>
                </template>
              </el-table-column>
              <el-table-column label="内存%" width="120">
                <template #default="{ row }">{{ formatPercentText(row.memory) }}</template>
              </el-table-column>
            </el-table>
          </el-card>
          <el-card class="inspect-card" shadow="never">
            <template #header>Top 内存进程</template>
            <el-table
              :fit="true"
              :data="inspectData.topMem"
              size="small"
              max-height="300"
              empty-text="暂无进程数据"
              :row-class-name="processRowClassName"
            >
              <el-table-column prop="pid" label="PID" width="90" />
              <el-table-column prop="command" label="进程" min-width="180" show-overflow-tooltip />
              <el-table-column label="内存%" width="130">
                <template #default="{ row }">
                  <el-progress
                    :percentage="toPercentNumber(row.memory)"
                    :stroke-width="8"
                    :show-text="false"
                    status="warning"
                  />
                  <div class="mini-percent">{{ formatPercentText(row.memory) }}</div>
                </template>
              </el-table-column>
              <el-table-column label="CPU%" width="120">
                <template #default="{ row }">{{ formatPercentText(row.cpu) }}</template>
              </el-table-column>
            </el-table>
          </el-card>
        </div>

        <div v-else class="inspect-tcp-block">
          <div class="tcp-summary">
            <el-tag type="info" effect="plain">总连接 {{ inspectData.tcpSummary.total || 0 }}</el-tag>
            <el-tag type="success" effect="plain">ESTABLISHED {{ inspectData.tcpSummary.established || 0 }}</el-tag>
            <el-tag type="warning" effect="plain">LISTEN {{ inspectData.tcpSummary.listen || 0 }}</el-tag>
            <el-tag type="danger" effect="plain">TIME_WAIT {{ inspectData.tcpSummary.time_wait || 0 }}</el-tag>
            <el-tag type="danger" effect="plain">风险连接 {{ tcpRiskCount }}</el-tag>
          </div>
          <el-table
            :fit="true"
            :data="inspectData.tcpRows"
            size="small"
            max-height="460"
            empty-text="暂无TCP连接数据"
            :row-class-name="tcpRowClassName"
          >
            <el-table-column prop="proto" label="协议" width="90" />
            <el-table-column label="状态" width="130">
              <template #default="{ row }">
                <el-tag size="small" :type="tcpStateTagType(row.state)" effect="plain">{{ row.state || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="local" label="本地地址" min-width="220" show-overflow-tooltip />
            <el-table-column prop="remote" label="远端地址" min-width="220" show-overflow-tooltip />
            <el-table-column prop="process" label="进程" min-width="220" show-overflow-tooltip />
          </el-table>
        </div>
      </template>

      <template #footer>
        <el-button @click="inspectVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="importVisible" title="批量导入主机" width="720px">
      <el-alert type="info" :closable="false" show-icon>
        格式：name,ip,port,username,password,group_name,status,os（第一行可写表头）
      </el-alert>
      <el-input
        v-model="importText"
        type="textarea"
        :rows="10"
        placeholder="例如：&#10;web-1,192.168.1.10,22,root,pass,prod,1,Ubuntu"
      />
      <div class="import-actions">
        <el-button @click="importVisible = false">取消</el-button>
        <el-button type="primary" :loading="importLoading" @click="submitImport">开始导入</el-button>
      </div>
    </el-dialog>

    <el-dialog append-to-body v-model="batchStatusVisible" title="批量修改状态" width="420px">
      <el-form label-width="80px">
        <el-form-item label="状态">
          <el-select v-model="batchStatus" placeholder="选择状态" style="width: 100%">
            <el-option label="在线" :value="1" />
            <el-option label="离线" :value="0" />
            <el-option label="维护" :value="2" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchStatusVisible = false">取消</el-button>
        <el-button type="primary" :loading="batchStatusLoading" @click="submitBatchStatus">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog append-to-body v-model="batchGroupVisible" title="批量设置分组" width="440px">
      <el-form label-width="90px">
        <el-form-item label="已选主机">
          <span style="font-weight: 600; color: #409eff;">{{ selectedRows.length }} 台</span>
        </el-form-item>
        <el-form-item label="目标分组">
          <el-select v-model="batchTargetGroupId" placeholder="请选择目标分组" style="width: 100%" clearable>
            <el-option label="未分组 (清空分组)" value="" />
            <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchGroupVisible = false">取消</el-button>
        <el-button type="primary" :loading="batchGroupLoading" @click="submitBatchGroup">确认保存</el-button>
      </template>
    </el-dialog>

    <!-- 批量命令执行弹窗 (含高危命令拦截) -->
    <el-dialog
      append-to-body
      v-model="batchExecVisible"
      title="主机批量下发 Shell 命令"
      width="880px"
      :close-on-click-modal="false"
    >
      <div class="batch-exec-container">
        <div class="exec-targets-bar">
          <span class="label">目标主机 ({{ selectedRows.length }} 台):</span>
          <div class="host-tags-scroll">
            <el-tag
              v-for="h in selectedRows"
              :key="h.id"
              size="small"
              type="info"
              class="mr-1 mb-1"
            >
              {{ h.name }} ({{ h.ip }})
            </el-tag>
          </div>
        </div>

        <div class="preset-commands-bar">
          <span class="preset-label">常用预设:</span>
          <el-button
            v-for="p in execPresets"
            :key="p.label"
            size="small"
            class="preset-btn"
            @click="applyPreset(p.cmd)"
          >
            {{ p.label }}
          </el-button>
        </div>

        <div class="exec-input-wrap">
          <el-input
            v-model="batchExecCommand"
            type="textarea"
            :rows="3"
            placeholder="输入要下发的 Shell 指令，如: uptime && df -h"
            class="command-textarea"
          />
        </div>

        <div v-if="isDangerousCommand" class="danger-warning-alert">
          <el-alert
            title="⚠️ 高危命令拦截防护"
            type="error"
            :description="dangerReason || '检测到潜在的高危破坏性指令，请仔细确认是否强制执行！'"
            show-icon
            :closable="false"
          />
          <div class="force-confirm-check">
            <el-checkbox v-model="forceConfirmExec">
              <span class="text-danger font-bold">我已知晓该命令风险，确认强制下发执行</span>
            </el-checkbox>
          </div>
        </div>

        <!-- 执行结果区 -->
        <div v-if="batchExecResults.length > 0" class="exec-results-section">
          <div class="results-summary">
            <span><strong>执行结果：</strong></span>
            <el-tag type="success" size="small">成功 {{ batchExecSummary.success }} 台</el-tag>
            <el-tag type="danger" size="small" class="ml-2">失败 {{ batchExecSummary.failed }} 台</el-tag>
          </div>

          <div class="results-list">
            <el-collapse v-model="activeCollapseHosts">
              <el-collapse-item
                v-for="res in batchExecResults"
                :key="res.host_id"
                :name="res.host_id"
              >
                <template #title>
                  <div class="collapse-title-row">
                    <el-tag :type="res.status === 'success' ? 'success' : 'danger'" size="small">
                      {{ res.status === 'success' ? 'SUCCESS' : 'FAILED' }}
                    </el-tag>
                    <strong class="ml-2">{{ res.hostname }}</strong>
                    <span class="text-gray-400 ml-1">({{ res.ip }}:{{ res.port }})</span>
                    <span class="dur-tag ml-auto mr-2">{{ res.duration_ms }}ms</span>
                  </div>
                </template>
                
                <div class="terminal-output-box">
                  <div v-if="res.stdout" class="stdout-block">
                    <div class="out-label">标准输出 (Stdout):</div>
                    <pre class="term-pre stdout-text">{{ res.stdout }}</pre>
                  </div>
                  <div v-if="res.stderr || res.error" class="stderr-block">
                    <div class="out-label text-danger">错误输出 (Stderr):</div>
                    <pre class="term-pre stderr-text">{{ res.stderr || res.error }}</pre>
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="batchExecVisible = false">关闭</el-button>
        <el-button
          type="primary"
          icon="VideoPlay"
          :loading="batchExecRunning"
          :disabled="!batchExecCommand.trim() || (isDangerousCommand && !forceConfirmExec)"
          @click="submitBatchExec"
        >
          立即并发执行
        </el-button>
      </template>
    </el-dialog>

    <!-- Web SSH 终端抽屉组件 -->
    <WebTerminalDrawer
      v-model="terminalDrawerVisible"
      :host-info="activeTerminalHost"
    />
  </el-card>
</template>

<script setup>
import { computed, ref, reactive, onBeforeUnmount, onMounted, nextTick, watch } from 'vue'
import axios from 'axios'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as echarts from 'echarts'
import StatusBadge from '@/components/common/StatusBadge.vue'
import WebTerminalDrawer from '@/components/common/WebTerminalDrawer.vue'
import { cmdbHostStatusMeta } from '@/utils/status'
import { useHostsStore } from '@/store/hosts'

const router = useRouter()
const hostsStore = useHostsStore()

const terminalDrawerVisible = ref(false)
const activeTerminalHost = ref(null)

const openWebTerminal = (row) => {
  if (!row) return
  activeTerminalHost.value = {
    id: row.id,
    name: row.name,
    hostname: row.name,
    ip: row.ip,
    port: row.port || 22,
    username: row.credential?.username || 'root',
    group_name: row.group?.name || '',
    os: row.os || '',
    provider: hostProvider(row)
  }
  terminalDrawerVisible.value = true
}

const UNGROUPED_GROUP_ID = '__ungrouped__'
const DEFAULT_HOST_PORT = 22

const loading = ref(false)
const tableData = ref([])
const currentPage = computed({
  get: () => hostsStore.currentPage,
  set: (val) => { hostsStore.currentPage = val }
})
const pageSize = computed({
  get: () => hostsStore.pageSize,
  set: (val) => { hostsStore.pageSize = val }
})
const totalHosts = ref(0)
const groups = ref([])
const selectedRows = ref([])
const searchKeyword = computed({
  get: () => hostsStore.searchKeyword,
  set: (val) => { hostsStore.searchKeyword = val }
})
const activeGroupId = computed({
  get: () => hostsStore.activeGroupId,
  set: (val) => { hostsStore.activeGroupId = val }
})
const groupKeyword = ref('')
const groupTreeRef = ref(null)

const showColumns = ref({
  ip: true,
  provider: true,
  os: true,
  cpu: true,
  memory: true,
  disk: true,
  status: true,
  desc: true
})

watch(showColumns, (val) => {
  localStorage.setItem('lazy_hosts_columns', JSON.stringify(val))
}, { deep: true })

const dialogVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const currentId = ref('')
const groupManageVisible = ref(false)
const groupEditorVisible = ref(false)
const groupEditorEdit = ref(false)
const groupSubmitting = ref(false)
const currentGroupId = ref('')

const testVisible = ref(false)
const testLoading = ref(false)
const testResult = ref(null)
const testError = ref('')

const importVisible = ref(false)
const importLoading = ref(false)
const importText = ref('')

const batchStatusVisible = ref(false)
const batchStatusLoading = ref(false)
const batchStatus = ref(null)
const passwordVisible = ref(false)

const syncingStatus = ref(false)
const hostMetricsMap = ref({})
const hostProviderMap = ref({})

const inspectVisible = ref(false)
const inspectLoading = ref(false)
const inspectHost = ref(null)
const inspectMode = ref('process')
const inspectAutoRefresh = ref(true)
const inspectData = ref({
  updatedAt: '',
  topCpu: [],
  topMem: [],
  tcpRows: [],
  tcpSummary: {}
})

const detailVisible = ref(false)
const detailHost = ref(null)
const detailLoading = ref(false)
const detailRangeHours = ref(1)
const detailAutoRefresh = ref(true)
const detailInstanceLabel = ref('')
const detailConnForm = reactive({ port: 22, username: '', password: '' })
const detailConnSaving = ref(false)
const detailAlerts = ref([])
const detailDockerHost = ref(null)
const detailDockerContainers = ref([])

const detailCpuChartRef = ref(null)
const detailMemChartRef = ref(null)
const detailDiskChartRef = ref(null)
const detailLoadChartRef = ref(null)
const detailNetInChartRef = ref(null)
const detailNetOutChartRef = ref(null)

let detailCpuChart = null
let detailMemChart = null
let detailDiskChart = null
let detailLoadChart = null
let detailNetInChart = null
let detailNetOutChart = null
let autoSyncTimer = null
let inspectAutoTimer = null
let detailAutoTimer = null

const form = reactive({
  name: '',
  ip: '',
  port: DEFAULT_HOST_PORT,
  status: 1,
  username: '',
  password: '',
  group_name: ''
})

const groupForm = reactive({
  name: '',
  description: '',
  parent_id: ''
})

const providerConfig = {
  aliyun: { label: '阿里云', type: 'warning' },
  aws: { label: 'AWS', type: 'danger' },
  huawei: { label: '华为云', type: 'success' },
  tencent: { label: '腾讯云', type: 'primary' },
  baidu: { label: '百度云', type: 'info' },
  azure: { label: 'Azure', type: 'primary' },
  gcp: { label: 'GCP', type: 'success' },
  onprem: { label: '自建', type: '' },
  unknown: { label: '-', type: 'info' }
}

const authHeaders = () => ({ Authorization: 'Bearer ' + localStorage.getItem('token') })

const getErrorMessage = (e, fallback) => {
  if (typeof e === 'string' && e.trim()) return e.trim()
  if (e?.response?.data?.message) return e.response.data.message
  if (e?.response?.data?.error) return e.response.data.error
  if (e?.message) return e.message
  return fallback
}

const toTime = (value) => {
  if (!value) return null
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? null : ts
}

const formatTime = (value) => {
  const ts = toTime(value)
  if (!ts) return '-'
  return new Date(ts).toLocaleString()
}

const toNumber = (value) => {
  if (value === null || value === undefined) return NaN
  if (typeof value === 'number') return value
  const str = String(value).trim()
  const match = str.match(/-?\d+(?:\.\d+)?/)
  if (match) return parseFloat(match[0])
  return NaN
}

const normalizeHostAddress = (value) => {
  const text = String(value || '').trim()
  if (!text) return ''
  if (text.startsWith('[')) {
    const idx = text.indexOf(']')
    if (idx > 0) return text.slice(1, idx)
  }
  const lastColon = text.lastIndexOf(':')
  if (lastColon > 0 && text.indexOf(':') === lastColon) {
    return text.slice(0, lastColon)
  }
  return text
}

const hostStatusMeta = (row) => cmdbHostStatusMeta(row || {}, { staleMinutes: 3 })

const hostMetric = (row) => {
  const ip = normalizeHostAddress(row?.ip)
  const byIP = hostMetricsMap.value[ip]
  if (byIP) return byIP
  const byName = hostMetricsMap.value[String(row?.name || '').trim()]
  if (byName) return byName
  return {}
}

const normalizeProvider = (value) => {
  const text = String(value || '').trim().toLowerCase()
  if (!text || text === 'unknown') return 'unknown'
  if (text.includes('ali') || text.includes('alibaba')) return 'aliyun'
  if (text.includes('aws') || text.includes('amazon')) return 'aws'
  if (text.includes('huawei') || text.includes('华为')) return 'huawei'
  if (text.includes('tencent') || text.includes('腾讯')) return 'tencent'
  if (text.includes('baidu') || text.includes('百度')) return 'baidu'
  if (text.includes('azure') || text.includes('7783-7084')) return 'azure'
  if (text.includes('gcp') || text.includes('google')) return 'gcp'
  if (
    text.includes('onprem') ||
    text.includes('自建') ||
    text.includes('private') ||
    text.includes('microsoft') ||
    text.includes('hyper-v') ||
    text.includes('hyperv') ||
    text.includes('vmware') ||
    text.includes('qemu') ||
    text.includes('kvm') ||
    text.includes('virtualbox') ||
    text.includes('innotek') ||
    text.includes('dell') ||
    text.includes('hpe') ||
    text.includes('hp') ||
    text.includes('inspur') ||
    text.includes('lenovo') ||
    text.includes('supermicro') ||
    text.includes('xen') ||
    text.includes('bochs')
  ) return 'onprem'
  return 'onprem'
}

const providerLabel = (key) => (providerConfig[key] || providerConfig.unknown).label
const providerTagType = (key) => (providerConfig[key] || providerConfig.unknown).type

const parseProviderFromTags = (tags) => {
  const text = String(tags || '').toLowerCase()
  if (!text) return ''
  const providerMatch = text.match(/provider\s*[:=]\s*([a-zA-Z0-9_-]+)/)
  if (providerMatch?.[1]) return normalizeProvider(providerMatch[1])
  const vendorMatch = text.match(/vendor\s*[:=]\s*([a-zA-Z0-9_-]+)/)
  if (vendorMatch?.[1]) return normalizeProvider(vendorMatch[1])
  return ''
}

const hostProvider = (row) => {
  const providerFromMetric = normalizeProvider(hostMetric(row)?.provider)
  if (providerFromMetric && providerFromMetric !== 'unknown') return providerFromMetric
  const ip = normalizeHostAddress(row?.ip)
  if (ip && hostProviderMap.value[ip] && hostProviderMap.value[ip] !== 'unknown') return hostProviderMap.value[ip]
  const providerFromTags = parseProviderFromTags(row?.tags)
  if (providerFromTags && providerFromTags !== 'unknown') return providerFromTags
  return 'onprem'
}

const formatPercent = (metricValue, fallback = '') => {
  const metricNum = toNumber(metricValue)
  if (Number.isFinite(metricNum)) return `${metricNum.toFixed(1)}%`
  const fallbackNum = toNumber(fallback)
  if (Number.isFinite(fallbackNum)) return `${fallbackNum.toFixed(1)}%`
  return '-'
}

const metricValue = (row, key) => {
  const metricNum = toNumber(hostMetric(row)?.[key])
  return Number.isFinite(metricNum) ? metricNum : NaN
}

const hasMetricValue = (row, key) => Number.isFinite(metricValue(row, key))

const metricText = (row, key) => {
  const value = metricValue(row, key)
  if (!Number.isFinite(value)) return '--'
  return `${value.toFixed(1)}%`
}

const hardwareUsageText = (row, key, unit) => {
  if (!row) return 'N/A'
  
  // 1. 优先尝试从 CMDB 静态资产字段提取
  let total = toNumber(row[key])
  
  // 2. 若静态字段未填，智能从 Prometheus / Monitor 监控指标对象中提取物理总容量
  const metric = hostMetric(row) || {}
  if (!Number.isFinite(total)) {
    if (key === 'cpu') {
      total = toNumber(metric.cpu_cores || metric.cores || metric.total_cpu || metric.cpu_total)
    } else if (key === 'memory') {
      total = toNumber(metric.mem_total_gb || metric.memory_total || metric.mem_total || metric.total_mem)
    } else if (key === 'disk') {
      total = toNumber(metric.disk_total_gb || metric.disk_total || metric.total_disk)
    }
  }

  const usagePct = metricValue(row, key)

  // 如果主机未在线（离线状态）或监控不可用
  const statusMeta = hostStatusMeta(row)
  const isOffline = statusMeta.type === 'offline' || row.status === 0

  if (isOffline) {
    if (Number.isFinite(total)) {
      const totalFormatted = (total % 1 === 0) ? total.toFixed(0) : total.toFixed(1)
      return `- / ${totalFormatted}${unit}`
    }
    return 'N/A'
  }

  // 无百分比监控数据
  if (!Number.isFinite(usagePct)) {
    if (Number.isFinite(total)) {
      const totalFormatted = (total % 1 === 0) ? total.toFixed(0) : total.toFixed(1)
      return `- / ${totalFormatted}${unit}`
    }
    return 'N/A'
  }

  // 在线且拥有使用率
  if (!Number.isFinite(total)) {
    return `${usagePct.toFixed(1)}%`
  }

  // 既有总容量又有使用率，计算精确使用量与总量
  const used = (total * (usagePct / 100))
  const usedFormatted = (used % 1 === 0) ? used.toFixed(0) : used.toFixed(1)
  const totalFormatted = (total % 1 === 0) ? total.toFixed(0) : total.toFixed(1)
  return `${usedFormatted}${unit} / ${totalFormatted}${unit}`
}

const metricTagType = (value) => {
  const num = toNumber(value)
  if (!Number.isFinite(num)) return 'info'
  if (num >= 85) return 'danger'
  if (num >= 70) return 'warning'
  return 'success'
}

const globalHostList = ref([])

const fetchGlobalHostsSummary = async () => {
  try {
    const res = await axios.get('/api/v1/cmdb/hosts', {
      headers: authHeaders(),
      params: { page: 1, page_size: 10000 }
    })
    if (res.data?.code === 0 && res.data?.data?.list) {
      globalHostList.value = res.data.data.list
    }
  } catch (e) {}
}

const groupHostCountMap = computed(() => {
  const map = {}
  let ungrouped = 0
  const source = globalHostList.value.length ? globalHostList.value : tableData.value
  source.forEach((row) => {
    const gid = String(row?.group?.id || row?.group_id || '').trim()
    if (!gid) {
      ungrouped += 1
      return
    }
    map[gid] = (map[gid] || 0) + 1
  })
  map[UNGROUPED_GROUP_ID] = ungrouped
  return map
})

const groupTreeData = computed(() => {
  const nodes = {}
  const roots = []

  groups.value.forEach((g) => {
    nodes[g.id] = {
      id: g.id,
      label: g.name,
      count: groupHostCountMap.value[g.id] || 0,
      parent_id: g.parent_id || '',
      children: []
    }
  })

  Object.values(nodes).forEach((node) => {
    if (node.parent_id && nodes[node.parent_id]) {
      nodes[node.parent_id].children.push(node)
    } else {
      roots.push(node)
    }
  })

  const totalAllCount = globalHostList.value.length ? globalHostList.value.length : totalHosts.value
  return [
    { id: 'all', label: '全部主机', count: totalAllCount, children: [] },
    ...roots,
    { id: UNGROUPED_GROUP_ID, label: '未分组', count: groupHostCountMap.value[UNGROUPED_GROUP_ID] || 0, children: [] }
  ]
})

const groupNodeFilter = (value, data) => {
  if (!value) return true
  return String(data?.label || '').toLowerCase().includes(String(value).toLowerCase())
}

const statusFilter = ref('all')

const toggleStatusFilter = (target) => {
  if (statusFilter.value === target) {
    statusFilter.value = 'all'
  } else {
    statusFilter.value = target
  }
}

const totalCount = computed(() => tableData.value.length)
const onlineCount = computed(() => tableData.value.filter((row) => hostStatusMeta(row).key === 'online').length)
const offlineCount = computed(() => tableData.value.filter((row) => hostStatusMeta(row).key !== 'online').length)

const selectAllPage = () => {
  if (pageSize.value >= 1000) {
    pageSize.value = 20
  } else {
    pageSize.value = 1000
  }
  currentPage.value = 1
  fetchData()
}

const filteredTableData = computed(() => {
  if (statusFilter.value === 'online') {
    return tableData.value.filter((row) => hostStatusMeta(row).key === 'online')
  }
  if (statusFilter.value === 'offline') {
    return tableData.value.filter((row) => hostStatusMeta(row).key !== 'online')
  }
  return tableData.value
})

watch(activeGroupId, () => {
  currentPage.value = 1
  fetchData()
})

const providerSummaryList = computed(() => {
  const counter = {}
  filteredTableData.value.forEach((row) => {
    const key = hostProvider(row)
    counter[key] = (counter[key] || 0) + 1
  })
  const baseKeys = ['onprem', 'aliyun', 'tencent', 'huawei', 'aws', 'baidu', 'azure', 'gcp', 'unknown']
  return baseKeys
    .filter((key) => (counter[key] || 0) > 0)
    .map((key) => ({ key, label: providerLabel(key), count: counter[key] || 0 }))
})
const inspectTitle = computed(() => (inspectMode.value === 'tcp' ? 'TCP 连接监控' : '进程监控'))
const tcpRiskCount = computed(() => inspectData.value.tcpRows.filter((row) => isRiskTCPRow(row)).length)

const handleGroupKeywordChange = (value) => {
  groupTreeRef.value?.filter(value)
}

const onGroupNodeClick = (node) => {
  if (!node) return
  if (node.id === 'all') {
    activeGroupId.value = ''
    return
  }
  activeGroupId.value = node.id
}

const onGroupSelectChange = (value) => {
  const key = value || 'all'
  nextTick(() => {
    groupTreeRef.value?.setCurrentKey(key)
  })
}

const clearGroupFilter = () => {
  activeGroupId.value = ''
  groupKeyword.value = ''
  groupTreeRef.value?.filter('')
  nextTick(() => {
    groupTreeRef.value?.setCurrentKey('all')
  })
}

const resetGroupForm = () => {
  groupForm.name = ''
  groupForm.description = ''
  groupForm.parent_id = ''
}

const openGroupManager = async () => {
  groupManageVisible.value = true
  await fetchGroups()
}

const openCreateGroup = () => {
  currentGroupId.value = ''
  groupEditorEdit.value = false
  resetGroupForm()
  groupEditorVisible.value = true
}

const openEditGroup = (row) => {
  currentGroupId.value = row.id
  groupEditorEdit.value = true
  groupForm.name = row.name || ''
  groupForm.description = row.description || ''
  groupForm.parent_id = row.parent_id || ''
  groupEditorVisible.value = true
}

const openConnectionEditor = () => {
  if (selectedRows.value.length !== 1) {
    ElMessage.warning('请选择 1 台主机后再编辑连接信息')
    return
  }
  handleEdit(selectedRows.value[0])
}

const saveGroup = async () => {
  if (!groupForm.name.trim()) {
    ElMessage.warning('请填写分组名称')
    return
  }
  groupSubmitting.value = true
  try {
    const url = groupEditorEdit.value ? `/api/v1/cmdb/groups/${currentGroupId.value}` : '/api/v1/cmdb/groups'
    const method = groupEditorEdit.value ? 'put' : 'post'
    const res = await axios({
      url,
      method,
      headers: authHeaders(),
      data: {
        name: groupForm.name.trim(),
        description: groupForm.description,
        parent_id: groupForm.parent_id
      }
    })
    if (res.data?.code === 0) {
      ElMessage.success(groupEditorEdit.value ? '分组更新成功' : '分组创建成功')
      groupEditorVisible.value = false
      await Promise.all([fetchGroups(), fetchData()])
    } else {
      ElMessage.error(res.data?.message || '分组保存失败')
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '分组保存失败'))
  } finally {
    groupSubmitting.value = false
  }
}

const handleGroupDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确定删除分组“${row.name}”吗？`, '提示', {
      type: 'warning'
    })
    await axios.delete(`/api/v1/cmdb/groups/${row.id}`, { headers: authHeaders() })
    if (String(activeGroupId.value || '') === String(row.id)) {
      clearGroupFilter()
    }
    ElMessage.success('分组删除成功')
    await Promise.all([fetchGroups(), fetchData()])
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(getErrorMessage(e, '分组删除失败'))
    }
  }
}

const fetchData = async () => {
  loading.value = true
  try {
    const params = {
      keyword: searchKeyword.value,
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (activeGroupId.value) {
      params.group_id = activeGroupId.value
    }
    const hostReq = axios.get('/api/v1/cmdb/hosts', {
      headers: authHeaders(),
      params
    })
    const metricsReq = axios.get('/api/v1/monitor/servers', { headers: authHeaders() })
    const cloudReq = axios.get('/api/v1/cmdb/cloud/resources', { headers: authHeaders() })
    const [hostRes, metricsRes, cloudRes] = await Promise.allSettled([hostReq, metricsReq, cloudReq])

    if (hostRes.status === 'fulfilled' && hostRes.value.data?.code === 0) {
      const resData = hostRes.value.data.data
      if (resData && typeof resData === 'object' && Array.isArray(resData.list)) {
        tableData.value = resData.list
        totalHosts.value = resData.total || 0
      } else {
        tableData.value = Array.isArray(resData) ? resData : []
        totalHosts.value = tableData.value.length
      }
    } else {
      const reason = hostRes.status === 'rejected' ? hostRes.reason : hostRes.value?.data?.message
      ElMessage.error(getErrorMessage(reason, '加载主机列表失败'))
    }

    if (metricsRes.status === 'fulfilled' && metricsRes.value.data?.code === 0) {
      const rows = Array.isArray(metricsRes.value.data.data) ? metricsRes.value.data.data : []
      const map = {}
      rows.forEach((item) => {
        const ipKey = normalizeHostAddress(item?.ip || item?.instance)
        const nameKey = String(item?.hostname || '').trim()
        if (ipKey) map[ipKey] = item
        if (nameKey) map[nameKey] = item
      })
      hostMetricsMap.value = map
    } else {
      hostMetricsMap.value = {}
    }

    if (cloudRes.status === 'fulfilled' && cloudRes.value.data?.code === 0) {
      const rows = Array.isArray(cloudRes.value.data.data) ? cloudRes.value.data.data : []
      const providerMap = {}
      rows.forEach((item) => {
        const ip = normalizeHostAddress(item?.ip)
        if (!ip) return
        const provider = normalizeProvider(item?.account?.provider || item?.provider)
        if (!providerMap[ip] && provider !== 'unknown') providerMap[ip] = provider
      })
      hostProviderMap.value = providerMap
    } else {
      hostProviderMap.value = {}
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '加载失败'))
  } finally {
    loading.value = false
  }
}

const syncStatuses = async (silent = false) => {
  syncingStatus.value = true
  try {
    const res = await axios.post('/api/v1/cmdb/hosts/sync-status', {}, { headers: authHeaders() })
    if (res.data?.code === 0 && !silent) {
      const info = res.data?.data || {}
      ElMessage.success(`巡检完成：在线 ${info.online ?? 0}，离线 ${info.offline ?? 0}，维护 ${info.maintenance ?? 0}`)
    }
  } catch (e) {
    if (!silent) ElMessage.error(getErrorMessage(e, '巡检失败'))
  } finally {
    syncingStatus.value = false
    await fetchData()
  }
}

const fetchGroups = async () => {
  try {
    const res = await axios.get('/api/v1/cmdb/groups', { headers: authHeaders() })
    if (res.data?.code === 0) groups.value = res.data.data || []
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '分组加载失败'))
  }
}

const handleAdd = () => {
  isEdit.value = false
  form.name = ''
  form.ip = ''
  form.port = DEFAULT_HOST_PORT
  form.status = 1
  form.username = 'root'
  form.password = ''
  form.group_name = ''
  dialogVisible.value = true
}

const handleEdit = async (row) => {
  isEdit.value = true
  currentId.value = row.id
  try {
    const res = await axios.get(`/api/v1/cmdb/hosts/${row.id}`, { headers: authHeaders() })
    if (res.data.code === 0) {
      const data = res.data.data
      form.name = data.name
      form.ip = data.ip
      form.port = data.port
      form.status = data.status ?? 1
      form.username = data.credential ? data.credential.username : ''
      form.password = data.credential ? data.credential.password : ''
      form.group_name = data.group ? data.group.name : ''
      dialogVisible.value = true
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '获取详情失败'))
  }
}

const submitForm = async () => {
  submitting.value = true
  try {
    const url = isEdit.value ? `/api/v1/cmdb/hosts/${currentId.value}` : '/api/v1/cmdb/hosts'
    const method = isEdit.value ? 'put' : 'post'

    const res = await axios({
      method,
      url,
      data: form,
      headers: authHeaders()
    })

    if (res.data.code === 0) {
      ElMessage.success(isEdit.value ? '更新成功' : '添加成功')
      dialogVisible.value = false
      syncStatuses(true)
    } else {
      ElMessage.error(res.data.message)
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, isEdit.value ? '更新失败' : '添加失败'))
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm('确定删除该主机吗?', '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await axios.delete(`/api/v1/cmdb/hosts/${row.id}`, { headers: authHeaders() })
    ElMessage.success('删除成功')
    await fetchData()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') ElMessage.error(getErrorMessage(e, '删除失败'))
  }
}

const handleBatchDelete = async () => {
  if (selectedRows.value.length === 0) return
  const ids = selectedRows.value.map(row => row.id).filter(Boolean)
  if (ids.length === 0) return

  try {
    await ElMessageBox.confirm(`确定批量删除选中的 ${ids.length} 台主机吗？`, '批量删除确认', {
      confirmButtonText: '确定删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    const res = await axios.post('/api/v1/cmdb/hosts/batch-delete', { ids }, { headers: authHeaders() })
    if (res.data?.code === 0) {
      ElMessage.success(res.data.message || `成功批量删除 ${ids.length} 台主机`)
    } else {
      for (const id of ids) {
        await axios.delete(`/api/v1/cmdb/hosts/${id}`, { headers: authHeaders() })
      }
      ElMessage.success(`成功批量删除 ${ids.length} 台主机`)
    }
    selectedRows.value = []
    await fetchData()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') ElMessage.error(getErrorMessage(e, '批量删除失败'))
  }
}

const openImport = () => {
  importText.value = ''
  importVisible.value = true
}

const parseCSV = (text) => {
  const lines = text.split(/\r?\n/).map((l) => l.trim()).filter(Boolean)
  if (lines.length === 0) return []
  const delim = lines[0].includes('\t') ? '\t' : ','
  const headers = lines[0].toLowerCase().split(delim).map((s) => s.trim())
  const hasHeader = headers.includes('name') || headers.includes('ip')
  const start = hasHeader ? 1 : 0
  const cols = hasHeader ? headers : ['name', 'ip', 'port', 'username', 'password', 'group_name', 'status', 'os']
  return lines.slice(start).map((line) => {
    const parts = line.split(delim).map((s) => s.trim())
    const obj = {}
    cols.forEach((k, idx) => { obj[k] = parts[idx] || '' })
    return obj
  })
}

const submitImport = async () => {
  const rows = parseCSV(importText.value)
  if (rows.length === 0) {
    ElMessage.warning('请填写导入内容')
    return
  }
  importLoading.value = true
  try {
    for (const row of rows) {
      if (!row.name || !row.ip) continue
      await axios.post('/api/v1/cmdb/hosts', {
        name: row.name,
        ip: row.ip,
        port: row.port ? Number(row.port) : DEFAULT_HOST_PORT,
        username: row.username || '',
        password: row.password || '',
        group_name: row.group_name || '',
        status: row.status ? Number(row.status) : 1,
        os: row.os || ''
      }, {
        headers: authHeaders()
      })
    }
    ElMessage.success('导入完成')
    importVisible.value = false
    await syncStatuses(true)
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '导入失败'))
  } finally {
    importLoading.value = false
  }
}

const exportCSV = () => {
  const headers = ['name', 'ip', 'provider', 'port', 'os', 'status', 'group', 'username']
  const rows = filteredTableData.value.map((h) => [
    h.name,
    h.ip,
    providerLabel(hostProvider(h)),
    h.port,
    h.os,
    h.status,
    h.group?.name || '',
    h.credential?.username || ''
  ])
  const csv = [headers.join(','), ...rows.map((r) => r.map((v) => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))].join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'cmdb_hosts.csv'
  a.click()
  URL.revokeObjectURL(url)
}

const openBatchStatus = () => {
  batchStatus.value = 1
  batchStatusVisible.value = true
}

const batchGroupVisible = ref(false)
const batchTargetGroupId = ref('')
const batchGroupLoading = ref(false)

const openBatchGroup = () => {
  if (selectedRows.value.length === 0) {
    ElMessage.warning('请至少选择一台主机')
    return
  }
  batchTargetGroupId.value = ''
  batchGroupVisible.value = true
}

const submitBatchGroup = async () => {
  if (selectedRows.value.length === 0) return
  batchGroupLoading.value = true
  try {
    const ids = selectedRows.value.map((r) => r.id)
    const res = await axios.post('/api/v1/cmdb/hosts/batch-update-group', {
      ids,
      group_id: batchTargetGroupId.value
    }, { headers: authHeaders() })
    if (res.data?.code === 0) {
      ElMessage.success(res.data.message || `成功为 ${ids.length} 台主机设置分组`)
      batchGroupVisible.value = false
      selectedRows.value = []
      await fetchData()
      await fetchGlobalHostsSummary()
    } else {
      ElMessage.error(res.data?.message || '批量设置分组失败')
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '批量设置分组失败'))
  } finally {
    batchGroupLoading.value = false
  }
}

// 批量执行 Shell 命令
const batchExecVisible = ref(false)
const batchExecCommand = ref('')
const batchExecRunning = ref(false)
const forceConfirmExec = ref(false)
const batchExecResults = ref([])
const activeCollapseHosts = ref([])

const execPresets = [
  { label: '系统信息 (uptime)', cmd: 'uptime && uname -a' },
  { label: '磁盘占用 (df -h)', cmd: 'df -h' },
  { label: '内存使用 (free -m)', cmd: 'free -m' },
  { label: '清理缓存 (drop_caches)', cmd: 'sync && echo 3 > /proc/sys/vm/drop_caches' },
  { label: 'Docker 容器状态', cmd: 'docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}" || true' },
  { label: '端口监听 (netstat)', cmd: 'netstat -tulnp || ss -tulnp' }
]

const dangerRegexList = [
  /\brm\s+-[rRfF]*\s+(\/|\*|\/\*)/i,
  /\bmkfs\b/i,
  /\bdd\s+if=/i,
  />\s*\/dev\/sd[a-z]/i,
  /\bdrop\s+database\b/i,
  /\btruncate\s+table\b/i,
  /\biptables\s+-F\b/i,
  /:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:/,
  /\bshutdown\b/i,
  /\binit\s+0\b/i,
  /\breboot\b/i
]

const isDangerousCommand = computed(() => {
  const cmd = batchExecCommand.value.trim()
  if (!cmd) return false
  return dangerRegexList.some(r => r.test(cmd))
})

const dangerReason = computed(() => {
  if (!isDangerousCommand.value) return ''
  return '检测到可能包含关机/重启/删除根目录/格式化磁盘的高危指令！'
})

const batchExecSummary = computed(() => {
  let success = 0
  let failed = 0
  let totalDuration = 0
  for (const r of batchExecResults.value) {
    if (r.status === 'success') success++
    else failed++
    totalDuration += (r.duration_ms || 0)
  }
  return { success, failed, totalDuration }
})

const applyPreset = (cmd) => {
  batchExecCommand.value = cmd
}

const openBatchExec = () => {
  if (selectedRows.value.length === 0) {
    ElMessage.warning('请至少选择一台主机')
    return
  }
  batchExecCommand.value = 'uptime && df -h'
  forceConfirmExec.value = false
  batchExecResults.value = []
  activeCollapseHosts.value = []
  batchExecVisible.value = true
}

const submitBatchExec = async () => {
  if (!batchExecCommand.value.trim()) {
    ElMessage.warning('请输入要执行的指令')
    return
  }
  if (isDangerousCommand.value && !forceConfirmExec.value) {
    ElMessage.error('高危命令已被拦截，如需执行请勾选下方强制确认')
    return
  }

  batchExecRunning.value = true
  try {
    const host_ids = selectedRows.value.map(r => r.id)
    const res = await axios.post('/api/v1/cmdb/hosts/batch-exec', {
      host_ids,
      command: batchExecCommand.value.trim(),
      timeout_seconds: 30,
      force_confirm: forceConfirmExec.value
    }, { headers: authHeaders() })

    if (res.data?.code === 0) {
      batchExecResults.value = res.data.data.results || []
      // 默认展开所有主机的输出结果
      activeCollapseHosts.value = batchExecResults.value.map(r => r.host_id)
      ElMessage.success(res.data.message || '批量执行完成')
    } else if (res.data?.code === 403) {
      ElMessage.error(res.data.message || '命令被安全策略拦截')
    } else {
      ElMessage.error(res.data?.message || '批量执行失败')
    }
  } catch (err) {
    ElMessage.error(getErrorMessage(err, '下发批量命令失败'))
  } finally {
    batchExecRunning.value = false
  }
}

const handleTest = async (row) => {
  testVisible.value = true
  testLoading.value = true
  testResult.value = null
  testError.value = ''
  try {
    const res = await axios.post(`/api/v1/cmdb/hosts/${row.id}/test`, {}, { headers: authHeaders() })
    if (res.data.code === 0) {
      testResult.value = res.data.data
    } else {
      testError.value = res.data.message || '测试失败'
    }
  } catch (e) {
    testError.value = getErrorMessage(e, '测试失败')
  } finally {
    testLoading.value = false
    await fetchData()
  }
}

const toPercentNumber = (value) => {
  const num = toNumber(value)
  if (!Number.isFinite(num) || num < 0) return 0
  if (num > 100) return 100
  return Number(num.toFixed(2))
}

const formatPercentText = (value) => {
  const num = toNumber(value)
  if (!Number.isFinite(num)) return '-'
  return `${num.toFixed(2)}%`
}

const normalizeProcessRows = (rows) =>
  Array.isArray(rows)
    ? rows.map((item) => ({
        pid: item?.pid || '-',
        command: item?.command || '-',
        cpu: item?.cpu || '0',
        memory: item?.memory || '0'
      }))
    : []

const normalizeTCPRows = (rows) =>
  Array.isArray(rows)
    ? rows.map((item) => ({
        proto: item?.proto || '-',
        state: item?.state || '-',
        local: item?.local || '-',
        remote: item?.remote || '-',
        process: item?.process || '-'
      }))
    : []

const getLocalPort = (localAddr) => {
  const text = String(localAddr || '').trim()
  if (!text) return ''
  const idx = text.lastIndexOf(':')
  if (idx < 0) return ''
  return text.slice(idx + 1)
}

const riskyStates = new Set(['syn-recv', 'close-wait', 'fin-wait-2', 'last-ack'])
const commonListenPorts = new Set(['22', '80', '443', '2379', '2380', '6443', '3306', '5432', '6379', '27017'])

const isRiskTCPRow = (row) => {
  const state = String(row?.state || '').toLowerCase()
  if (riskyStates.has(state)) return true
  if (state === 'listen') {
    const port = getLocalPort(row?.local)
    if (port && !commonListenPorts.has(port)) return true
  }
  return false
}

const processRowClassName = ({ row }) => {
  const cpu = toNumber(row?.cpu)
  const mem = toNumber(row?.memory)
  if ((Number.isFinite(cpu) && cpu >= 70) || (Number.isFinite(mem) && mem >= 80)) return 'danger-row'
  if ((Number.isFinite(cpu) && cpu >= 40) || (Number.isFinite(mem) && mem >= 60)) return 'warn-row'
  return ''
}

const tcpRowClassName = ({ row }) => (isRiskTCPRow(row) ? 'danger-row' : '')

const tcpStateTagType = (state) => {
  const text = String(state || '').toLowerCase()
  if (text === 'established') return 'success'
  if (text === 'listen') return 'warning'
  if (text.includes('time')) return 'info'
  if (riskyStates.has(text)) return 'danger'
  return 'info'
}

const loadInspect = async (row, { silent = false } = {}) => {
  if (!row?.id) return
  inspectLoading.value = true
  try {
    const res = await axios.post(`/api/v1/cmdb/hosts/${row.id}/test`, {}, { headers: authHeaders() })
    if (res.data?.code !== 0) throw new Error(res.data?.message || '检测失败')
    const payload = res.data?.data || {}
    inspectData.value = {
      updatedAt: new Date().toISOString(),
      topCpu: normalizeProcessRows(payload?.processes?.top_cpu),
      topMem: normalizeProcessRows(payload?.processes?.top_mem),
      tcpRows: normalizeTCPRows(payload?.tcp_connections),
      tcpSummary: payload?.tcp_summary || {}
    }
  } catch (e) {
    if (!silent) ElMessage.error(getErrorMessage(e, '加载主机监控详情失败'))
  } finally {
    inspectLoading.value = false
  }
}

const clearInspectAutoTimer = () => {
  if (inspectAutoTimer) {
    window.clearInterval(inspectAutoTimer)
    inspectAutoTimer = null
  }
}

const ensureInspectAutoTimer = () => {
  clearInspectAutoTimer()
  if (!inspectVisible.value || !inspectAutoRefresh.value) return
  inspectAutoTimer = window.setInterval(() => {
    if (document.hidden || inspectLoading.value || !inspectHost.value) return
    loadInspect(inspectHost.value, { silent: true })
  }, 15000)
}

const openInspect = async (row, mode = 'process') => {
  if (!row) return
  inspectHost.value = row
  inspectMode.value = mode
  inspectVisible.value = true
  await loadInspect(row)
  ensureInspectAutoTimer()
}

const refreshInspect = async () => {
  if (!inspectHost.value) return
  await loadInspect(inspectHost.value)
}

const openMonitor = (row) => {
  const keyword = String(row?.ip || row?.name || '').trim()
  router.push({
    path: '/monitor/hosts',
    query: keyword ? { keyword } : {}
  })
}

const openDiagnose = async (row) => {
  try {
    const res = await axios.post(`/api/v1/cmdb/hosts/${row.id}/diagnose`, {}, { headers: authHeaders() })
    if (res.data?.code === 0) {
      const diag = res.data.data
      const portsText = (diag.tcp_ports || []).map(p => `端口 ${p.port} (${p.protocol}): ${p.status === 'OPEN' ? '🟢 开放' : '🔴 关闭'}`).join('\n')
      ElMessageBox.alert(
        `目标主机: ${diag.target_host} (${diag.target_ip})\nICMP Ping 状态: ${diag.ping_status} (${diag.ping_latency_ms}ms, 丢包率: ${diag.packet_loss})\n\n关键服务端口连通性:\n${portsText}`,
        '网络连通性深度诊断结果',
        { confirmButtonText: '确定', type: 'success' }
      )
    }
  } catch (e) {
    ElMessage.error(getErrorMessage(e, '诊断失败'))
  }
}

const handleRowCommand = (row, command) => {
  if (command === 'diagnose') {
    openDiagnose(row)
    return
  }
  if (command === 'process') {
    openInspect(row, 'process')
    return
  }
  if (command === 'tcp') {
    openInspect(row, 'tcp')
    return
  }
  if (command === 'monitor') {
    openMonitor(row)
    return
  }
  if (command === 'delete') {
    handleDelete(row)
  }
}

const buildSelector = (base, instance) => {
  const baseText = String(base || '').trim()
  if (!instance) return baseText
  return baseText ? `${baseText},instance="${instance}"` : `instance="${instance}"`
}

const queryCpu = (instance) => `100 - (avg by(instance) (irate(node_cpu_seconds_total{${buildSelector('mode="idle"', instance)}}[5m])) * 100)`
const queryMem = (instance) => `100 * (1 - (node_memory_MemAvailable_bytes{${buildSelector('', instance)}} / node_memory_MemTotal_bytes{${buildSelector('', instance)}}))`
const queryDisk = (instance) => `max by(instance) (100 - (node_filesystem_free_bytes{${buildSelector('fstype!="tmpfs",mountpoint="/"', instance)}} / node_filesystem_size_bytes{${buildSelector('fstype!="tmpfs",mountpoint="/"', instance)}}) * 100)`
const queryLoad = (instance) => `node_load1{${buildSelector('', instance)}}`
const queryNetIn = (instance) => `sum by(instance) (rate(node_network_receive_bytes_total{${buildSelector('', instance)}}[5m])) / 1024`
const queryNetOut = (instance) => `sum by(instance) (rate(node_network_transmit_bytes_total{${buildSelector('', instance)}}[5m])) / 1024`

const calcStep = (hours) => {
  if (hours <= 1) return 30
  if (hours <= 6) return 60
  return 120
}

const fetchPromRange = async (query, start, end, step) => {
  const res = await axios.get('/api/v1/monitor/prometheus/query_range', {
    headers: authHeaders(),
    params: {
      query: query.replace(/\s+/g, ' ').trim(),
      start,
      end,
      step
    }
  })
  if (res.data?.status && res.data.status !== 'success') {
    throw new Error(res.data?.error || 'Prometheus 查询失败')
  }
  return res.data?.data?.result || []
}

const resolveChartEl = (value) => {
  if (!value) return null
  if (value?.$el) return resolveChartEl(value.$el)
  if (value instanceof HTMLElement) return value
  return null
}

const ensureChartInstance = (instance, holderRef) => {
  const el = resolveChartEl(holderRef.value)
  if (!el || !el.isConnected || el.clientWidth <= 0 || el.clientHeight <= 0) {
    if (instance) {
      try { instance.dispose() } catch {}
    }
    return null
  }
  const existing = echarts.getInstanceByDom(el)
  if (existing && existing !== instance) {
    try { existing.dispose() } catch {}
  }
  if (instance && instance.getDom && instance.getDom() !== el) {
    try { instance.dispose() } catch {}
    instance = null
  }
  if (!instance) {
    try {
      instance = echarts.init(el, null, { renderer: 'svg' })
    } catch {
      return null
    }
  }
  return instance
}

const parseRangeSeries = (results) => {
  if (!Array.isArray(results) || !results.length) return { labels: [], data: [] }
  const points = Array.isArray(results[0]?.values) ? results[0].values : []
  return {
    labels: points.map((item) => new Date(Number(item[0]) * 1000).toLocaleTimeString()),
    data: points.map((item) => Number(item[1] || 0))
  }
}

const renderDetailLineChart = (chart, title, labels, values, unit = '%', color = '#3b82f6') => {
  if (!chart) return null
  const option = {
    color: [color],
    tooltip: {
      trigger: 'axis',
      valueFormatter: (val) => `${Number(val || 0).toFixed(2)} ${unit}`
    },
    grid: { left: 42, right: 16, top: 24, bottom: 24 },
    xAxis: {
      type: 'category',
      data: labels,
      boundaryGap: false,
      axisLabel: { fontSize: 11 }
    },
    yAxis: {
      type: 'value',
      axisLabel: { fontSize: 11 }
    },
    series: [
      {
        name: title,
        type: 'line',
        showSymbol: false,
        smooth: true,
        data: values,
        areaStyle: { opacity: 0.15 }
      }
    ]
  }
  try {
    chart.setOption(option, true)
    chart.resize()
    return chart
  } catch {
    return chart
  }
}

const initDetailCharts = async () => {
  await nextTick()
  detailCpuChart = ensureChartInstance(detailCpuChart, detailCpuChartRef)
  detailMemChart = ensureChartInstance(detailMemChart, detailMemChartRef)
  detailDiskChart = ensureChartInstance(detailDiskChart, detailDiskChartRef)
  detailLoadChart = ensureChartInstance(detailLoadChart, detailLoadChartRef)
  detailNetInChart = ensureChartInstance(detailNetInChart, detailNetInChartRef)
  detailNetOutChart = ensureChartInstance(detailNetOutChart, detailNetOutChartRef)
}

const disposeDetailCharts = () => {
  const charts = [detailCpuChart, detailMemChart, detailDiskChart, detailLoadChart, detailNetInChart, detailNetOutChart]
  charts.forEach((chart) => {
    if (!chart) return
    try { chart.dispose() } catch {}
  })
  detailCpuChart = null
  detailMemChart = null
  detailDiskChart = null
  detailLoadChart = null
  detailNetInChart = null
  detailNetOutChart = null
}

const resolveHostInstance = (row) => {
  if (!row) return ''
  const metric = hostMetric(row)
  const direct = String(metric?.instance || '').trim()
  if (direct) return direct

  const ip = normalizeHostAddress(row?.ip)
  if (!ip) return ''

  const candidates = [ip, `${ip}:9100`, `${ip}:9101`, `${ip}:9102`]
  const matched = Object.values(hostMetricsMap.value).find((item) => {
    const inst = String(item?.instance || item?.ip || '').trim()
    return candidates.includes(inst) || normalizeHostAddress(inst) === ip
  })
  return String(matched?.instance || '').trim() || `${ip}:9100`
}

const clearDetailAutoTimer = () => {
  if (detailAutoTimer) {
    window.clearInterval(detailAutoTimer)
    detailAutoTimer = null
  }
}

const ensureDetailAutoTimer = () => {
  clearDetailAutoTimer()
  if (!detailVisible.value || !detailAutoRefresh.value) return
  detailAutoTimer = window.setInterval(() => {
    if (document.hidden || detailLoading.value || !detailHost.value) return
    fetchDetailMetrics({ silent: true })
  }, 30000)
}

const fetchDetailMetrics = async ({ silent = false } = {}) => {
  if (!detailHost.value) return
  const instance = resolveHostInstance(detailHost.value)
  detailInstanceLabel.value = instance
  if (!instance) return

  detailLoading.value = true
  try {
    await initDetailCharts()
    const end = Math.floor(Date.now() / 1000)
    const start = end - detailRangeHours.value * 3600
    const step = calcStep(detailRangeHours.value)

    const settled = await Promise.allSettled([
      fetchPromRange(queryCpu(instance), start, end, step),
      fetchPromRange(queryMem(instance), start, end, step),
      fetchPromRange(queryDisk(instance), start, end, step),
      fetchPromRange(queryLoad(instance), start, end, step),
      fetchPromRange(queryNetIn(instance), start, end, step),
      fetchPromRange(queryNetOut(instance), start, end, step)
    ])

    const cpuSeries = parseRangeSeries(settled[0].status === 'fulfilled' ? settled[0].value : [])
    const memSeries = parseRangeSeries(settled[1].status === 'fulfilled' ? settled[1].value : [])
    const diskSeries = parseRangeSeries(settled[2].status === 'fulfilled' ? settled[2].value : [])
    const loadSeries = parseRangeSeries(settled[3].status === 'fulfilled' ? settled[3].value : [])
    const netInSeries = parseRangeSeries(settled[4].status === 'fulfilled' ? settled[4].value : [])
    const netOutSeries = parseRangeSeries(settled[5].status === 'fulfilled' ? settled[5].value : [])

    detailCpuChart = renderDetailLineChart(detailCpuChart, 'CPU', cpuSeries.labels, cpuSeries.data, '%', '#3b82f6')
    detailMemChart = renderDetailLineChart(detailMemChart, 'MEM', memSeries.labels, memSeries.data, '%', '#f59e0b')
    detailDiskChart = renderDetailLineChart(detailDiskChart, 'DISK', diskSeries.labels, diskSeries.data, '%', '#ef4444')
    detailLoadChart = renderDetailLineChart(detailLoadChart, 'LOAD', loadSeries.labels, loadSeries.data, '', '#10b981')
    detailNetInChart = renderDetailLineChart(detailNetInChart, 'NET IN', netInSeries.labels, netInSeries.data, 'KB/s', '#6366f1')
    detailNetOutChart = renderDetailLineChart(detailNetOutChart, 'NET OUT', netOutSeries.labels, netOutSeries.data, 'KB/s', '#22c55e')

    if (!silent) {
      const failedCount = settled.filter((item) => item.status === 'rejected').length
      if (failedCount > 0 && failedCount < settled.length) {
        ElMessage.warning(`部分趋势图查询失败（${failedCount}/${settled.length}）`)
      }
      if (failedCount === settled.length) {
        ElMessage.error('趋势图查询失败，请检查 Prometheus 指标')
      }
    }
  } catch (e) {
    if (!silent) ElMessage.error(getErrorMessage(e, '加载主机趋势失败'))
  } finally {
    detailLoading.value = false
  }
}

const openDetail = async (row) => {
  if (!row) return
  detailHost.value = row
  detailVisible.value = true
  detailRangeHours.value = 1
  detailConnForm.port = row.port || 22
  detailConnForm.username = row.username || ''
  detailConnForm.password = ''
  detailAlerts.value = []
  detailDockerHost.value = null
  detailDockerContainers.value = []
  await fetchDetailMetrics()
  fetchDetailAlerts(row)
  fetchDetailDocker(row)
  ensureDetailAutoTimer()
}

const fmtTimeAgo = (val) => {
  if (!val) return ''
  const diff = Math.floor((Date.now() - new Date(val).getTime()) / 1000)
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)}m前`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h前`
  return `${Math.floor(diff / 86400)}d前`
}

const fetchDetailAlerts = async (host) => {
  try {
    const res = await axios.get('/api/v1/alert/alerts', { headers: authHeaders() })
    const allAlerts = res.data?.data || []
    const ip = (host.ip || '').trim()
    const name = (host.name || '').trim()
    detailAlerts.value = allAlerts.filter(a =>
      a.status === 'firing' && ((a.target || '').includes(ip) || (a.target || '').includes(name))
    ).slice(0, 10)
  } catch (e) { /* silent */ }
}

const fetchDetailDocker = async (host) => {
  try {
    const res = await axios.get('/api/v1/docker/hosts', { headers: authHeaders() })
    const dockerHosts = res.data?.data || []
    const ip = (host.ip || '').trim()
    const matched = dockerHosts.find(d => (d.ip || '').trim() === ip)
    if (matched) {
      detailDockerHost.value = matched
      detailDockerContainers.value = matched.containers || matched.container_list || []
    }
  } catch (e) { /* silent */ }
}

const saveDetailConn = async () => {
  if (!detailHost.value?.id) return
  detailConnSaving.value = true
  try {
    const payload = {
      port: detailConnForm.port,
      username: detailConnForm.username
    }
    if (detailConnForm.password) payload.password = detailConnForm.password
    await axios.put(`/api/v1/cmdb/hosts/${detailHost.value.id}`, payload, { headers: authHeaders() })
    ElMessage.success('连接信息已更新')
    detailHost.value.port = detailConnForm.port
    detailHost.value.username = detailConnForm.username
  } catch (err) {
    ElMessage.error('保存失败: ' + (err.response?.data?.message || err.message))
  } finally {
    detailConnSaving.value = false
  }
}

const onResize = () => {
  const charts = [detailCpuChart, detailMemChart, detailDiskChart, detailLoadChart, detailNetInChart, detailNetOutChart]
  charts.forEach((chart) => {
    if (!chart) return
    try { chart.resize() } catch {}
  })
}

watch([inspectVisible, inspectAutoRefresh], ensureInspectAutoTimer)
watch([detailVisible, detailAutoRefresh], ensureDetailAutoTimer)
watch(detailVisible, (visible) => {
  if (!visible) {
    clearDetailAutoTimer()
    detailHost.value = null
    detailInstanceLabel.value = ''
  }
})

onMounted(async () => {
  const cached = localStorage.getItem('lazy_hosts_columns')
  if (cached) {
    try {
      showColumns.value = { ...showColumns.value, ...JSON.parse(cached) }
    } catch (e) {}
  }

  await fetchGroups()
  await fetchGlobalHostsSummary()
  await syncStatuses(true)
  nextTick(() => {
    groupTreeRef.value?.setCurrentKey('all')
  })

  autoSyncTimer = window.setInterval(() => {
    if (document.hidden || syncingStatus.value || loading.value) return
    syncStatuses(true)
  }, 60 * 1000)

  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  if (autoSyncTimer) {
    window.clearInterval(autoSyncTimer)
    autoSyncTimer = null
  }
  clearInspectAutoTimer()
  clearDetailAutoTimer()
  disposeDetailCharts()
  window.removeEventListener('resize', onResize)
})

const editingHostId = ref(null)
const editingDescription = ref('')
const descInputRefs = ref({})

const startEditDescription = (row) => {
  editingHostId.value = row.id
  editingDescription.value = row.description || ''
  nextTick(() => {
    const inputEl = descInputRefs.value[row.id]
    if (inputEl) {
      inputEl.focus()
    }
  })
}

const saveDescription = async (row) => {
  if (editingHostId.value !== row.id) return
  const oldVal = row.description || ''
  const newVal = editingDescription.value.trim()
  editingHostId.value = null
  
  if (oldVal === newVal) return
  
  try {
    const h = { Authorization: 'Bearer ' + localStorage.getItem('token') }
    await axios.put(`/api/v1/cmdb/hosts/${row.id}`, { description: newVal }, { headers: h })
    row.description = newVal
    ElMessage.success('备注更新成功')
  } catch (error) {
    ElMessage.error(error?.response?.data?.message || error?.message || '更新备注失败')
  }
}
</script>

<style scoped>
.flex { display: flex; }
.justify-between { justify-content: space-between; }
.items-center { align-items: center; }
.gap-2 { gap: 8px; }
.font-bold { font-weight: 600; }
.mb-4 { margin-bottom: 16px; }
.w-64 { width: 256px; }
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.host-page-card { min-height: 400px; }
.header-wrap { flex-wrap: wrap; gap: 10px; }
.header-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }

.host-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 14px;
}

.host-aside {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.group-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 600;
}

.group-card-header-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.group-search { margin-bottom: 10px; }

.group-tree-node {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding-right: 8px;
}

.group-tree-label {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-total {
  font-weight: 600;
  color: var(--el-text-color-secondary);
}

.provider-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.provider-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  background: var(--el-fill-color-extra-light);
  border-radius: 8px;
}

.provider-name {
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.host-main { min-width: 0; }
.filters-row { flex-wrap: wrap; }

.table-scroll {
  padding-bottom: 2px;
}

.op-row {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  white-space: nowrap !important;
  flex-wrap: nowrap !important;
}

.op-row :deep(.el-button) {
  margin-left: 0 !important;
  padding: 5px 8px;
  height: 28px;
  font-size: 12px;
  flex-shrink: 0;
  white-space: nowrap !important;
}

:deep(.fixed-op-col) {
  background-color: var(--el-bg-color, #ffffff) !important;
}
:deep(.el-table__row:hover .fixed-op-col) {
  background-color: var(--el-table-row-hover-bg-color, #f5f7fa) !important;
}

.group-manage-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-bottom: 12px;
}

.detail-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 14px;
}

.detail-host-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.detail-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.detail-conn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  margin-bottom: 8px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  font-size: 13px;
}
.conn-label { font-weight: 700; color: var(--el-text-color-secondary); margin-right: 4px; }

.detail-cross-ref { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 12px; }
.cross-section { background: var(--el-fill-color-lighter); border-radius: 8px; padding: 10px 12px; }
.cross-title { font-size: 12px; font-weight: 700; color: var(--el-text-color-secondary); display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.cross-count { background: var(--el-color-primary); color: #fff; font-size: 10px; padding: 1px 6px; border-radius: 8px; }
.cross-empty { font-size: 12px; color: var(--el-text-color-placeholder); padding: 8px 0; }
.cross-item { display: flex; align-items: center; gap: 6px; padding: 4px 0; font-size: 12px; cursor: pointer; border-bottom: 1px solid var(--el-border-color-lighter); }
.cross-item:last-child { border-bottom: none; }
.cross-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.cross-time { font-size: 11px; color: var(--el-text-color-placeholder); margin-left: auto; }

.detail-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.metric-card {
  background: var(--el-fill-color-extra-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.metric-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.detail-chart-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.chart-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  padding: 10px;
  background: #fff;
}

.chart-title {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 6px;
}

.detail-chart {
  height: 200px;
  width: 100%;
}

.import-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}

.test-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.test-title { font-weight: 600; }

.test-pre {
  background: #0f172a;
  color: #e2e8f0;
  padding: 12px;
  border-radius: 6px;
  overflow: auto;
  max-height: 200px;
}

.helper-row {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}

.inspect-toolbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.inspect-host {
  display: flex;
  gap: 12px;
  align-items: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  flex-wrap: wrap;
}

.inspect-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.inspect-process-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.inspect-card :deep(.el-card__header) {
  padding: 10px 14px;
  font-weight: 600;
}

.inspect-card :deep(.el-card__body) {
  padding: 10px 12px;
}

.mini-percent {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 3px;
}

.inspect-tcp-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tcp-summary {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

:deep(.danger-row td) {
  background: rgba(245, 108, 108, 0.08);
}

:deep(.warn-row td) {
  background: rgba(230, 162, 60, 0.08);
}

@media (max-width: 1440px) {
  .host-layout { grid-template-columns: 250px minmax(0, 1fr); }
}

@media (max-width: 1280px) {
  .detail-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 1100px) {
  .host-layout { grid-template-columns: 1fr; }
  .detail-chart-grid { grid-template-columns: 1fr; }
  .inspect-process-grid { grid-template-columns: 1fr; }
}

/* 批量命令执行样式 */
.batch-exec-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.exec-targets-bar {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  background: #f8fafc;
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
}
.exec-targets-bar .label {
  font-size: 13px;
  font-weight: 600;
  color: #475569;
  white-space: nowrap;
  margin-top: 2px;
}
.host-tags-scroll {
  display: flex;
  flex-wrap: wrap;
  max-height: 80px;
  overflow-y: auto;
}
.preset-commands-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.preset-label {
  font-size: 12.5px;
  color: #64748b;
  font-weight: 600;
}
.preset-btn {
  background: #f1f5f9;
  border-color: #cbd5e1;
  font-size: 12px;
}
.preset-btn:hover {
  background: #e2e8f0;
  color: #0284c7;
}
.command-textarea :deep(textarea) {
  font-family: Consolas, monospace;
  font-size: 13px;
  background: #0f172a;
  color: #38bdf8;
  border-radius: 6px;
}
.danger-warning-alert {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.force-confirm-check {
  padding-left: 4px;
}
.text-danger {
  color: #ef4444;
}
.exec-results-section {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  padding: 10px;
}
.results-summary {
  display: flex;
  align-items: center;
  font-size: 13px;
  margin-bottom: 8px;
}
.collapse-title-row {
  display: flex;
  align-items: center;
  width: 100%;
  padding-right: 12px;
}
.dur-tag {
  font-size: 12px;
  color: #64748b;
}
.terminal-output-box {
  background: #0d1117;
  padding: 10px 14px;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.out-label {
  font-size: 11.5px;
  font-weight: 600;
  color: #94a3b8;
  margin-bottom: 2px;
}
.term-pre {
  margin: 0;
  font-family: Consolas, monospace;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 220px;
  overflow-y: auto;
}
.stdout-text {
  color: #38bdf8;
}
.stderr-text {
  color: #f87171;
}
</style>
