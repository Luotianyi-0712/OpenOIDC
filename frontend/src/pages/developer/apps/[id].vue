<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'
import { useI18n } from 'vue-i18n'
import {
  Loader2, Copy, Check, RefreshCw, Trash2, ArrowLeft, Save,
  AlertTriangle, Eye, EyeOff, X,
} from 'lucide-vue-next'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appId = route.params.id as string

interface AppDetail {
  id: string
  client_id: string
  client_secret: string
  client_name: string
  description: string
  logo_url: string
  homepage_url: string
  redirect_uris: string[]
  grant_types: string[]
  scopes: string[]
  min_security_level: number
  is_active: boolean
  created_at: string
  user_count?: number
  endpoints: {
    issuer: string
    authorize_url: string
    token_url: string
    userinfo_url: string
    jwks_url: string
    discovery_url: string
  }
}

const app = ref<AppDetail | null>(null)
const loading = ref(false)
const error = ref('')
const saveSuccess = ref(false)

// Edit form
const form = ref({
  client_name: '',
  description: '',
  logo_url: '',
  homepage_url: '',
  redirect_uris: '',
  scopes: [] as string[],
  grant_types: [] as string[],
  min_security_level: 0,
})
const saving = ref(false)

const allScopes = ['openid', 'profile', 'email', 'security_level']
const allGrantTypes = ['authorization_code', 'refresh_token', 'client_credentials']

// Secret state
const showSecretModal = ref(false)
const revealedSecret = ref('')
const secretCopied = ref(false)
const secretVisible = ref(false)

// Rotate confirm modal
const showRotateModal = ref(false)

// Delete modal
const showDeleteModal = ref(false)
const deleting = ref(false)

// Copy tracking
const copiedField = ref('')

onMounted(fetchApp)

