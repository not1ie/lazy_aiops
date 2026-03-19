const CRITICAL_STATUSES = new Set([
  'Failed',
  'CrashLoopBackOff',
  'ImagePullBackOff',
  'ErrImagePull',
  'CreateContainerConfigError',
  'CreateContainerError',
  'RunContainerError',
  'InvalidImageName',
  'OOMKilled',
  'DeadlineExceeded',
  'Evicted'
])

const WARNING_STATUSES = new Set([
  'Pending',
  'NotReady',
  'ContainerCreating',
  'PodInitializing',
  'Init:Error',
  'Init:CrashLoopBackOff',
  'Terminating',
  'Unknown'
])

const SUCCESS_STATUSES = new Set(['Running', 'Succeeded', 'Completed'])

export const COMMON_POD_STATUS_ORDER = [
  'Running',
  'NotReady',
  'Pending',
  'CrashLoopBackOff',
  'ImagePullBackOff',
  'OOMKilled',
  'Failed',
  'Succeeded',
  'Terminating',
  'Unknown'
]

export const normalizePodStatus = (status, phase = '') => {
  const text = String(status || '').trim()
  if (text) return text
  const p = String(phase || '').trim()
  return p || 'Unknown'
}

export const podStatusType = (status) => {
  const normalized = normalizePodStatus(status)
  if (SUCCESS_STATUSES.has(normalized)) return 'success'
  if (CRITICAL_STATUSES.has(normalized)) return 'danger'
  if (WARNING_STATUSES.has(normalized) || normalized.startsWith('Init:')) return 'warning'
  return 'info'
}

export const isPodStatusCritical = (status) => {
  const normalized = normalizePodStatus(status)
  return CRITICAL_STATUSES.has(normalized)
}

export const isPodStatusPending = (status) => {
  const normalized = normalizePodStatus(status)
  return WARNING_STATUSES.has(normalized) || normalized.startsWith('Init:')
}

export const isPodStatusHealthy = (status) => {
  const normalized = normalizePodStatus(status)
  return SUCCESS_STATUSES.has(normalized)
}

export const buildPodStatusOptions = (statuses = []) => {
  const uniq = new Set(statuses.map((item) => normalizePodStatus(item)).filter(Boolean))
  const ordered = []
  COMMON_POD_STATUS_ORDER.forEach((name) => {
    if (uniq.has(name)) {
      ordered.push(name)
      uniq.delete(name)
    }
  })
  Array.from(uniq)
    .sort((a, b) => a.localeCompare(b, 'zh-CN'))
    .forEach((name) => ordered.push(name))
  return ordered
}
