<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { Loader2, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface AuthorizedApp {
  id: string
  client_id: string
  client_name: string
  scopes: string[]
  granted_at: string
}

const authorizedApps = ref<AuthorizedApp[]>([])
const loading = ref(false)
const revokeTarget = ref<AuthorizedApp | null>(null)
const showRevokeModal = ref(false)
const revoking = ref(false)

onMounted(loadApps)

async function loadApps() {
  loading.value = true
  try {
    const res = await api.get<AuthorizedApp[]>('/me/authorized-apps')
    authorizedApps.value = res.data || []
  } catch { /* ignore */ }
  loading.value = false
}

function confirmRevoke(app: AuthorizedApp) {
  revokeTarget.value = app
  showRevokeModal.value = true
}

async function revokeApp() {
  if (!revokeTarget.value) return
  revoking.value = true
  try {
    await api.del(`/me/authorized-apps/${revokeTarget.value.client_id}`)
    authorizedApps.value = authorizedApps.value.filter(a => a.client_id !== revokeTarget.value!.client_id)
    showRevokeModal.value = false
    revokeTarget.value = null
  } catch { /* ignore */ }
  revoking.value = false
}
</script>

<template>
  <div>
    <p class="text-sm text-muted-foreground mb-6">{{ $t('authorizedApps.desc') }}</p>

    <div v-if="loading" class="text-sm text-muted-foreground flex items-center gap-2">
      <Loader2 class="w-4 h-4 animate-spin" /> {{ $t('loading') }}
    </div>
    <div v-else-if="authorizedApps.length === 0" class="text-sm text-muted-foreground py-8 text-center">
      {{ $t('authorizedApps.noApps') }}
    </div>
    <div v-else class="space-y-3 max-w-lg">
      <div
        v-for="app in authorizedApps"
        :key="app.id"
        class="border border-border rounded-lg p-4 flex items-center justify-between"
      >
        <div class="flex-1 min-w-0">
          <div class="font-medium text-sm">{{ app.client_name }}</div>
          <div v-if="app.scopes && app.scopes.length" class="mt-1.5 flex flex-wrap gap-1">
            <span
              v-for="scope in app.scopes"
              :key="scope"
              class="px-2 py-0.5 text-xs bg-muted rounded"
            >{{ scope }}</span>
          </div>
          <div v-if="app.granted_at" class="mt-1.5 text-xs text-muted-foreground">
            {{ $t('authorizedApps.grantedAt') }}: {{ new Date(app.granted_at).toLocaleDateString() }}
          </div>
        </div>
        <button
          @click="confirmRevoke(app)"
          class="shrink-0 ml-4 px-3 py-1.5 text-xs font-medium text-destructive border border-destructive/30 rounded-lg hover:bg-destructive/5 transition-colors"
        >
          {{ $t('authorizedApps.revoke') }}
        </button>
      </div>
    </div>

    <!-- Revoke Confirmation Modal -->
    <div v-if="showRevokeModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showRevokeModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('authorizedApps.revoke') }}</h2>
          <button @click="showRevokeModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <p class="text-sm text-muted-foreground mb-1">
          <strong>{{ revokeTarget?.client_name }}</strong>
        </p>
        <p class="text-sm text-muted-foreground mb-5">
          {{ $t('authorizedApps.revokeConfirm') }}
        </p>
        <div class="flex justify-end gap-2">
          <button @click="showRevokeModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">
            {{ $t('cancel') }}
          </button>
          <button @click="revokeApp" :disabled="revoking" class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2">
            <Loader2 v-if="revoking" class="w-4 h-4 animate-spin" />
            {{ $t('authorizedApps.revoke') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
