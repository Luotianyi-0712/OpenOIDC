<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { Loader2, Shield, ShieldCheck, ShieldAlert, Check, X, ArrowUp, Link2, Fingerprint, Plus, Trash2, Pencil } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { usePasskey, type PasskeyCredential } from '@/composables/usePasskey'

const { t } = useI18n()

interface MissingCondition {
  provider: string
  min_binding_days: number
  is_bound: boolean
  bound_days: number
}

interface NextLevel {
  level: number
  rule_name: string
  missing: MissingCondition[]
}

interface Binding {
  provider: string
  bound_at: string
}

interface LevelInfo {
  level: number
  max_level: number
  bindings: Binding[]
  next_level?: NextLevel
}

const auth = useAuthStore()
const info = ref<LevelInfo | null>(null)
const loading = ref(true)
const error = ref('')

onMounted(fetchSecurity)

async function fetchSecurity() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<LevelInfo>('/me/security-level')
    info.value = res.data || null
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

const maxLevel = computed(() => info.value?.max_level || 1)
const percentage = computed(() => {
  if (!info.value) return 0
  return Math.min(100, Math.round((info.value.level / maxLevel.value) * 100))
})

const levelColor = computed(() => {
  if (!info.value) return 'text-muted-foreground'
  const p = percentage.value
  if (p >= 80) return 'text-success'
  if (p >= 40) return 'text-brand'
  return 'text-destructive'
})

const levelBg = computed(() => {
  if (!info.value) return 'bg-muted'
  const p = percentage.value
  if (p >= 80) return 'bg-success/10'
  if (p >= 40) return 'bg-brand/10'
  return 'bg-destructive/10'
})

const ShieldIcon = computed(() => {
  if (!info.value) return Shield
  const p = percentage.value
  if (p >= 80) return ShieldCheck
  if (p >= 40) return Shield
  return ShieldAlert
})

const levelLabel = computed(() => {
  if (!info.value) return ''
  const p = percentage.value
  if (p >= 80) return t('security.strong')
  if (p >= 40) return t('security.moderate')
  return t('security.weak')
})

function providerLabel(provider: string): string {
  const key = `adminRules.providerOptions.${provider}`
  const translated = t(key)
  return translated !== key ? translated : provider
}

// Passkey management
const { loading: passkeyLoading, error: passkeyError, registerPasskey, listPasskeys, deletePasskey, renamePasskey } = usePasskey()
const passkeys = ref<PasskeyCredential[]>([])
const passkeyListLoading = ref(false)
const showRenameModal = ref(false)
const renameTarget = ref<{ id: string; name: string } | null>(null)
const renameInput = ref('')
const showDeleteModal = ref(false)
const deleteTarget = ref<{ id: string; name: string } | null>(null)

async function fetchPasskeys() {
  passkeyListLoading.value = true
  try {
    passkeys.value = await listPasskeys()
  } catch { /* ignore */ }
  finally { passkeyListLoading.value = false }
}

async function handleRegisterPasskey() {
  const ok = await registerPasskey()
  if (ok) fetchPasskeys()
}

function confirmDeletePasskey(pk: PasskeyCredential) {
  deleteTarget.value = { id: pk.id, name: pk.name || t('passkey.unnamed') }
  showDeleteModal.value = true
}

async function doDeletePasskey() {
  if (!deleteTarget.value) return
  showDeleteModal.value = false
  await deletePasskey(deleteTarget.value.id)
  fetchPasskeys()
}

function openRename(pk: PasskeyCredential) {
  renameTarget.value = { id: pk.id, name: pk.name }
  renameInput.value = pk.name
  showRenameModal.value = true
}

async function doRename() {
  if (!renameTarget.value || !renameInput.value.trim()) return
  showRenameModal.value = false
  await renamePasskey(renameTarget.value.id, renameInput.value.trim())
  fetchPasskeys()
}

