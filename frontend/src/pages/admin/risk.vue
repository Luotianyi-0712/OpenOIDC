<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { useToastStore } from '@/stores/toast'
import { useI18n } from 'vue-i18n'
import { Loader2, AlertTriangle, ShieldX, Check, X, Plus } from 'lucide-vue-next'

const { t } = useI18n()
const toast = useToastStore()

interface RiskReport {
  id: string
  client_id: string
  reporter_id: string
  reporter_uid?: number
  reporter_email?: string
  reporter_display_name?: string
  target_id: string
  target_uid?: number
  target_email?: string
  target_display_name?: string
  reason: string
  category: string
  status: string
  created_at: string
  client_name?: string
  app_name?: string
}

interface RiskListEntry {
  id: string
  provider: string
  provider_uid: string
  user_id: string | null
  user_uid?: number
  user_email?: string
  user_display_name?: string
  reason: string
  created_at: string
}

interface RiskPolicy {
  enabled: boolean
  blocked_ips: string
  blocked_emails: string
  blocked_email_domains: string
  blocked_email_patterns: string
}

const tab = ref<'reports' | 'blacklist' | 'policy'>('reports')
const reports = ref<RiskReport[]>([])
const blacklist = ref<RiskListEntry[]>([])
const loading = ref(false)
const reportTotal = ref(0)
const blacklistTotal = ref(0)
const riskPolicyEnabled = ref(true)
const blockedIPs = ref('')
const blockedEmails = ref('')
const blockedEmailDomains = ref('')
const blockedEmailPatterns = ref('')
const policyLoading = ref(false)
const policySaving = ref(false)
const policyLoaded = ref(false)

// Confirm/Dismiss dialog
const actionDialog = ref(false)
const actionType = ref<'confirm' | 'dismiss'>('confirm')
const actionReportId = ref('')
const actionReport = ref<RiskReport | null>(null)
const actionNote = ref('')
const actionLoading = ref(false)
const disableApp = ref(false)

const addDialog = ref(false)
const addLoading = ref(false)
const addForm = ref({
  provider: '',
  provider_uid: '',
  user_id: '',
  reason: '',
})

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
  } catch (e: any) {
    toast.error(e.message || t('adminRisk.loadReportsFailed'))
  }
  loading.value = false
}

async function loadBlacklist() {
  try {
    const res = await api.get<RiskListEntry[]>('/admin/risk/list')
    blacklist.value = res.data || []
    blacklistTotal.value = res.meta?.total || 0
  } catch (e: any) {
    toast.error(e.message || t('adminRisk.loadBlacklistFailed'))
  }
}

async function loadPolicySettings() {
  policyLoading.value = true
  try {
    const res = await api.get<RiskPolicy>('/admin/risk/policy')
    const policy = res.data
    if (policy) {
      riskPolicyEnabled.value = policy.enabled
      blockedIPs.value = policy.blocked_ips || ''
      blockedEmails.value = policy.blocked_emails || ''
      blockedEmailDomains.value = policy.blocked_email_domains || ''
      blockedEmailPatterns.value = policy.blocked_email_patterns || ''
    }
    policyLoaded.value = true
  } catch (e: any) {
    toast.error(e.message || t('adminRisk.loadPolicyFailed'))
  } finally {
    policyLoading.value = false
  }
}

function openPolicyTab() {
  tab.value = 'policy'
  if (!policyLoaded.value) loadPolicySettings()
}

async function savePolicySettings() {
  if (!policyLoaded.value || policyLoading.value) return
  policySaving.value = true
  try {
    await api.put('/admin/risk/policy', {
      enabled: riskPolicyEnabled.value,
      blocked_ips: blockedIPs.value,
      blocked_emails: blockedEmails.value,
      blocked_email_domains: blockedEmailDomains.value,
      blocked_email_patterns: blockedEmailPatterns.value,
    })
    toast.success(t('adminRisk.policySaved'))
    await loadPolicySettings()
  } catch (e: any) {
    toast.error(e.message || t('adminRisk.savePolicyFailed'))
  } finally {
    policySaving.value = false
  }
}

function openAction(type: 'confirm' | 'dismiss', id: string) {
  actionType.value = type
  actionReportId.value = id
  actionReport.value = reports.value.find(r => r.id === id) || null
  actionNote.value = ''
  disableApp.value = false
  actionDialog.value = true
}

