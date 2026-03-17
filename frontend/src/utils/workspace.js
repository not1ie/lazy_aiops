export const requestApplyWorkspaceCategory = (category, source = 'hub', silent = false) => {
  if (typeof window === 'undefined') return
  window.dispatchEvent(
    new CustomEvent('lao:apply-workspace-category', {
      detail: {
        category,
        source,
        silent
      }
    })
  )
}