async function fetchApp() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<AppDetail>(`/developer/apps/${appId}`)
    app.value = res.data ?? null
    if (app.value) {
      form.value = {
        client_name: app.value.client_name,
        description: app.value.description || '',
        logo_url: app.value.logo_url || '',
        homepage_url: app.value.homepage_url || '',
        redirect_uris: (app.value.redirect_uris || []).join('\n'),
        scopes: [...(app.value.scopes || [])],
        grant_types: [...(app.value.grant_types || [])],
        min_security_level: app.value.min_security_level || 0,
      }
      if (app.value.client_secret) {
        revealedSecret.value = app.value.client_secret
      }
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

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

async function saveApp() {
  saving.value = true
  error.value = ''
  saveSuccess.value = false
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
    await api.put(`/developer/apps/${appId}`, payload)
    saveSuccess.value = true
    setTimeout(() => (saveSuccess.value = false), 3000)
    await fetchApp()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

async function rotateSecret() {
  showRotateModal.value = false
  try {
    const res = await api.post<{ client_secret: string }>(`/developer/apps/${appId}/rotate-secret`)
    if (res.data?.client_secret) {
      revealedSecret.value = res.data.client_secret
      secretVisible.value = true
      showSecretModal.value = true
    }
  } catch (e: any) {
    error.value = e.message
  }
}

async function copyToClipboard(text: string, field: string) {
  await navigator.clipboard.writeText(text)
  copiedField.value = field
  setTimeout(() => (copiedField.value = ''), 2000)
}

async function copySecret() {
  await navigator.clipboard.writeText(revealedSecret.value)
  secretCopied.value = true
  setTimeout(() => (secretCopied.value = false), 2000)
}

async function deleteApp() {
  deleting.value = true
  try {
    await api.del(`/developer/apps/${appId}`)
    showDeleteModal.value = false
    router.push('/developer')
  } catch (e: any) {
    error.value = e.message
  } finally {
    deleting.value = false
  }
}

const integrationItems = computed(() => {
  if (!app.value) return []
  const ep = app.value.endpoints || {} as AppDetail['endpoints']
  return [
    { label: t('developer.clientId'), key: 'client_id', value: app.value.client_id },
    { label: t('developer.clientSecret'), key: 'client_secret', value: '', isSecret: true },
    { label: t('developer.authorizeUrl'), key: 'authorize', value: ep.authorize_url },
    { label: t('developer.tokenUrl'), key: 'token', value: ep.token_url },
    { label: t('developer.userinfoUrl'), key: 'userinfo', value: ep.userinfo_url },
    { label: t('developer.discoveryUrl'), key: 'discovery', value: ep.discovery_url },
  ]
})
</script>

<template>
  <div class="max-w-3xl">
    <!-- Back link -->
    <router-link to="/developer" class="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-6">
      <ArrowLeft class="w-4 h-4" /> {{ $t('devNav.myApps') }}
    </router-link>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <!-- Error -->
    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <template v-if="app">
      <h2 class="text-lg font-semibold mb-6">{{ $t('developer.appDetail') }}</h2>

      <!-- Section 1: App Info -->
      <div class="border border-border rounded-xl p-6 mb-6">
        <h3 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground mb-4">{{ $t('developer.appInfo') }}</h3>
        <form @submit.prevent="saveApp" class="flex flex-col gap-4">
          <!-- App Name -->
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('developer.appName') }}</label>
            <input
              v-model="form.client_name"
              type="text"
              required
              class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
            />
          </div>

          <!-- Description -->
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('developer.description') }}</label>
            <textarea
              v-model="form.description"
              rows="3"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-none"
            />
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
            <div v-if="form.logo_url" class="mt-2">
              <img :src="form.logo_url" alt="Logo preview" class="w-12 h-12 rounded-lg object-cover border border-border" />
            </div>
          </div>

          <!-- Website URL -->
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('developer.homepageUrl') }}</label>
            <input
              v-model="form.homepage_url"
              type="url"
              placeholder="https://example.com"
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
              placeholder="https://yourapp.com/callback"
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

          <!-- Save -->
          <div class="flex items-center gap-3 mt-1">
            <button
              type="submit"
              :disabled="saving"
              class="bg-foreground text-white px-5 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              <Loader2 v-if="saving" class="w-4 h-4 animate-spin" />
              <Save v-else class="w-4 h-4" />
              {{ $t('save') }}
            </button>
            <span v-if="saveSuccess" class="text-sm text-green-600 flex items-center gap-1">
              <Check class="w-4 h-4" /> {{ $t('developer.copied') }}
            </span>
          </div>
        </form>
      </div>

      <!-- Section 2: Integration Info (Credentials + Endpoints) -->
      <div class="border border-border rounded-xl p-6 mb-6">
        <!-- User Count -->
        <div v-if="app.user_count !== undefined" class="mb-5 pb-5 border-b border-border">
          <div class="text-xs font-medium text-muted-foreground mb-1">{{ $t('developerApp.userCount') }}</div>
          <div class="text-2xl font-semibold">{{ app.user_count }}</div>
        </div>

        <h3 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground mb-1">{{ $t('developer.credentials') }}</h3>
        <p class="text-xs text-muted-foreground mb-5">{{ $t('developer.credentialsHint') }}</p>

        <div class="flex flex-col divide-y divide-border">
          <div
            v-for="item in integrationItems"
            :key="item.key"
            class="py-3.5 first:pt-0 last:pb-0"
          >
            <div class="text-xs font-medium text-muted-foreground mb-1.5">{{ item.label }}</div>

            <!-- Client Secret row -->
            <div v-if="item.isSecret" class="flex items-center gap-2">
              <code class="flex-1 text-sm font-mono break-all">{{ secretVisible && revealedSecret ? revealedSecret : '••••••••••' }}</code>
              <button @click="secretVisible = !secretVisible" class="shrink-0 p-1.5 rounded-lg hover:bg-muted transition-colors">
                <EyeOff v-if="secretVisible && revealedSecret" class="w-4 h-4 text-muted-foreground" />
                <Eye v-else class="w-4 h-4 text-muted-foreground" />
              </button>
              <button @click="revealedSecret && copyToClipboard(revealedSecret, 'client_secret')" class="shrink-0 p-1.5 rounded-lg hover:bg-muted transition-colors">
                <Check v-if="copiedField === 'client_secret'" class="w-4 h-4 text-green-600" />
                <Copy v-else class="w-4 h-4 text-muted-foreground" />
              </button>
              <button
                @click="showRotateModal = true"
                class="shrink-0 flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium border border-border rounded-lg hover:bg-muted transition-colors"
              >
                <RefreshCw class="w-3.5 h-3.5" /> {{ $t('developer.rotateSecret') }}
              </button>
            </div>

            <!-- Normal rows (Client ID + Endpoints) -->
            <div v-else class="flex items-center gap-2">
              <code class="flex-1 text-sm font-mono break-all">{{ item.value }}</code>
              <button @click="copyToClipboard(item.value, item.key)" class="shrink-0 p-1.5 rounded-lg hover:bg-muted transition-colors">
                <Check v-if="copiedField === item.key" class="w-4 h-4 text-green-600" />
                <Copy v-else class="w-4 h-4 text-muted-foreground" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Section 4: Danger Zone -->
      <div class="border-2 border-destructive/30 rounded-xl p-6">
        <h3 class="text-sm font-semibold uppercase tracking-wider text-destructive mb-2 flex items-center gap-2">
          <AlertTriangle class="w-4 h-4" />
          {{ $t('developer.dangerZone') }}
        </h3>
        <p class="text-sm text-muted-foreground mb-4">{{ $t('developer.deleteConfirm') }}</p>
        <button
          @click="showDeleteModal = true"
          class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors flex items-center gap-2"
        >
          <Trash2 class="w-4 h-4" /> {{ $t('developer.deleteApp') }}
        </button>
      </div>
    </template>

    <!-- Secret Reveal Modal -->
    <div v-if="showSecretModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4 p-6">
        <h2 class="text-lg font-semibold mb-2">{{ $t('developer.clientSecret') }}</h2>
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
          <button @click="showSecretModal = false" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors">
            {{ $t('developer.done') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Rotate Secret Confirmation Modal -->
    <div v-if="showRotateModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showRotateModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('developer.rotateSecret') }}</h2>
          <button @click="showRotateModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <p class="text-sm text-muted-foreground mb-5">
          {{ $t('developer.rotateConfirm') }}
        </p>
        <div class="flex justify-end gap-2">
          <button @click="showRotateModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">
            {{ $t('cancel') }}
          </button>
          <button @click="rotateSecret" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors">
            {{ $t('confirm') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showDeleteModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <h2 class="text-lg font-semibold mb-2">{{ $t('developer.deleteApp') }}</h2>
        <p class="text-sm text-muted-foreground mb-5">
          {{ $t('developer.deleteConfirm') }}
        </p>
        <div class="flex justify-end gap-2">
          <button @click="showDeleteModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">
            {{ $t('cancel') }}
          </button>
          <button @click="deleteApp" :disabled="deleting" class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2">
            <Loader2 v-if="deleting" class="w-4 h-4 animate-spin" />
            {{ $t('delete') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
