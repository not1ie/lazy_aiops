import { defineStore } from 'pinia'
import { login as loginApi, getUserInfo } from '../api/auth'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    userInfo: null
  }),
  
  getters: {
    isLoggedIn: (state) => !!state.token
  },
  
  actions: {
    async login(username, password) {
      try {
        const res = await loginApi(username, password)
        this.token = res.data.token
        localStorage.setItem('token', res.data.token)
        return true
      } catch (error) {
        console.error('Login failed:', error)
        return false
      }
    },
    
    async fetchUserInfo() {
      try {
        const res = await getUserInfo()
        this.userInfo = res.data
      } catch (error) {
        console.error('Fetch user info failed:', error)
      }
    },
    
    logout() {
      this.token = ''
      this.userInfo = null
      localStorage.removeItem('token')
    }
  }
})
