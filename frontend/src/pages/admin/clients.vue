<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { Plus, Pencil, Trash2, Loader2, RefreshCw, X, Copy, Check, AlertTriangle } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface Client {
  id: string
  client_id: string
  client_name: string
  description?: string
  owner_user_id?: string
  owner_email?: string
  owner_display_name?: string
  client_secret?: string
  redirect_uris: string[]
  grant_types: string[]
  scopes: string[]
  min_security_level: number
  protocol_type: string
  is_confidential: boolean
  is_active: boolean
  created_at: string
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
  redirect_uris: '',
  grant_types: [] as string[],
  scopes: '',
  min_security_level: 0,
  protocol_type: 'oidc',
  is_confidential: true,
  is_active: true,
})

const allGrantTypes = ['authorization_code', 'refresh_token', 'client_credentials', 'implicit', 'device_code']

const showSecretModal = ref(false)
const revealedSecret = ref('')
const secretCopied = ref(false)

const showRotateModal = ref(false)
const rotatingClient = ref<Client | null>(null)
const rotating = ref(false)

const showDeleteModal = ref(false)
const deletingClient = ref<Client | null>(null)
const deleting = ref(false)

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
    redirect_uris: '',
    grant_types: ['authorization_code', 'refresh_token'],
    scopes: 'openid profile email',
    min_security_level: 0,
    protocol_type: 'oidc',
    is_confidential: true,
    is_active: true,
  }
  showModal.value = true
}

function openEdit(client: Client) {
  isCreate.value = false
  editingClient.value = client
  form.value = {
    client_name: client.client_name,
    description: client.description || '',
    redirect_uris: (client.redirect_uris || []).join(', '),
    grant_types: [...(client.grant_types || [])],
    scopes: (client.scopes || []).join(' '),
    min_security_level: client.min_security_level,
    protocol_type: client.protocol_type,
    is_confidential: client.is_confidential,
    is_active: client.is_active,
  }
  showModal.value = true
}

function toggleGrant(grant: string) {
  const idx = form.value.grant_types.indexOf(grant)
  if (idx >= 0) form.value.grant_types.splice(idx, 1)
  else form.value.grant_types.push(grant)
}

