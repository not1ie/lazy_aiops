const normalizeText = (value) => String(value ?? '').trim().toLowerCase()

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
  if (status === 2) return { key: 'maintenance', text: '维护', type: 'warning' }
  if (status === 1) {
    if (isStaleByMinutes(row?.last_check_at, staleMinutes, options?.nowMs)) {
      return { key: 'stale', text: '状态过期', type: 'warning' }
    }
    return { key: 'online', text: '在线', type: 'success' }
  }
  if (status === 0) return { key: 'offline', text: '离线', type: 'danger' }
  return { key: 'unknown', text: '未知', type: 'info' }
}

export const dockerHostStatusMeta = (row, options = {}) => {
  const staleMinutes = Number(options?.staleMinutes || 3)
  const status = normalizeText(row?.status)
  if (status === 'maintenance') return { key: 'maintenance', text: '维护', type: 'warning' }
  if (status === 'online') {
    if (isStaleByMinutes(row?.last_check_at, staleMinutes, options?.nowMs)) {
      return { key: 'stale', text: '状态过期', type: 'warning' }
    }
    return { key: 'online', text: '在线', type: 'success' }
  }
  if (status === 'offline') return { key: 'offline', text: '离线', type: 'danger' }
  if (status === 'error') return { key: 'error', text: '异常', type: 'danger' }
  return { key: 'unknown', text: '未知', type: 'info' }
}

export const k8sClusterStatusMeta = (row, options = {}) => {
  const staleMinutes = Number(options?.staleMinutes || 5)
  const status = Number(row?.status)
  if (status === 2) return { key: 'maintenance', text: '维护', type: 'warning' }
  if (status === 1) {
    if (isStaleByMinutes(row?.last_check_at, staleMinutes, options?.nowMs)) {
      return { key: 'stale', text: '状态过期', type: 'warning' }
    }
    return { key: 'online', text: '在线', type: 'success' }
  }
  if (status === 0) return { key: 'abnormal', text: '异常', type: 'danger' }
  return { key: 'unknown', text: '未知', type: 'info' }
}

export const jumpSessionStatusMeta = (row, options = {}) => {
  const pendingTimeoutMinutes = Number(options?.pendingTimeoutMinutes || 15)
  const activeLongMinutes = Number(options?.activeLongMinutes || 180)
  const nowMs = Number(options?.nowMs || Date.now())
  const status = normalizeText(row?.status)
  if (status === 'pending_approval') {
    const stale = isStaleByMinutes(row?.created_at || row?.started_at, pendingTimeoutMinutes, nowMs)
    if (stale) return { key: 'pending_timeout', text: '审批超时', type: 'warning' }
    return { key: 'pending', text: '待审批', type: 'warning' }
  }
  if (status === 'active') {
    const longRunning = isStaleByMinutes(row?.started_at || row?.created_at, activeLongMinutes, nowMs)
    if (longRunning) return { key: 'active_long', text: '活跃(长时)', type: 'warning' }
    return { key: 'active', text: '活跃', type: 'success' }
  }
  if (status === 'blocked') return { key: 'blocked', text: '已阻断', type: 'danger' }
  if (status === 'rejected') return { key: 'rejected', text: '已拒绝', type: 'danger' }
  if (status === 'closed' || status === 'finished' || status === 'done') return { key: 'closed', text: '已关闭', type: 'info' }
  return { key: 'unknown', text: '未知', type: 'info' }
}
