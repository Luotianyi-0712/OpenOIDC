<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Mail, Lock, Loader2, Fingerprint, Eye, EyeOff } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { usePublicConfig, getProviderIcon, isGoogleProvider, GOOGLE_SVG } from '@/composables/usePublicConfig'
import { useToastStore } from '@/stores/toast'
import { useCaptcha } from '@/composables/useCaptcha'
import { usePasskey } from '@/composables/usePasskey'

const { t } = useI18n()

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const toastStore = useToastStore()

const { providers, settings, loaded } = usePublicConfig()
const { token: captchaToken, containerId: captchaId, reset: resetCaptcha } = useCaptcha(() => settings.value.captcha_provider, () => settings.value.captcha_site_key)

const email = ref('')
const password = ref('')
const showPassword = ref(false)
const error = ref('')
const loading = ref(false)

const { loading: passkeyLoading, error: passkeyError, loginWithPasskey } = usePasskey()
const loginProviders = computed(() => providers.value.filter(p => p.login_enabled !== false))

function safeReturnTo(value: unknown) {
  return typeof value === 'string' && value.startsWith('/') && !value.startsWith('//') ? value : '/me'
}

async function handlePasskeyLogin() {
  error.value = ''
  const returnTo = safeReturnTo(route.query.return_to)
  const ok = await loginWithPasskey(returnTo)
  if (!ok && passkeyError.value && passkeyError.value !== 'cancelled') {
    error.value = passkeyError.value
  }
}

onMounted(() => {
  if (route.query.registered === '1') {
    toastStore.success(t('login.registered'))
    router.replace({ path: route.path, query: { ...route.query, registered: undefined } })
  }
  if (route.query.error) {
    const code = String(route.query.error)
    toastStore.error(t(`bindings.errors.${code}`, code))
    router.replace({ path: route.path, query: { ...route.query, error: undefined } })
  }
  if (route.query.result === 'login_success') {
    router.replace(safeReturnTo(route.query.return_to))
  }
})

function socialLogin(provider: string) {
  const returnTo = safeReturnTo(route.query.return_to)
  window.location.href = `/api/v1/social/${provider}/begin?intent=login&return_to=${encodeURIComponent(returnTo)}`
}

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(email.value, password.value, captchaToken.value || undefined)
    const returnTo = safeReturnTo(route.query.return_to)
    router.push(returnTo)
  } catch (e: any) {
    const msg = e.message || 'Login failed. Please try again.'
    if (msg.includes('locked') || msg.includes('too many')) {
      error.value = t('login.accountLocked')
    } else {
      error.value = msg
    }
    resetCaptcha()
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="w-full max-w-[420px]">
    <div class="text-center mb-8">
      <h1 class="text-2xl font-bold tracking-tight">{{ $t('login.title') }}</h1>
      <p class="text-muted-foreground text-[0.9375rem] mt-1.5">
        {{ $t('login.subtitle') }}
      </p>
    </div>

    <!-- Error alert -->
    <div
      v-if="error"
      class="mb-5 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
    >
      {{ error }}
    </div>

    <!-- Passkey login -->
    <button
      v-if="settings.passkey_enabled"
      type="button"
      :disabled="passkeyLoading"
      class="w-full border border-border rounded-lg py-2.5 px-4 text-sm font-medium hover:bg-muted transition-colors flex items-center justify-center gap-2.5 mb-2.5"
      @click="handlePasskeyLogin"
    >
      <Loader2 v-if="passkeyLoading" class="w-4 h-4 animate-spin" />
      <Fingerprint v-else class="w-[18px] h-[18px]" />
      {{ $t('login.passkey') }}
    </button>

    <!-- Social buttons (dynamic from backend) -->
    <div v-if="settings.social_login_enabled && loginProviders.length" class="flex flex-col gap-2.5">
      <button
        v-for="p in loginProviders"
        :key="p.name"
        type="button"
        class="w-full border border-border rounded-lg py-2.5 px-4 text-sm font-medium hover:bg-muted transition-colors flex items-center justify-center gap-2.5"
        @click="socialLogin(p.name)"
      >
        <img
          v-if="p.icon_url"
          :src="p.icon_url"
          :alt="p.display_name"
          class="w-[18px] h-[18px] object-contain"
        />
        <span v-else-if="isGoogleProvider(p.name)" v-html="GOOGLE_SVG" />
        <svg v-else-if="getProviderIcon(p.name)?.path" class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none">
          <path :d="getProviderIcon(p.name)!.path" :fill="getProviderIcon(p.name)!.color" />
        </svg>
        <span v-else class="text-xs font-semibold text-muted-foreground">
          {{ p.display_name.slice(0, 1).toUpperCase() }}
        </span>
        {{ p.display_name }}
      </button>
    </div>

    <!-- Divider -->
    <div v-if="settings.social_login_enabled && providers.length && settings.password_login_enabled" class="flex items-center gap-3.5 my-6">
      <div class="flex-1 h-px bg-border" />
      <span class="text-muted-foreground text-xs font-medium uppercase tracking-wider">{{ $t('or') }}</span>
      <div class="flex-1 h-px bg-border" />
    </div>

    <!-- Form -->
    <form v-if="settings.password_login_enabled" @submit.prevent="onSubmit" class="flex flex-col gap-4">
      <div>
        <label class="block text-sm font-medium mb-1.5" for="email">{{ $t('login.email') }}</label>
        <div class="relative">
          <Mail class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
          <input
            id="email"
            v-model="email"
            type="email"
            required
            autocomplete="username"
            placeholder="name@example.com"
            class="w-full pl-9.5 pr-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
          />
        </div>
      </div>
      <div>
        <div class="flex items-center justify-between mb-1.5">
          <label class="text-sm font-medium" for="password">{{ $t('login.password') }}</label>
          <RouterLink
            to="/forgot-password"
            class="text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            {{ $t('login.forgot') }}
          </RouterLink>
        </div>
        <div class="relative">
          <Lock class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
          <input
            id="password"
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            required
            autocomplete="current-password"
            :placeholder="$t('login.passwordPlaceholder')"
            class="w-full pl-9.5 pr-10 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
          />
          <button
            type="button"
            @click="showPassword = !showPassword"
            class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
          >
            <EyeOff v-if="showPassword" class="w-4 h-4" />
            <Eye v-else class="w-4 h-4" />
          </button>
        </div>
      </div>
      <div v-if="settings.captcha_enabled && settings.captcha_site_key" :id="captchaId" class="flex justify-center"></div>
      <button
        type="submit"
        :disabled="loading"
        class="w-full bg-foreground text-white rounded-full py-2.5 text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2 mt-1"
      >
        <Loader2 v-if="loading" class="w-4 h-4 animate-spin" />
        {{ loading ? $t('login.submitting') : $t('login.submit') }}
      </button>
    </form>

    <p class="text-center text-sm text-muted-foreground mt-6">
      {{ $t('login.noAccount') }}
      <RouterLink to="/register" class="text-foreground font-medium hover:underline">
        {{ $t('login.signUp') }}
      </RouterLink>
    </p>
  </div>
</template>
