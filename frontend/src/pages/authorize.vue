<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'
import { User, Mail, Shield, KeyRound, RefreshCw, HelpCircle, Building2, Link as LinkIcon } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const route = useRoute()
const loading = ref<'allow' | 'deny' | null>(null)
const pageLoading = ref(true)
const error = ref('')

interface ConsentContext {
  client: {
    client_id: string
    name: string
    description: string
    logo_url: string
    redirect_uri: string
    homepage_url: string
  }
  developer: {
    id: string
  }
  scopes: string[]
}

const consent = ref<ConsentContext | null>(null)
const consentChallenge = computed(() => route.query.consent_challenge as string)

const clientName = computed(() => consent.value?.client.name || t('authorize.unknownApp'))
const clientInitial = computed(() => clientName.value.charAt(0).toUpperCase())
const requestedScopes = computed(() => consent.value?.scopes?.length ? consent.value.scopes : ['openid'])

const scopeMeta = computed<Record<string, { icon: any; label: string; desc: string }>>(() => ({
  openid: { icon: KeyRound, label: t('authorize.scopeOpenID'), desc: t('authorize.scopeOpenIDDesc') },
  profile: { icon: User, label: t('authorize.profile'), desc: t('authorize.profileDesc') },
  email: { icon: Mail, label: t('authorize.emailLabel'), desc: t('authorize.emailDesc') },
  security_level: { icon: Shield, label: t('authorize.securityLevel'), desc: t('authorize.securityLevelDesc') },
  offline_access: { icon: RefreshCw, label: t('authorize.offlineAccess'), desc: t('authorize.offlineAccessDesc') },
}))

const permissions = computed(() => requestedScopes.value.map(scope => {
  const meta = scopeMeta.value[scope]
  if (meta) return { ...meta, scope }
  return { icon: HelpCircle, label: scope, desc: t('authorize.unknownScopeDesc'), scope }
}))

onMounted(fetchConsentContext)

async function fetchConsentContext() {
  pageLoading.value = true
  error.value = ''
  try {
    if (!consentChallenge.value) throw new Error(t('authorize.missingChallenge'))
    const res = await api.get<ConsentContext>(`/consent/context?consent_challenge=${encodeURIComponent(consentChallenge.value)}`)
    consent.value = res.data ?? null
  } catch (e: any) {
    error.value = e.message || t('authorize.loadFailed')
  } finally {
    pageLoading.value = false
  }
}

async function handleAllow() {
  loading.value = 'allow'
  error.value = ''
  try {
    const res = await api.post<{ redirect_uri: string }>('/consent/accept', {
      consent_challenge: consentChallenge.value,
    })
    if (res.data?.redirect_uri) {
      window.location.href = res.data.redirect_uri
    }
  } catch (e: any) {
    error.value = e.message || t('authorize.allowFailed')
    loading.value = null
  }
}

async function handleDeny() {
  loading.value = 'deny'
  error.value = ''
  try {
    const res = await api.post<{ redirect_uri: string }>('/consent/reject', {
      consent_challenge: consentChallenge.value,
    })
    if (res.data?.redirect_uri) {
      window.location.href = res.data.redirect_uri
    }
  } catch (e: any) {
    error.value = e.message || t('authorize.denyFailed')
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

    <div v-if="pageLoading" class="bg-muted rounded-xl p-5 mb-6 text-sm text-muted-foreground text-center">
      {{ $t('authorize.loading') }}
    </div>

    <!-- Client info -->
    <div v-else-if="consent" class="bg-muted rounded-xl p-5 flex items-start gap-4 mb-6">
      <img
        v-if="consent.client.logo_url"
        :src="consent.client.logo_url"
        :alt="clientName"
        class="w-11 h-11 rounded-lg object-cover border border-border shrink-0 bg-background"
      />
      <div v-else class="w-11 h-11 rounded-lg bg-foreground text-white flex items-center justify-center text-lg font-bold shrink-0">
        {{ clientInitial }}
      </div>
      <div class="min-w-0 flex-1">
        <div class="text-sm font-semibold text-foreground truncate">{{ clientName }}</div>
        <div v-if="consent.client.description" class="text-xs text-muted-foreground mt-1 leading-relaxed">
          {{ consent.client.description }}
        </div>
        <div class="mt-3 grid gap-2 text-xs text-muted-foreground">
          <div class="flex items-center gap-2 min-w-0">
            <Building2 class="w-3.5 h-3.5 shrink-0" />
            <span class="shrink-0">{{ $t('authorize.developerId') }}</span>
            <span class="font-mono text-foreground truncate">{{ consent.developer.id || $t('authorize.unknownDeveloper') }}</span>
          </div>
          <div v-if="consent.client.homepage_url" class="flex items-center gap-2 min-w-0">
            <LinkIcon class="w-3.5 h-3.5 shrink-0" />
            <span class="shrink-0">{{ $t('authorize.website') }}</span>
            <a
              :href="consent.client.homepage_url"
              target="_blank"
              rel="noopener noreferrer"
              class="font-mono text-foreground truncate hover:underline"
            >{{ consent.client.homepage_url }}</a>
          </div>
          <div class="flex items-center gap-2 min-w-0">
            <LinkIcon class="w-3.5 h-3.5 shrink-0" />
            <span class="shrink-0">{{ $t('authorize.redirectTo') }}</span>
            <span class="font-mono truncate">{{ consent.client.redirect_uri }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Permissions -->
    <div class="mb-6">
      <div class="text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-3">{{ $t('authorize.permissions') }}</div>
      <div class="space-y-3">
        <div
          v-for="perm in permissions"
          :key="perm.scope"
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
        :disabled="loading !== null || pageLoading || !consent"
        class="flex-1 h-11 rounded-full border border-border text-sm font-medium text-foreground hover:bg-muted transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {{ loading === 'deny' ? $t('authorize.denying') : $t('authorize.deny') }}
      </button>
      <button
        @click="handleAllow"
        :disabled="loading !== null || pageLoading || !consent"
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
