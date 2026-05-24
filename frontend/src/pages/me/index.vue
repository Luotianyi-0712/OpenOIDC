<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/api/client'
import { Loader2, Check, BadgeCheck, AtSign, KeyRound, UserPen, ImagePlus, Mail, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { usePasswordPolicy } from '@/composables/usePasswordPolicy'

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

// Password form
const oldPassword = ref('')
const newPassword = ref('')
const passwordLoading = ref(false)
const passwordMsg = ref('')
const passwordError = ref('')

const newPasswordErrors = computed(() => newPassword.value ? validate(newPassword.value) : [])

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
        <div v-if="auth.user?.alias" class="text-sm text-muted-foreground mt-0.5">
          @{{ auth.user.alias }}
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
            {{ $t('profile.aliasHint') }}
          </p>
          <input
            id="alias"
            v-model="alias"
            type="text"
            required
            pattern="[a-zA-Z0-9_-]+"
            placeholder="my-alias"
            class="w-full px-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all font-mono"
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
            :disabled="aliasLoading"
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
          <input
            id="oldPassword"
            v-model="oldPassword"
            type="password"
            required
            autocomplete="current-password"
            class="w-full px-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
          />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1.5" for="newPassword">{{ $t('profile.newPassword') }}</label>
          <input
            id="newPassword"
            v-model="newPassword"
            type="password"
            required
            minlength="8"
            autocomplete="new-password"
            class="w-full px-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
          />
          <!-- Password policy hints -->
          <div v-if="hasRequirements && newPassword" class="mt-2 space-y-1">
            <div class="flex items-center gap-1.5 text-xs" :class="newPassword.length >= policy.min_length ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="newPassword.length >= policy.min_length" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.minLength', { n: policy.min_length }) }}
            </div>
            <div v-if="policy.require_upper" class="flex items-center gap-1.5 text-xs" :class="/[A-Z]/.test(newPassword) ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="/[A-Z]/.test(newPassword)" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireUpper') }}
            </div>
            <div v-if="policy.require_lower" class="flex items-center gap-1.5 text-xs" :class="/[a-z]/.test(newPassword) ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="/[a-z]/.test(newPassword)" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireLower') }}
            </div>
            <div v-if="policy.require_digit" class="flex items-center gap-1.5 text-xs" :class="/[0-9]/.test(newPassword) ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="/[0-9]/.test(newPassword)" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireDigit') }}
            </div>
            <div v-if="policy.require_symbol" class="flex items-center gap-1.5 text-xs" :class="/[!@#$%^&*()\-_=+\[\]{};:,.<>/?\\|`~]/.test(newPassword) ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="/[!@#$%^&*()\-_=+\[\]{};:,.<>/?\\|`~]/.test(newPassword)" class="w-3 h-3" />
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
            :disabled="passwordLoading"
            class="px-4 py-2 text-sm font-medium bg-foreground text-white rounded-lg hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2"
          >
            <Loader2 v-if="passwordLoading" class="w-4 h-4 animate-spin" />
            {{ $t('profile.changePassword') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
