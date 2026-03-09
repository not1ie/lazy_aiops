export const getErrorMessage = (error, fallback = '操作失败') => {
  return error?.response?.data?.message || error?.message || fallback
}

export const isCancelError = (error) => {
  return error === 'cancel' || error === 'close' || error === 'abort'
}