async function submitAction() {
  actionLoading.value = true
  try {
    const endpoint = actionType.value === 'confirm'
      ? `/admin/risk/reports/${actionReportId.value}/confirm`
      : `/admin/risk/reports/${actionReportId.value}/dismiss`
    await api.put(endpoint, { note: actionNote.value })

    // If confirming an app report and user wants to disable the app
    if (actionType.value === 'confirm' && disableApp.value && actionReport.value && isAppReport(actionReport.value)) {
      try {
        await api.put(`/admin/clients/${actionReport.value.target_id}`, { is_active: false })
        toast.success(t('adminRisk.appDisabled'))
      } catch (e: any) {
        toast.error(e.message || t('adminRisk.disableAppFailed'))
      }
    }

    actionDialog.value = false
    await loadReports()
    await loadBlacklist()
  } catch (e: any) {
    toast.error(e.message || t('adminRisk.actionFailed'))
  }
  actionLoading.value = false
}

function openAddDialog() {
  addForm.value = { provider: '', provider_uid: '', user_id: '', reason: '' }
  addDialog.value = true
}

async function submitAddEntry() {
  addLoading.value = true
  try {
    await api.post('/admin/risk/list', {
      provider: addForm.value.provider,
      provider_uid: addForm.value.provider_uid,
      user_id: addForm.value.user_id || undefined,
      reason: addForm.value.reason,
    })
    addDialog.value = false
    await loadBlacklist()
  } catch (e: any) {
    toast.error(e.message || t('adminRisk.addFailed'))
  }
  addLoading.value = false
}

async function removeEntry(id: string) {
  if (!window.confirm(t('adminRisk.removeConfirm'))) return
  try {
    await api.del(`/admin/risk/list/${id}`)
    await loadBlacklist()
  } catch (e: any) {
    toast.error(e.message || t('adminRisk.removeFailed'))
  }
}

function categoryLabel(cat: string): string {
  const key = `adminRisk.categories.${cat}`
  return t(key)
}

function userLabel(uid?: number, email?: string, fallback?: string | null): string {
  if (uid) return `UID ${uid}`
  if (email) return email
  return fallback || '-'
}

function isAppReport(report: RiskReport): boolean {
  // If client_id is all zeros (uuid.Nil), it's a user reporting an app
  return report.client_id === '00000000-0000-0000-0000-000000000000'
}

