<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { useI18n } from 'vue-i18n'
import { Loader2, AlertTriangle, ShieldX, Check, X } from 'lucide-vue-next'

const { t } = useI18n()

interface RiskReport {
  id: string
  client_id: string
  reporter_id: string
  target_id: string
  reason: string
  category: string
  status: string
  created_at: string
}

interface RiskListEntry {
  id: string
  provider: string
  provider_uid: string
  user_id: string | null
  reason: string
  created_at: string
}

const tab = ref<'reports' | 'blacklist'>('reports')
const reports = ref<RiskReport[]>([])
const blacklist = ref<RiskListEntry[]>([])
const loading = ref(false)
const reportTotal = ref(0)
const blacklistTotal = ref(0)

// Confirm/Dismiss dialog
const actionDialog = ref(false)
const actionType = ref<'confirm' | 'dismiss'>('confirm')
const actionReportId = ref('')
const actionNote = ref('')
const actionLoading = ref(false)

onMounted(() => {
  loadReports()
  loadBlacklist()
})

async function loadReports() {
  loading.value = true
  try {
    const res = await api.get<RiskReport[]>('/admin/risk/reports')
    reports.value = res.data || []
    reportTotal.value = res.meta?.total || 0
  } catch { /* ignore */ }
  loading.value = false
}

async function loadBlacklist() {
  try {
    const res = await api.get<RiskListEntry[]>('/admin/risk/list')
    blacklist.value = res.data || []
    blacklistTotal.value = res.meta?.total || 0
  } catch { /* ignore */ }
}

function openAction(type: 'confirm' | 'dismiss', id: string) {
  actionType.value = type
  actionReportId.value = id
  actionNote.value = ''
  actionDialog.value = true
}

async function submitAction() {
  actionLoading.value = true
  try {
    const endpoint = actionType.value === 'confirm'
      ? `/admin/risk/reports/${actionReportId.value}/confirm`
      : `/admin/risk/reports/${actionReportId.value}/dismiss`
    await api.put(endpoint, { note: actionNote.value })
    actionDialog.value = false
    await loadReports()
    await loadBlacklist()
  } catch { /* ignore */ }
  actionLoading.value = false
}

async function removeEntry(id: string) {
  if (!window.confirm(t('adminRisk.removeConfirm'))) return
  try {
    await api.del(`/admin/risk/list/${id}`)
    await loadBlacklist()
  } catch { /* ignore */ }
}

