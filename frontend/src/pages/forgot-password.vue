<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { api } from '@/api/client'
import { Mail, ArrowLeft, Loader2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { usePublicConfig } from '@/composables/usePublicConfig'
import { useCaptcha } from '@/composables/useCaptcha'

const { t } = useI18n()
const { settings } = usePublicConfig()

const { token: captchaToken, containerId: captchaId, reset: resetCaptcha } = useCaptcha(
  () => settings.value.captcha_provider,
  () => settings.value.captcha_site_key
)

const email = ref('')
const error = ref('')
const success = ref(false)
const loading = ref(false)

async function onSubmit() {
  error.value = ''
  success.value = false
  loading.value = true
  try {
    const headers: Record<string, string> = {}
    if (captchaToken.value) {
      headers['X-Captcha-Token'] = captchaToken.value
      headers['X-Turnstile-Token'] = captchaToken.value
    }
    await api.post('/auth/forgot-password', { email: email.value }, headers)
    success.value = true
  } catch (e: any) {
    error.value = e.message || 'Something went wrong. Please try again.'
    resetCaptcha()
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="w-full max-w-[420px]">
    <div class="text-center mb-8">
      <h1 class="text-2xl font-bold tracking-tight">{{ $t('forgot.title') }}</h1>
      <p class="text-muted-foreground text-[0.9375rem] mt-1.5">
        {{ $t('forgot.subtitle') }}
      </p>
    </div>

    <!-- Success alert -->
    <div
      v-if="success"
      class="mb-5 rounded-lg border border-success/30 bg-success/5 px-4 py-3 text-sm text-success"
    >
      {{ $t('forgot.success') }}
    </div>

    <!-- Error alert -->
    <div
      v-if="error"
      class="mb-5 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
    >
      {{ error }}
    </div>

    <!-- Form -->
    <form @submit.prevent="onSubmit" class="flex flex-col gap-4">
      <div>
        <label class="block text-sm font-medium mb-1.5" for="email">{{ $t('forgot.email') }}</label>
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
      <div v-if="settings.captcha_enabled && settings.captcha_site_key" :id="captchaId" class="flex justify-center"></div>
      <button
        type="submit"
        :disabled="loading"
        class="w-full bg-foreground text-white rounded-full py-2.5 text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2 mt-1"
      >
        <Loader2 v-if="loading" class="w-4 h-4 animate-spin" />
        {{ loading ? $t('forgot.submitting') : $t('forgot.submit') }}
      </button>
    </form>

    <p class="text-center text-sm text-muted-foreground mt-6">
      <RouterLink
        to="/login"
        class="inline-flex items-center gap-1.5 text-foreground font-medium hover:underline"
      >
        <ArrowLeft class="w-3.5 h-3.5" />
        {{ $t('forgot.back') }}
      </RouterLink>
    </p>
  </div>
</template>
