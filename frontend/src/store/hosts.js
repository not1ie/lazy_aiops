import { defineStore } from 'pinia'

export const useHostsStore = defineStore('hosts', {
  state: () => ({
    searchKeyword: '',
    activeGroupId: '',
    currentPage: 1,
    pageSize: 20
  }),
  actions: {
    setFilters(filters) {
      if (filters.searchKeyword !== undefined) this.searchKeyword = filters.searchKeyword
      if (filters.activeGroupId !== undefined) this.activeGroupId = filters.activeGroupId
      if (filters.currentPage !== undefined) this.currentPage = filters.currentPage
      if (filters.pageSize !== undefined) this.pageSize = filters.pageSize
    },
    resetFilters() {
      this.searchKeyword = ''
      this.activeGroupId = ''
      this.currentPage = 1
      this.pageSize = 20
    }
  }
})
