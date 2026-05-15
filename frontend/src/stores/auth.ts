import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api/client'

export interface User {
  id: string
  email: string
  email_verified: boolean
  display_name: string
  alias: string | null
  avatar_url: string
  security_level: number
  role: 'super_admin' | 'admin' | 'user'
  status: string
  last_login_at: string | null
  created_at: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(true)
  const developerMinTrustLevel = ref(1)

  const isLoggedIn = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin' || user.value?.role === 'super_admin')
  const isSuperAdmin = computed(() => user.value?.role === 'super_admin')
  const isDeveloper = computed(() => isAdmin.value || (user.value?.security_level ?? 0) >= developerMinTrustLevel.value)

  async function fetchUser() {
    try {
      const res = await api.get<User>('/me')
      user.value = res.data!
    } catch {
      user.value = null
    } finally {
      loading.value = false
    }
  }

  async function fetchPublicSettings() {
    try {
      const res = await api.get<Record<string, string>>('/settings/public')
      if (res.data?.developer_min_trust_level) {
        developerMinTrustLevel.value = parseInt(res.data.developer_min_trust_level) || 1
      }
    } catch { /* use default */ }
  }

  async function login(email: string, password: string, turnstileToken?: string) {
    const headers: Record<string, string> = {}
    if (turnstileToken) headers['X-Turnstile-Token'] = turnstileToken
    await api.post('/auth/login', { email, password }, headers)
    await fetchUser()
  }

  async function register(email: string, password: string, display_name: string, turnstileToken?: string) {
    const headers: Record<string, string> = {}
    if (turnstileToken) headers['X-Turnstile-Token'] = turnstileToken
    await api.post('/auth/register', { email, password, display_name }, headers)
  }

  async function logout() {
    try { await api.post('/auth/logout') } catch { /* ignore */ }
    user.value = null
  }

  return { user, loading, developerMinTrustLevel, isLoggedIn, isAdmin, isSuperAdmin, isDeveloper, fetchUser, fetchPublicSettings, login, register, logout }
})
