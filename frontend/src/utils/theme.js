import { computed, ref } from 'vue'

const THEME_KEY = 'lazy-theme'
const theme = ref('light')
let mediaQuery = null
let mediaListenerBound = false

const isBrowser = () => typeof window !== 'undefined'

const detectSystemTheme = () => {
  if (!isBrowser() || !window.matchMedia) return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export const getStoredTheme = () => {
  if (!isBrowser()) return ''
  return localStorage.getItem(THEME_KEY) || ''
}

export const resolveTheme = () => {
  const stored = getStoredTheme()
  if (stored === 'dark' || stored === 'light') return stored
  return detectSystemTheme()
}

export const applyTheme = (nextTheme) => {
  const normalized = nextTheme === 'dark' ? 'dark' : 'light'
  theme.value = normalized
  if (!isBrowser()) return normalized
  document.documentElement.setAttribute('data-theme', normalized)
  document.documentElement.style.colorScheme = normalized
  localStorage.setItem(THEME_KEY, normalized)
  window.dispatchEvent(new CustomEvent('lazy-theme-change', { detail: normalized }))
  return normalized
}

export const initTheme = () => {
  const current = applyTheme(resolveTheme())
  if (!isBrowser() || !window.matchMedia || mediaListenerBound) return current
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener?.('change', (event) => {
    const stored = getStoredTheme()
    if (stored === 'dark' || stored === 'light') return
    applyTheme(event.matches ? 'dark' : 'light')
  })
  mediaListenerBound = true
  return current
}

export const setTheme = (nextTheme) => applyTheme(nextTheme)

export const toggleTheme = () => applyTheme(theme.value === 'dark' ? 'light' : 'dark')

export const useTheme = () => ({
  theme,
  isDark: computed(() => theme.value === 'dark'),
  setTheme,
  toggleTheme
})
