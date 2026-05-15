<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'
import { User, Mail, Shield } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const route = useRoute()
const loading = ref<'allow' | 'deny' | null>(null)
const error = ref('')

const clientName = (route.query.client_name as string) || 'Unknown App'
const clientUri = (route.query.client_uri as string) || ''
const clientInitial = clientName.charAt(0).toUpperCase()

const permissions = computed(() => [
  { icon: User, label: t('authorize.profile'), desc: t('authorize.profileDesc') },
  { icon: Mail, label: t('authorize.emailLabel'), desc: t('authorize.emailDesc') },
  { icon: Shield, label: t('authorize.securityLevel'), desc: t('authorize.securityLevelDesc') },
])

async function handleAllow() {
  loading.value = 'allow'
  error.value = ''
  try {
    const res = await api.post<{ redirect_uri: string }>('/consent/accept', {
      consent_challenge: route.query.consent_challenge,
    })
    if (res.data?.redirect_uri) {
      window.location.href = res.data.redirect_uri
    }
  } catch (e: any) {
    error.value = e.message || 'Failed to authorize. Please try again.'
    loading.value = null
  }
}

async function handleDeny() {
  loading.value = 'deny'
  error.value = ''
  try {
    const res = await api.post<{ redirect_uri: string }>('/consent/reject', {
      consent_challenge: route.query.consent_challenge,
    })
    if (res.data?.redirect_uri) {
      window.location.href = res.data.redirect_uri
    }
  } catch (e: any) {
    error.value = e.message || 'Failed to deny. Please try again.'
    loading.value = null
  }
}
</script>

<template>
  <div class="w-full max-w-[460px]">
    <!-- Heading -->
    <div class="text-center mb-8">
      <h1 class="text-2xl font-bold tracking-tight text-foreground">{{ $t('authorize.title') }}</h1>
      <p class="text-sm text-muted-foreground mt-2">
        {{ $t('authorize.subtitle') }}
      </p>
    </div>

    <!-- Client info -->
    <div class="bg-muted rounded-xl p-5 flex items-center gap-4 mb-6">
      <div class="w-11 h-11 rounded-lg bg-foreground text-white flex items-center justify-center text-lg font-bold shrink-0">
        {{ clientInitial }}
      </div>
      <div class="min-w-0">
        <div class="text-sm font-semibold text-foreground truncate">{{ clientName }}</div>
        <div v-if="clientUri" class="text-xs text-muted-foreground truncate mt-0.5">{{ clientUri }}</div>
      </div>
    </div>

    <!-- Permissions -->
    <div class="mb-6">
      <div class="text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-3">{{ $t('authorize.permissions') }}</div>
      <div class="space-y-3">
        <div
          v-for="perm in permissions"
          :key="perm.label"
          class="flex items-center gap-3.5"
        >
          <div class="w-9 h-9 rounded-lg border border-border flex items-center justify-center text-muted-foreground shrink-0">
            <component :is="perm.icon" class="w-4 h-4" />
          </div>
          <div class="min-w-0">
            <div class="text-sm font-medium text-foreground">{{ perm.label }}</div>
            <div class="text-xs text-muted-foreground">{{ perm.desc }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Error -->
    <div v-if="error" class="text-sm text-destructive bg-destructive/10 rounded-lg px-4 py-2.5 mb-4">
      {{ error }}
    </div>

    <!-- Buttons -->
    <div class="flex items-center gap-3">
      <button
        @click="handleDeny"
        :disabled="loading !== null"
        class="flex-1 h-11 rounded-full border border-border text-sm font-medium text-foreground hover:bg-muted transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {{ loading === 'deny' ? $t('authorize.denying') : $t('authorize.deny') }}
      </button>
      <button
        @click="handleAllow"
        :disabled="loading !== null"
        class="flex-1 h-11 rounded-full bg-foreground text-white text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {{ loading === 'allow' ? $t('authorize.allowing') : $t('authorize.allow') }}
      </button>
    </div>

    <!-- Footer text -->
    <i18n-t keypath="authorize.revokeHint" tag="p" class="text-xs text-muted-foreground text-center mt-6 leading-relaxed">
      <template #link>
        <RouterLink to="/me/bindings" class="underline underline-offset-2 hover:text-foreground transition-colors">{{ $t('authorize.accountSettings') }}</RouterLink>
      </template>
    </i18n-t>
  </div>
</template>
