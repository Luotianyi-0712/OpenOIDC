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

export interface DeveloperStatus {
  eligible: boolean
  has_clients: boolean
  can_access: boolean
  can_create: boolean
  current_trust_level: number
  min_trust_level: number
  email_verified: boolean
  requires_email_verify: boolean
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(true)
  const developerMinTrustLevel = ref(1)
  const developerStatus = ref<DeveloperStatus | null>(null)

  const isLoggedIn = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin' || user.value?.role === 'super_admin')
  const isSuperAdmin = computed(() => user.value?.role === 'super_admin')
  const isDeveloper = computed(() => canShowDeveloperConsole.value)
  const canCreateDeveloperApp = computed(() => developerStatus.value?.can_create || false)
  const canShowDeveloperConsole = computed(() => developerStatus.value?.can_access || false)

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

  async function fetchDeveloperStatus() {
    if (!user.value) {
      developerStatus.value = null
      return
    }
    try {
      const res = await api.get<DeveloperStatus>('/developer/status')
      developerStatus.value = res.data ?? null
      if (res.data?.min_trust_level !== undefined) {
        developerMinTrustLevel.value = res.data.min_trust_level
      }
    } catch {
      developerStatus.value = null
    }
  }

  async function login(email: string, password: string, turnstileToken?: string) {
    const headers: Record<string, string> = {}
    if (turnstileToken) headers['X-Turnstile-Token'] = turnstileToken
    await api.post('/auth/login', { email, password }, headers)
    await fetchUser()
    await fetchDeveloperStatus()
  }

  async function sendRegisterCode(email: string, turnstileToken?: string) {
    const headers: Record<string, string> = {}
    if (turnstileToken) headers['X-Turnstile-Token'] = turnstileToken
    await api.post('/auth/register/code', { email }, headers)
  }

  async function register(email: string, password: string, display_name: string, code: string, turnstileToken?: string) {
    const headers: Record<string, string> = {}
    if (turnstileToken) headers['X-Turnstile-Token'] = turnstileToken
    await api.post('/auth/register', { email, password, display_name, code }, headers)
  }

  async function logout() {
    try { await api.post('/auth/logout') } catch { /* ignore */ }
    user.value = null
    developerStatus.value = null
  }

  return { user, loading, developerMinTrustLevel, developerStatus, isLoggedIn, isAdmin, isSuperAdmin, isDeveloper, canShowDeveloperConsole, canCreateDeveloperApp, fetchUser, fetchPublicSettings, fetchDeveloperStatus, login, sendRegisterCode, register, logout }
})
