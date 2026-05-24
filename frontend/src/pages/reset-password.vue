<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'
import { KeyRound, Loader2, CheckCircle, ArrowLeft, Check, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { usePasswordPolicy } from '@/composables/usePasswordPolicy'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { policy, hasRequirements, validate } = usePasswordPolicy()

const token = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const error = ref('')
const success = ref(false)
const loading = ref(false)
const noToken = ref(false)

const passwordErrors = computed(() => newPassword.value ? validate(newPassword.value) : [])
const passwordValid = computed(() => newPassword.value.length > 0 && passwordErrors.value.length === 0)

onMounted(() => {
  token.value = (route.query.token as string) || ''
  if (!token.value) {
    noToken.value = true
  }
})

async function onSubmit() {
  error.value = ''
  if (passwordErrors.value.length > 0) {
    error.value = t('passwordPolicy.notMet')
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    error.value = t('resetPassword.mismatch')
    return
  }
  loading.value = true
  try {
    await api.post('/auth/reset-password', {
      token: token.value,
      new_password: newPassword.value,
    })
    success.value = true
  } catch (e: any) {
    error.value = e.message || t('resetPassword.failed')
  } finally {
    loading.value = false
  }
}

function goLogin() {
  router.push('/login')
}
</script>

<template>
  <div class="w-full max-w-[420px]">
    <!-- No token state -->
    <div v-if="noToken" class="text-center">
      <h1 class="text-2xl font-bold tracking-tight mb-3">{{ $t('resetPassword.title') }}</h1>
      <p class="text-sm text-muted-foreground mb-6">{{ $t('resetPassword.noToken') }}</p>
      <router-link
        to="/forgot-password"
        class="inline-flex items-center gap-1.5 text-foreground font-medium hover:underline text-sm"
      >
        <ArrowLeft class="w-3.5 h-3.5" />
        {{ $t('forgot.title') }}
      </router-link>
    </div>

    <!-- Success state -->
    <div v-else-if="success" class="text-center">
      <CheckCircle class="w-12 h-12 text-success mx-auto mb-4" />
      <h1 class="text-2xl font-bold tracking-tight mb-2">{{ $t('resetPassword.title') }}</h1>
      <p class="text-sm text-muted-foreground mb-6">{{ $t('resetPassword.success') }}</p>
      <button
        @click="goLogin"
        class="bg-foreground text-white px-6 py-2.5 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors"
      >
        {{ $t('resetPassword.goLogin') }}
      </button>
    </div>

    <!-- Form state -->
    <template v-else>
      <div class="text-center mb-8">
        <h1 class="text-2xl font-bold tracking-tight">{{ $t('resetPassword.title') }}</h1>
        <p class="text-muted-foreground text-[0.9375rem] mt-1.5">
          {{ $t('resetPassword.subtitle') }}
        </p>
      </div>

      <!-- Error alert -->
      <div
        v-if="error"
        class="mb-5 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
      >
        {{ error }}
      </div>

      <form @submit.prevent="onSubmit" class="flex flex-col gap-4">
        <div>
          <label class="block text-sm font-medium mb-1.5" for="newPwd">{{ $t('resetPassword.newPassword') }}</label>
          <div class="relative">
            <KeyRound class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
            <input
              id="newPwd"
              v-model="newPassword"
              type="password"
              required
              minlength="8"
              autocomplete="new-password"
              class="w-full pl-9.5 pr-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
            />
          </div>
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
          <label class="block text-sm font-medium mb-1.5" for="confirmPwd">{{ $t('resetPassword.confirmPassword') }}</label>
          <div class="relative">
            <KeyRound class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
            <input
              id="confirmPwd"
              v-model="confirmPassword"
              type="password"
              required
              minlength="8"
              autocomplete="new-password"
              class="w-full pl-9.5 pr-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
            />
          </div>
        </div>
        <button
          type="submit"
          :disabled="loading"
          class="w-full bg-foreground text-white rounded-full py-2.5 text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2 mt-1"
        >
          <Loader2 v-if="loading" class="w-4 h-4 animate-spin" />
          {{ loading ? $t('resetPassword.submitting') : $t('resetPassword.submit') }}
        </button>
      </form>

      <p class="text-center text-sm text-muted-foreground mt-6">
        <router-link
          to="/login"
          class="inline-flex items-center gap-1.5 text-foreground font-medium hover:underline"
        >
          <ArrowLeft class="w-3.5 h-3.5" />
          {{ $t('forgot.back') }}
        </router-link>
      </p>
    </template>
  </div>
</template>
