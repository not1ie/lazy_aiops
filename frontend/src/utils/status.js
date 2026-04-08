const normalizeText = (value) => String(value ?? '').trim().toLowerCase()

const cleanText = (value) => {
  const text = String(value ?? '').trim()
  return text || ''
}

const pickText = (...values) => {
  for (const value of values) {
    const text = cleanText(value)
    if (text) return text
  }
  return ''
}

const inferSource = (row, fallback = '') => {
  return pickText(
    row?.status_source,
    row?.source,
    row?.provider,
    row?.origin,
    row?.from,
    fallback
  )
}

const withStatusMeta = (base, extra = {}) => {
  return {
    ...base,
    reason: pickText(extra.reason, base.reason),
    source: pickText(extra.source, base.source),
    checkAt: extra.checkAt ?? base.checkAt ?? '',
    isStale: Boolean(extra.isStale ?? base.isStale),
    staleText: pickText(extra.staleText, base.staleText)
  }
}

export const toTimestamp = (value) => {
  if (!value) return null
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? null : ts
}

export const isStaleByMinutes = (value, minutes = 3, nowMs = Date.now()) => {
  const ts = toTimestamp(value)
  if (!ts) return true
  return nowMs - ts > minutes * 60 * 1000
}

export const cmdbHostStatusMeta = (row, options = {}) => {
  const staleMinutes = Number(options?.staleMinutes || 3)
  const status = Number(row?.status)
  const source = inferSource(row, 'CMDB巡检')
  const checkAt = row?.last_check_at || row?.updated_at || ''
  if (status === 2) {
    return withStatusMeta(
      { key: 'maintenance', text: '维护', type: 'warning' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '主机处于维护状态'
      }
    )
  }
  if (status === 1) {
    const stale = isStaleByMinutes(checkAt, staleMinutes, options?.nowMs)
    if (stale) {
      return withStatusMeta(
        { key: 'stale', text: '状态过期', type: 'warning' },
        {
          source,
          checkAt,
          reason: row?.status_reason || `超过 ${staleMinutes} 分钟未更新主机状态`,
          isStale: true,
          staleText: `状态已超过 ${staleMinutes} 分钟未更新`
        }
      )
    }
    return withStatusMeta(
      { key: 'online', text: '在线', type: 'success' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '主机连接正常'
      }
    )
  }
  if (status === 0) {
    return withStatusMeta(
      { key: 'offline', text: '离线', type: 'danger' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '主机不可达或探测失败'
      }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    {
      source,
      checkAt,
      reason: row?.status_reason || '未获取到主机状态'
    }
  )
}

export const dockerHostStatusMeta = (row, options = {}) => {
  const staleMinutes = Number(options?.staleMinutes || 3)
  const status = normalizeText(row?.status)
  const source = inferSource(row, 'Docker巡检')
  const checkAt = row?.last_check_at || row?.updated_at || ''
  if (status === 'maintenance') {
    return withStatusMeta(
      { key: 'maintenance', text: '维护', type: 'warning' },
      {
        source,
        checkAt,
        reason: row?.last_error || row?.status_reason || '环境处于维护状态'
      }
    )
  }
  if (status === 'online') {
    const stale = isStaleByMinutes(checkAt, staleMinutes, options?.nowMs)
    if (stale) {
      return withStatusMeta(
        { key: 'stale', text: '状态过期', type: 'warning' },
        {
          source,
          checkAt,
          reason: row?.last_error || row?.status_reason || `超过 ${staleMinutes} 分钟未更新 Docker 状态`,
          isStale: true,
          staleText: `状态已超过 ${staleMinutes} 分钟未更新`
        }
      )
    }
    return withStatusMeta(
      { key: 'online', text: '在线', type: 'success' },
      {
        source,
        checkAt,
        reason: row?.status_reason || 'Docker API 连接正常'
      }
    )
  }
  if (status === 'offline') {
    return withStatusMeta(
      { key: 'offline', text: '离线', type: 'danger' },
      {
        source,
        checkAt,
        reason: row?.last_error || row?.status_reason || 'Docker 引擎不可达'
      }
    )
  }
  if (status === 'error') {
    return withStatusMeta(
      { key: 'error', text: '异常', type: 'danger' },
      {
        source,
        checkAt,
        reason: row?.last_error || row?.status_reason || 'Docker 状态采集异常'
      }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    {
      source,
      checkAt,
      reason: row?.status_reason || '未获取到 Docker 环境状态'
    }
  )
}

export const k8sClusterStatusMeta = (row, options = {}) => {
  const staleMinutes = Number(options?.staleMinutes || 5)
  const status = Number(row?.status)
  const source = inferSource(row, 'K8s巡检')
  const checkAt = row?.last_check_at || row?.updated_at || ''
  if (status === 2) {
    return withStatusMeta(
      { key: 'maintenance', text: '维护', type: 'warning' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '集群处于维护状态'
      }
    )
  }
  if (status === 1) {
    const stale = isStaleByMinutes(checkAt, staleMinutes, options?.nowMs)
    if (stale) {
      return withStatusMeta(
        { key: 'stale', text: '状态过期', type: 'warning' },
        {
          source,
          checkAt,
          reason: row?.status_reason || `超过 ${staleMinutes} 分钟未更新集群状态`,
          isStale: true,
          staleText: `状态已超过 ${staleMinutes} 分钟未更新`
        }
      )
    }
    return withStatusMeta(
      { key: 'online', text: '在线', type: 'success' },
      {
        source,
        checkAt,
        reason: row?.status_reason || 'K8s API 探测正常'
      }
    )
  }
  if (status === 0) {
    return withStatusMeta(
      { key: 'abnormal', text: '异常', type: 'danger' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '集群状态异常或不可达'
      }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    {
      source,
      checkAt,
      reason: row?.status_reason || '未获取到集群状态'
    }
  )
}

export const jumpSessionStatusMeta = (row, options = {}) => {
  const pendingTimeoutMinutes = Number(options?.pendingTimeoutMinutes || 15)
  const activeLongMinutes = Number(options?.activeLongMinutes || 180)
  const nowMs = Number(options?.nowMs || Date.now())
  const status = normalizeText(row?.status)
  const source = inferSource(row, 'JumpServer会话')
  const checkAt = row?.updated_at || row?.ended_at || row?.started_at || row?.created_at || ''

  if (status === 'pending_approval') {
    const stale = isStaleByMinutes(row?.created_at || row?.started_at, pendingTimeoutMinutes, nowMs)
    if (stale) {
      return withStatusMeta(
        { key: 'pending_timeout', text: '审批超时', type: 'warning' },
        {
          source,
          checkAt,
          reason: row?.status_reason || `审批等待超过 ${pendingTimeoutMinutes} 分钟`,
          isStale: true,
          staleText: `审批等待超过 ${pendingTimeoutMinutes} 分钟`
        }
      )
    }
    return withStatusMeta(
      { key: 'pending', text: '待审批', type: 'warning' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '会话正在等待审批'
      }
    )
  }

  if (status === 'active') {
    const longRunning = isStaleByMinutes(row?.started_at || row?.created_at, activeLongMinutes, nowMs)
    if (longRunning) {
      return withStatusMeta(
        { key: 'active_long', text: '活跃(长时)', type: 'warning' },
        {
          source,
          checkAt,
          reason: row?.status_reason || `会话持续超过 ${activeLongMinutes} 分钟`,
          isStale: true,
          staleText: `会话持续超过 ${activeLongMinutes} 分钟`
        }
      )
    }
    return withStatusMeta(
      { key: 'active', text: '活跃', type: 'success' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '会话正在进行'
      }
    )
  }

  if (status === 'blocked') {
    return withStatusMeta(
      { key: 'blocked', text: '已阻断', type: 'danger' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '会话已被阻断'
      }
    )
  }
  if (status === 'rejected') {
    return withStatusMeta(
      { key: 'rejected', text: '已拒绝', type: 'danger' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '审批被拒绝'
      }
    )
  }
  if (status === 'closed' || status === 'finished' || status === 'done') {
    return withStatusMeta(
      { key: 'closed', text: '已关闭', type: 'info' },
      {
        source,
        checkAt,
        reason: row?.close_reason || row?.status_reason || '会话已结束'
      }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    {
      source,
      checkAt,
      reason: row?.status_reason || '会话状态未知'
    }
  )
}

export const monitorAgentStatusMeta = (row, options = {}) => {
  const staleMinutes = Number(options?.staleMinutes || 3)
  const status = normalizeText(row?.status)
  const heartbeatAt = row?.last_seen || row?.last_heartbeat || row?.updated_at
  const source = inferSource(row, 'Agent心跳')

  if (status === 'online' || status === 'connected' || status === 'running' || status === 'up') {
    if (isStaleByMinutes(heartbeatAt, staleMinutes, options?.nowMs)) {
      return withStatusMeta(
        { key: 'stale', text: '状态过期', type: 'warning' },
        {
          source,
          checkAt: heartbeatAt,
          reason: row?.status_reason || `超过 ${staleMinutes} 分钟未收到心跳`,
          isStale: true,
          staleText: `心跳超过 ${staleMinutes} 分钟未更新`
        }
      )
    }
    return withStatusMeta(
      { key: 'online', text: '在线', type: 'success' },
      {
        source,
        checkAt: heartbeatAt,
        reason: row?.status_reason || '心跳正常'
      }
    )
  }
  if (status === 'offline' || status === 'down' || status === 'disconnected' || status === 'error') {
    return withStatusMeta(
      { key: 'offline', text: '离线', type: 'danger' },
      {
        source,
        checkAt: heartbeatAt,
        reason: row?.status_reason || 'Agent 不可达'
      }
    )
  }
  if (status === 'maintenance') {
    return withStatusMeta(
      { key: 'maintenance', text: '维护', type: 'warning' },
      {
        source,
        checkAt: heartbeatAt,
        reason: row?.status_reason || 'Agent 处于维护状态'
      }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    {
      source,
      checkAt: heartbeatAt,
      reason: row?.status_reason || '未获取到 Agent 状态'
    }
  )
}

export const monitorAlertStatusMeta = (statusOrRow) => {
  const row = statusOrRow && typeof statusOrRow === 'object' ? statusOrRow : null
  const status = row ? row.status : statusOrRow
  const normalized = normalizeText(status)
  const numeric = Number(status)

  const source = inferSource(row, '告警中心')
  const checkAt = row?.updated_at || row?.fired_at || row?.created_at || ''

  if (
    numeric === 0 ||
    normalized === 'open' ||
    normalized === 'firing' ||
    normalized === 'pending' ||
    normalized === 'new'
  ) {
    return withStatusMeta(
      { key: 'open', text: '未处理', type: 'danger' },
      {
        source,
        checkAt,
        reason: row?.message || '告警处于未处理状态'
      }
    )
  }
  if (
    numeric === 1 ||
    normalized === 'closed' ||
    normalized === 'resolved' ||
    normalized === 'ack' ||
    normalized === 'handled' ||
    normalized === 'done'
  ) {
    return withStatusMeta(
      { key: 'closed', text: '已处理', type: 'success' },
      {
        source,
        checkAt,
        reason: row?.message || '告警已处理'
      }
    )
  }
  if (
    numeric === 2 ||
    normalized === 'ignored' ||
    normalized === 'mute' ||
    normalized === 'silenced'
  ) {
    return withStatusMeta(
      { key: 'ignored', text: '已忽略', type: 'info' },
      {
        source,
        checkAt,
        reason: row?.message || '告警已忽略'
      }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    {
      source,
      checkAt,
      reason: row?.message || '告警状态未知'
    }
  )
}

export const jumpIntegrationSyncStatusMeta = (status, options = {}) => {
  const enabled = options?.enabled !== false
  const staleMinutes = Number(options?.staleMinutes || 30)
  const lastSyncAt = options?.lastSyncAt
  const normalized = normalizeText(status)
  const source = pickText(options?.source, 'JumpServer同步')

  if (!enabled) {
    return withStatusMeta(
      { key: 'disabled', text: '未启用', type: 'info' },
      {
        source,
        checkAt: lastSyncAt,
        reason: '未启用自动同步'
      }
    )
  }
  if (normalized === 'failed') {
    return withStatusMeta(
      { key: 'failed', text: '同步失败', type: 'danger' },
      {
        source,
        checkAt: lastSyncAt,
        reason: pickText(options?.reason, options?.lastError, '最近一次同步失败')
      }
    )
  }
  if (normalized === 'partial' || normalized === 'warning') {
    return withStatusMeta(
      { key: 'partial', text: '部分成功', type: 'warning' },
      {
        source,
        checkAt: lastSyncAt,
        reason: pickText(options?.reason, options?.lastError, '部分资产同步成功')
      }
    )
  }
  if (normalized === 'ok' || normalized === 'success') {
    if (isStaleByMinutes(lastSyncAt, staleMinutes, options?.nowMs)) {
      return withStatusMeta(
        { key: 'stale', text: '状态过期', type: 'warning' },
        {
          source,
          checkAt: lastSyncAt,
          reason: pickText(options?.reason, `超过 ${staleMinutes} 分钟未完成同步`),
          isStale: true,
          staleText: `同步状态超过 ${staleMinutes} 分钟未更新`
        }
      )
    }
    return withStatusMeta(
      { key: 'ok', text: '同步成功', type: 'success' },
      {
        source,
        checkAt: lastSyncAt,
        reason: pickText(options?.reason, '最近一次同步成功')
      }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    {
      source,
      checkAt: lastSyncAt,
      reason: pickText(options?.reason, '尚未获取同步状态')
    }
  )
}

export const databaseAssetStatusMeta = (row, options = {}) => {
  const staleDays = Number(options?.staleDays || 30)
  const status = Number(row?.status)
  const nowMs = Number(options?.nowMs || Date.now())
  const checkAt = row?.last_check_at || row?.updated_at
  const stale = isStaleByMinutes(checkAt, staleDays * 24 * 60, nowMs)
  const hasEndpoint = String(row?.host || '').trim() !== '' && Number(row?.port || 0) > 0
  const source = inferSource(row, '数据库资产巡检')

  if (status === 0) {
    return withStatusMeta(
      { key: 'disabled', text: '禁用', type: 'info' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '数据库资产已禁用'
      }
    )
  }
  if (!hasEndpoint) {
    return withStatusMeta(
      { key: 'error', text: '配置异常', type: 'danger' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '缺少有效的主机或端口配置'
      }
    )
  }
  if (status === 1) {
    if (stale) {
      return withStatusMeta(
        { key: 'stale', text: '状态过期', type: 'warning' },
        {
          source,
          checkAt,
          reason: row?.status_reason || `超过 ${staleDays} 天未更新数据库状态`,
          isStale: true,
          staleText: `状态超过 ${staleDays} 天未更新`
        }
      )
    }
    return withStatusMeta(
      { key: 'online', text: '可用', type: 'success' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '数据库连接可用'
      }
    )
  }
  if (stale) {
    return withStatusMeta(
      { key: 'stale', text: '状态过期', type: 'warning' },
      {
        source,
        checkAt,
        reason: row?.status_reason || `超过 ${staleDays} 天未更新数据库状态`,
        isStale: true,
        staleText: `状态超过 ${staleDays} 天未更新`
      }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    {
      source,
      checkAt,
      reason: row?.status_reason || '数据库状态未知'
    }
  )
}

export const cloudResourceStatusMeta = (row, options = {}) => {
  const staleDays = Number(options?.staleDays || 7)
  const nowMs = Number(options?.nowMs || Date.now())
  const status = normalizeText(row?.status)
  const accountStatus = Number(row?.account?.status)
  const checkAt = row?.last_check_at || row?.updated_at
  const stale = isStaleByMinutes(checkAt, staleDays * 24 * 60, nowMs)
  const source = inferSource(row, pickText(row?.vendor, row?.provider_name, '云资源巡检'))

  if (accountStatus === 0) {
    return withStatusMeta(
      { key: 'account_disabled', text: '账号禁用', type: 'warning' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '云账号已禁用'
      }
    )
  }
  if (status === 'running' || status === 'online' || status === 'active' || status === 'available' || status === 'normal') {
    if (stale) {
      return withStatusMeta(
        { key: 'stale', text: '状态过期', type: 'warning' },
        {
          source,
          checkAt,
          reason: row?.status_reason || `超过 ${staleDays} 天未更新云资源状态`,
          isStale: true,
          staleText: `状态超过 ${staleDays} 天未更新`
        }
      )
    }
    return withStatusMeta(
      { key: 'online', text: '在线', type: 'success' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '云资源运行正常'
      }
    )
  }
  if (status === 'stopped' || status === 'offline' || status === 'inactive' || status === 'disabled') {
    return withStatusMeta(
      { key: 'offline', text: '离线', type: 'danger' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '云资源已停止或离线'
      }
    )
  }
  if (status === 'error' || status === 'failed' || status === 'abnormal') {
    return withStatusMeta(
      { key: 'error', text: '异常', type: 'danger' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '云资源状态异常'
      }
    )
  }
  if (status === 'pending' || status === 'creating' || status === 'updating' || status === 'starting') {
    return withStatusMeta(
      { key: 'pending', text: '初始化中', type: 'warning' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '云资源正在变更'
      }
    )
  }
  if (!status) {
    if (stale) {
      return withStatusMeta(
        { key: 'stale', text: '状态过期', type: 'warning' },
        {
          source,
          checkAt,
          reason: row?.status_reason || `超过 ${staleDays} 天未更新云资源状态`,
          isStale: true,
          staleText: `状态超过 ${staleDays} 天未更新`
        }
      )
    }
    return withStatusMeta(
      { key: 'unknown', text: '未知', type: 'info' },
      {
        source,
        checkAt,
        reason: row?.status_reason || '未获取到云资源状态'
      }
    )
  }
  if (stale) {
    return withStatusMeta(
      { key: 'stale', text: '状态过期', type: 'warning' },
      {
        source,
        checkAt,
        reason: row?.status_reason || `超过 ${staleDays} 天未更新云资源状态`,
        isStale: true,
        staleText: `状态超过 ${staleDays} 天未更新`
      }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    {
      source,
      checkAt,
      reason: row?.status_reason || '云资源状态未知'
    }
  )
}

const parseBooleanLike = (value, fallback = false) => {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value === 1
  const normalized = normalizeText(value)
  if (!normalized) return fallback
  if (['1', 'true', 'enabled', 'enable', 'active', 'on', 'yes', 'ok', 'normal', 'success', 'up', 'running'].includes(normalized)) return true
  if (['0', 'false', 'disabled', 'disable', 'inactive', 'off', 'no', 'error', 'failed', 'down', 'stopped'].includes(normalized)) return false
  return fallback
}

export const booleanEnabledStatusMeta = (statusOrRow, options = {}) => {
  const row = statusOrRow && typeof statusOrRow === 'object' ? statusOrRow : null
  const raw = row ? (row.enabled ?? row.status ?? row.state) : statusOrRow
  const enabled = parseBooleanLike(raw, Boolean(raw))
  const source = inferSource(row, options?.source || '状态中心')
  const checkAt = row?.last_check_at || row?.updated_at || row?.created_at || options?.checkAt || ''
  const enabledText = pickText(options?.enabledText, '启用')
  const disabledText = pickText(options?.disabledText, '停用')
  const enabledType = pickText(options?.enabledType, 'success')
  const disabledType = pickText(options?.disabledType, 'info')
  const enabledReason = pickText(options?.enabledReason, row?.status_reason, '当前状态已启用')
  const disabledReason = pickText(options?.disabledReason, row?.status_reason, '当前状态已停用')
  return withStatusMeta(
    {
      key: enabled ? 'enabled' : 'disabled',
      text: enabled ? enabledText : disabledText,
      type: enabled ? enabledType : disabledType
    },
    {
      source,
      checkAt,
      reason: enabled ? enabledReason : disabledReason
    }
  )
}

export const serviceHealthStatusMeta = (statusOrRow, options = {}) => {
  const row = statusOrRow && typeof statusOrRow === 'object' ? statusOrRow : null
  const raw = row ? (row.status ?? row.state ?? row.health) : statusOrRow
  const normalized = normalizeText(raw)
  const numeric = Number(raw)
  const source = inferSource(row, options?.source || '服务巡检')
  const checkAt = row?.last_check_at || row?.updated_at || row?.checked_at || options?.checkAt || ''
  const healthyText = pickText(options?.healthyText, '正常')
  const unhealthyText = pickText(options?.unhealthyText, '异常')
  const unknownText = pickText(options?.unknownText, '未知')

  const healthy = (
    numeric === 1 ||
    ['1', 'ok', 'healthy', 'normal', 'online', 'success', 'up', 'running', 'available', 'connected'].includes(normalized) ||
    parseBooleanLike(raw, false) === true
  )
  const unhealthy = (
    numeric === 0 ||
    ['0', 'error', 'failed', 'abnormal', 'offline', 'down', 'unhealthy', 'unavailable', 'disconnected'].includes(normalized) ||
    (parseBooleanLike(raw, true) === false && normalized !== '')
  )

  if (healthy) {
    return withStatusMeta(
      { key: 'healthy', text: healthyText, type: 'success' },
      {
        source,
        checkAt,
        reason: pickText(options?.healthyReason, row?.status_reason, `${healthyText}状态`)
      }
    )
  }
  if (unhealthy) {
    return withStatusMeta(
      { key: 'unhealthy', text: unhealthyText, type: 'danger' },
      {
        source,
        checkAt,
        reason: pickText(options?.unhealthyReason, row?.status_reason, `${unhealthyText}状态`)
      }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: unknownText, type: 'info' },
    {
      source,
      checkAt,
      reason: pickText(options?.unknownReason, row?.status_reason, '状态未知')
    }
  )
}

export const workflowEventStatusMeta = (statusOrRow, options = {}) => {
  const row = statusOrRow && typeof statusOrRow === 'object' ? statusOrRow : null
  const status = row ? row.status : statusOrRow
  const normalized = normalizeText(status)
  const source = inferSource(row, options?.source || '编排事件')
  const checkAt = row?.updated_at || row?.received_at || row?.created_at || options?.checkAt || ''

  if (normalized === 'dispatched') {
    return withStatusMeta(
      { key: 'dispatched', text: '已分发', type: 'success' },
      { source, checkAt, reason: pickText(row?.status_reason, '事件分发成功') }
    )
  }
  if (normalized === 'partial') {
    return withStatusMeta(
      { key: 'partial', text: '部分成功', type: 'warning' },
      { source, checkAt, reason: pickText(row?.status_reason, '事件仅部分分发成功') }
    )
  }
  if (normalized === 'failed') {
    return withStatusMeta(
      { key: 'failed', text: '分发失败', type: 'danger' },
      { source, checkAt, reason: pickText(row?.status_reason, row?.error, '事件分发失败') }
    )
  }
  if (normalized === 'ignored') {
    return withStatusMeta(
      { key: 'ignored', text: '已忽略', type: 'info' },
      { source, checkAt, reason: pickText(row?.status_reason, '事件被规则忽略') }
    )
  }
  if (normalized === 'received') {
    return withStatusMeta(
      { key: 'received', text: '已接入', type: 'primary' },
      { source, checkAt, reason: pickText(row?.status_reason, '事件已接入，等待规则分发') }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: pickText(status, '未知'), type: 'info' },
    { source, checkAt, reason: pickText(row?.status_reason, '事件状态未知') }
  )
}

export const workflowDispatchStatusMeta = (statusOrRow, options = {}) => {
  const row = statusOrRow && typeof statusOrRow === 'object' ? statusOrRow : null
  const status = row ? row.status : statusOrRow
  const normalized = normalizeText(status)
  const source = inferSource(row, options?.source || '编排分发')
  const checkAt = row?.finished_at || row?.updated_at || row?.started_at || options?.checkAt || ''

  if (normalized === 'success') {
    return withStatusMeta(
      { key: 'success', text: '成功', type: 'success' },
      { source, checkAt, reason: pickText(row?.status_reason, '分发执行成功') }
    )
  }
  if (normalized === 'failed') {
    return withStatusMeta(
      { key: 'failed', text: '失败', type: 'danger' },
      { source, checkAt, reason: pickText(row?.error, row?.status_reason, '分发执行失败') }
    )
  }
  if (normalized === 'skipped') {
    return withStatusMeta(
      { key: 'skipped', text: '已跳过', type: 'warning' },
      { source, checkAt, reason: pickText(row?.status_reason, '分发被跳过') }
    )
  }
  if (normalized === 'running' || normalized === 'pending') {
    return withStatusMeta(
      { key: 'running', text: '执行中', type: 'primary' },
      { source, checkAt, reason: pickText(row?.status_reason, '分发任务执行中') }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: pickText(status, '未知'), type: 'info' },
    { source, checkAt, reason: pickText(row?.status_reason, '分发状态未知') }
  )
}

export const workflowExecutionStatusMeta = (statusOrRow, options = {}) => {
  const row = statusOrRow && typeof statusOrRow === 'object' ? statusOrRow : null
  const status = row ? row.status : statusOrRow
  const normalized = normalizeText(status)
  const code = Number(status)
  const source = inferSource(row, options?.source || '工作流执行')
  const checkAt = row?.finished_at || row?.updated_at || row?.started_at || row?.created_at || options?.checkAt || ''

  if (code === 0 || normalized === 'running') {
    return withStatusMeta(
      { key: 'running', text: '运行中', type: 'primary' },
      { source, checkAt, reason: pickText(row?.status_reason, '工作流正在运行') }
    )
  }
  if (code === 1 || normalized === 'success') {
    return withStatusMeta(
      { key: 'success', text: '成功', type: 'success' },
      { source, checkAt, reason: pickText(row?.status_reason, '工作流执行成功') }
    )
  }
  if (code === 2 || normalized === 'failed') {
    return withStatusMeta(
      { key: 'failed', text: '失败', type: 'danger' },
      { source, checkAt, reason: pickText(row?.error, row?.status_reason, '工作流执行失败') }
    )
  }
  if (code === 3 || normalized === 'canceled' || normalized === 'cancelled') {
    return withStatusMeta(
      { key: 'canceled', text: '取消', type: 'warning' },
      { source, checkAt, reason: pickText(row?.status_reason, '工作流已取消') }
    )
  }
  if (code === 4 || normalized === 'pending_approval' || normalized === 'approval') {
    return withStatusMeta(
      { key: 'pending_approval', text: '待审批', type: 'warning' },
      { source, checkAt, reason: pickText(row?.status_reason, '工作流等待人工审批') }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    { source, checkAt, reason: pickText(row?.status_reason, '工作流状态未知') }
  )
}

export const sqlauditOrderStatusMeta = (statusOrRow, options = {}) => {
  const row = statusOrRow && typeof statusOrRow === 'object' ? statusOrRow : null
  const status = row ? row.status : statusOrRow
  const code = Number(status)
  const source = inferSource(row, options?.source || 'SQL工单')
  const checkAt = row?.updated_at || row?.executed_at || row?.created_at || options?.checkAt || ''

  if (code === 0) return withStatusMeta({ key: 'pending', text: '待审核', type: 'warning' }, { source, checkAt, reason: pickText(row?.status_reason, '工单等待审核') })
  if (code === 1) return withStatusMeta({ key: 'approved', text: '审核通过', type: 'success' }, { source, checkAt, reason: pickText(row?.status_reason, '工单审核已通过') })
  if (code === 2) return withStatusMeta({ key: 'rejected', text: '审核拒绝', type: 'danger' }, { source, checkAt, reason: pickText(row?.status_reason, '工单审核被拒绝') })
  if (code === 3) return withStatusMeta({ key: 'running', text: '执行中', type: 'primary' }, { source, checkAt, reason: pickText(row?.status_reason, '工单正在执行') })
  if (code === 4) return withStatusMeta({ key: 'success', text: '执行成功', type: 'success' }, { source, checkAt, reason: pickText(row?.status_reason, '工单执行成功') })
  if (code === 5) return withStatusMeta({ key: 'failed', text: '执行失败', type: 'danger' }, { source, checkAt, reason: pickText(row?.status_reason, row?.error, '工单执行失败') })
  if (code === 6) return withStatusMeta({ key: 'rollback', text: '已回滚', type: 'info' }, { source, checkAt, reason: pickText(row?.status_reason, '工单已回滚') })
  return withStatusMeta({ key: 'unknown', text: '未知', type: 'info' }, { source, checkAt, reason: pickText(row?.status_reason, '工单状态未知') })
}

export const sqlauditAuditLevelMeta = (levelOrRow, options = {}) => {
  const row = levelOrRow && typeof levelOrRow === 'object' ? levelOrRow : null
  const level = row ? (row.audit_level ?? row.level) : levelOrRow
  const code = Number(level)
  const source = inferSource(row, options?.source || 'SQL审核')
  const checkAt = row?.updated_at || row?.created_at || options?.checkAt || ''

  if (code === 0) return withStatusMeta({ key: 'pass', text: '通过', type: 'success' }, { source, checkAt, reason: pickText(row?.status_reason, '未发现明显风险') })
  if (code === 1) return withStatusMeta({ key: 'warning', text: '警告', type: 'warning' }, { source, checkAt, reason: pickText(row?.status_reason, '存在可控风险') })
  if (code === 2) return withStatusMeta({ key: 'error', text: '错误', type: 'danger' }, { source, checkAt, reason: pickText(row?.status_reason, '存在高风险或违规语句') })
  return withStatusMeta({ key: 'unknown', text: '未知', type: 'info' }, { source, checkAt, reason: pickText(row?.status_reason, '风险等级未知') })
}

export const gitRepoStatusMeta = (statusOrRow, options = {}) => {
  const row = statusOrRow && typeof statusOrRow === 'object' ? statusOrRow : null
  const status = row ? row.status : statusOrRow
  const code = Number(status)
  const normalized = normalizeText(status)
  const source = inferSource(row, options?.source || 'GitOps同步')
  const checkAt = row?.last_sync_at || row?.updated_at || options?.checkAt || ''

  if (code === 1 || normalized === 'syncing' || normalized === 'running') {
    return withStatusMeta(
      { key: 'syncing', text: '同步中', type: 'warning' },
      { source, checkAt, reason: pickText(row?.status_reason, '仓库正在同步') }
    )
  }
  if (code === 2 || normalized === 'failed' || normalized === 'error') {
    return withStatusMeta(
      { key: 'error', text: '异常', type: 'danger' },
      { source, checkAt, reason: pickText(row?.last_error, row?.status_reason, '仓库同步异常') }
    )
  }
  if (code === 0 || normalized === 'ok' || normalized === 'success' || normalized === 'normal') {
    return withStatusMeta(
      { key: 'ok', text: '正常', type: 'success' },
      { source, checkAt, reason: pickText(row?.status_reason, '仓库状态正常') }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    { source, checkAt, reason: pickText(row?.status_reason, '仓库状态未知') }
  )
}

export const oncallShiftStatusMeta = (statusOrRow, options = {}) => {
  const row = statusOrRow && typeof statusOrRow === 'object' ? statusOrRow : null
  const status = row ? row.status : statusOrRow
  const code = Number(status)
  const source = inferSource(row, options?.source || '值班排班')
  const checkAt = row?.updated_at || row?.end_at || row?.start_at || options?.checkAt || ''

  if (code === 1) {
    return withStatusMeta(
      { key: 'normal', text: '正常', type: 'success' },
      { source, checkAt, reason: pickText(row?.status_reason, '班次正常值守') }
    )
  }
  if (code === 2 || code === 0) {
    return withStatusMeta(
      { key: 'swapped', text: '已换班', type: 'warning' },
      { source, checkAt, reason: pickText(row?.reason, row?.status_reason, '班次已发生换班') }
    )
  }
  return withStatusMeta(
    { key: 'unknown', text: '未知', type: 'info' },
    { source, checkAt, reason: pickText(row?.status_reason, '班次状态未知') }
  )
}
