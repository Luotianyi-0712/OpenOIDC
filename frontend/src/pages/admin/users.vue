<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { Search, Pencil, Trash2, Loader2, ShieldCheck, X, Plus, Eye, Monitor, KeyRound } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const auth = useAuthStore()

interface User {
  id: string
  email: string
  email_verified: boolean
  display_name: string
  alias?: string
  avatar_url?: string
  status: string
  role: 'super_admin' | 'admin' | 'user'
  security_level: number
  last_login_at?: string
  created_at: string
  updated_at: string
}

interface Client {
  id: string
  client_id: string
  client_name: string
  owner_email?: string
  redirect_uris: string[]
  scopes: string[]
  min_security_level: number
  is_active: boolean
  created_at: string
}

interface UserSession {
  id: string
  ip: string
  user_agent: string
  created_at: string
  expires_at: string
}

interface UserBinding {
  id: string
  provider: string
  provider_uid: string
  provider_name: string
  bound_at: string
}

interface RiskReport {
  id: string
  reason: string
  category: string
  status: string
  created_at: string
}

const users = ref<User[]>([])
const total = ref(0)
const offset = ref(0)
const limit = ref(20)
const search = ref('')
const statusFilter = ref('')
const loading = ref(false)
const error = ref('')

const showCreateModal = ref(false)
const createForm = ref({ email: '', password: '', display_name: '', role: 'user' })
const creating = ref(false)

const showModal = ref(false)
const editingUser = ref<User | null>(null)
const form = ref({ display_name: '', status: 'active', role: 'user' as string })
const saving = ref(false)

const showDetailModal = ref(false)
const detailUser = ref<User | null>(null)
const detailClients = ref<Client[]>([])
const detailSessions = ref<UserSession[]>([])
const detailBindings = ref<UserBinding[]>([])
const detailRiskReports = ref<RiskReport[]>([])
const loadingDetail = ref(false)

const showSecurityModal = ref(false)
const securityForm = ref({ level: 0 })
const securityUserId = ref('')

const showDeleteModal = ref(false)
const deletingUser = ref<User | null>(null)
const deleting = ref(false)

const showResetPasswordModal = ref(false)
const resetPasswordUser = ref<User | null>(null)
const resetPasswordForm = ref({ new_password: '' })
const resettingPassword = ref(false)

let searchTimer: ReturnType<typeof setTimeout> | null = null

watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    offset.value = 0
    fetchUsers()
  }, 300)
})

watch(statusFilter, () => {
  offset.value = 0
  fetchUsers()
})

onMounted(fetchUsers)

async function fetchUsers() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams()
    if (search.value) params.set('search', search.value)
    if (statusFilter.value) params.set('status', statusFilter.value)
    params.set('offset', String(offset.value))
    params.set('limit', String(limit.value))
    const res = await api.get<User[]>(`/admin/users?${params}`)
    users.value = res.data ?? []
    total.value = res.meta?.total ?? 0
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function openCreate() {
  createForm.value = { email: '', password: '', display_name: '', role: 'user' }
  showCreateModal.value = true
}

async function createUser() {
  creating.value = true
  error.value = ''
  try {
    await api.post('/admin/users', createForm.value)
    showCreateModal.value = false
    offset.value = 0
    await fetchUsers()
  } catch (e: any) {
    error.value = e.message
  } finally {
    creating.value = false
  }
}

function openEdit(user: User) {
  editingUser.value = user
  form.value = {
    display_name: user.display_name,
    status: user.status,
    role: user.role,
  }
  showModal.value = true
}