async function saveClient() {
  saving.value = true
  error.value = ''
  try {
    const payload = {
      client_name: form.value.client_name,
      description: form.value.description,
      redirect_uris: form.value.redirect_uris.split(',').map(s => s.trim()).filter(Boolean),
      grant_types: form.value.grant_types,
      scopes: form.value.scopes.split(/[\s,]+/).filter(Boolean),
      min_security_level: form.value.min_security_level,
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

function ownerLabel(client: Client) {
  return client.owner_email || client.owner_display_name || t('adminClients.noOwner')
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

    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 mb-4">
      <div class="rounded-xl border border-border bg-muted/30 p-4 text-sm">
        <div class="font-medium mb-1">{{ $t('adminClients.adminScopeTitle') }}</div>
        <p class="text-muted-foreground">{{ $t('adminClients.adminScopeDesc') }}</p>
      </div>
      <div class="rounded-xl border border-border bg-muted/30 p-4 text-sm">
        <div class="font-medium mb-1">{{ $t('adminClients.developerScopeTitle') }}</div>
        <p class="text-muted-foreground">{{ $t('adminClients.developerScopeDesc') }}</p>
      </div>
      <div class="rounded-xl border border-border bg-muted/30 p-4 text-sm">
        <div class="font-medium mb-1">{{ $t('adminClients.securityScopeTitle') }}</div>
        <p class="text-muted-foreground">{{ $t('adminClients.securityScopeDesc') }}</p>
      </div>
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
            <th class="px-4 py-3">{{ $t('adminClients.owner') }}</th>
            <th class="px-4 py-3">{{ $t('adminClients.clientId') }}</th>
            <th class="px-4 py-3">{{ $t('adminClients.grantTypes') }}</th>
            <th class="px-4 py-3">{{ $t('adminClients.minSecurityLevel') }}</th>
            <th class="px-4 py-3">{{ $t('adminClients.isActive') }}</th>
            <th class="px-4 py-3">{{ $t('actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-if="clients.length === 0">
            <td colspan="7" class="px-4 py-8 text-center text-muted-foreground">{{ $t('adminClients.noClients') }}</td>
          </tr>
          <tr v-for="client in clients" :key="client.id" class="hover:bg-muted/30 transition-colors">
            <td class="px-4 py-3">
              <div class="font-medium">{{ client.client_name }}</div>
              <div class="text-xs text-muted-foreground max-w-56 truncate">{{ client.description || $t('adminClients.noDescription') }}</div>
            </td>
            <td class="px-4 py-3 text-muted-foreground text-xs">{{ ownerLabel(client) }}</td>
            <td class="px-4 py-3 font-mono text-xs text-muted-foreground">{{ client.client_id }}</td>
            <td class="px-4 py-3">
              <div class="flex flex-wrap gap-1">
                <span v-for="g in client.grant_types" :key="g" class="text-xs bg-muted px-1.5 py-0.5 rounded">{{ grantLabel(g) }}</span>
              </div>
            </td>
            <td class="px-4 py-3 text-muted-foreground">L{{ client.min_security_level }}</td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium" :class="client.is_active ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'">
                {{ client.is_active ? $t('adminProviders.enabled') : $t('adminProviders.disabled') }}
              </span>
            </td>
            <td class="px-4 py-3">
              <div class="flex items-center gap-1 flex-wrap">
                <button @click="openEdit(client)" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <Pencil class="w-3 h-3" /> {{ $t('edit') }}
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
      <div class="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-5">
          <h2 class="text-lg font-semibold">{{ isCreate ? $t('adminClients.createClient') : $t('adminClients.editClient') }}</h2>
          <button @click="showModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="saveClient" class="flex flex-col gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.clientName') }}</label>
            <input v-model="form.client_name" type="text" required :placeholder="$t('adminClients.clientNamePlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
            <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.clientNameHint') }}</p>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.description') }}</label>
            <input v-model="form.description" type="text" :placeholder="$t('adminClients.descriptionPlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.redirectUris') }}</label>
            <input v-model="form.redirect_uris" type="text" placeholder="https://app.example.com/callback" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
            <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.redirectUrisHint') }}</p>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.grantTypes') }}</label>
            <div class="flex flex-col gap-2">
              <label v-for="grant in allGrantTypes" :key="grant" class="flex items-center justify-between gap-3 text-sm cursor-pointer px-3 py-2 border rounded-lg transition-colors" :class="form.grant_types.includes(grant) ? 'border-foreground bg-foreground/5' : 'border-border'">
                <span>{{ grantLabel(grant) }}</span>
                <input type="checkbox" :checked="form.grant_types.includes(grant)" @change="toggleGrant(grant)" class="rounded border-border" />
              </label>
            </div>
            <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.grantTypesHint') }}</p>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.scopes') }}</label>
            <input v-model="form.scopes" type="text" placeholder="openid profile email" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
            <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.scopesHint') }}</p>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.minSecurityLevel') }}</label>
              <input v-model.number="form.min_security_level" type="number" min="0" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
              <p class="text-xs text-muted-foreground mt-1">{{ $t('adminClients.minSecurityLevelHint') }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ $t('adminClients.protocol') }}</label>
              <select v-model="form.protocol_type" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10">
                <option value="oidc">OIDC</option>
                <option value="oauth2">OAuth2</option>
              </select>
            </div>
          </div>
          <div class="space-y-2">
            <label class="flex items-center gap-2 text-sm font-medium cursor-pointer">
              <input type="checkbox" v-model="form.is_confidential" class="rounded border-border" /> {{ $t('adminClients.confidential') }}
            </label>
            <p class="text-xs text-muted-foreground">{{ $t('adminClients.confidentialHint') }}</p>
            <label v-if="!isCreate" class="flex items-center gap-2 text-sm font-medium cursor-pointer">
              <input type="checkbox" v-model="form.is_active" class="rounded border-border" /> {{ $t('adminClients.isActive') }}
            </label>
          </div>
          <div class="flex justify-end gap-2 mt-2">
            <button type="button" @click="showModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
            <button type="submit" :disabled="saving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="saving" class="w-4 h-4 animate-spin" /> {{ isCreate ? $t('adminClients.createClient') : $t('save') }}
            </button>
          </div>
        </form>
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
