<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/api/client'
import { Loader2, Check, BadgeCheck, AtSign, KeyRound, UserPen, ImagePlus, Mail, X, Fingerprint, Plus, Trash2, Pencil, Eye, EyeOff } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { usePasswordPolicy } from '@/composables/usePasswordPolicy'
import { usePasskey, type PasskeyCredential } from '@/composables/usePasskey'

const { t } = useI18n()
const auth = useAuthStore()
const { policy, hasRequirements, validate } = usePasswordPolicy()

// Profile form
const displayName = ref('')
const avatarUrl = ref('')
const profileLoading = ref(false)
const profileMsg = ref('')
const profileError = ref('')

// Alias form
const alias = ref('')
const aliasLoading = ref(false)
const aliasMsg = ref('')
const aliasError = ref('')
const aliasLocked = computed(() => !!auth.user?.alias)

// Password form
const oldPassword = ref('')
const newPassword = ref('')
const passwordLoading = ref(false)
const passwordMsg = ref('')
const passwordError = ref('')
const showOldPassword = ref(false)
const showNewPassword = ref(false)

const newPasswordErrors = computed(() => newPassword.value ? validate(newPassword.value) : [])
const newPasswordValid = computed(() => newPassword.value.length > 0 && newPasswordErrors.value.length === 0)
const newPasswordChecks = computed(() => ({
  minLength: !newPasswordErrors.value.includes('min_length'),
  upper: !newPasswordErrors.value.includes('require_upper'),
  lower: !newPasswordErrors.value.includes('require_lower'),
  digit: !newPasswordErrors.value.includes('require_digit'),
  symbol: !newPasswordErrors.value.includes('require_symbol'),
}))

// Passkey management
const { loading: passkeyLoading, error: passkeyError, registerPasskey, listPasskeys, deletePasskey, renamePasskey } = usePasskey()
const passkeys = ref<PasskeyCredential[]>([])
const passkeyListLoading = ref(false)
const showRenameModal = ref(false)
const renameTarget = ref<{ id: string; name: string } | null>(null)
const renameInput = ref('')
const showDeleteModal = ref(false)
const deleteTarget = ref<{ id: string; name: string } | null>(null)

// Email verification
const verificationSending = ref(false)
const verificationSent = ref(false)
const verificationError = ref('')

onMounted(() => {
  if (auth.user) {
    displayName.value = auth.user.display_name
    avatarUrl.value = auth.user.avatar_url
    alias.value = auth.user.alias || ''
  }
  fetchPasskeys()
})

async function sendVerification() {
  verificationSending.value = true
  verificationError.value = ''
  verificationSent.value = false
  try {
    await api.post('/me/resend-verification')
    verificationSent.value = true
  } catch (e: any) {
    verificationError.value = e.message
  } finally {
    verificationSending.value = false
  }
}

async function updateProfile() {
  profileMsg.value = ''
  profileError.value = ''
  profileLoading.value = true
  try {
    await api.put('/me', {
      display_name: displayName.value,
      avatar_url: avatarUrl.value,
    })
    await auth.fetchUser()
    profileMsg.value = t('profile.profileUpdated')
  } catch (e: any) {
    profileError.value = e.message || 'Failed to update profile.'
  } finally {
    profileLoading.value = false
  }
}

async function updateAlias() {
  aliasMsg.value = ''
  aliasError.value = ''
  if (aliasLocked.value) {
    aliasError.value = t('profile.aliasLocked')
    return
  }
  aliasLoading.value = true
  try {
    await api.put('/me/alias', { alias: alias.value })
    await auth.fetchUser()
    aliasMsg.value = t('profile.aliasUpdated')
  } catch (e: any) {
    aliasError.value = e.message || 'Failed to update alias.'
  } finally {
    aliasLoading.value = false
  }
}