function formatPasskeyDate(iso: string | null) {
  if (!iso) return '-'
  return new Date(iso).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

onMounted(fetchPasskeys)
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-lg font-semibold">{{ $t('security.title') }}</h2>
      <p class="text-sm text-muted-foreground mt-0.5">{{ $t('security.desc') }}</p>
    </div>

    <div v-if="loading" class="flex items-center gap-2 text-sm text-muted-foreground py-12 justify-center">
      <Loader2 class="w-4 h-4 animate-spin" /> {{ $t('security.loadingSecurity') }}
    </div>

    <div v-else-if="error" class="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <template v-else-if="info">
      <!-- Level Badge Card -->
      <div class="border border-border rounded-xl p-8 mb-6">
        <div class="flex items-center gap-6">
          <div class="w-20 h-20 rounded-2xl flex items-center justify-center shrink-0" :class="levelBg">
            <component :is="ShieldIcon" class="w-10 h-10" :class="levelColor" />
          </div>
          <div class="flex-1">
            <div class="flex items-baseline gap-3 mb-2">
              <span class="text-4xl font-bold tabular-nums" :class="levelColor">{{ info.level }}</span>
              <span class="text-lg text-muted-foreground font-medium">/ {{ maxLevel }}</span>
            </div>
            <div class="text-sm font-medium" :class="levelColor">{{ levelLabel }}</div>
            <div class="mt-3 w-full h-2 rounded-full bg-muted overflow-hidden">
              <div class="h-full rounded-full transition-all duration-500" :class="{ 'bg-success': percentage >= 80, 'bg-brand': percentage >= 40 && percentage < 80, 'bg-destructive': percentage < 40 }" :style="{ width: percentage + '%' }" />
            </div>
          </div>
        </div>
      </div>

      <!-- Current Bindings -->
      <div v-if="info.bindings && info.bindings.length" class="border border-border rounded-xl p-6 mb-6">
        <h3 class="text-sm font-medium mb-3 flex items-center gap-2">
          <Link2 class="w-4 h-4 text-muted-foreground" /> {{ $t('security.currentBindings') }}
        </h3>
        <div class="flex flex-wrap gap-2">
          <span v-for="b in info.bindings" :key="b.provider" class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-green-50 text-green-700 text-xs font-medium">
            <Check class="w-3 h-3" /> {{ providerLabel(b.provider) }}
          </span>
        </div>
      </div>

      <!-- Next Level Requirements -->
      <div v-if="info.next_level" class="border border-border rounded-xl p-6 mb-6">
        <div class="flex items-center gap-2 mb-3">
          <ArrowUp class="w-4 h-4 text-brand" />
          <h3 class="text-sm font-medium">{{ $t('security.nextLevelTitle', { level: info.next_level.level }) }}</h3>
        </div>
        <p class="text-xs text-muted-foreground mb-4">{{ $t('security.nextLevelDesc', { name: info.next_level.rule_name }) }}</p>
        <div class="space-y-2">
          <div v-for="cond in info.next_level.missing" :key="cond.provider" class="flex items-center gap-3 px-4 py-3 rounded-lg" :class="cond.is_bound ? 'bg-green-50' : 'bg-muted/50'">
            <div class="w-6 h-6 rounded-full flex items-center justify-center shrink-0" :class="cond.is_bound ? 'bg-green-100 text-green-600' : 'bg-muted text-muted-foreground'">
              <Check v-if="cond.is_bound" class="w-3.5 h-3.5" />
              <X v-else class="w-3.5 h-3.5" />
            </div>
            <div class="flex-1">
              <div class="text-sm font-medium" :class="cond.is_bound ? 'text-green-700' : 'text-foreground'">
                {{ $t('security.bindProvider', { provider: providerLabel(cond.provider) }) }}
              </div>
              <div v-if="cond.min_binding_days > 0" class="text-xs text-muted-foreground">
                <template v-if="cond.is_bound">
                  {{ $t('security.boundDays', { current: cond.bound_days, required: cond.min_binding_days }) }}
                </template>
                <template v-else>
                  {{ $t('security.requireDays', { days: cond.min_binding_days }) }}
                </template>
              </div>
            </div>
            <span class="text-xs font-medium px-2 py-0.5 rounded-full" :class="cond.is_bound && (cond.min_binding_days <= 0 || cond.bound_days >= cond.min_binding_days) ? 'bg-green-100 text-green-700' : 'bg-muted text-muted-foreground'">
              {{ cond.is_bound && (cond.min_binding_days <= 0 || cond.bound_days >= cond.min_binding_days) ? $t('security.completed') : $t('security.incomplete') }}
            </span>
          </div>
        </div>
        <router-link to="/me/bindings" class="inline-flex items-center gap-1.5 mt-4 text-sm text-brand hover:underline">
          <Link2 class="w-3.5 h-3.5" /> {{ $t('security.goBindings') }}
        </router-link>
      </div>

      <!-- Already at max or no rules -->
      <div v-else-if="info.level >= maxLevel" class="border border-border rounded-xl p-6 mb-6 text-center">
        <ShieldCheck class="w-8 h-8 text-success mx-auto mb-2" />
        <p class="text-sm font-medium">{{ $t('security.maxReached') }}</p>
      </div>

      <!-- Passkey Management -->
      <div class="border border-border rounded-xl p-6 mb-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-sm font-medium flex items-center gap-2">
            <Fingerprint class="w-4 h-4 text-muted-foreground" /> {{ $t('passkey.title') }}
          </h3>
          <button
            @click="handleRegisterPasskey"
            :disabled="passkeyLoading"
            class="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium border border-border rounded-lg hover:bg-muted transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="passkeyLoading" class="w-3 h-3 animate-spin" />
            <Plus v-else class="w-3 h-3" />
            {{ $t('passkey.register') }}
          </button>
        </div>
        <div v-if="passkeyError" class="text-xs text-destructive mb-3">{{ passkeyError }}</div>
        <div v-if="passkeyListLoading" class="flex items-center gap-2 text-xs text-muted-foreground py-4 justify-center">
          <Loader2 class="w-3 h-3 animate-spin" /> {{ $t('passkey.loading') }}
        </div>
        <div v-else-if="passkeys.length === 0" class="text-sm text-muted-foreground text-center py-4">
          {{ $t('passkey.empty') }}
        </div>
        <div v-else class="space-y-2.5">
          <div v-for="pk in passkeys" :key="pk.id" class="flex items-center justify-between px-4 py-3 rounded-lg bg-muted/30">
            <div>
              <div class="text-sm font-medium">{{ pk.name || $t('passkey.unnamed') }}</div>
              <div class="text-xs text-muted-foreground mt-0.5">
                {{ $t('passkey.created') }}: {{ formatPasskeyDate(pk.created_at) }}
                <span v-if="pk.last_used_at" class="ml-2">{{ $t('passkey.lastUsed') }}: {{ formatPasskeyDate(pk.last_used_at) }}</span>
              </div>
            </div>
            <div class="flex items-center gap-1.5">
              <button @click="openRename(pk)" class="p-1.5 rounded hover:bg-muted transition-colors text-muted-foreground hover:text-foreground">
                <Pencil class="w-3.5 h-3.5" />
              </button>
              <button @click="confirmDeletePasskey(pk)" class="p-1.5 rounded hover:bg-destructive/10 transition-colors text-muted-foreground hover:text-destructive">
                <Trash2 class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Account Info -->
      <div class="border border-border rounded-xl p-6">
        <h3 class="text-sm font-medium mb-3">{{ $t('security.accountDetails') }}</h3>
        <div class="space-y-2.5 text-sm">
          <div class="flex justify-between gap-4 py-2 border-b border-border">
            <span class="text-muted-foreground">{{ $t('profile.uid') }}</span>
            <span class="font-mono text-xs text-right break-all">{{ auth.user?.id || '-' }}</span>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <span class="text-muted-foreground">{{ $t('security.emailVerified') }}</span>
            <span :class="auth.user?.email_verified ? 'text-success' : 'text-muted-foreground'">
              {{ auth.user?.email_verified ? $t('yes') : $t('no') }}
            </span>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <span class="text-muted-foreground">{{ $t('security.accountStatus') }}</span>
            <span class="font-medium">{{ auth.user?.status }}</span>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <span class="text-muted-foreground">{{ $t('security.securityLevel') }}</span>
            <span class="font-medium" :class="levelColor">{{ info.level }} / {{ maxLevel }}</span>
          </div>
          <div class="flex justify-between py-2">
            <span class="text-muted-foreground">{{ $t('security.accountCreated') }}</span>
            <span>{{ auth.user?.created_at ? new Date(auth.user.created_at).toLocaleDateString('zh-CN') : '-' }}</span>
          </div>
        </div>
      </div>
    </template>

    <!-- Passkey Rename Modal -->
    <div v-if="showRenameModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showRenameModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('passkey.rename') }}</h2>
          <button @click="showRenameModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <input v-model="renameInput" class="w-full border border-border rounded-lg px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-foreground/10 mb-4" :placeholder="$t('passkey.namePlaceholder')" @keyup.enter="doRename" />
        <div class="flex justify-end gap-2">
          <button @click="showRenameModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
          <button @click="doRename" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors">{{ $t('confirm') }}</button>
        </div>
      </div>
    </div>

    <!-- Passkey Delete Modal -->
    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showDeleteModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('passkey.delete') }}</h2>
          <button @click="showDeleteModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('passkey.deleteConfirm', { name: deleteTarget?.name }) }}</p>
        <div class="flex justify-end gap-2">
          <button @click="showDeleteModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
          <button @click="doDeletePasskey" class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors">{{ $t('confirm') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>
