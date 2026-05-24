<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Mail, Lock, User, Loader2, Check, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { usePublicConfig, getProviderIcon, isGoogleProvider, GOOGLE_SVG } from '@/composables/usePublicConfig'
import { useTurnstile } from '@/composables/useTurnstile'
import { usePasswordPolicy } from '@/composables/usePasswordPolicy'

const { t } = useI18n()

const router = useRouter()
const auth = useAuthStore()

const { providers, settings, loaded } = usePublicConfig()
const { token: turnstileToken, containerId: turnstileId, reset: resetTurnstile, renderWidget } = useTurnstile(() => settings.value.turnstile_site_key)
const { policy, hasRequirements, validate } = usePasswordPolicy()

const displayName = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const passwordErrors = computed(() => password.value ? validate(password.value) : [])
const passwordValid = computed(() => password.value.length > 0 && passwordErrors.value.length === 0)

function socialLogin(provider: string) {
  window.location.href = `/api/v1/social/${provider}/begin?return_to=/`
}

async function onSubmit() {
  error.value = ''
  if (passwordErrors.value.length > 0) {
    error.value = t('passwordPolicy.notMet')
    return
  }
  loading.value = true
  try {
    await auth.register(email.value, password.value, displayName.value, turnstileToken.value || undefined)
    router.push('/login?registered=1')
  } catch (e: any) {
    error.value = e.message || 'Registration failed. Please try again.'
    resetTurnstile()
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="w-full max-w-[420px]">
    <div class="text-center mb-8">
      <h1 class="text-2xl font-bold tracking-tight">{{ $t('register.title') }}</h1>
      <p class="text-muted-foreground text-[0.9375rem] mt-1.5">
        {{ $t('register.subtitle') }}
      </p>
    </div>

    <!-- Registration disabled -->
    <div v-if="loaded && !settings.registration_enabled" class="text-center text-muted-foreground text-sm py-8">
      {{ $t('register.disabled') }}
      <p class="mt-2">
        <RouterLink to="/login" class="text-foreground font-medium hover:underline">
          {{ $t('register.signIn') }}
        </RouterLink>
      </p>
    </div>

    <template v-else>
      <!-- Error alert -->
      <div
        v-if="error"
        class="mb-5 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
      >
        {{ error }}
      </div>

      <!-- Social buttons (dynamic from backend) -->
      <div v-if="settings.social_register_enabled && providers.length" class="flex flex-col gap-2.5">
        <button
          v-for="p in providers"
          :key="p.name"
          type="button"
          class="w-full border border-border rounded-lg py-2.5 px-4 text-sm font-medium hover:bg-muted transition-colors flex items-center justify-center gap-2.5"
          @click="socialLogin(p.name)"
        >
          <span v-if="isGoogleProvider(p.name)" v-html="GOOGLE_SVG" />
          <svg v-else-if="getProviderIcon(p.name)" class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none">
            <path :d="getProviderIcon(p.name)!.path" :fill="getProviderIcon(p.name)!.color" />
          </svg>
          {{ p.display_name }}
        </button>
      </div>

      <!-- Divider -->
      <div v-if="settings.social_register_enabled && providers.length && settings.registration_enabled" class="flex items-center gap-3.5 my-6">
        <div class="flex-1 h-px bg-border" />
        <span class="text-muted-foreground text-xs font-medium uppercase tracking-wider">{{ $t('or') }}</span>
        <div class="flex-1 h-px bg-border" />
      </div>

      <!-- Form -->
      <form v-if="settings.registration_enabled" @submit.prevent="onSubmit" class="flex flex-col gap-4">
        <div>
          <label class="block text-sm font-medium mb-1.5" for="name">{{ $t('register.name') }}</label>
          <div class="relative">
            <User class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
            <input
              id="name"
              v-model="displayName"
              type="text"
              required
              autocomplete="name"
              :placeholder="t('register.namePlaceholder')"
              class="w-full pl-9.5 pr-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
            />
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1.5" for="email">{{ $t('register.email') }}</label>
          <div class="relative">
            <Mail class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
            <input
              id="email"
              v-model="email"
              type="email"
              required
              autocomplete="email"
              placeholder="name@example.com"
              class="w-full pl-9.5 pr-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
            />
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1.5" for="password">{{ $t('register.password') }}</label>
          <div class="relative">
            <Lock class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
            <input
              id="password"
              v-model="password"
              type="password"
              required
              autocomplete="new-password"
              :placeholder="t('register.passwordPlaceholder')"
              class="w-full pl-9.5 pr-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
            />
          </div>
          <!-- Password policy hints -->
          <div v-if="hasRequirements && password" class="mt-2 space-y-1">
            <div class="flex items-center gap-1.5 text-xs" :class="password.length >= policy.min_length ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="password.length >= policy.min_length" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.minLength', { n: policy.min_length }) }}
            </div>
            <div v-if="policy.require_upper" class="flex items-center gap-1.5 text-xs" :class="/[A-Z]/.test(password) ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="/[A-Z]/.test(password)" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireUpper') }}
            </div>
            <div v-if="policy.require_lower" class="flex items-center gap-1.5 text-xs" :class="/[a-z]/.test(password) ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="/[a-z]/.test(password)" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireLower') }}
            </div>
            <div v-if="policy.require_digit" class="flex items-center gap-1.5 text-xs" :class="/[0-9]/.test(password) ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="/[0-9]/.test(password)" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireDigit') }}
            </div>
            <div v-if="policy.require_symbol" class="flex items-center gap-1.5 text-xs" :class="/[!@#$%^&*()\-_=+\[\]{};:,.<>/?\\|`~]/.test(password) ? 'text-success' : 'text-muted-foreground'">
              <Check v-if="/[!@#$%^&*()\-_=+\[\]{};:,.<>/?\\|`~]/.test(password)" class="w-3 h-3" />
              <X v-else class="w-3 h-3" />
              {{ $t('passwordPolicy.requireSymbol') }}
            </div>
          </div>
        </div>
        <div v-if="settings.turnstile_site_key" :id="turnstileId" class="flex justify-center"></div>
        <button
          type="submit"
          :disabled="loading"
          class="w-full bg-foreground text-white rounded-full py-2.5 text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2 mt-1"
        >
          <Loader2 v-if="loading" class="w-4 h-4 animate-spin" />
          {{ loading ? $t('register.submitting') : $t('register.submit') }}
        </button>
      </form>

      <p class="text-center text-sm text-muted-foreground mt-6">
        {{ $t('register.hasAccount') }}
        <RouterLink to="/login" class="text-foreground font-medium hover:underline">
          {{ $t('register.signIn') }}
        </RouterLink>
      </p>
    </template>
  </div>
</template>
