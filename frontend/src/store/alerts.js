import { defineStore } from 'pinia'

export const useAlertsStore = defineStore('alerts', {
  state: () => ({
    status: '',
    severity: '',
    target: ''
  }),
  actions: {
    setFilters(filters) {
      if (filters.status !== undefined) this.status = filters.status
      if (filters.severity !== undefined) this.severity = filters.severity
      if (filters.target !== undefined) this.target = filters.target
    },
    resetFilters() {
      this.status = ''
      this.severity = ''
      this.target = ''
    }
  }
})
