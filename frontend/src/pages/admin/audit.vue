<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { api } from '@/api/client'
import { Loader2, Search } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { tm, locale } = useI18n()

interface AuditEntry {
  id: string
  action: string
  user_id: string | null
  user_uid?: number | string
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
  'user.register',
  'user.register_social',
  'user.email_verified',
  'user.login',
  'user.login_social',
  'user.login_failed',
  'user.logout',
  'user.password_changed',
  'user.password_reset_requested',
  'user.password_reset',
  'social.bound',
  'social.unbound',
  'social_bind',
  'social_unbind',
  'client.created',
  'client.updated',
  'client.deleted',
  'client.secret_rotated',
  'admin.user_created',
  'admin.user_updated',
  'admin.user_deleted',
  'admin.user_password_reset',
  'admin.user_passkey_deleted',
  'admin.provider_created',
  'admin.provider_updated',
  'admin.provider_deleted',
  'admin.signing_key_rotated',
  'admin.settings_updated',
  'security_level.changed',
  'security_level.upgraded',
  'security_level.downgraded',
  'risk.user_reported',
  'risk.app_reported',
  'risk.report_confirmed',
  'risk.report_dismissed',
  'risk.enforcement',
  'risk.list_added',
  'risk.list_removed',
  'risk.restored',
  'token_issue',
  'token_revoke',
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
  const displayLocale = String(locale.value).startsWith('en') ? 'en-US' : 'zh-CN'
  return new Date(d).toLocaleString(displayLocale, {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  })
}

function recordLabel(path: string, key: string | undefined | null, fallback = '-') {
  if (!key) return fallback
  const record = tm(path)
  if (record && typeof record === 'object') {
    const value = (record as Record<string, unknown>)[key]
    if (typeof value === 'string') return value
  }
  return key
}

function formatAction(action: string): string {
  return recordLabel('adminAudit.actions', action, action)
}

function formatResource(resourceType?: string): string {
  return recordLabel('adminAudit.resources', resourceType, '-')
}

function formatDetailKey(key: string): string {
  return recordLabel('adminAudit.detailKeys', key, key)
}

function formatDetailValue(value: any): string {
  if (value === null || value === undefined || value === '') return '-'
  const raw = String(value)
  return recordLabel('adminAudit.detailValues', raw, raw)
}

function formatDetails(entry: AuditEntry): string {
  const details = entry.details ?? {}
  const keys = Object.keys(details)
  if (keys.length === 0) return entry.details_text || '-'
  return keys
    .map(key => `${formatDetailKey(key)}=${formatDetailValue(details[key])}`)
    .join(', ')
}

function rawDetails(entry: AuditEntry): string {
  if (entry.details && Object.keys(entry.details).length > 0) {
    return JSON.stringify(entry.details, null, 2)
  }
  return entry.details_text || ''
}

function actorDisplay(entry: AuditEntry): string {
  if (entry.user_email) return entry.user_email
  return '-'
}

