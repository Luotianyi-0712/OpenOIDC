<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { Plus, Pencil, Trash2, Loader2, RefreshCw, X, Copy, Check, AlertTriangle, Power, Users, Ban, CheckCircle2, Search } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface Client {
  id: string
  client_id: string
  client_name: string
  description?: string
  logo_url?: string
  homepage_url?: string
  owner_user_id?: string
  owner_uid?: number
  owner_email?: string
  owner_display_name?: string
  client_secret?: string
  redirect_uris: string[]
  post_logout_redirect_uris?: string[]
  grant_types: string[]
  response_types: string[]
  scopes: string[]
  token_endpoint_auth_method: string
  min_security_level: number
  require_email_verified: boolean
  protocol_type: string
  is_confidential: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

interface ClientUser {
  id: string
  uid: number
  display_name: string
  email: string
  security_level: number
  providers: string[]
  blocked: boolean
  granted_at: string
  last_used_at: string
}

const clients = ref<Client[]>([])
const total = ref(0)
const offset = ref(0)
const limit = ref(20)
const loading = ref(false)
const error = ref('')

const showModal = ref(false)
const isCreate = ref(false)
const editingClient = ref<Client | null>(null)
const saving = ref(false)
const form = ref({
  client_name: '',
  description: '',
  logo_url: '',
  homepage_url: '',
  redirect_uris: '',
  post_logout_redirect_uris: '',
  grant_types: [] as string[],
  response_types: 'code',
  scopes: ['openid', 'profile', 'email'] as string[],
  token_endpoint_auth_method: 'client_secret_basic',
  min_security_level: 0,
  require_email_verified: true,
  protocol_type: 'oidc',
  is_confidential: true,
  is_active: true,
})

const allGrantTypes = ['authorization_code', 'refresh_token', 'client_credentials']
const allScopes = ['openid', 'profile', 'email', 'security_level', 'offline_access']
const allTokenEndpointAuthMethods = ['client_secret_basic', 'client_secret_post']

const showSecretModal = ref(false)
const revealedSecret = ref('')
const secretCopied = ref(false)
const detailSecretCopied = ref(false)

const showRotateModal = ref(false)
const rotatingClient = ref<Client | null>(null)
const rotating = ref(false)

const showDeleteModal = ref(false)
const deletingClient = ref<Client | null>(null)
const deleting = ref(false)
const togglingClientId = ref('')

const usersModal = ref(false)
const usersClient = ref<Client | null>(null)
const clientUsers = ref<ClientUser[]>([])
const clientUsersTotal = ref(0)
const clientUsersOffset = ref(0)
const clientUsersLimit = ref(20)
const clientUsersSearch = ref('')
const clientUsersLoading = ref(false)
const clientUserActionId = ref('')

const clientUsersFrom = computed(() => (clientUsersTotal.value === 0 ? 0 : clientUsersOffset.value + 1))
const clientUsersTo = computed(() => Math.min(clientUsersOffset.value + clientUsersLimit.value, clientUsersTotal.value))

onMounted(fetchClients)

async function fetchClients() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({ offset: String(offset.value), limit: String(limit.value) })
    const res = await api.get<Client[]>(`/admin/clients?${params}`)
    clients.value = res.data ?? []
    total.value = res.meta?.total ?? 0
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function openCreate() {
  isCreate.value = true
  editingClient.value = null
  form.value = {
    client_name: '',
    description: '',
    logo_url: '',
    homepage_url: '',
    redirect_uris: '',
    post_logout_redirect_uris: '',
    grant_types: ['authorization_code', 'refresh_token'],
    response_types: 'code',
    scopes: ['openid', 'profile', 'email'],
    token_endpoint_auth_method: 'client_secret_basic',
    min_security_level: 0,
    require_email_verified: true,
    protocol_type: 'oidc',
    is_confidential: true,
    is_active: true,
  }
  showModal.value = true
}

async function openEdit(client: Client) {
  isCreate.value = false
  error.value = ''
  try {
    const res = await api.get<Client>(`/admin/clients/${client.id}`)
    const detail = res.data ?? client
    editingClient.value = detail
    form.value = {
      client_name: detail.client_name,
      description: detail.description || '',
      logo_url: detail.logo_url || '',
      homepage_url: detail.homepage_url || '',
      redirect_uris: (detail.redirect_uris || []).join('\n'),
      post_logout_redirect_uris: (detail.post_logout_redirect_uris || []).join('\n'),
      grant_types: [...(detail.grant_types || [])],
      response_types: (detail.response_types || ['code']).join(' '),
      scopes: [...(detail.scopes || [])],
      token_endpoint_auth_method: detail.token_endpoint_auth_method || (detail.is_confidential ? 'client_secret_basic' : 'none'),
      min_security_level: detail.min_security_level,
      require_email_verified: detail.require_email_verified,
      protocol_type: detail.protocol_type,
      is_confidential: detail.is_confidential,
      is_active: detail.is_active,
    }
    showModal.value = true
  } catch (e: any) {
    error.value = e.message
  }
}

function toggleGrant(grant: string) {
  const idx = form.value.grant_types.indexOf(grant)
  if (idx >= 0) form.value.grant_types.splice(idx, 1)
  else form.value.grant_types.push(grant)
}

function toggleScope(scope: string) {
  const idx = form.value.scopes.indexOf(scope)
  if (idx >= 0) form.value.scopes.splice(idx, 1)
  else form.value.scopes.push(scope)
}

function normalizeClientType() {
  if (!form.value.is_confidential) {
    form.value.token_endpoint_auth_method = 'none'
  } else if (form.value.token_endpoint_auth_method === 'none' || !form.value.token_endpoint_auth_method) {
    form.value.token_endpoint_auth_method = 'client_secret_basic'
  }
}

async function saveClient() {
  normalizeClientType()
  saving.value = true
  error.value = ''
  try {
    const payload = {
      client_name: form.value.client_name,
      description: form.value.description,
      logo_url: form.value.logo_url,
      homepage_url: form.value.homepage_url,
      redirect_uris: form.value.redirect_uris.split(/[\n,]+/).map(s => s.trim()).filter(Boolean),
      post_logout_redirect_uris: form.value.post_logout_redirect_uris.split(/[\n,]+/).map(s => s.trim()).filter(Boolean),
      grant_types: form.value.grant_types,
      response_types: form.value.response_types.split(/[\s,]+/).filter(Boolean),
      scopes: form.value.scopes,
      token_endpoint_auth_method: form.value.is_confidential ? form.value.token_endpoint_auth_method : 'none',
      min_security_level: form.value.min_security_level,
      require_email_verified: form.value.require_email_verified,
      protocol_type: form.value.protocol_type,
      is_confidential: form.value.is_confidential,
      is_active: form.value.is_active,
    }
    if (isCreate.value) {
      const res = await api.post<Client>('/admin/clients', payload)
      showModal.value = false
      if (res.data?.client_secret) {
        revealedSecret.value = res.data.client_secret
        showSecretModal.value = true
      }
    } else if (editingClient.value) {
      await api.put(`/admin/clients/${editingClient.value.id}`, payload)
      showModal.value = false
    }
    await fetchClients()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

function confirmRotate(client: Client) {
  rotatingClient.value = client
  showRotateModal.value = true
}

async function rotateSecret() {
  if (!rotatingClient.value) return
  rotating.value = true
  error.value = ''
  try {
    const res = await api.post<{ client_secret: string }>(`/admin/clients/${rotatingClient.value.id}/rotate-secret`)
    showRotateModal.value = false
    if (res.data?.client_secret) {
      revealedSecret.value = res.data.client_secret
      showSecretModal.value = true
    }
    await fetchClients()
  } catch (e: any) {
    error.value = e.message
  } finally {
    rotating.value = false
  }
}

async function copySecret() {
  await navigator.clipboard.writeText(revealedSecret.value)
  secretCopied.value = true
  setTimeout(() => (secretCopied.value = false), 2000)
}

async function copyClientSecret(secret?: string) {
  if (!secret) return
  await navigator.clipboard.writeText(secret)
  detailSecretCopied.value = true
  setTimeout(() => (detailSecretCopied.value = false), 2000)
}

function confirmDelete(client: Client) {
  deletingClient.value = client
  showDeleteModal.value = true
}

async function deleteClient() {
  if (!deletingClient.value) return
  deleting.value = true
  error.value = ''
  try {
    await api.del(`/admin/clients/${deletingClient.value.id}`)
    showDeleteModal.value = false
    await fetchClients()
  } catch (e: any) {
    error.value = e.message
  } finally {
    deleting.value = false
  }
}

async function toggleClientActive(client: Client) {
  togglingClientId.value = client.id
  error.value = ''
  try {
    await api.put(`/admin/clients/${client.id}`, { is_active: !client.is_active })
    await fetchClients()
  } catch (e: any) {
    error.value = e.message
  } finally {
    togglingClientId.value = ''
  }
}

async function openUsers(client: Client) {
  usersClient.value = client
  clientUsersOffset.value = 0
  clientUsersSearch.value = ''
  usersModal.value = true
  await fetchClientUsers()
}

async function fetchClientUsers() {
  if (!usersClient.value) return
  clientUsersLoading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({
      offset: String(clientUsersOffset.value),
      limit: String(clientUsersLimit.value),
    })
    if (clientUsersSearch.value.trim()) params.set('q', clientUsersSearch.value.trim())
    const res = await api.get<ClientUser[]>(`/admin/clients/${usersClient.value.id}/users?${params}`)
    clientUsers.value = res.data ?? []
    clientUsersTotal.value = res.meta?.total ?? 0
  } catch (e: any) {
    error.value = e.message
  } finally {
    clientUsersLoading.value = false
  }
}

function searchClientUsers() {
  clientUsersOffset.value = 0
  fetchClientUsers()
}

function prevClientUsersPage() {
  if (clientUsersOffset.value === 0) return
  clientUsersOffset.value = Math.max(0, clientUsersOffset.value - clientUsersLimit.value)
  fetchClientUsers()
}

function nextClientUsersPage() {
  if (clientUsersOffset.value + clientUsersLimit.value >= clientUsersTotal.value) return
  clientUsersOffset.value += clientUsersLimit.value
  fetchClientUsers()
}

async function blockClientUser(user: ClientUser) {
  if (!usersClient.value || !window.confirm(t('adminClients.blockUserConfirm', { uid: user.uid }))) return
  clientUserActionId.value = user.id
  try {
    await api.post(`/admin/clients/${usersClient.value.id}/users/${user.id}/block`)
    user.blocked = true
  } catch (e: any) {
    error.value = e.message
  } finally {
    clientUserActionId.value = ''
  }
}

async function unblockClientUser(user: ClientUser) {
  if (!usersClient.value || !window.confirm(t('adminClients.unblockUserConfirm', { uid: user.uid }))) return
  clientUserActionId.value = user.id
  try {
    await api.del(`/admin/clients/${usersClient.value.id}/users/${user.id}/block`)
    user.blocked = false
  } catch (e: any) {
    error.value = e.message
  } finally {
    clientUserActionId.value = ''
  }
}

async function revokeClientUser(user: ClientUser) {
  if (!usersClient.value || !window.confirm(t('adminClients.revokeUserConfirm', { uid: user.uid }))) return
  clientUserActionId.value = user.id
  try {
    await api.del(`/admin/clients/${usersClient.value.id}/users/${user.id}/authorization`)
    await fetchClientUsers()
  } catch (e: any) {
    error.value = e.message
  } finally {
    clientUserActionId.value = ''
  }
}

function prevPage() {
  if (offset.value > 0) {
    offset.value = Math.max(0, offset.value - limit.value)
    fetchClients()
  }
}

function nextPage() {
  if (offset.value + limit.value < total.value) {
    offset.value += limit.value
    fetchClients()
  }
}

function grantLabel(grant: string) {
  return t(`adminClients.grantTypeLabels.${grant}`)
}

function scopeDescription(scope: string) {
  return t(`adminClients.scopeDescriptions.${scope}`)
}

function tokenEndpointAuthMethodLabel(method: string) {
  return t(`adminClients.tokenEndpointAuthMethodLabels.${method}`)
}

function ownerLabel(client: Client) {
  return client.owner_email || client.owner_display_name || t('adminClients.noOwner')
}

function formatDateTime(value: string) {
  return value ? new Date(value).toLocaleString() : '-'
}
</script>

<template>
  <div>
    <div class="flex items-start justify-between gap-4 mb-6">
      <div>
        <h2 class="text-lg font-semibold">{{ $t('adminClients.title') }}</h2>
        <p class="text-sm text-muted-foreground mt-1">{{ $t('adminClients.subtitle') }}</p>
      </div>
      <button @click="openCreate" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors flex items-center gap-2">
        <Plus class="w-4 h-4" /> {{ $t('adminClients.createClient') }}
      </button>
    </div>

    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <div v-if="loading && clients.length === 0" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <div v-else class="border border-border rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
          <tr>
            <th class="px-4 py-3">{{ $t('adminClients.clientName') }}</th>
            <th class="px-4 py-3">{{ $t('adminClients.clientId') }}</th>
            <th class="px-4 py-3">{{ $t('adminClients.owner') }}</th>
            <th class="px-4 py-3">{{ $t('adminClients.isActive') }}</th>
            <th class="px-4 py-3">{{ $t('adminClients.updatedAt') }}</th>
            <th class="px-4 py-3 w-56 text-right">{{ $t('actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-if="clients.length === 0">
            <td colspan="6" class="px-4 py-8 text-center text-muted-foreground">{{ $t('adminClients.noClients') }}</td>
          </tr>
          <tr v-for="client in clients" :key="client.id" class="hover:bg-muted/30 transition-colors">
            <td class="px-4 py-3">
              <div class="flex items-center gap-3 min-w-0">
                <img v-if="client.logo_url" :src="client.logo_url" :alt="client.client_name" class="w-9 h-9 rounded-lg object-cover border border-border shrink-0" />
                <div v-else class="w-9 h-9 rounded-lg bg-muted shrink-0"></div>
                <div class="min-w-0">
                  <div class="font-medium truncate max-w-72">{{ client.client_name }}</div>
                  <div class="text-xs text-muted-foreground truncate max-w-72">{{ client.description || $t('adminClients.noDescription') }}</div>
                </div>
              </div>
            </td>
            <td class="px-4 py-3 font-mono text-xs text-muted-foreground max-w-56 truncate">{{ client.client_id }}</td>
            <td class="px-4 py-3 text-muted-foreground text-xs">
              <div class="truncate max-w-56">{{ ownerLabel(client) }}</div>
              <div v-if="client.owner_uid" class="font-mono truncate max-w-56">{{ $t('adminUsers.uid') }} {{ client.owner_uid }}</div>
            </td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium" :class="client.is_active ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'">
                {{ client.is_active ? $t('adminProviders.enabled') : $t('adminProviders.disabled') }}
              </span>
            </td>
            <td class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">{{ formatDateTime(client.updated_at || client.created_at) }}</td>
            <td class="px-4 py-3">
              <div class="flex items-center justify-end gap-1 flex-wrap">
                <button @click="openEdit(client)" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <Pencil class="w-3 h-3" /> {{ $t('edit') }}
                </button>
                <button @click="toggleClientActive(client)" :disabled="togglingClientId === client.id" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <Loader2 v-if="togglingClientId === client.id" class="w-3 h-3 animate-spin" />
                  <Power v-else class="w-3 h-3" /> {{ client.is_active ? $t('adminClients.disable') : $t('adminClients.enable') }}
                </button>
                <button @click="openUsers(client)" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <Users class="w-3 h-3" /> {{ $t('adminClients.authorizedUsers') }}
                </button>
                <button v-if="client.is_confidential" @click="confirmRotate(client)" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <RefreshCw class="w-3 h-3" /> {{ $t('adminClients.rotateSecret') }}
                </button>
                <button @click="confirmDelete(client)" class="text-xs font-medium px-2 py-1 rounded hover:bg-destructive/5 transition-colors text-destructive flex items-center gap-1">
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

    <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-4xl mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-5">
          <h2 class="text-lg font-semibold">{{ isCreate ? $t('adminClients.createClient') : $t('adminClients.editClient') }}</h2>
          <button @click="showModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="saveClient" class="flex flex-col gap-5">
          <section v-if="!isCreate && editingClient" class="rounded-xl border border-border bg-muted/20 p-4 space-y-3">
            <h3 class="text-sm font-semibold">{{ $t('adminClients.detailSection') }}</h3>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
              <div>
                <div class="text-xs text-muted-foreground mb-1">{{ $t('adminClients.clientId') }}</div>
                <div class="font-mono text-xs break-all">{{ editingClient.client_id }}</div>
              </div>
              <div>
                <div class="text-xs text-muted-foreground mb-1">{{ $t('adminClients.owner') }}</div>
                <div>{{ ownerLabel(editingClient) }}</div>
                <div v-if="editingClient.owner_uid" class="font-mono text-xs text-muted-foreground break-all">{{ $t('adminUsers.uid') }} {{ editingClient.owner_uid }}</div>
              </div>
              <div v-if="editingClient.client_secret" class="md:col-span-2">
                <div class="text-xs text-muted-foreground mb-1">{{ $t('adminClients.clientSecret') }}</div>
                <div class="flex items-center gap-2 bg-white rounded-lg border border-border p-3">
                  <code class="flex-1 text-xs font-mono break-all">{{ editingClient.client_secret }}</code>
                  <button type="button" @click="copyClientSecret(editingClient.client_secret)" class="shrink-0 p-2 rounded-lg hover:bg-muted transition-colors">
                    <Check v-if="detailSecretCopied" class="w-4 h-4 text-green-600" />
                    <Copy v-else class="w-4 h-4 text-muted-foreground" />
                  </button>
                </div>
              </div>
              <div>
                <div class="text-xs text-muted-foreground mb-1">{{ $t('adminClients.createdAt') }}</div>
                <div>{{ formatDateTime(editingClient.created_at) }}</div>
              </div>
              <div>
                <div class="text-xs text-muted-foreground mb-1">{{ $t('adminClients.updatedAt') }}</div>
                <div>{{ formatDateTime(editingClient.updated_at) }}</div>
              </div>
            </div>
          </section>

          <section class="space-y-4">
            <h3 class="text-sm font-semibold">{{ $t('adminClients.basicSection') }}</h3>
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.clientName') }}</label>
              <input v-model="form.client_name" type="text" required :placeholder="$t('adminClients.clientNamePlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
              <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.clientNameHint') }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.description') }}</label>
              <input v-model="form.description" type="text" :placeholder="$t('adminClients.descriptionPlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.logoUrl') }}</label>
                <input v-model="form.logo_url" type="url" placeholder="https://example.com/logo.png" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
              </div>
              <div>
                <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.homepageUrl') }}</label>
                <input v-model="form.homepage_url" type="url" placeholder="https://example.com" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
              </div>
            </div>
          </section>

          <section class="space-y-4">
            <h3 class="text-sm font-semibold">{{ $t('adminClients.oauthSection') }}</h3>
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.redirectUris') }}</label>
              <textarea v-model="form.redirect_uris" rows="3" placeholder="https://app.example.com/callback" class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-none" />
              <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.redirectUrisHint') }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.postLogoutRedirectUris') }}</label>
              <textarea v-model="form.post_logout_redirect_uris" rows="2" placeholder="https://app.example.com/logout-callback" class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-none" />
              <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.postLogoutRedirectUrisHint') }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.grantTypes') }}</label>
              <div class="grid grid-cols-1 md:grid-cols-3 gap-2">
                <label v-for="grant in allGrantTypes" :key="grant" class="flex items-center justify-between gap-3 text-sm cursor-pointer px-3 py-2 border rounded-lg transition-colors" :class="form.grant_types.includes(grant) ? 'border-foreground bg-foreground/5' : 'border-border'">
                  <span>{{ grantLabel(grant) }}</span>
                  <input type="checkbox" :checked="form.grant_types.includes(grant)" @change="toggleGrant(grant)" class="rounded border-border" />
                </label>
              </div>
              <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.grantTypesHint') }}</p>
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.responseTypes') }}</label>
                <input v-model="form.response_types" type="text" placeholder="code" class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10" />
                <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.responseTypesHint') }}</p>
              </div>
              <div>
                <label class="block text-sm font-medium mb-2">{{ $t('adminClients.scopes') }}</label>
                <div class="flex flex-col gap-2">
                  <label
                    v-for="scope in allScopes"
                    :key="scope"
                    class="flex items-center gap-3 text-sm cursor-pointer px-3 py-2.5 border rounded-lg transition-colors"
                    :class="form.scopes.includes(scope) ? 'border-foreground bg-foreground/5' : 'border-border'"
                  >
                    <input type="checkbox" :checked="form.scopes.includes(scope)" @change="toggleScope(scope)" class="sr-only" />
                    <div class="flex flex-col">
                      <span class="font-mono text-xs">{{ scope }}</span>
                      <span class="text-xs text-muted-foreground">{{ scopeDescription(scope) }}</span>
                    </div>
                  </label>
                </div>
              </div>
            </div>
            <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.protocol') }}</label>
                <select v-model="form.protocol_type" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10">
                  <option value="oidc">OIDC</option>
                  <option value="oauth2">OAuth2</option>
                </select>
              </div>
              <div>
                <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.tokenEndpointAuthMethod') }}</label>
                <select v-model="form.token_endpoint_auth_method" :disabled="!form.is_confidential" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10 disabled:bg-muted/40 disabled:text-muted-foreground">
                  <option v-if="!form.is_confidential" value="none">none</option>
                  <option v-for="method in allTokenEndpointAuthMethods" :key="method" :value="method">{{ tokenEndpointAuthMethodLabel(method) }}</option>
                </select>
              </div>
              <div>
                <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.minSecurityLevel') }}</label>
                <input v-model.number="form.min_security_level" type="number" min="0" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
                <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.minSecurityLevelHint') }}</p>
              </div>
            </div>
          </section>

          <section class="space-y-3">
            <h3 class="text-sm font-semibold">{{ $t('adminClients.policySection') }}</h3>
            <label class="flex items-center gap-2 text-sm font-medium cursor-pointer">
              <input type="checkbox" v-model="form.require_email_verified" class="rounded border-border" /> {{ $t('adminClients.requireEmailVerified') }}
            </label>
            <label class="flex items-center gap-2 text-sm font-medium cursor-pointer">
              <input type="checkbox" v-model="form.is_confidential" @change="normalizeClientType" class="rounded border-border" /> {{ $t('adminClients.confidential') }}
            </label>
            <p class="text-xs text-muted-foreground">{{ $t('adminClients.confidentialHint') }}</p>
            <label v-if="!isCreate" class="flex items-center gap-2 text-sm font-medium cursor-pointer">
              <input type="checkbox" v-model="form.is_active" class="rounded border-border" /> {{ $t('adminClients.isActive') }}
            </label>
          </section>

          <div class="flex justify-end gap-2 mt-2">
            <button type="button" @click="showModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
            <button type="submit" :disabled="saving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="saving" class="w-4 h-4 animate-spin" /> {{ isCreate ? $t('adminClients.createClient') : $t('save') }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="usersModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="usersModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-6xl mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-start justify-between gap-4 mb-5">
          <div>
            <h2 class="text-lg font-semibold">{{ $t('adminClients.authorizedUsers') }}</h2>
            <p class="text-sm text-muted-foreground mt-1">{{ usersClient?.client_name }}</p>
          </div>
          <button @click="usersModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>

        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between mb-4">
          <div class="relative w-full sm:max-w-sm">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
            <input
              v-model="clientUsersSearch"
              type="text"
              :placeholder="$t('adminClients.searchUsersPlaceholder')"
              class="w-full pl-9 pr-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
              @keyup.enter="searchClientUsers"
            />
          </div>
          <button @click="searchClientUsers" :disabled="clientUsersLoading" class="inline-flex items-center gap-2 px-3 py-2 border border-border rounded-lg text-sm font-medium hover:bg-muted transition-colors disabled:opacity-50">
            <Loader2 v-if="clientUsersLoading" class="w-4 h-4 animate-spin" />
            <Search v-else class="w-4 h-4" />
            {{ $t('search') }}
          </button>
        </div>

        <div v-if="clientUsersLoading && clientUsers.length === 0" class="flex items-center justify-center py-12 text-muted-foreground">
          <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
        </div>
        <div v-else class="border border-border rounded-xl overflow-hidden">
          <div class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead class="bg-muted/50 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                <tr>
                  <th class="px-4 py-3">{{ $t('adminClients.userUid') }}</th>
                  <th class="px-4 py-3">{{ $t('adminUsers.name') }}</th>
                  <th class="px-4 py-3">{{ $t('adminUsers.email') }}</th>
                  <th class="px-4 py-3">{{ $t('adminClients.securityLevel') }}</th>
                  <th class="px-4 py-3">{{ $t('adminClients.providers') }}</th>
                  <th class="px-4 py-3">{{ $t('adminClients.authorizationStatus') }}</th>
                  <th class="px-4 py-3">{{ $t('adminClients.lastUsedAt') }}</th>
                  <th class="px-4 py-3 text-right">{{ $t('actions') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                <tr v-if="clientUsers.length === 0">
                  <td colspan="8" class="px-4 py-10 text-center text-muted-foreground">{{ $t('adminClients.noAuthorizedUsers') }}</td>
                </tr>
                <tr v-for="user in clientUsers" :key="user.id" class="hover:bg-muted/30 transition-colors">
                  <td class="px-4 py-3 align-top font-mono text-xs whitespace-nowrap">{{ user.uid }}</td>
                  <td class="px-4 py-3 align-top font-medium whitespace-nowrap">{{ user.display_name || '-' }}</td>
                  <td class="px-4 py-3 align-top text-muted-foreground whitespace-nowrap">{{ user.email || '-' }}</td>
                  <td class="px-4 py-3 align-top whitespace-nowrap">
                    <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted">L{{ user.security_level }}</span>
                  </td>
                  <td class="px-4 py-3 align-top">
                    <div v-if="user.providers?.length" class="flex flex-wrap gap-1.5">
                      <span v-for="provider in user.providers" :key="provider" class="px-2 py-0.5 rounded-full bg-muted text-xs text-muted-foreground">{{ provider }}</span>
                    </div>
                    <span v-else class="text-muted-foreground">-</span>
                  </td>
                  <td class="px-4 py-3 align-top whitespace-nowrap">
                    <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium" :class="user.blocked ? 'bg-destructive/10 text-destructive' : 'bg-green-50 text-green-700'">
                      <Ban v-if="user.blocked" class="w-3 h-3" />
                      <CheckCircle2 v-else class="w-3 h-3" />
                      {{ user.blocked ? $t('adminClients.blocked') : $t('adminClients.allowed') }}
                    </span>
                  </td>
                  <td class="px-4 py-3 align-top text-muted-foreground whitespace-nowrap text-xs">{{ formatDateTime(user.last_used_at) }}</td>
                  <td class="px-4 py-3 align-top">
                    <div class="flex justify-end gap-2 flex-wrap">
                      <button
                        v-if="user.blocked"
                        @click="unblockClientUser(user)"
                        :disabled="clientUserActionId === user.id"
                        class="inline-flex items-center gap-1.5 px-3 py-1.5 border border-border rounded-lg text-xs font-medium hover:bg-muted transition-colors disabled:opacity-50"
                      >
                        <Loader2 v-if="clientUserActionId === user.id" class="w-3.5 h-3.5 animate-spin" />
                        {{ $t('adminClients.unblockUser') }}
                      </button>
                      <button
                        v-else
                        @click="blockClientUser(user)"
                        :disabled="clientUserActionId === user.id"
                        class="inline-flex items-center gap-1.5 px-3 py-1.5 border border-destructive/30 text-destructive rounded-lg text-xs font-medium hover:bg-destructive/5 transition-colors disabled:opacity-50"
                      >
                        <Loader2 v-if="clientUserActionId === user.id" class="w-3.5 h-3.5 animate-spin" />
                        {{ $t('adminClients.blockUser') }}
                      </button>
                      <button
                        @click="revokeClientUser(user)"
                        :disabled="clientUserActionId === user.id"
                        class="inline-flex items-center gap-1.5 px-3 py-1.5 border border-border rounded-lg text-xs font-medium hover:bg-muted transition-colors disabled:opacity-50"
                      >
                        <Loader2 v-if="clientUserActionId === user.id" class="w-3.5 h-3.5 animate-spin" />
                        {{ $t('adminClients.revokeAuthorization') }}
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-if="clientUsersTotal > 0" class="flex items-center justify-between mt-4 text-sm text-muted-foreground">
          <span>{{ $t('showing', { from: clientUsersFrom, to: clientUsersTo, total: clientUsersTotal }) }}</span>
          <div class="flex gap-2">
            <button @click="prevClientUsersPage" :disabled="clientUsersOffset === 0" class="px-3 py-1.5 border border-border rounded-lg text-sm hover:bg-muted transition-colors disabled:opacity-40">{{ $t('prev') }}</button>
            <button @click="nextClientUsersPage" :disabled="clientUsersOffset + clientUsersLimit >= clientUsersTotal" class="px-3 py-1.5 border border-border rounded-lg text-sm hover:bg-muted transition-colors disabled:opacity-40">{{ $t('next') }}</button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showRotateModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showRotateModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-md mx-4 p-6">
        <div class="flex items-start gap-3 mb-4">
          <AlertTriangle class="w-5 h-5 text-yellow-600 mt-0.5" />
          <div>
            <h2 class="text-lg font-semibold">{{ $t('adminClients.rotateSecret') }}</h2>
            <p class="text-sm text-muted-foreground mt-1">{{ $t('adminClients.rotateConfirm', { name: rotatingClient?.client_name || '' }) }}</p>
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <button @click="showRotateModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
          <button @click="rotateSecret" :disabled="rotating" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
            <Loader2 v-if="rotating" class="w-4 h-4 animate-spin" /> {{ $t('confirm') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showSecretModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4 p-6">
        <h2 class="text-lg font-semibold mb-2">{{ $t('adminClients.clientSecret') }}</h2>
        <p class="text-sm text-muted-foreground mb-4">{{ $t('adminClients.copySecretNow') }}</p>
        <div class="flex items-center gap-2 bg-muted rounded-lg p-3">
          <code class="flex-1 text-sm font-mono break-all">{{ revealedSecret }}</code>
          <button @click="copySecret" class="shrink-0 p-2 rounded-lg hover:bg-white transition-colors">
            <Check v-if="secretCopied" class="w-4 h-4 text-green-600" />
            <Copy v-else class="w-4 h-4 text-muted-foreground" />
          </button>
        </div>
        <div class="flex justify-end mt-5">
          <button @click="showSecretModal = false" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors">{{ $t('adminClients.done') }}</button>
        </div>
      </div>
    </div>

    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showDeleteModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <h2 class="text-lg font-semibold mb-2">{{ $t('adminClients.deleteClient') }}</h2>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('adminClients.deleteConfirm', { name: deletingClient?.client_name || '' }) }}</p>
        <div class="flex justify-end gap-2">
          <button @click="showDeleteModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
          <button @click="deleteClient" :disabled="deleting" class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2">
            <Loader2 v-if="deleting" class="w-4 h-4 animate-spin" /> {{ $t('delete') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
