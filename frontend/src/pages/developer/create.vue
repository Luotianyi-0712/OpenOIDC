<script setup lang="ts">
import { ref } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { Loader2, Copy, Check, X, ArrowLeft } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()

const form = ref({
  client_name: '',
  description: '',
  logo_url: '',
  homepage_url: '',
  redirect_uris: '',
  scopes: ['openid'] as string[],
  grant_types: ['authorization_code', 'refresh_token'] as string[],
  min_security_level: 0,
})

const allScopes = ['openid', 'profile', 'email', 'security_level']
const allGrantTypes = ['authorization_code', 'refresh_token', 'client_credentials']

const saving = ref(false)
const error = ref('')

async function refreshDeveloperStatus() {
  await auth.fetchDeveloperStatus()
}

refreshDeveloperStatus()

// Secret modal
const showSecretModal = ref(false)
const revealedSecret = ref('')
const createdAppId = ref('')
const secretCopied = ref(false)

function toggleScope(scope: string) {
  const idx = form.value.scopes.indexOf(scope)
  if (idx >= 0) form.value.scopes.splice(idx, 1)
  else form.value.scopes.push(scope)
}

function toggleGrant(grant: string) {
  const idx = form.value.grant_types.indexOf(grant)
  if (idx >= 0) form.value.grant_types.splice(idx, 1)
  else form.value.grant_types.push(grant)
}

