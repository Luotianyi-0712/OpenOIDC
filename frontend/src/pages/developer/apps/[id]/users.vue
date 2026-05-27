<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'
import { useToastStore } from '@/stores/toast'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Ban, CheckCircle2, Loader2, RefreshCw, Search, ShieldAlert } from 'lucide-vue-next'

const route = useRoute()
const toast = useToastStore()
const { t } = useI18n()
const appId = route.params.id as string

interface AppDetail {
  id: string
  client_name: string
  user_count?: number
}

interface AppUserSummary {
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

const app = ref<AppDetail | null>(null)
const users = ref<AppUserSummary[]>([])
const total = ref(0)
const offset = ref(0)
const limit = ref(20)
const search = ref('')
const loading = ref(false)
const error = ref('')
const actionUserId = ref('')

const reportDialog = ref(false)
const reportUser = ref<AppUserSummary | null>(null)
const reportForm = ref({ category: 'other', reason: '' })
const reporting = ref(false)

let searchTimer: ReturnType<typeof setTimeout> | null = null

const pageFrom = computed(() => (total.value === 0 ? 0 : offset.value + 1))
const pageTo = computed(() => Math.min(offset.value + limit.value, total.value))

watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    offset.value = 0
    fetchUsers()
  }, 300)
})

onMounted(async () => {
  await Promise.all([fetchApp(), fetchUsers()])
})

async function fetchApp() {
  try {
    const res = await api.get<AppDetail>(`/developer/apps/${appId}`)
    app.value = res.data ?? null
  } catch {
    app.value = null
  }
}

async function fetchUsers() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({
      offset: String(offset.value),
      limit: String(limit.value),
    })
    if (search.value.trim()) params.set('q', search.value.trim())
    const res = await api.get<AppUserSummary[]>(`/developer/apps/${appId}/users?${params}`)
    users.value = res.data ?? []
    total.value = res.meta?.total ?? 0
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function prevPage() {
  if (offset.value === 0) return
  offset.value = Math.max(0, offset.value - limit.value)
  fetchUsers()
}

function nextPage() {
  if (offset.value + limit.value >= total.value) return
  offset.value += limit.value
  fetchUsers()
}

async function refreshUsers() {
  await fetchUsers()
  await fetchApp()
}

async function blockUser(user: AppUserSummary) {
  if (!window.confirm(t('developerApp.blockConfirm', { uid: user.uid }))) return
  actionUserId.value = user.id
  try {
    await api.post(`/developer/apps/${appId}/users/${user.id}/block`)
    user.blocked = true
    toast.success(t('developerApp.blockSuccess'))
  } catch (e: any) {
    toast.error(e.message)
  } finally {
    actionUserId.value = ''
  }
}

async function unblockUser(user: AppUserSummary) {
  if (!window.confirm(t('developerApp.unblockConfirm', { uid: user.uid }))) return
  actionUserId.value = user.id
  try {
    await api.del(`/developer/apps/${appId}/users/${user.id}/block`)
    user.blocked = false
    toast.success(t('developerApp.unblockSuccess'))
  } catch (e: any) {
    toast.error(e.message)
  } finally {
    actionUserId.value = ''
  }
}

function openReport(user: AppUserSummary) {
  reportUser.value = user
  reportForm.value = { category: 'other', reason: '' }
  reportDialog.value = true
}

async function submitReport() {
  if (!reportUser.value || !reportForm.value.reason.trim()) return
  reporting.value = true
  try {
    await api.post(`/developer/apps/${appId}/users/${reportUser.value.id}/report`, {
      category: reportForm.value.category,
      reason: reportForm.value.reason.trim(),
    })
    reportDialog.value = false
    toast.success(t('developerApp.reportSuccess'))
  } catch (e: any) {
    toast.error(e.message)
  } finally {
    reporting.value = false
  }
}

function formatTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime()) || date.getFullYear() <= 1) return '-'
  return date.toLocaleString()
}

function categoryLabel(category: string) {
  const key = `developerApp.reportCategories.${category}`
  const translated = t(key)
  return translated === key ? category : translated
}
</script>