function uidDisplay(entry: AuditEntry): string {
  if (entry.user_uid !== undefined && entry.user_uid !== null) return String(entry.user_uid)
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

    <div class="flex flex-col sm:flex-row sm:items-center gap-3 mb-4">
      <select v-model="actionFilter" class="w-full sm:w-auto px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10">
        <option value="">{{ $t('adminAudit.allActions') }}</option>
        <option v-for="action in actionOptions" :key="action" :value="action">{{ formatAction(action) }}</option>
      </select>
      <div class="relative w-full sm:w-64">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
        <input v-model="userIdFilter" type="text" :placeholder="$t('adminAudit.filterByUser')" class="w-full pl-9 pr-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
      </div>
    </div>

    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{{ error }}</div>

    <div v-if="loading && entries.length === 0" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <div v-else>
      <div class="hidden md:block border border-border rounded-xl overflow-hidden">
        <table class="w-full text-sm table-fixed">
          <thead class="bg-muted/50 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
            <tr>
              <th class="px-4 py-3 w-40">{{ $t('adminAudit.time') }}</th>
              <th class="px-4 py-3 w-52">{{ $t('adminAudit.actor') }}</th>
              <th class="px-4 py-3 w-32">{{ $t('adminUsers.uid') }}</th>
              <th class="px-4 py-3 w-40">{{ $t('adminAudit.action') }}</th>
              <th class="px-4 py-3 w-44">{{ $t('adminAudit.resourceType') }}</th>
              <th class="px-4 py-3">{{ $t('adminAudit.details') }}</th>
              <th class="px-4 py-3 w-32">{{ $t('adminAudit.ip') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-if="entries.length === 0">
              <td colspan="7" class="px-4 py-8 text-center text-muted-foreground">{{ $t('adminAudit.noLogs') }}</td>
            </tr>
            <tr v-for="entry in entries" :key="entry.id" class="hover:bg-muted/30 transition-colors">
              <td class="px-4 py-3 text-muted-foreground whitespace-nowrap text-xs">{{ formatTimestamp(entry.created_at) }}</td>
              <td class="px-4 py-3 min-w-0">
                <div class="text-xs min-w-0">
                  <div class="font-medium truncate" :title="actorDisplay(entry)">{{ actorDisplay(entry) }}</div>
                </div>
              </td>
              <td class="px-4 py-3 text-muted-foreground text-xs font-mono truncate" :title="uidDisplay(entry)">{{ uidDisplay(entry) }}</td>
              <td class="px-4 py-3">
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted max-w-full truncate">{{ formatAction(entry.action) }}</span>
              </td>
              <td class="px-4 py-3 text-muted-foreground text-xs min-w-0">
                <div class="truncate">{{ formatResource(entry.resource_type) }}</div>
                <div v-if="entry.resource_id" class="font-mono text-[11px] mt-0.5 truncate" :title="entry.resource_id">
                  {{ entry.resource_id }}
                </div>
              </td>
              <td class="px-4 py-3 text-muted-foreground text-xs truncate" :title="rawDetails(entry)">{{ formatDetails(entry) }}</td>
              <td class="px-4 py-3 text-muted-foreground text-xs font-mono truncate">{{ entry.ip_address || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="md:hidden space-y-3">
        <div v-if="entries.length === 0" class="border border-border rounded-xl px-4 py-8 text-center text-muted-foreground text-sm">
          {{ $t('adminAudit.noLogs') }}
        </div>
        <div v-for="entry in entries" :key="entry.id" class="border border-border rounded-xl p-4 bg-background">
          <div class="flex flex-col gap-2">
            <span class="px-2 py-0.5 rounded-full text-xs font-medium bg-muted break-words min-w-0 w-fit max-w-full">{{ formatAction(entry.action) }}</span>
            <span class="text-xs text-muted-foreground break-words">{{ formatTimestamp(entry.created_at) }}</span>
          </div>
          <div class="mt-3 space-y-2 text-xs text-muted-foreground">
            <div class="break-words"><span class="font-medium text-foreground">{{ $t('adminAudit.actor') }}：</span>{{ actorDisplay(entry) }}</div>
            <div class="break-words"><span class="font-medium text-foreground">{{ $t('adminUsers.uid') }}：</span>{{ uidDisplay(entry) }}</div>
            <div class="break-words"><span class="font-medium text-foreground">{{ $t('adminAudit.resourceType') }}：</span>{{ formatResource(entry.resource_type) }}<span v-if="entry.resource_id"> / {{ entry.resource_id }}</span></div>
            <div class="break-words"><span class="font-medium text-foreground">{{ $t('adminAudit.details') }}：</span>{{ formatDetails(entry) }}</div>
            <div class="break-words"><span class="font-medium text-foreground">{{ $t('adminAudit.ip') }}：</span>{{ entry.ip_address || '-' }}</div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="total > 0" class="flex flex-col gap-3 mt-4 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
      <span>{{ $t('showing', { from: offset + 1, to: Math.min(offset + limit, total), total }) }}</span>
      <div class="flex gap-2">
        <button @click="prevPage" :disabled="offset === 0" class="px-3 py-1.5 border border-border rounded-lg text-sm hover:bg-muted transition-colors disabled:opacity-40">{{ $t('prev') }}</button>
        <button @click="nextPage" :disabled="offset + limit >= total" class="px-3 py-1.5 border border-border rounded-lg text-sm hover:bg-muted transition-colors disabled:opacity-40">{{ $t('next') }}</button>
      </div>
    </div>
  </div>
</template>