async function submit() {
  if (!auth.canCreateDeveloperApp) {
    error.value = t('developer.createLevelRequired', {
      current: auth.developerStatus?.current_trust_level ?? auth.user?.security_level ?? 0,
      min: auth.developerStatus?.min_trust_level ?? auth.developerMinTrustLevel,
    })
    return
  }
  saving.value = true
  error.value = ''
  try {
    const payload = {
      client_name: form.value.client_name,
      description: form.value.description,
      logo_url: form.value.logo_url || undefined,
      homepage_url: form.value.homepage_url || undefined,
      redirect_uris: form.value.redirect_uris.split('\n').map(s => s.trim()).filter(Boolean),
      scopes: form.value.scopes,
      grant_types: form.value.grant_types,
      min_security_level: form.value.min_security_level,
    }
    const res = await api.post<{ id: string; client_secret: string }>('/developer/apps', payload)
    await auth.fetchDeveloperStatus()
    if (res.data) {
      createdAppId.value = res.data.id
      if (res.data.client_secret) {
        revealedSecret.value = res.data.client_secret
        showSecretModal.value = true
      } else {
        router.push(`/developer/apps/${res.data.id}`)
      }
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

async function copySecret() {
  await navigator.clipboard.writeText(revealedSecret.value)
  secretCopied.value = true
  setTimeout(() => (secretCopied.value = false), 2000)
}

function closeSecretModal() {
  showSecretModal.value = false
  router.push(`/developer/apps/${createdAppId.value}`)
}
</script>

<template>
  <div class="max-w-2xl">
    <!-- Back link -->
    <router-link to="/developer" class="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-6">
      <ArrowLeft class="w-4 h-4" /> {{ $t('devNav.myApps') }}
    </router-link>

    <div class="mb-6">
      <h2 class="text-lg font-semibold">{{ $t('developer.createApp') }}</h2>
      <p class="text-sm text-muted-foreground mt-1">{{ $t('developer.createSubtitle') }}</p>
    </div>

    <!-- Error -->
    <div v-if="!auth.canCreateDeveloperApp" class="mb-4 rounded-lg border border-yellow-200 bg-yellow-50 px-4 py-3 text-sm text-yellow-800">
      {{ $t('developer.createLevelRequired', {
        current: auth.developerStatus?.current_trust_level ?? auth.user?.security_level ?? 0,
        min: auth.developerStatus?.min_trust_level ?? auth.developerMinTrustLevel,
      }) }}
    </div>
    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <form @submit.prevent="submit" class="flex flex-col gap-5">
      <!-- App Name -->
      <div>
        <label class="block text-sm font-medium mb-1.5">{{ $t('developer.appName') }}</label>
        <input
          v-model="form.client_name"
          type="text"
          required
          :placeholder="$t('developer.appNamePlaceholder')"
          class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
        />
        <p class="text-xs text-muted-foreground mt-1">{{ $t('developer.appNameHint') }}</p>
      </div>

      <!-- Description -->
      <div>
        <label class="block text-sm font-medium mb-1.5">{{ $t('developer.description') }}</label>
        <textarea
          v-model="form.description"
          rows="3"
          :placeholder="$t('developer.descriptionPlaceholder')"
          class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-none"
        />
        <p class="text-xs text-muted-foreground mt-1">{{ $t('developer.descriptionHint') }}</p>
      </div>

      <!-- Logo URL -->
      <div>
        <label class="block text-sm font-medium mb-1.5">{{ $t('developer.logoUrl') }}</label>
        <input
          v-model="form.logo_url"
          type="url"
          placeholder="https://example.com/logo.png"
          class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
        />
        <p class="text-xs text-muted-foreground mt-1">{{ $t('developer.logoUrlHint') }}</p>
        <div v-if="form.logo_url" class="mt-2">
          <img :src="form.logo_url" :alt="$t('developer.logoPreview')" class="w-12 h-12 rounded-lg object-cover border border-border" />
        </div>
      </div>

      <!-- Website URL -->
      <div>
        <label class="block text-sm font-medium mb-1.5">{{ $t('developer.homepageUrl') }}</label>
        <input
          v-model="form.homepage_url"
          type="url"
          :placeholder="$t('developer.homepageUrlPlaceholder')"
          class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
        />
        <p class="text-xs text-muted-foreground mt-1">{{ $t('developer.homepageUrlHint') }}</p>
      </div>

      <!-- Redirect URIs -->
      <div>
        <label class="block text-sm font-medium mb-1.5">{{ $t('developer.redirectUris') }}</label>
        <textarea
          v-model="form.redirect_uris"
          rows="3"
          :placeholder="$t('developer.redirectUrisPlaceholder')"
          class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-none"
        />
        <p class="text-xs text-muted-foreground mt-1">{{ $t('developer.redirectUrisHint') }}</p>
      </div>

      <!-- Scopes -->
      <div>
        <label class="block text-sm font-medium mb-2">{{ $t('developer.scopes') }}</label>
        <div class="flex flex-col gap-2">
          <label
            v-for="scope in allScopes"
            :key="scope"
            class="flex items-center gap-3 text-sm cursor-pointer px-3 py-2.5 border rounded-lg transition-colors"
            :class="form.scopes.includes(scope) ? 'border-foreground bg-foreground/5' : 'border-border'"
          >
            <input
              type="checkbox"
              :checked="form.scopes.includes(scope)"
              @change="toggleScope(scope)"
              class="sr-only"
            />
            <div class="flex flex-col">
              <span class="font-mono text-xs">{{ scope }}</span>
              <span class="text-xs text-muted-foreground">{{ $t(`developer.scopeDescriptions.${scope}`) }}</span>
            </div>
          </label>
        </div>
      </div>

      <!-- Grant Types -->
      <div>
        <label class="block text-sm font-medium mb-2">{{ $t('developer.grantTypes') }}</label>
        <p class="text-xs text-muted-foreground mb-2">{{ $t('developer.grantTypesHint') }}</p>
        <div class="flex flex-col gap-2">
          <label
            v-for="grant in allGrantTypes"
            :key="grant"
            class="flex items-center justify-between gap-3 text-sm cursor-pointer px-3 py-2.5 border rounded-lg transition-colors"
            :class="form.grant_types.includes(grant) ? 'border-foreground bg-foreground/5' : 'border-border'"
          >
            <span>{{ $t(`adminClients.grantTypeLabels.${grant}`) }}</span>
            <input type="checkbox" :checked="form.grant_types.includes(grant)" @change="toggleGrant(grant)" class="rounded border-border" />
          </label>
        </div>
      </div>

      <!-- Min Security Level -->
      <div>
        <label class="block text-sm font-medium mb-1.5">{{ $t('developer.minSecurityLevel') }}</label>
        <input
          v-model.number="form.min_security_level"
          type="number"
          min="0"
          class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
        />
        <p class="text-xs text-muted-foreground mt-1">{{ $t('developer.minSecurityLevelHint') }}</p>
      </div>

      <!-- Submit -->
      <div class="flex justify-end gap-2 mt-2">
        <router-link to="/developer" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">
          {{ $t('cancel') }}
        </router-link>
        <button
          type="submit"
          :disabled="saving || !auth.canCreateDeveloperApp"
          class="bg-foreground text-white px-5 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          <Loader2 v-if="saving" class="w-4 h-4 animate-spin" />
          {{ saving ? $t('developer.creating') : $t('developer.createApp') }}
        </button>
      </div>
    </form>

    <!-- Secret Reveal Modal -->
    <div v-if="showSecretModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4 p-6">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('developer.clientSecret') }}</h2>
        </div>
        <p class="text-sm text-muted-foreground mb-4">
          {{ $t('developer.secretWarning') }}
        </p>
        <div class="flex items-center gap-2 bg-muted rounded-lg p-3">
          <code class="flex-1 text-sm font-mono break-all">{{ revealedSecret }}</code>
          <button @click="copySecret" class="shrink-0 p-2 rounded-lg hover:bg-white transition-colors">
            <Check v-if="secretCopied" class="w-4 h-4 text-green-600" />
            <Copy v-else class="w-4 h-4 text-muted-foreground" />
          </button>
        </div>
        <div class="flex justify-end mt-5">
          <button @click="closeSecretModal" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors">
            {{ $t('developer.done') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