<template>
  <div class="max-w-6xl">
    <router-link :to="`/developer/apps/${appId}`" class="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-6">
      <ArrowLeft class="w-4 h-4" /> {{ $t('developer.appDetail') }}
    </router-link>

    <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between mb-6">
      <div>
        <h2 class="text-lg font-semibold">{{ $t('developerApp.usersTitle') }}</h2>
        <p class="text-sm text-muted-foreground mt-1">
          {{ app?.client_name || $t('developerApp.usersSubtitle') }}
        </p>
      </div>
      <button @click="refreshUsers" :disabled="loading" class="inline-flex items-center justify-center gap-2 px-3 py-2 border border-border rounded-lg text-sm font-medium hover:bg-muted transition-colors disabled:opacity-50 w-full sm:w-auto">
        <RefreshCw class="w-4 h-4" :class="loading ? 'animate-spin' : ''" />
        {{ $t('refresh') }}
      </button>
    </div>

    <div class="grid gap-4 sm:grid-cols-3 mb-6">
      <div class="border border-border rounded-xl p-4">
        <div class="text-xs text-muted-foreground mb-1">{{ $t('developerApp.userCount') }}</div>
        <div class="text-2xl font-semibold">{{ app?.user_count ?? total }}</div>
      </div>
      <div class="border border-border rounded-xl p-4 sm:col-span-2">
        <div class="relative">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
          <input
            v-model="search"
            type="text"
            :placeholder="$t('developerApp.searchPlaceholder')"
            class="w-full pl-9 pr-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
          />
        </div>
      </div>
    </div>

    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <div v-if="loading && users.length === 0" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <div v-else>
      <div class="hidden md:block border border-border rounded-xl overflow-hidden bg-white">
        <table class="w-full min-w-[960px] text-sm">
          <thead class="bg-muted/50 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
            <tr>
              <th class="px-4 py-3">{{ $t('developerApp.uid') }}</th>
              <th class="px-4 py-3">{{ $t('developerApp.name') }}</th>
              <th class="px-4 py-3">{{ $t('developerApp.email') }}</th>
              <th class="px-4 py-3">{{ $t('developerApp.securityLevel') }}</th>
              <th class="px-4 py-3">{{ $t('developerApp.providers') }}</th>
              <th class="px-4 py-3">{{ $t('developerApp.blocked') }}</th>
              <th class="px-4 py-3">{{ $t('developerApp.lastUsedAt') }}</th>
              <th class="px-4 py-3 text-right">{{ $t('actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-if="users.length === 0">
              <td colspan="8" class="px-4 py-10 text-center text-muted-foreground">
                {{ $t('developerApp.noUsers') }}
              </td>
            </tr>
            <tr v-for="user in users" :key="user.id" class="hover:bg-muted/30 transition-colors">
              <td class="px-4 py-3 align-top">
                <code class="text-xs font-mono break-all">{{ user.uid }}</code>
              </td>
              <td class="px-4 py-3 align-top font-medium whitespace-nowrap">{{ user.display_name || '-' }}</td>
              <td class="px-4 py-3 align-top text-muted-foreground whitespace-nowrap">{{ user.email || '-' }}</td>
              <td class="px-4 py-3 align-top whitespace-nowrap">
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted">
                  L{{ user.security_level }}
                </span>
              </td>
              <td class="px-4 py-3 align-top">
                <div v-if="user.providers?.length" class="flex flex-wrap gap-1.5">
                  <span v-for="provider in user.providers" :key="provider" class="px-2 py-0.5 rounded-full bg-muted text-xs text-muted-foreground">
                    {{ provider }}
                  </span>
                </div>
                <span v-else class="text-muted-foreground">-</span>
              </td>
              <td class="px-4 py-3 align-top whitespace-nowrap">
                <span
                  class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
                  :class="user.blocked ? 'bg-destructive/10 text-destructive' : 'bg-green-50 text-green-700'"
                >
                  <Ban v-if="user.blocked" class="w-3 h-3" />
                  <CheckCircle2 v-else class="w-3 h-3" />
                  {{ user.blocked ? $t('developerApp.blockedYes') : $t('developerApp.blockedNo') }}
                </span>
              </td>
              <td class="px-4 py-3 align-top text-muted-foreground whitespace-nowrap text-xs">{{ formatTime(user.last_used_at) }}</td>
              <td class="px-4 py-3 align-top">
                <div class="flex flex-wrap justify-end gap-2">
                  <button @click="openReport(user)" class="inline-flex items-center gap-1.5 px-3 py-1.5 border border-border rounded-lg text-xs font-medium hover:bg-muted transition-colors">
                    <ShieldAlert class="w-3.5 h-3.5" />
                    {{ $t('developerApp.report') }}
                  </button>
                  <button
                    v-if="user.blocked"
                    @click="unblockUser(user)"
                    :disabled="actionUserId === user.id"
                    class="inline-flex items-center gap-1.5 px-3 py-1.5 border border-border rounded-lg text-xs font-medium hover:bg-muted transition-colors disabled:opacity-50"
                  >
                    <Loader2 v-if="actionUserId === user.id" class="w-3.5 h-3.5 animate-spin" />
                    {{ $t('developerApp.unblock') }}
                  </button>
                  <button
                    v-else
                    @click="blockUser(user)"
                    :disabled="actionUserId === user.id"
                    class="inline-flex items-center gap-1.5 px-3 py-1.5 border border-destructive/30 text-destructive rounded-lg text-xs font-medium hover:bg-destructive/5 transition-colors disabled:opacity-50"
                  >
                    <Loader2 v-if="actionUserId === user.id" class="w-3.5 h-3.5 animate-spin" />
                    {{ $t('developerApp.block') }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="md:hidden space-y-3">
        <div v-if="users.length === 0" class="border border-border rounded-xl px-4 py-10 text-center text-muted-foreground text-sm">
          {{ $t('developerApp.noUsers') }}
        </div>
        <div v-for="user in users" :key="user.id" class="border border-border rounded-xl p-4 bg-white space-y-3">
          <div class="flex flex-col gap-2">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-mono text-xs text-muted-foreground">UID {{ user.uid }}</span>
              <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted">L{{ user.security_level }}</span>
              <span
                class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
                :class="user.blocked ? 'bg-destructive/10 text-destructive' : 'bg-green-50 text-green-700'"
              >
                <Ban v-if="user.blocked" class="w-3 h-3" />
                <CheckCircle2 v-else class="w-3 h-3" />
                {{ user.blocked ? $t('developerApp.blockedYes') : $t('developerApp.blockedNo') }}
              </span>
            </div>
            <div class="font-medium text-sm break-words">{{ user.display_name || '-' }}</div>
            <div class="text-xs text-muted-foreground break-all">{{ user.email || '-' }}</div>
          </div>

          <div v-if="user.providers?.length" class="flex flex-wrap gap-1.5">
            <span v-for="provider in user.providers" :key="provider" class="px-2 py-0.5 rounded-full bg-muted text-xs text-muted-foreground">
              {{ provider }}
            </span>
          </div>

          <div class="text-xs text-muted-foreground">
            <span class="font-medium text-foreground">{{ $t('developerApp.lastUsedAt') }}：</span>{{ formatTime(user.last_used_at) }}
          </div>

          <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <button @click="openReport(user)" class="inline-flex items-center justify-center gap-1.5 px-3 py-2 border border-border rounded-lg text-xs font-medium hover:bg-muted transition-colors">
              <ShieldAlert class="w-3.5 h-3.5" />
              {{ $t('developerApp.report') }}
            </button>
            <button
              v-if="user.blocked"
              @click="unblockUser(user)"
              :disabled="actionUserId === user.id"
              class="inline-flex items-center justify-center gap-1.5 px-3 py-2 border border-border rounded-lg text-xs font-medium hover:bg-muted transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="actionUserId === user.id" class="w-3.5 h-3.5 animate-spin" />
              {{ $t('developerApp.unblock') }}
            </button>
            <button
              v-else
              @click="blockUser(user)"
              :disabled="actionUserId === user.id"
              class="inline-flex items-center justify-center gap-1.5 px-3 py-2 border border-destructive/30 text-destructive rounded-lg text-xs font-medium hover:bg-destructive/5 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="actionUserId === user.id" class="w-3.5 h-3.5 animate-spin" />
              {{ $t('developerApp.block') }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="total > 0" class="flex flex-col gap-3 mt-4 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
      <span>{{ $t('showing', { from: pageFrom, to: pageTo, total }) }}</span>
      <div class="flex gap-2">
        <button @click="prevPage" :disabled="offset === 0" class="px-3 py-1.5 border border-border rounded-lg text-sm hover:bg-muted transition-colors disabled:opacity-40">
          {{ $t('prev') }}
        </button>
        <button @click="nextPage" :disabled="offset + limit >= total" class="px-3 py-1.5 border border-border rounded-lg text-sm hover:bg-muted transition-colors disabled:opacity-40">
          {{ $t('next') }}
        </button>
      </div>
    </div>

    <div v-if="reportDialog" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="reportDialog = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4 p-6">
        <h3 class="text-lg font-semibold mb-2">{{ $t('developerApp.reportUser') }}</h3>
        <p class="text-sm text-muted-foreground mb-4">
          {{ reportUser?.email || reportUser?.uid }}
        </p>
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('developerApp.reportCategory') }}</label>
            <select v-model="reportForm.category" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10">
              <option v-for="category in ['spam', 'abuse', 'fraud', 'bot', 'other']" :key="category" :value="category">
                {{ categoryLabel(category) }}
              </option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('developerApp.reportReason') }}</label>
            <textarea
              v-model="reportForm.reason"
              rows="4"
              :placeholder="$t('developerApp.reportReasonPlaceholder')"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-none"
            />
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-6">
          <button @click="reportDialog = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">
            {{ $t('cancel') }}
          </button>
          <button
            @click="submitReport"
            :disabled="reporting || !reportForm.reason.trim()"
            class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2"
          >
            <Loader2 v-if="reporting" class="w-4 h-4 animate-spin" />
            {{ $t('developerApp.submitReport') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>