async function changePassword() {
  passwordMsg.value = ''
  passwordError.value = ''
  if (newPasswordErrors.value.length > 0) {
    passwordError.value = t('passwordPolicy.notMet')
    return
  }
  passwordLoading.value = true
  try {
    await api.put('/me/password', {
      old_password: oldPassword.value,
      new_password: newPassword.value,
    })
    oldPassword.value = ''
    newPassword.value = ''
    passwordMsg.value = t('profile.passwordChanged')
  } catch (e: any) {
    passwordError.value = e.message || 'Failed to change password.'
  } finally {
    passwordLoading.value = false
  }
}

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
</script>

<template>
  <div class="space-y-0">
    <!-- User Info Header -->
    <div class="flex items-center gap-5 pb-8 border-b border-border">
      <img
        v-if="auth.user?.avatar_url"
        :src="auth.user.avatar_url"
        :alt="auth.user.display_name"
        class="w-16 h-16 rounded-full object-cover border border-border"
      />
      <div
        v-else
        class="w-16 h-16 rounded-full bg-muted flex items-center justify-center text-muted-foreground text-xl font-semibold"
      >
        {{ auth.user?.display_name?.charAt(0)?.toUpperCase() || '?' }}
      </div>
      <div>
        <h2 class="text-lg font-semibold">{{ auth.user?.display_name }}</h2>
        <div class="flex items-center gap-2 mt-0.5">
          <span class="text-sm text-muted-foreground">{{ auth.user?.email }}</span>
          <span
            v-if="auth.user?.email_verified"
            class="inline-flex items-center gap-1 text-xs text-success font-medium"
          >
            <BadgeCheck class="w-3.5 h-3.5" /> {{ $t('profile.verified') }}
          </span>
          <template v-else>
            <span class="text-xs text-muted-foreground">({{ $t('profile.unverified') }})</span>
            <button
              v-if="!verificationSent"
              @click="sendVerification"
              :disabled="verificationSending"
              class="text-xs text-brand hover:underline flex items-center gap-1 disabled:opacity-50"
            >
              <Mail class="w-3 h-3" />
              {{ verificationSending ? $t('profile.sending') : $t('profile.sendVerification') }}
            </button>
            <span v-else class="text-xs text-success flex items-center gap-1">
              <Check class="w-3 h-3" /> {{ $t('profile.verificationSent') }}
            </span>
          </template>
        </div>
        <div v-if="verificationError" class="text-xs text-destructive mt-1">{{ verificationError }}</div>
        <div v-if="auth.user?.uid || auth.user?.alias" class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground font-mono">
          <span v-if="auth.user?.uid">{{ $t('profile.uid') }} {{ auth.user.uid }}</span>
          <span v-if="auth.user?.alias">@{{ auth.user.alias }}</span>
        </div>
      </div>
    </div>

    <!-- Account Details -->
    <div class="py-8 border-b border-border">
      <h3 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground mb-5">
        {{ $t('profile.accountDetails') }}
      </h3>
      <div class="grid gap-3 sm:grid-cols-2 max-w-2xl text-sm">
        <div class="rounded-lg bg-muted/40 px-4 py-3">
          <div class="text-xs text-muted-foreground mb-1">{{ $t('profile.uid') }}</div>
          <div class="font-mono text-xs break-all">{{ auth.user?.uid || '-' }}</div>
        </div>
        <div class="rounded-lg bg-muted/40 px-4 py-3">
          <div class="text-xs text-muted-foreground mb-1">{{ $t('profile.emailVerified') }}</div>
          <div class="font-medium" :class="auth.user?.email_verified ? 'text-success' : 'text-muted-foreground'">
            {{ auth.user?.email_verified ? $t('yes') : $t('no') }}
          </div>
        </div>
        <div class="rounded-lg bg-muted/40 px-4 py-3">
          <div class="text-xs text-muted-foreground mb-1">{{ $t('profile.accountStatus') }}</div>
          <div class="font-medium">{{ auth.user?.status || '-' }}</div>
        </div>
        <div class="rounded-lg bg-muted/40 px-4 py-3">
          <div class="text-xs text-muted-foreground mb-1">{{ $t('profile.accountCreated') }}</div>
          <div>{{ auth.user?.created_at ? new Date(auth.user.created_at).toLocaleDateString('zh-CN') : '-' }}</div>
        </div>
      </div>
    </div>

    <!-- Edit Profile Section -->
    <div class="py-8 border-b border-border">
      <h3 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground mb-5 flex items-center gap-2">
        <UserPen class="w-4 h-4" /> {{ $t('profile.editProfile') }}
      </h3>
      <form @submit.prevent="updateProfile" class="space-y-4 max-w-md">
        <div>
          <label class="block text-sm font-medium mb-1.5" for="displayName">{{ $t('profile.displayName') }}</label>
          <input
            id="displayName"
            v-model="displayName"
            type="text"
            required
            class="w-full px-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
          />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1.5" for="avatarUrl">{{ $t('profile.avatarUrl') }}</label>
          <div class="flex items-center gap-3">
            <input
              id="avatarUrl"
              v-model="avatarUrl"
              type="url"
              placeholder="https://example.com/avatar.png"
              class="flex-1 px-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
            />
            <img
              v-if="avatarUrl"
              :src="avatarUrl"
              class="w-10 h-10 rounded-full object-cover border border-border shrink-0"
              @error="($event.target as HTMLImageElement).style.display = 'none'"
            />
          </div>
        </div>
        <div>
          <div
            v-if="profileMsg"
            class="mb-3 rounded-lg border border-success/30 bg-success/5 px-4 py-2.5 text-sm text-success flex items-center gap-2"
          >
            <Check class="w-4 h-4 shrink-0" /> {{ profileMsg }}
          </div>
          <div
            v-if="profileError"
            class="mb-3 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-2.5 text-sm text-destructive"
          >
            {{ profileError }}
          </div>
          <button
            type="submit"
            :disabled="profileLoading"
            class="px-4 py-2 text-sm font-medium bg-foreground text-white rounded-lg hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2"
          >
            <Loader2 v-if="profileLoading" class="w-4 h-4 animate-spin" />
            {{ $t('profile.saveChanges') }}
          </button>
        </div>
      </form>
    </div>

    <!-- Alias Section -->
    <div class="py-8 border-b border-border">
      <h3 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground mb-5 flex items-center gap-2">
        <AtSign class="w-4 h-4" /> {{ $t('profile.alias') }}
      </h3>
      <form @submit.prevent="updateAlias" class="space-y-4 max-w-md">
        <div>
          <label class="block text-sm font-medium mb-1.5" for="alias">{{ $t('profile.aliasLabel') }}</label>
          <p class="text-xs text-muted-foreground mb-2">
            {{ aliasLocked ? $t('profile.aliasLocked') : $t('profile.aliasHint') }}
          </p>
          <input
            id="alias"
            v-model="alias"
            type="text"
            required
            pattern="[a-zA-Z0-9_-]+"
            placeholder="my-alias"
            :readonly="aliasLocked"
            class="w-full px-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all font-mono read-only:bg-muted/40 read-only:text-muted-foreground"
          />
        </div>
        <div>
          <div
            v-if="aliasMsg"
            class="mb-3 rounded-lg border border-success/30 bg-success/5 px-4 py-2.5 text-sm text-success flex items-center gap-2"
          >
            <Check class="w-4 h-4 shrink-0" /> {{ aliasMsg }}
          </div>
          <div
            v-if="aliasError"
            class="mb-3 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-2.5 text-sm text-destructive"
          >
            {{ aliasError }}
          </div>
          <button
            type="submit"
            :disabled="aliasLoading || aliasLocked"
            class="px-4 py-2 text-sm font-medium bg-foreground text-white rounded-lg hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2"
          >
            <Loader2 v-if="aliasLoading" class="w-4 h-4 animate-spin" />
            {{ $t('profile.setAlias') }}
          </button>
        </div>
      </form>
    </div>

    <!-- Change Password Section -->
    <div class="py-8 border-b border-border">
      <h3 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground mb-5 flex items-center gap-2">
        <KeyRound class="w-4 h-4" /> {{ $t('profile.changePassword') }}
      </h3>
      <form @submit.prevent="changePassword" class="space-y-4 max-w-md">
        <div>
          <label class="block text-sm font-medium mb-1.5" for="oldPassword">{{ $t('profile.currentPassword') }}</label>
          <div class="relative">
            <input
              id="oldPassword"
              v-model="oldPassword"
              :type="showOldPassword ? 'text' : 'password'"
              required
              autocomplete="current-password"
              class="w-full px-3.5 pr-10 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
            />
            <button
              type="button"
              @click="showOldPassword = !showOldPassword"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
            >
              <EyeOff v-if="showOldPassword" class="w-4 h-4" />
              <Eye v-else class="w-4 h-4" />
            </button>
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1.5" for="newPassword">{{ $t('profile.newPassword') }}</label>
          <div class="relative">
            <input
              id="newPassword"
              v-model="newPassword"
              :type="showNewPassword ? 'text' : 'password'"
              required
              :minlength="policy.min_length"
              autocomplete="new-password"
              class="w-full px-3.5 pr-10 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
            />
            <button
              type="button"
              @click="showNewPassword = !showNewPassword"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
            >
              <EyeOff v-if="showNewPassword" class="w-4 h-4" />
              <Eye v-else class="w-4 h-4" />
            </button>
          </div>
          <!-- Password policy hints -->
          <div v-if="hasRequirements && newPassword" class="mt-2 space-y-1">
            <div class="flex items-center gap-1.5 text-xs" :class="newPasswordChecks.minLength ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="newPasswordChecks.minLength" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.minLength', { n: policy.min_length }) }}
            </div>
            <div v-if="policy.require_upper" class="flex items-center gap-1.5 text-xs" :class="newPasswordChecks.upper ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="newPasswordChecks.upper" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireUpper') }}
            </div>
            <div v-if="policy.require_lower" class="flex items-center gap-1.5 text-xs" :class="newPasswordChecks.lower ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="newPasswordChecks.lower" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireLower') }}
            </div>
            <div v-if="policy.require_digit" class="flex items-center gap-1.5 text-xs" :class="newPasswordChecks.digit ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="newPasswordChecks.digit" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireDigit') }}
            </div>
            <div v-if="policy.require_symbol" class="flex items-center gap-1.5 text-xs" :class="newPasswordChecks.symbol ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="newPasswordChecks.symbol" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireSymbol') }}
            </div>
          </div>
        </div>
        <div>
          <div
            v-if="passwordMsg"
            class="mb-3 rounded-lg border border-success/30 bg-success/5 px-4 py-2.5 text-sm text-success flex items-center gap-2"
          >
            <Check class="w-4 h-4 shrink-0" /> {{ passwordMsg }}
          </div>
          <div
            v-if="passwordError"
            class="mb-3 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-2.5 text-sm text-destructive"
          >
            {{ passwordError }}
          </div>
          <button
            type="submit"
            :disabled="passwordLoading || !oldPassword || !newPasswordValid"
            class="px-4 py-2 text-sm font-medium bg-foreground text-white rounded-lg hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2"
          >
            <Loader2 v-if="passwordLoading" class="w-4 h-4 animate-spin" />
            {{ $t('profile.changePassword') }}
          </button>
        </div>
      </form>
    </div>

    <!-- Passkey Management -->
    <div class="py-8 border-b border-border">
      <div class="flex items-center justify-between mb-5">
        <h3 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-2">
          <Fingerprint class="w-4 h-4" /> {{ $t('passkey.title') }}
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
      <div v-else class="space-y-2.5 max-w-2xl">
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