function getReportTypeLabel(report: RiskReport): { targetLabel: string; reporterLabel: string; targetValue: string; reporterValue: string } {
  if (isAppReport(report)) {
    // User reporting app
    return {
      targetLabel: t('adminRisk.reportedApp'),
      reporterLabel: t('adminRisk.reportingUser'),
      targetValue: report.app_name || report.target_id,
      reporterValue: userLabel(report.reporter_uid, report.reporter_email, report.reporter_id)
    }
  } else {
    // Developer reporting user
    return {
      targetLabel: t('adminRisk.reportedUser'),
      reporterLabel: t('adminRisk.reportingApp'),
      targetValue: userLabel(report.target_uid, report.target_email, report.target_id),
      reporterValue: report.client_name || report.client_id
    }
  }
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
      <button
        @click="openPolicyTab"
        class="px-4 py-2.5 text-sm font-medium border-b-2 transition-colors"
        :class="tab === 'policy' ? 'border-foreground text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'"
      >
        <AlertTriangle class="w-4 h-4 inline mr-1.5" />
        {{ $t('adminRisk.policyTab') }}
      </button>
    </div>

    <!-- Loading -->
    <div v-if="tab === 'reports' && loading" class="flex items-center justify-center py-12 text-muted-foreground">
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
          <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 text-sm">
                <span class="px-2 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-700">
                  {{ categoryLabel(report.category) }}
                </span>
                <span class="text-muted-foreground">{{ new Date(report.created_at).toLocaleString() }}</span>
              </div>
              <p class="mt-2 text-sm">{{ report.reason }}</p>
              <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span>{{ getReportTypeLabel(report).targetLabel }}: <code class="font-mono">{{ getReportTypeLabel(report).targetValue }}</code></span>
                <span>{{ getReportTypeLabel(report).reporterLabel }}: <code class="font-mono">{{ getReportTypeLabel(report).reporterValue }}</code></span>
              </div>
            </div>
            <div class="flex flex-wrap gap-2 sm:justify-end sm:shrink-0">
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
      <div class="flex justify-end mb-4">
        <button
          @click="openAddDialog"
          class="px-3 py-2 text-sm font-medium bg-foreground text-white rounded-md hover:bg-foreground/90 transition-colors flex items-center gap-2"
        >
          <Plus class="w-4 h-4" /> {{ $t('adminRisk.addEntry') }}
        </button>
      </div>
      <div v-if="blacklist.length === 0" class="text-center text-muted-foreground py-12 text-sm">
        {{ $t('adminRisk.noBlacklist') }}
      </div>
      <div v-else class="hidden md:block overflow-x-auto">
        <table class="w-full min-w-[760px] text-sm">
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
              <td class="py-2.5 pr-4 font-mono text-xs whitespace-nowrap">{{ userLabel(entry.user_uid, entry.user_email, entry.user_id) }}</td>
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
      <div v-if="blacklist.length > 0" class="md:hidden space-y-3">
        <div v-for="entry in blacklist" :key="entry.id" class="border border-border rounded-xl p-4 bg-background space-y-3">
          <div class="flex flex-wrap gap-2">
            <span class="px-2 py-0.5 rounded-full text-xs font-medium bg-muted font-mono">{{ entry.provider }}</span>
            <span class="px-2 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground">{{ userLabel(entry.user_uid, entry.user_email, entry.user_id) }}</span>
          </div>
          <div class="space-y-2 text-xs text-muted-foreground">
            <div class="break-all"><span class="font-medium text-foreground">{{ $t('adminRisk.providerUid') }}：</span>{{ entry.provider_uid }}</div>
            <div class="break-words"><span class="font-medium text-foreground">{{ $t('adminRisk.reason') }}：</span>{{ entry.reason }}</div>
            <div><span class="font-medium text-foreground">{{ $t('adminRisk.time') }}：</span>{{ new Date(entry.created_at).toLocaleString() }}</div>
          </div>
          <button @click="removeEntry(entry.id)" class="w-full px-3 py-2 text-xs text-destructive border border-destructive/30 rounded-lg hover:bg-destructive/5 transition-colors">
            {{ $t('adminRisk.removeEntry') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Platform Policy Tab -->
    <div v-else-if="tab === 'policy'" class="space-y-4">
      <div class="rounded-xl border border-border bg-muted/30 p-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h3 class="text-base font-semibold">{{ $t('adminRisk.policyTitle') }}</h3>
            <p class="text-sm text-muted-foreground mt-1">{{ $t('adminRisk.policyDesc') }}</p>
          </div>
          <label class="inline-flex items-center gap-2 text-sm font-medium cursor-pointer shrink-0">
            <input v-model="riskPolicyEnabled" type="checkbox" class="rounded border-border" />
            {{ $t('adminRisk.policyEnabled') }}
          </label>
        </div>
      </div>

      <div v-if="policyLoading" class="rounded-lg border border-border bg-muted/30 px-4 py-3 text-sm text-muted-foreground flex items-center gap-2">
        <Loader2 class="w-4 h-4 animate-spin" /> {{ $t('loading') }}
      </div>
      <form @submit.prevent="savePolicySettings" class="grid gap-4 lg:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-xl border border-border p-4 bg-white">
          <label class="block text-sm font-medium mb-1.5">{{ $t('adminRisk.blockedIPs') }}</label>
          <p class="text-xs text-muted-foreground mb-3">{{ $t('adminRisk.blockedIPsHint') }}</p>
          <textarea
            v-model="blockedIPs"
            rows="8"
            placeholder="203.0.113.1\n2001:db8::/32"
            class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-y"
          />
        </div>

        <div class="rounded-xl border border-border p-4 bg-white">
          <label class="block text-sm font-medium mb-1.5">{{ $t('adminRisk.blockedEmails') }}</label>
          <p class="text-xs text-muted-foreground mb-3">{{ $t('adminRisk.blockedEmailsHint') }}</p>
          <textarea
            v-model="blockedEmails"
            rows="8"
            placeholder="bad@example.com\nspam@example.net"
            class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-y"
          />
        </div>

        <div class="rounded-xl border border-border p-4 bg-white">
          <label class="block text-sm font-medium mb-1.5">{{ $t('adminRisk.blockedDomains') }}</label>
          <p class="text-xs text-muted-foreground mb-3">{{ $t('adminRisk.blockedDomainsHint') }}</p>
          <textarea
            v-model="blockedEmailDomains"
            rows="8"
            placeholder="tempmail.example\ndisposable.test"
            class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-y"
          />
        </div>

        <div class="rounded-xl border border-border p-4 bg-white">
          <label class="block text-sm font-medium mb-1.5">{{ $t('adminRisk.blockedEmailPatterns') }}</label>
          <p class="text-xs text-muted-foreground mb-3">{{ $t('adminRisk.blockedEmailPatternsHint') }}</p>
          <textarea
            v-model="blockedEmailPatterns"
            rows="8"
            placeholder="^[^@]*[.+][^@]*@"
            class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-y"
          />
        </div>

        <div class="lg:col-span-2 xl:col-span-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p class="text-xs text-muted-foreground">{{ $t('adminRisk.policyFormatHint') }}</p>
          <div class="flex gap-2">
            <button type="button" @click="loadPolicySettings" :disabled="policySaving || policyLoading" class="px-4 py-2 text-sm font-medium border border-border rounded-lg hover:bg-muted transition-colors disabled:opacity-50">
              {{ $t('adminRisk.reloadPolicy') }}
            </button>
            <button type="submit" :disabled="policySaving || policyLoading || !policyLoaded" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2">
              <Loader2 v-if="policySaving" class="w-4 h-4 animate-spin" />
              {{ $t('adminRisk.savePolicy') }}
            </button>
          </div>
        </div>
      </form>
    </div>

    <!-- Action Dialog -->
    <div v-if="actionDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-md p-6 max-h-[90vh] overflow-y-auto">
        <h3 class="text-lg font-semibold mb-2">
          {{ actionType === 'confirm' ? $t('adminRisk.confirmReport') : $t('adminRisk.dismissReport') }}
        </h3>
        <p class="text-sm text-muted-foreground mb-4">
          {{ actionType === 'confirm' ? $t('adminRisk.confirmHint') : $t('adminRisk.dismissHint') }}
        </p>

        <!-- Disable App Option (only for app reports when confirming) -->
        <div v-if="actionType === 'confirm' && actionReport && isAppReport(actionReport)" class="mb-4 p-3 bg-amber-50 border border-amber-200 rounded-lg">
          <label class="flex items-start gap-2 cursor-pointer">
            <input
              type="checkbox"
              v-model="disableApp"
              class="mt-0.5 w-4 h-4 rounded border-gray-300 text-foreground focus:ring-2 focus:ring-foreground/20"
            />
            <div class="flex-1">
              <div class="text-sm font-medium text-amber-900">{{ $t('adminRisk.disableAppOption') }}</div>
              <div class="text-xs text-amber-700 mt-0.5">{{ $t('adminRisk.disableAppHint') }}</div>
            </div>
          </label>
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium mb-1">
            {{ actionType === 'dismiss' ? $t('adminRisk.dismissReasonLabel') : $t('adminRisk.noteLabel') }}
            <span v-if="actionType === 'dismiss'" class="text-xs text-muted-foreground font-normal ml-1">{{ $t('adminRisk.dismissReasonOptional') }}</span>
          </label>
          <textarea
            v-model="actionNote"
            rows="3"
            :placeholder="actionType === 'dismiss' ? $t('adminRisk.dismissReasonPlaceholder') : $t('adminRisk.notePlaceholder')"
            class="w-full px-3 py-2 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 resize-y"
          />
        </div>
        <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button
            @click="actionDialog = false"
            class="px-4 py-2 text-sm font-medium border border-border rounded-lg hover:bg-muted transition-colors w-full sm:w-auto"
          >
            {{ $t('cancel') }}
          </button>
          <button
            @click="submitAction"
            :disabled="actionLoading"
            class="px-4 py-2 text-sm font-medium rounded-lg transition-colors disabled:opacity-50 flex items-center justify-center gap-2 w-full sm:w-auto"
            :class="actionType === 'confirm' ? 'bg-destructive text-white hover:bg-destructive/90' : 'bg-foreground text-white hover:bg-foreground/90'"
          >
            <Loader2 v-if="actionLoading" class="w-4 h-4 animate-spin" />
            {{ $t('confirm') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Add Risk Entry Dialog -->
    <div v-if="addDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-md p-6 max-h-[90vh] overflow-y-auto">
        <h3 class="text-lg font-semibold mb-2">{{ $t('adminRisk.addEntry') }}</h3>
        <p class="text-sm text-muted-foreground mb-4">{{ $t('adminRisk.addHint') }}</p>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-1">{{ $t('adminRisk.provider') }}</label>
            <input
              v-model="addForm.provider"
              type="text"
              :placeholder="$t('adminRisk.providerPlaceholder')"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10"
            />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">{{ $t('adminRisk.providerUid') }}</label>
            <input
              v-model="addForm.provider_uid"
              type="text"
              :placeholder="$t('adminRisk.providerUidPlaceholder')"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10"
            />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">{{ $t('adminRisk.userIdOptional') }}</label>
            <input
              v-model="addForm.user_id"
              type="text"
              :placeholder="$t('adminRisk.userIdPlaceholder')"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10"
            />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">{{ $t('adminRisk.reason') }}</label>
            <input
              v-model="addForm.reason"
              type="text"
              :placeholder="$t('adminRisk.reasonPlaceholder')"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10"
            />
          </div>
        </div>
        <div class="flex flex-col-reverse gap-2 mt-6 sm:flex-row sm:justify-end">
          <button
            @click="addDialog = false"
            class="px-4 py-2 text-sm font-medium border border-border rounded-lg hover:bg-muted transition-colors w-full sm:w-auto"
          >
            {{ $t('cancel') }}
          </button>
          <button
            @click="submitAddEntry"
            :disabled="addLoading || !addForm.provider || !addForm.provider_uid || !addForm.reason"
            class="px-4 py-2 text-sm font-medium bg-foreground text-white rounded-lg hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2 w-full sm:w-auto"
          >
            <Loader2 v-if="addLoading" class="w-4 h-4 animate-spin" />
            {{ $t('confirm') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
