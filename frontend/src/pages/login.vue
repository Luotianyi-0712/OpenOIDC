<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Mail, Lock, Loader2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { usePublicConfig, getProviderIcon, isGoogleProvider, GOOGLE_SVG } from '@/composables/usePublicConfig'
import { useToastStore } from '@/stores/toast'
import { useTurnstile } from '@/composables/useTurnstile'

const { t } = useI18n()

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const toastStore = useToastStore()

const { providers, settings, loaded } = usePublicConfig()
const { token: turnstileToken, containerId: turnstileId, reset: resetTurnstile, renderWidget } = useTurnstile(() => settings.value.turnstile_site_key)

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

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
    router.replace((route.query.return_to as string) || '/me')
  }
})

function socialLogin(provider: string) {
  const returnTo = (route.query.return_to as string) || '/me'
  window.location.href = `/api/v1/social/${provider}/begin?return_to=${encodeURIComponent(returnTo)}`
}

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(email.value, password.value, turnstileToken.value || undefined)
    const returnTo = (route.query.return_to as string) || '/me'
    router.push(returnTo)
  } catch (e: any) {
    const msg = e.message || 'Login failed. Please try again.'
    if (msg.includes('locked') || msg.includes('too many')) {
      error.value = t('login.accountLocked')
    } else {
      error.value = msg
    }
    resetTurnstile()
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

    <!-- Social buttons (dynamic from backend) -->
    <div v-if="settings.social_login_enabled && providers.length" class="flex flex-col gap-2.5">
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
            autocomplete="email"
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
            type="password"
            required
            autocomplete="current-password"
            placeholder="Enter your password"
            class="w-full pl-9.5 pr-3.5 py-2.5 border border-border rounded-lg text-sm outline-none focus:ring-2 focus:ring-foreground/10 focus:border-foreground transition-all"
          />
        </div>
      </div>
      <div v-if="settings.turnstile_site_key" :id="turnstileId" class="flex justify-center"></div>
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