async function saveUser() {
  if (!editingUser.value) return
  saving.value = true
  error.value = ''
  try {
    await api.put(`/admin/users/${editingUser.value.id}`, form.value)
    showModal.value = false
    await fetchUsers()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

async function openDetail(user: User) {
  detailUser.value = user
  detailClients.value = []
  detailSessions.value = []
  detailBindings.value = []
  detailRiskReports.value = []
  showDetailModal.value = true
  loadingDetail.value = true
  try {
    const [clientsRes, detailRes] = await Promise.all([
      api.get<Client[]>(`/admin/users/${user.id}/clients`),
      api.get<any>(`/admin/users/${user.id}/detail`),
    ])
    detailClients.value = clientsRes.data ?? []
    if (detailRes.data) {
      detailSessions.value = detailRes.data.sessions || []
      detailBindings.value = detailRes.data.bindings || []
      detailRiskReports.value = detailRes.data.risk_reports || []
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loadingDetail.value = false
  }
}

function openSecurityLevel(user: User) {
  securityUserId.value = user.id
  securityForm.value = { level: user.security_level }
  showSecurityModal.value = true
}

async function saveSecurityLevel() {
  saving.value = true
  error.value = ''
  try {
    await api.put(`/admin/users/${securityUserId.value}/security-level`, securityForm.value)
    showSecurityModal.value = false
    await fetchUsers()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

function confirmDelete(user: User) {
  deletingUser.value = user
  showDeleteModal.value = true
}

async function deleteUser() {
  if (!deletingUser.value) return
  deleting.value = true
  error.value = ''
  try {
    await api.del(`/admin/users/${deletingUser.value.id}`)
    showDeleteModal.value = false
    await fetchUsers()
  } catch (e: any) {
    error.value = e.message
  } finally {
    deleting.value = false
  }
}

function openResetPassword(user: User) {
  resetPasswordUser.value = user
  resetPasswordForm.value = { new_password: '' }
  showResetPasswordModal.value = true
}

async function resetPassword() {
  if (!resetPasswordUser.value) return
  resettingPassword.value = true
  error.value = ''
  try {
    await api.post(`/admin/users/${resetPasswordUser.value.id}/reset-password`, resetPasswordForm.value)
    showResetPasswordModal.value = false
  } catch (e: any) {
    error.value = e.message
  } finally {
    resettingPassword.value = false
  }
}

function prevPage() {
  if (offset.value > 0) {
    offset.value = Math.max(0, offset.value - limit.value)
    fetchUsers()
  }
}

function nextPage() {
  if (offset.value + limit.value < total.value) {
    offset.value += limit.value
    fetchUsers()
  }
}

function formatDate(d?: string) {
  if (!d) return '-'
  return new Date(d).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    active: t('adminUsers.active'),
    suspended: t('adminUsers.suspended'),
    deleted: t('adminUsers.deleted'),
  }
  return labels[status] || status
}
</script>

<template>
  <div>
    <div class="flex items-start justify-between gap-4 mb-6">
      <div>
        <h2 class="text-lg font-semibold">{{ $t('adminUsers.title') }}</h2>
        <p class="text-sm text-muted-foreground mt-1">{{ $t('adminUsers.subtitle') }}</p>
      </div>
      <button @click="openCreate" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors flex items-center gap-2">
        <Plus class="w-4 h-4" /> {{ $t('adminUsers.createUser') }}
      </button>
    </div>

    <div class="rounded-xl border border-border bg-muted/30 p-4 mb-4 text-sm text-muted-foreground">
      {{ $t('adminUsers.pageHint') }}
    </div>

    <div class="flex items-center gap-3 mb-4">
      <div class="relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
        <input v-model="search" type="text" :placeholder="$t('adminUsers.searchPlaceholder')" class="pl-9 pr-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10 w-64" />
      </div>
      <select v-model="statusFilter" class="px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10">
        <option value="">{{ $t('adminUsers.allStatuses') }}</option>
        <option value="active">{{ $t('adminUsers.active') }}</option>
        <option value="suspended">{{ $t('adminUsers.suspended') }}</option>
        <option value="deleted">{{ $t('adminUsers.deleted') }}</option>
      </select>
    </div>

    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <div v-if="loading && users.length === 0" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <div v-else class="border border-border rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
          <tr>
            <th class="px-4 py-3">{{ $t('adminUsers.user') }}</th>
            <th class="px-4 py-3">{{ $t('adminUsers.status') }}</th>
            <th class="px-4 py-3">{{ $t('adminUsers.role') }}</th>
            <th class="px-4 py-3">{{ $t('adminUsers.securityLevel') }}</th>
            <th class="px-4 py-3">{{ $t('adminUsers.lastLogin') }}</th>
            <th class="px-4 py-3">{{ $t('adminUsers.created') }}</th>
            <th class="px-4 py-3">{{ $t('actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-if="users.length === 0">
            <td colspan="7" class="px-4 py-8 text-center text-muted-foreground">{{ $t('adminUsers.noUsers') }}</td>
          </tr>
          <tr v-for="user in users" :key="user.id" class="hover:bg-muted/30 transition-colors">
            <td class="px-4 py-3">
              <div class="font-medium">{{ user.email }}</div>
              <div class="text-xs text-muted-foreground">{{ user.display_name || $t('adminUsers.noDisplayName') }} · {{ user.email_verified ? $t('adminUsers.emailVerified') : $t('adminUsers.emailUnverified') }}</div>
            </td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium" :class="{ 'bg-green-50 text-green-700': user.status === 'active', 'bg-yellow-50 text-yellow-700': user.status === 'suspended', 'bg-red-50 text-red-700': user.status === 'deleted' }">
                {{ statusLabel(user.status) }}
              </span>
            </td>
            <td class="px-4 py-3">
              <span class="text-xs font-medium px-2 py-0.5 rounded-full" :class="{ 'text-white bg-foreground': user.role === 'super_admin', 'text-foreground bg-foreground/10': user.role === 'admin', 'text-muted-foreground bg-muted': user.role === 'user' }">
                {{ $t(`role.${user.role}`) }}
              </span>
            </td>
            <td class="px-4 py-3 text-muted-foreground">L{{ user.security_level }}</td>
            <td class="px-4 py-3 text-muted-foreground whitespace-nowrap">{{ formatDate(user.last_login_at) }}</td>
            <td class="px-4 py-3 text-muted-foreground whitespace-nowrap">{{ formatDate(user.created_at) }}</td>
            <td class="px-4 py-3">
              <div class="flex items-center gap-1 flex-wrap">
                <button @click="openDetail(user)" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <Eye class="w-3 h-3" /> {{ $t('adminUsers.view') }}
                </button>
                <button @click="openEdit(user)" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <Pencil class="w-3 h-3" /> {{ $t('edit') }}
                </button>
                <button @click="openSecurityLevel(user)" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <ShieldCheck class="w-3 h-3" /> {{ $t('adminUsers.level') }}
                </button>
                <button @click="openResetPassword(user)" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <KeyRound class="w-3 h-3" /> {{ $t('adminUsers.resetPwd') }}
                </button>
                <button @click="confirmDelete(user)" class="text-xs font-medium px-2 py-1 rounded hover:bg-destructive/5 transition-colors text-destructive flex items-center gap-1">
                  <Trash2 class="w-3 h-3" /> {{ $t('delete') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="total > 0" class="flex items-center justify-between mt-4 text-sm text-muted-foreground">
      <span>{{ $t('showing', { from: offset + 1, to: Math.min(offset + limit, total), total }) }}</span>
      <div class="flex gap-2">
        <button @click="prevPage" :disabled="offset === 0" class="px-3 py-1.5 border border-border rounded-lg text-sm hover:bg-muted transition-colors disabled:opacity-40">{{ $t('prev') }}</button>
        <button @click="nextPage" :disabled="offset + limit >= total" class="px-3 py-1.5 border border-border rounded-lg text-sm hover:bg-muted transition-colors disabled:opacity-40">{{ $t('next') }}</button>
      </div>
    </div>

    <div v-if="showCreateModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showCreateModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4 p-6">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('adminUsers.createUserTitle') }}</h2>
          <button @click="showCreateModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('adminUsers.createUserHint') }}</p>
        <form @submit.prevent="createUser" class="flex flex-col gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminUsers.emailLabel') }}</label>
            <input v-model="createForm.email" type="email" required :placeholder="$t('adminUsers.emailPlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminUsers.passwordLabel') }}</label>
            <input v-model="createForm.password" type="password" required minlength="6" :placeholder="$t('adminUsers.passwordPlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminUsers.displayName') }}</label>
            <input v-model="createForm.display_name" type="text" :placeholder="$t('adminUsers.displayNamePlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminUsers.roleLabel') }}</label>
            <select v-model="createForm.role" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10">
              <option value="user">{{ $t('role.user') }}</option>
              <option v-if="auth.isSuperAdmin" value="admin">{{ $t('role.admin') }}</option>
              <option v-if="auth.isSuperAdmin" value="super_admin">{{ $t('role.super_admin') }}</option>
            </select>
            <p class="text-xs text-muted-foreground mt-1">{{ $t('adminUsers.roleHint') }}</p>
          </div>
          <div class="flex justify-end gap-2 mt-2">
            <button type="button" @click="showCreateModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
            <button type="submit" :disabled="creating" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="creating" class="w-4 h-4 animate-spin" /> {{ $t('adminUsers.createUser') }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showDetailModal && detailUser" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showDetailModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-3xl mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-5">
          <h2 class="text-lg font-semibold">{{ $t('adminUsers.userDetails') }}</h2>
          <button @click="showDetailModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <div class="grid grid-cols-2 gap-4 text-sm mb-6">
          <div><div class="text-muted-foreground">{{ $t('adminUsers.email') }}</div><div class="font-medium">{{ detailUser.email }}</div></div>
          <div><div class="text-muted-foreground">{{ $t('adminUsers.name') }}</div><div class="font-medium">{{ detailUser.display_name || '-' }}</div></div>
          <div><div class="text-muted-foreground">{{ $t('adminUsers.alias') }}</div><div class="font-medium">{{ detailUser.alias || '-' }}</div></div>
          <div><div class="text-muted-foreground">{{ $t('adminUsers.role') }}</div><div class="font-medium">{{ $t(`role.${detailUser.role}`) }}</div></div>
          <div><div class="text-muted-foreground">{{ $t('adminUsers.status') }}</div><div class="font-medium">{{ statusLabel(detailUser.status) }}</div></div>
          <div><div class="text-muted-foreground">{{ $t('adminUsers.securityLevel') }}</div><div class="font-medium">L{{ detailUser.security_level }}</div></div>
          <div><div class="text-muted-foreground">{{ $t('adminUsers.lastLogin') }}</div><div class="font-medium">{{ formatDate(detailUser.last_login_at) }}</div></div>
          <div><div class="text-muted-foreground">{{ $t('adminUsers.created') }}</div><div class="font-medium">{{ formatDate(detailUser.created_at) }}</div></div>
        </div>
        <div class="flex items-center gap-2 mb-3">
          <Monitor class="w-4 h-4 text-muted-foreground" />
          <h3 class="font-medium">{{ $t('adminUsers.userClients') }}</h3>
        </div>
        <div v-if="loadingDetail" class="text-sm text-muted-foreground py-4">{{ $t('loading') }}</div>
        <div v-else-if="detailClients.length === 0" class="text-sm text-muted-foreground py-4 border border-dashed border-border rounded-lg text-center">{{ $t('adminUsers.noUserClients') }}</div>
        <div v-else class="border border-border rounded-lg overflow-hidden">
          <table class="w-full text-sm">
            <thead class="bg-muted/50 text-left text-xs text-muted-foreground"><tr><th class="px-3 py-2">{{ $t('adminClients.clientName') }}</th><th class="px-3 py-2">{{ $t('adminClients.clientId') }}</th><th class="px-3 py-2">{{ $t('adminClients.minSecurityLevel') }}</th><th class="px-3 py-2">{{ $t('adminUsers.status') }}</th></tr></thead>
            <tbody class="divide-y divide-border">
              <tr v-for="client in detailClients" :key="client.id">
                <td class="px-3 py-2 font-medium">{{ client.client_name }}</td>
                <td class="px-3 py-2 font-mono text-xs text-muted-foreground">{{ client.client_id }}</td>
                <td class="px-3 py-2">L{{ client.min_security_level }}</td>
                <td class="px-3 py-2">{{ client.is_active ? $t('adminProviders.enabled') : $t('adminProviders.disabled') }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Sessions -->
        <div class="mt-6">
          <h3 class="font-medium mb-2">{{ $t('adminUserDetail.sessions') }}</h3>
          <div v-if="detailSessions.length === 0" class="text-sm text-muted-foreground py-3 border border-dashed border-border rounded-lg text-center">{{ $t('adminUserDetail.noSessions') }}</div>
          <div v-else class="border border-border rounded-lg overflow-hidden">
            <table class="w-full text-sm">
              <thead class="bg-muted/50 text-left text-xs text-muted-foreground"><tr><th class="px-3 py-2">{{ $t('adminUserDetail.ip') }}</th><th class="px-3 py-2">{{ $t('adminUserDetail.userAgent') }}</th><th class="px-3 py-2">{{ $t('adminUserDetail.createdAt') }}</th></tr></thead>
              <tbody class="divide-y divide-border">
                <tr v-for="sess in detailSessions" :key="sess.id">
                  <td class="px-3 py-2 font-mono text-xs">{{ sess.ip }}</td>
                  <td class="px-3 py-2 text-xs truncate max-w-[200px]">{{ sess.user_agent }}</td>
                  <td class="px-3 py-2 text-xs text-muted-foreground">{{ formatDate(sess.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Bindings -->
        <div class="mt-6">
          <h3 class="font-medium mb-2">{{ $t('adminUserDetail.bindings') }}</h3>
          <div v-if="detailBindings.length === 0" class="text-sm text-muted-foreground py-3 border border-dashed border-border rounded-lg text-center">{{ $t('adminUserDetail.noBindings') }}</div>
          <div v-else class="border border-border rounded-lg overflow-hidden">
            <table class="w-full text-sm">
              <thead class="bg-muted/50 text-left text-xs text-muted-foreground"><tr><th class="px-3 py-2">{{ $t('adminUserDetail.provider') }}</th><th class="px-3 py-2">{{ $t('adminUserDetail.providerName') }}</th><th class="px-3 py-2">{{ $t('adminUserDetail.boundAt') }}</th></tr></thead>
              <tbody class="divide-y divide-border">
                <tr v-for="binding in detailBindings" :key="binding.id">
                  <td class="px-3 py-2 font-mono text-xs">{{ binding.provider }}</td>
                  <td class="px-3 py-2 text-xs">{{ binding.provider_name || binding.provider_uid }}</td>
                  <td class="px-3 py-2 text-xs text-muted-foreground">{{ formatDate(binding.bound_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Risk Reports -->
        <div class="mt-6">
          <h3 class="font-medium mb-2">{{ $t('adminUserDetail.riskReports') }}</h3>
          <div v-if="detailRiskReports.length === 0" class="text-sm text-muted-foreground py-3 border border-dashed border-border rounded-lg text-center">{{ $t('adminUserDetail.noRiskReports') }}</div>
          <div v-else class="border border-border rounded-lg overflow-hidden">
            <table class="w-full text-sm">
              <thead class="bg-muted/50 text-left text-xs text-muted-foreground"><tr><th class="px-3 py-2">{{ $t('adminRisk.category') }}</th><th class="px-3 py-2">{{ $t('adminRisk.reason') }}</th><th class="px-3 py-2">{{ $t('adminUsers.status') }}</th><th class="px-3 py-2">{{ $t('adminRisk.time') }}</th></tr></thead>
              <tbody class="divide-y divide-border">
                <tr v-for="report in detailRiskReports" :key="report.id">
                  <td class="px-3 py-2 text-xs">{{ report.category }}</td>
                  <td class="px-3 py-2 text-xs">{{ report.reason }}</td>
                  <td class="px-3 py-2 text-xs">{{ report.status }}</td>
                  <td class="px-3 py-2 text-xs text-muted-foreground">{{ formatDate(report.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4 p-6">
        <div class="flex items-center justify-between mb-5">
          <h2 class="text-lg font-semibold">{{ $t('adminUsers.editUser') }}</h2>
          <button @click="showModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="saveUser" class="flex flex-col gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminUsers.displayName') }}</label>
            <input v-model="form.display_name" type="text" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminUsers.status') }}</label>
            <select v-model="form.status" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10">
              <option value="active">{{ $t('adminUsers.active') }}</option>
              <option value="suspended">{{ $t('adminUsers.suspended') }}</option>
              <option value="deleted">{{ $t('adminUsers.deleted') }}</option>
            </select>
          </div>
          <div v-if="auth.isSuperAdmin">
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminUsers.role') }}</label>
            <select v-model="form.role" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10">
              <option value="user">{{ $t('role.user') }}</option>
              <option value="admin">{{ $t('role.admin') }}</option>
              <option value="super_admin">{{ $t('role.super_admin') }}</option>
            </select>
          </div>
          <div class="flex justify-end gap-2 mt-2">
            <button type="button" @click="showModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
            <button type="submit" :disabled="saving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="saving" class="w-4 h-4 animate-spin" /> {{ $t('save') }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showSecurityModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showSecurityModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('adminUsers.overrideSecurityLevel') }}</h2>
          <button @click="showSecurityModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('adminUsers.overrideLevelHint') }}</p>
        <form @submit.prevent="saveSecurityLevel" class="flex flex-col gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminUsers.securityLevel') }}</label>
            <input v-model.number="securityForm.level" type="number" min="0" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div class="flex justify-end gap-2 mt-2">
            <button type="button" @click="showSecurityModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
            <button type="submit" :disabled="saving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="saving" class="w-4 h-4 animate-spin" /> {{ $t('save') }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showDeleteModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <h2 class="text-lg font-semibold mb-2">{{ $t('adminUsers.deleteUser') }}</h2>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('adminUsers.deleteConfirm', { email: deletingUser?.email || '' }) }}</p>
        <div class="flex justify-end gap-2">
          <button @click="showDeleteModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
          <button @click="deleteUser" :disabled="deleting" class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2">
            <Loader2 v-if="deleting" class="w-4 h-4 animate-spin" /> {{ $t('delete') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showResetPasswordModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showResetPasswordModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('adminUsers.resetPassword') }}</h2>
          <button @click="showResetPasswordModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('adminUsers.resetPasswordHint', { email: resetPasswordUser?.email || '' }) }}</p>
        <form @submit.prevent="resetPassword" class="flex flex-col gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminUsers.newPassword') }}</label>
            <input v-model="resetPasswordForm.new_password" type="password" required minlength="6" :placeholder="$t('adminUsers.newPasswordPlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div class="flex justify-end gap-2 mt-2">
            <button type="button" @click="showResetPasswordModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
            <button type="submit" :disabled="resettingPassword" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="resettingPassword" class="w-4 h-4 animate-spin" /> {{ $t('adminUsers.resetPwd') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