function categoryLabel(cat: string): string {
  const key = `adminRisk.categories.${cat}`
  return t(key)
}
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-xl font-semibold">{{ $t('adminRisk.title') }}</h2>
      <p class="text-sm text-muted-foreground mt-1">{{ $t('adminRisk.subtitle') }}</p>
    </div>

    <!-- Tabs -->
    <div class="flex gap-0 border-b border-border mb-6">
      <button
        @click="tab = 'reports'"
        class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors"
        :class="tab === 'reports' ? 'border-foreground text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'"
      >
        <AlertTriangle class="w-4 h-4 inline mr-1.5" />
        {{ $t('adminRisk.reportsTab') }}
        <span v-if="reportTotal > 0" class="ml-1.5 px-1.5 py-0.5 text-xs bg-destructive/10 text-destructive rounded-full">{{ reportTotal }}</span>
      </button>
      <button
        @click="tab = 'blacklist'"
        class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors"
        :class="tab === 'blacklist' ? 'border-foreground text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'"
      >
        <ShieldX class="w-4 h-4 inline mr-1.5" />
        {{ $t('adminRisk.blacklistTab') }}
        <span v-if="blacklistTotal > 0" class="ml-1.5 px-1.5 py-0.5 text-xs bg-muted text-muted-foreground rounded-full">{{ blacklistTotal }}</span>
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <!-- Reports Tab -->
    <div v-else-if="tab === 'reports'">
      <div v-if="reports.length === 0" class="text-center text-muted-foreground py-12 text-sm">
        {{ $t('adminRisk.noReports') }}
      </div>
      <div v-else class="space-y-3">
        <div
          v-for="report in reports"
          :key="report.id"
          class="border border-border rounded-lg p-4"
        >
          <div class="flex items-start justify-between gap-4">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 text-sm">
                <span class="px-2 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-700">
                  {{ categoryLabel(report.category) }}
                </span>
                <span class="text-muted-foreground">{{ new Date(report.created_at).toLocaleString() }}</span>
              </div>
              <p class="mt-2 text-sm">{{ report.reason }}</p>
              <div class="mt-2 flex gap-4 text-xs text-muted-foreground">
                <span>{{ $t('adminRisk.target') }}: <code class="font-mono break-all">{{ report.target_id }}</code></span>
                <span>{{ $t('adminRisk.reporter') }}: <code class="font-mono break-all">{{ report.reporter_id }}</code></span>
              </div>
            </div>
            <div class="flex gap-2 shrink-0">
              <button
                @click="openAction('confirm', report.id)"
                class="px-3 py-1.5 text-xs font-medium bg-destructive text-white rounded-md hover:bg-destructive/90 transition-colors flex items-center gap-1"
              >
                <Check class="w-3.5 h-3.5" /> {{ $t('adminRisk.confirmReport') }}
              </button>
              <button
                @click="openAction('dismiss', report.id)"
                class="px-3 py-1.5 text-xs font-medium border border-border rounded-md hover:bg-muted transition-colors flex items-center gap-1"
              >
                <X class="w-3.5 h-3.5" /> {{ $t('adminRisk.dismissReport') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Blacklist Tab -->
    <div v-else-if="tab === 'blacklist'">
      <div v-if="blacklist.length === 0" class="text-center text-muted-foreground py-12 text-sm">
        {{ $t('adminRisk.noBlacklist') }}
      </div>
      <div v-else class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-border text-left text-muted-foreground">
              <th class="py-2 pr-4 font-medium">{{ $t('adminRisk.provider') }}</th>
              <th class="py-2 pr-4 font-medium">{{ $t('adminRisk.providerUid') }}</th>
              <th class="py-2 pr-4 font-medium">{{ $t('adminRisk.userUid') }}</th>
              <th class="py-2 pr-4 font-medium">{{ $t('adminRisk.reason') }}</th>
              <th class="py-2 pr-4 font-medium">{{ $t('adminRisk.time') }}</th>
              <th class="py-2 font-medium">{{ $t('actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="entry in blacklist" :key="entry.id" class="border-b border-border/50">
              <td class="py-2.5 pr-4 font-mono text-xs">{{ entry.provider }}</td>
              <td class="py-2.5 pr-4 font-mono text-xs truncate max-w-[200px]">{{ entry.provider_uid }}</td>
              <td class="py-2.5 pr-4 font-mono text-xs truncate max-w-[200px]">{{ entry.user_id || '-' }}</td>
              <td class="py-2.5 pr-4 text-xs">{{ entry.reason }}</td>
              <td class="py-2.5 pr-4 text-xs text-muted-foreground">{{ new Date(entry.created_at).toLocaleString() }}</td>
              <td class="py-2.5">
                <button
                  @click="removeEntry(entry.id)"
                  class="text-xs text-destructive hover:underline"
                >
                  {{ $t('adminRisk.removeEntry') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Action Dialog -->
    <div v-if="actionDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-md p-6">
        <h3 class="text-lg font-semibold mb-2">
          {{ actionType === 'confirm' ? $t('adminRisk.confirmReport') : $t('adminRisk.dismissReport') }}
        </h3>
        <p class="text-sm text-muted-foreground mb-4">
          {{ actionType === 'confirm' ? $t('adminRisk.confirmHint') : $t('adminRisk.dismissHint') }}
        </p>
        <div class="mb-4">
          <label class="block text-sm font-medium mb-1">{{ $t('adminRisk.noteLabel') }}</label>
          <input
            v-model="actionNote"
            type="text"
            :placeholder="$t('adminRisk.notePlaceholder')"
            class="w-full px-3 py-2 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10"
          />
        </div>
        <div class="flex justify-end gap-2">
          <button
            @click="actionDialog = false"
            class="px-4 py-2 text-sm font-medium border border-border rounded-lg hover:bg-muted transition-colors"
          >
            {{ $t('cancel') }}
          </button>
          <button
            @click="submitAction"
            :disabled="actionLoading"
            class="px-4 py-2 text-sm font-medium rounded-lg transition-colors disabled:opacity-50 flex items-center gap-2"
            :class="actionType === 'confirm' ? 'bg-destructive text-white hover:bg-destructive/90' : 'bg-foreground text-white hover:bg-foreground/90'"
          >
            <Loader2 v-if="actionLoading" class="w-4 h-4 animate-spin" />
            {{ $t('confirm') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
