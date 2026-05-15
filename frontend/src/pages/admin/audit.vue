<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { api } from '@/api/client'
import { Loader2, Search } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface AuditEntry {
  id: string
  action: string
  user_id: string | null
  user_email?: string
  user_display_name?: string
  resource_type?: string
  resource_id?: string
  ip_address?: string
  details?: Record<string, any>
  details_text?: string
  created_at: string
}

const entries = ref<AuditEntry[]>([])
const total = ref(0)
const offset = ref(0)
const limit = ref(30)
const loading = ref(false)
const error = ref('')

const actionFilter = ref('')
const userIdFilter = ref('')

const actionOptions = [
  'user.register', 'user.login', 'user.logout', 'user.password_changed',
  'social_bind', 'social_unbind',
  'client.created', 'client.updated', 'client.deleted', 'client.secret_rotated',
  'admin.user_created', 'admin.user_updated', 'admin.user_deleted',
  'admin.provider_updated', 'admin.signing_key_rotated', 'admin.settings_updated',
  'security_level.changed',
  'token_issue', 'token_revoke',
]

let filterTimer: ReturnType<typeof setTimeout> | null = null

watch([actionFilter], () => {
  offset.value = 0
  fetchAudit()
})

watch(userIdFilter, () => {
  if (filterTimer) clearTimeout(filterTimer)
  filterTimer = setTimeout(() => {
    offset.value = 0
    fetchAudit()
  }, 300)
})

onMounted(fetchAudit)

async function fetchAudit() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({
      offset: String(offset.value),
      limit: String(limit.value),
    })
    if (actionFilter.value) params.set('action', actionFilter.value)
    if (userIdFilter.value) params.set('user_id', userIdFilter.value)
    const res = await api.get<AuditEntry[]>(`/admin/audit-log?${params}`)
    entries.value = res.data ?? []
    total.value = res.meta?.total ?? 0
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function prevPage() {
  if (offset.value > 0) {
    offset.value = Math.max(0, offset.value - limit.value)
    fetchAudit()
  }
}

function nextPage() {
  if (offset.value + limit.value < total.value) {
    offset.value += limit.value
    fetchAudit()
  }
}

function formatTimestamp(d: string) {
  return new Date(d).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}

function actionLabel(action: string): string {
  const key = `adminOverview.actions.${action}`
  const translated = t(key)
  return translated !== key ? translated : action
}

function actorDisplay(entry: AuditEntry): string {
  if (entry.user_email) return entry.user_email
  if (entry.user_id) return entry.user_id
  return '-'
}
</script>

<template>
  <div>
    <div class="flex items-start justify-between gap-4 mb-6">
      <div>
        <h2 class="text-lg font-semibold">{{ $t('adminAudit.title') }}</h2>
        <p class="text-sm text-muted-foreground mt-1">{{ $t('adminAudit.subtitle') }}</p>
      </div>
    </div>

    <div class="flex items-center gap-3 mb-4">
      <select v-model="actionFilter" class="px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10">
        <option value="">{{ $t('adminAudit.allActions') }}</option>
        <option v-for="action in actionOptions" :key="action" :value="action">{{ actionLabel(action) }}</option>
      </select>
      <div class="relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
        <input v-model="userIdFilter" type="text" :placeholder="$t('adminAudit.filterByUser')" class="pl-9 pr-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10 w-64" />
      </div>
    </div>

    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{{ error }}</div>

    <div v-if="loading && entries.length === 0" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <div v-else class="border border-border rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
          <tr>
            <th class="px-4 py-3">{{ $t('adminAudit.time') }}</th>
            <th class="px-4 py-3">{{ $t('adminAudit.actor') }}</th>
            <th class="px-4 py-3">{{ $t('adminAudit.action') }}</th>
            <th class="px-4 py-3">{{ $t('adminAudit.details') }}</th>
            <th class="px-4 py-3">{{ $t('adminAudit.ip') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-if="entries.length === 0">
            <td colspan="5" class="px-4 py-8 text-center text-muted-foreground">{{ $t('adminAudit.noLogs') }}</td>
          </tr>
          <tr v-for="entry in entries" :key="entry.id" class="hover:bg-muted/30 transition-colors">
            <td class="px-4 py-3 text-muted-foreground whitespace-nowrap text-xs">{{ formatTimestamp(entry.created_at) }}</td>
            <td class="px-4 py-3">
              <div class="text-xs">
                <span v-if="entry.user_email" class="font-medium">{{ entry.user_email }}</span>
                <span v-else class="text-muted-foreground font-mono">{{ entry.user_id || '-' }}</span>
              </div>
            </td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted">{{ actionLabel(entry.action) }}</span>
            </td>
            <td class="px-4 py-3 text-muted-foreground text-xs max-w-80 truncate">{{ entry.details_text || '-' }}</td>
            <td class="px-4 py-3 text-muted-foreground text-xs font-mono">{{ entry.ip_address || '-' }}</td>
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
  </div>
</template>
