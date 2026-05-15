<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { Users, Monitor, Activity, Loader2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface AuditEvent {
  id: string
  action: string
  user_id: string | null
  user_email?: string
  user_display_name?: string
  resource_type?: string
  details_text?: string
  created_at: string
}

interface Stats {
  total_users: number
  total_clients: number
  total_sessions: number
  recent_events: AuditEvent[]
}

const stats = ref<Stats | null>(null)
const loading = ref(false)
const error = ref('')

async function fetchStats() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<Stats>('/admin/stats')
    stats.value = res.data ?? null
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function formatTime(iso: string) {
  return new Date(iso).toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  })
}

function actionLabel(action: string): string {
  const key = `adminOverview.actions.${action}`
  const translated = t(key)
  return translated !== key ? translated : action
}

const cards = [
  { key: 'total_users', labelKey: 'adminOverview.totalUsers', icon: Users },
  { key: 'total_clients', labelKey: 'adminOverview.totalClients', icon: Monitor },
  { key: 'total_sessions', labelKey: 'adminOverview.activeSessions', icon: Activity },
] as const

onMounted(fetchStats)
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-lg font-semibold">{{ $t('adminOverview.title') }}</h2>
      <p class="text-sm text-muted-foreground mt-1">{{ $t('adminOverview.subtitle') }}</p>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-20 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <div v-else-if="error" class="text-center py-20 text-destructive">{{ error }}</div>

    <template v-else-if="stats">
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-5 mb-10">
        <div v-for="card in cards" :key="card.key" class="relative border border-border rounded-xl p-6 bg-background">
          <component :is="card.icon" class="absolute top-5 right-5 w-5 h-5 text-muted-foreground/50" />
          <div class="text-3xl font-bold tracking-tight">{{ (stats as any)[card.key]?.toLocaleString() ?? '—' }}</div>
          <div class="text-sm text-muted-foreground mt-1">{{ t(card.labelKey) }}</div>
        </div>
      </div>

      <div>
        <h2 class="text-lg font-semibold mb-4">{{ t('adminOverview.recentActivity') }}</h2>

        <div v-if="!stats.recent_events || stats.recent_events.length === 0" class="border border-dashed border-border rounded-xl py-8 text-center text-muted-foreground text-sm">
          {{ t('adminOverview.noActivity') }}
        </div>

        <div v-else class="border border-border rounded-xl overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border bg-muted/40">
                <th class="text-left px-4 py-3 font-medium text-muted-foreground">{{ t('adminAudit.time') }}</th>
                <th class="text-left px-4 py-3 font-medium text-muted-foreground">{{ t('adminAudit.actor') }}</th>
                <th class="text-left px-4 py-3 font-medium text-muted-foreground">{{ t('adminAudit.action') }}</th>
                <th class="text-left px-4 py-3 font-medium text-muted-foreground">{{ t('adminAudit.details') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="event in stats.recent_events" :key="event.id" class="border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors">
                <td class="px-4 py-3 text-muted-foreground whitespace-nowrap text-xs">{{ formatTime(event.created_at) }}</td>
                <td class="px-4 py-3 text-xs">
                  <span v-if="event.user_email" class="font-medium">{{ event.user_email }}</span>
                  <span v-else class="text-muted-foreground">-</span>
                </td>
                <td class="px-4 py-3">
                  <span class="inline-block px-2 py-0.5 rounded-md bg-muted text-xs font-medium">{{ actionLabel(event.action) }}</span>
                </td>
                <td class="px-4 py-3 text-muted-foreground text-xs max-w-64 truncate">{{ event.details_text || '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>
