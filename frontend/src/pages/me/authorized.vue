<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { Loader2, X, User, Building2, Link as LinkIcon, ExternalLink, Flag } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const toast = useToast()

interface AuthorizedApp {
  id: string
  client_id: string
  client_name: string
  description: string
  logo_url: string
  homepage_url: string
  developer: {
    id: string
    uid: number
    display_name: string
    avatar_url: string
  }
  scopes: string[]
  granted_at: string
}

const authorizedApps = ref<AuthorizedApp[]>([])
const loading = ref(false)
const revokeTarget = ref<AuthorizedApp | null>(null)
const showRevokeModal = ref(false)
const revoking = ref(false)
const detailTarget = ref<AuthorizedApp | null>(null)
const showDetailModal = ref(false)
const reportTarget = ref<AuthorizedApp | null>(null)
const showReportModal = ref(false)
const reporting = ref(false)
const reportReason = ref('')
const reportCategory = ref('other')

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

function showDetail(app: AuthorizedApp) {
  detailTarget.value = app
  showDetailModal.value = true
}

function showReport(app: AuthorizedApp) {
  reportTarget.value = app
  reportReason.value = ''
  reportCategory.value = 'other'
  showReportModal.value = true
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

async function submitReport() {
  if (!reportTarget.value || !reportReason.value.trim()) return
  reporting.value = true
  try {
    await api.post(`/me/authorized-apps/${reportTarget.value.client_id}/report`, {
      reason: reportReason.value.trim(),
      category: reportCategory.value
    })
    showReportModal.value = false
    reportTarget.value = null
    reportReason.value = ''
    reportCategory.value = 'other'
    toast.success(t('authorizedApps.reportSuccess'))
  } catch (err: any) {
    const msg = err?.response?.data?.error_description || err?.response?.data?.error || t('authorizedApps.reportFailed')
    toast.error(msg)
  }
  reporting.value = false
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
    <div v-else class="space-y-3 max-w-2xl">
      <div
        v-for="app in authorizedApps"
        :key="app.id"
        class="border border-border rounded-lg p-4 flex flex-col gap-3"
      >
        <div class="flex items-start gap-3">
          <img
            v-if="app.logo_url"
            :src="app.logo_url"
            :alt="app.client_name"
            class="w-12 h-12 rounded-lg object-cover border border-border shrink-0 bg-background"
          />
          <div v-else class="w-12 h-12 rounded-lg bg-foreground text-white flex items-center justify-center text-lg font-bold shrink-0">
            {{ app.client_name.charAt(0).toUpperCase() }}
          </div>
          <div class="flex-1 min-w-0">
            <button @click="showDetail(app)" class="font-medium text-sm break-words hover:underline text-left">
              {{ app.client_name }}
            </button>
            <div v-if="app.description" class="mt-0.5 text-xs text-muted-foreground line-clamp-2">
              {{ app.description }}
            </div>
            <div v-if="app.developer?.display_name" class="mt-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
              <User class="w-3 h-3" />
              <span>{{ app.developer.display_name }}</span>
            </div>
            <div v-if="app.scopes && app.scopes.length" class="mt-2 flex flex-wrap gap-1">
              <span
                v-for="scope in app.scopes"
                :key="scope"
                class="px-2 py-0.5 text-xs bg-muted rounded break-all"
              >{{ scope }}</span>
            </div>
            <div v-if="app.granted_at" class="mt-1.5 text-xs text-muted-foreground">
              {{ $t('authorizedApps.grantedAt') }}: {{ new Date(app.granted_at).toLocaleDateString() }}
            </div>
          </div>
        </div>
        <button
          @click="confirmRevoke(app)"
          class="shrink-0 px-3 py-1.5 text-xs font-medium text-destructive border border-destructive/30 rounded-lg hover:bg-destructive/5 transition-colors w-full"
        >
          {{ $t('authorizedApps.revoke') }}
        </button>
      </div>
    </div>

    <!-- Detail Modal -->
    <div v-if="showDetailModal && detailTarget" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 py-4" @click.self="showDetailModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-md p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold">{{ $t('authorizedApps.appDetails') }}</h2>
          <button @click="showDetailModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>

        <div class="flex items-start gap-3 mb-4">
          <img
            v-if="detailTarget.logo_url"
            :src="detailTarget.logo_url"
            :alt="detailTarget.client_name"
            class="w-16 h-16 rounded-lg object-cover border border-border shrink-0 bg-background"
          />
          <div v-else class="w-16 h-16 rounded-lg bg-foreground text-white flex items-center justify-center text-2xl font-bold shrink-0">
            {{ detailTarget.client_name.charAt(0).toUpperCase() }}
          </div>
          <div class="flex-1 min-w-0">
            <div class="font-semibold text-base break-words">{{ detailTarget.client_name }}</div>
            <div v-if="detailTarget.description" class="mt-1 text-sm text-muted-foreground">
              {{ detailTarget.description }}
            </div>
          </div>
        </div>

        <div class="space-y-3 text-sm">
          <div v-if="detailTarget.developer?.display_name" class="flex items-start gap-2">
            <Building2 class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div class="flex-1 min-w-0">
              <div class="text-xs text-muted-foreground">{{ $t('authorizedApps.developer') }}</div>
              <div class="flex items-center gap-2 mt-0.5">
                <img
                  v-if="detailTarget.developer.avatar_url"
                  :src="detailTarget.developer.avatar_url"
                  :alt="detailTarget.developer.display_name"
                  class="w-5 h-5 rounded-full object-cover"
                />
                <span class="font-medium">{{ detailTarget.developer.display_name }}</span>
                <span v-if="detailTarget.developer.uid" class="text-xs text-muted-foreground">#{{ detailTarget.developer.uid }}</span>
              </div>
            </div>
          </div>

          <div v-if="detailTarget.homepage_url" class="flex items-start gap-2">
            <LinkIcon class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div class="flex-1 min-w-0">
              <div class="text-xs text-muted-foreground">{{ $t('authorizedApps.website') }}</div>
              <a
                :href="detailTarget.homepage_url"
                target="_blank"
                rel="noopener noreferrer"
                class="text-brand hover:underline break-all flex items-center gap-1 mt-0.5"
              >
                {{ detailTarget.homepage_url }}
                <ExternalLink class="w-3 h-3 shrink-0" />
              </a>
            </div>
          </div>

          <div class="flex items-start gap-2">
            <User class="w-4 h-4 mt-0.5 text-muted-foreground shrink-0" />
            <div class="flex-1 min-w-0">
              <div class="text-xs text-muted-foreground">{{ $t('authorizedApps.clientId') }}</div>
              <div class="font-mono text-xs break-all mt-0.5">{{ detailTarget.client_id }}</div>
            </div>
          </div>

          <div v-if="detailTarget.scopes && detailTarget.scopes.length">
            <div class="text-xs text-muted-foreground mb-2">{{ $t('authorizedApps.permissions') }}</div>
            <div class="flex flex-wrap gap-1">
              <span
                v-for="scope in detailTarget.scopes"
                :key="scope"
                class="px-2 py-1 text-xs bg-muted rounded break-all"
              >{{ scope }}</span>
            </div>
          </div>

          <div v-if="detailTarget.granted_at">
            <div class="text-xs text-muted-foreground">{{ $t('authorizedApps.grantedAt') }}</div>
            <div class="mt-0.5">{{ new Date(detailTarget.granted_at).toLocaleString() }}</div>
          </div>
        </div>

        <div class="flex gap-2 mt-6">
          <button
            @click="showDetailModal = false; showReport(detailTarget)"
            class="flex-1 px-4 py-2 text-sm font-medium border border-border rounded-lg hover:bg-muted transition-colors flex items-center justify-center gap-2"
          >
            <Flag class="w-4 h-4" />
            {{ $t('authorizedApps.report') }}
          </button>
          <button
            @click="showDetailModal = false; confirmRevoke(detailTarget)"
            class="flex-1 px-4 py-2 text-sm font-medium text-destructive border border-destructive/30 rounded-lg hover:bg-destructive/5 transition-colors"
          >
            {{ $t('authorizedApps.revoke') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Revoke Confirmation Modal -->
    <div v-if="showRevokeModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 py-4" @click.self="showRevokeModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('authorizedApps.revoke') }}</h2>
          <button @click="showRevokeModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <p class="text-sm text-muted-foreground mb-1 break-words">
          <strong>{{ revokeTarget?.client_name }}</strong>
        </p>
        <p class="text-sm text-muted-foreground mb-5">
          {{ $t('authorizedApps.revokeConfirm') }}
        </p>
        <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button @click="showRevokeModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors w-full sm:w-auto">
            {{ $t('cancel') }}
          </button>
          <button @click="revokeApp" :disabled="revoking" class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2 w-full sm:w-auto">
            <Loader2 v-if="revoking" class="w-4 h-4 animate-spin" />
            {{ $t('authorizedApps.revoke') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Report Modal -->
    <div v-if="showReportModal && reportTarget" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 py-4" @click.self="showReportModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-md p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold">{{ $t('authorizedApps.reportApp') }}</h2>
          <button @click="showReportModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>

        <p class="text-sm text-muted-foreground mb-4">
          {{ $t('authorizedApps.reportDesc') }}
        </p>

        <div class="mb-4">
          <label class="block text-sm font-medium mb-2">{{ $t('authorizedApps.reportCategory') }}</label>
          <select v-model="reportCategory" class="w-full px-3 py-2 border border-border rounded-lg text-sm">
            <option value="spam">{{ $t('authorizedApps.categorySpam') }}</option>
            <option value="abuse">{{ $t('authorizedApps.categoryAbuse') }}</option>
            <option value="fraud">{{ $t('authorizedApps.categoryFraud') }}</option>
            <option value="bot">{{ $t('authorizedApps.categoryBot') }}</option>
            <option value="other">{{ $t('authorizedApps.categoryOther') }}</option>
          </select>
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium mb-2">{{ $t('authorizedApps.reportReason') }}</label>
          <textarea
            v-model="reportReason"
            rows="4"
            class="w-full px-3 py-2 border border-border rounded-lg text-sm resize-none"
            :placeholder="$t('authorizedApps.reportReasonPlaceholder')"
          ></textarea>
        </div>

        <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button @click="showReportModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors w-full sm:w-auto">
            {{ $t('cancel') }}
          </button>
          <button
            @click="submitReport"
            :disabled="reporting || !reportReason.trim()"
            class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2 w-full sm:w-auto"
          >
            <Loader2 v-if="reporting" class="w-4 h-4 animate-spin" />
            {{ $t('authorizedApps.submitReport') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
