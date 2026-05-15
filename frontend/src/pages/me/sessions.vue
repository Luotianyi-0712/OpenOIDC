<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { Loader2, MonitorSmartphone, Trash2, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface Session {
  id: string
  ip: string
  user_agent: string
  expires_at: string
  created_at: string
}

const sessions = ref<Session[]>([])
const loading = ref(true)
const error = ref('')
const revokingId = ref<string | null>(null)

const showRevokeModal = ref(false)
const revokeTarget = ref<string | null>(null)

onMounted(() => {
  fetchSessions()
})

async function fetchSessions() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<Session[]>('/me/sessions')
    sessions.value = res.data || []
  } catch (e: any) {
    error.value = e.message || 'Failed to load sessions.'
  } finally {
    loading.value = false
  }
}

function confirmRevoke(id: string) {
  revokeTarget.value = id
  showRevokeModal.value = true
}

async function doRevoke() {
  if (!revokeTarget.value) return
  const id = revokeTarget.value
  showRevokeModal.value = false
  revokingId.value = id
  try {
    await api.del(`/me/sessions/${id}`)
    sessions.value = sessions.value.filter((s) => s.id !== id)
  } catch (e: any) {
    error.value = e.message
  } finally {
    revokingId.value = null
  }
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function truncateUA(ua: string, max = 60) {
  return ua.length > max ? ua.slice(0, max) + '...' : ua
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-lg font-semibold">{{ $t('sessions.title') }}</h2>
        <p class="text-sm text-muted-foreground mt-0.5">
          {{ $t('sessions.desc') }}
        </p>
      </div>
      <button
        v-if="!loading && sessions.length"
        @click="fetchSessions"
        class="px-3 py-1.5 text-xs font-medium border border-border rounded-lg hover:bg-muted transition-colors"
      >
        {{ $t('refresh') }}
      </button>
    </div>

    <div v-if="loading" class="flex items-center gap-2 text-sm text-muted-foreground py-12 justify-center">
      <Loader2 class="w-4 h-4 animate-spin" /> {{ $t('sessions.loadingSessions') }}
    </div>

    <div
      v-else-if="error"
      class="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
    >
      {{ error }}
    </div>

    <div v-else-if="!sessions.length" class="text-sm text-muted-foreground py-12 text-center">
      {{ $t('sessions.noSessions') }}
    </div>

    <div v-else class="border border-border rounded-xl overflow-hidden">
      <div class="grid grid-cols-[1fr_2fr_1fr_1fr_auto] gap-4 px-5 py-3 bg-muted/50 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        <div>{{ $t('sessions.ip') }}</div>
        <div>{{ $t('sessions.userAgent') }}</div>
        <div>{{ $t('sessions.created') }}</div>
        <div>{{ $t('sessions.expires') }}</div>
        <div class="w-20"></div>
      </div>
      <div
        v-for="session in sessions"
        :key="session.id"
        class="grid grid-cols-[1fr_2fr_1fr_1fr_auto] gap-4 px-5 py-3.5 border-t border-border text-sm items-center"
      >
        <div class="font-mono text-xs">{{ session.ip }}</div>
        <div class="text-muted-foreground text-xs truncate" :title="session.user_agent">
          {{ truncateUA(session.user_agent) }}
        </div>
        <div class="text-xs text-muted-foreground">{{ formatDate(session.created_at) }}</div>
        <div class="text-xs text-muted-foreground">{{ formatDate(session.expires_at) }}</div>
        <div class="w-20 flex justify-end">
          <button
            @click="confirmRevoke(session.id)"
            :disabled="revokingId === session.id"
            class="px-3 py-1.5 text-xs font-medium border rounded-lg transition-colors flex items-center gap-1.5 text-destructive border-destructive/30 hover:bg-destructive/5 disabled:opacity-50"
          >
            <Loader2 v-if="revokingId === session.id" class="w-3 h-3 animate-spin" />
            <Trash2 v-else class="w-3 h-3" />
            {{ $t('sessions.revoke') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Revoke Confirm Modal -->
    <div v-if="showRevokeModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showRevokeModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('sessions.revoke') }}</h2>
          <button @click="showRevokeModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('sessions.revokeConfirm') }}</p>
        <div class="flex justify-end gap-2">
          <button @click="showRevokeModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
          <button @click="doRevoke" class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors">{{ $t('confirm') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>
