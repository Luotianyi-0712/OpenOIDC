<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@/stores/toast'
import { useAuthStore } from '@/stores/auth'
import {
  Loader2,
  Link2,
  Unlink,
  ExternalLink,
  X,
} from 'lucide-vue-next'
import { usePublicConfig, getProviderIcon, isGoogleProvider, GOOGLE_SVG, type EnabledProvider } from '@/composables/usePublicConfig'

interface Binding {
  id: string
  provider: string
  provider_uid: string
  provider_name: string
  bound_at: string
}

interface BindingProvider {
  id: string
  name: string
  iconUrl?: string
  binding?: Binding
}

interface BoundBindingProvider extends BindingProvider {
  binding: Binding
}

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const toastStore = useToastStore()
const authStore = useAuthStore()

const { providers, settings, loaded } = usePublicConfig()

const bindings = ref<Binding[]>([])
const loading = ref(true)
const error = ref('')
const unbindingProvider = ref<string | null>(null)

const showUnbindModal = ref(false)
const unbindTarget = ref<string | null>(null)
const unbindTargetName = ref('')

onMounted(async () => {
  const result = route.query.result as string
  const errorParam = route.query.error as string
  if (result === 'bind_success') {
    toastStore.success(t('bindings.bindSuccess'))
    await refreshAuthState()
  } else if (errorParam) {
    toastStore.error(t(`bindings.errors.${errorParam}`, errorParam))
  }
  if (result || errorParam) {
    router.replace({ path: route.path })
  }
  fetchBindings()
})

async function refreshAuthState() {
  await authStore.fetchUser()
  await authStore.fetchDeveloperStatus()
}

async function fetchBindings() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<Binding[]>('/me/bindings')
    bindings.value = res.data || []
  } catch (e: any) {
    error.value = e.message || 'Failed to load bindings.'
  } finally {
    loading.value = false
  }
}

function enabledProviderItem(provider: EnabledProvider): BindingProvider {
  return {
    id: provider.name,
    name: provider.display_name || provider.name,
    iconUrl: provider.icon_url,
  }
}

const enabledBindingProviders = computed<BindingProvider[]>(() => {
  return providers.value.map(enabledProviderItem)
})

const pageLoading = computed(() => loading.value || !loaded.value)
const socialBindingDisabled = computed(() => loaded.value && !settings.value.social_binding_enabled)
const hasEnabledProviders = computed(() => enabledBindingProviders.value.length > 0)

const boundProviders = computed<BoundBindingProvider[]>(() => {
  return enabledBindingProviders.value
    .map((provider) => {
      const binding = bindings.value.find((b) => b.provider === provider.id)
      return binding ? { ...provider, binding } : null
    })
    .filter((provider): provider is BoundBindingProvider => provider !== null)
})

const unboundProviders = computed<BindingProvider[]>(() => {
  if (!settings.value.social_binding_enabled) return []
  const boundIds = new Set(bindings.value.map((b) => b.provider))
  return enabledBindingProviders.value.filter((p) => !boundIds.has(p.id))
})

function providerInitial(name: string) {
  return name.trim().slice(0, 1).toUpperCase() || '?'
}

function confirmUnbind(providerId: string, providerName: string) {
  unbindTarget.value = providerId
  unbindTargetName.value = providerName
  showUnbindModal.value = true
}

async function doUnbind() {
  if (!unbindTarget.value) return
  const provider = unbindTarget.value
  showUnbindModal.value = false
  unbindingProvider.value = provider
  try {
    await api.del(`/me/bindings/${provider}`)
    bindings.value = bindings.value.filter((b) => b.provider !== provider)
    await refreshAuthState()
  } catch (e: any) {
    error.value = e.message
  } finally {
    unbindingProvider.value = null
  }
}

function bindProvider(provider: string) {
  window.location.href = `/api/v1/social/${provider}/begin?return_to=/me/bindings`
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-lg font-semibold">{{ $t('bindings.title') }}</h2>
      <p class="text-sm text-muted-foreground mt-0.5">
        {{ $t('bindings.desc') }}
      </p>
    </div>

    <div v-if="error" class="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive mb-4">
      {{ error }}
    </div>

    <div v-if="pageLoading" class="flex items-center gap-2 text-sm text-muted-foreground py-12 justify-center">
      <Loader2 class="w-4 h-4 animate-spin" /> {{ $t('bindings.loadingBindings') }}
    </div>

    <template v-else>
      <div v-if="boundProviders.length" class="mb-8">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3 flex items-center gap-2">
          <Link2 class="w-3.5 h-3.5" /> {{ $t('bindings.connected') }}
        </h3>
        <div class="space-y-3">
          <div
            v-for="item in boundProviders"
            :key="item.id"
            class="border border-border rounded-xl p-5 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"
          >
            <div class="flex items-start gap-4 min-w-0">
              <div class="w-10 h-10 rounded-lg bg-muted flex items-center justify-center shrink-0">
                <img
                  v-if="item.iconUrl"
                  :src="item.iconUrl"
                  :alt="item.name"
                  class="w-5 h-5 object-contain"
                />
                <span v-else-if="isGoogleProvider(item.id)" v-html="GOOGLE_SVG" />
                <svg v-else-if="getProviderIcon(item.id)?.path" class="w-5 h-5" viewBox="0 0 24 24" fill="none">
                  <path :d="getProviderIcon(item.id)!.path" :fill="getProviderIcon(item.id)!.color" />
                </svg>
                <span v-else class="text-sm font-semibold text-muted-foreground">
                  {{ providerInitial(item.name) }}
                </span>
              </div>
              <div class="min-w-0">
                <div class="text-sm font-medium break-words">{{ item.name }}</div>
                <div class="text-xs text-muted-foreground mt-0.5 break-all">
                  {{ item.binding.provider_name || item.binding.provider_uid }}
                </div>
                <div class="text-xs text-muted-foreground mt-0.5">
                  {{ $t('bindings.bound', { date: formatDate(item.binding.bound_at) }) }}
                </div>
              </div>
            </div>
            <button
              @click="confirmUnbind(item.id, item.name)"
              :disabled="unbindingProvider === item.id"
              class="px-3 py-1.5 text-xs font-medium border rounded-lg transition-colors flex items-center justify-center gap-1.5 text-destructive border-destructive/30 hover:bg-destructive/5 disabled:opacity-50 w-full sm:w-auto"
            >
              <Loader2 v-if="unbindingProvider === item.id" class="w-3 h-3 animate-spin" />
              <Unlink v-else class="w-3 h-3" />
              {{ $t('bindings.unbind') }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="unboundProviders.length">
        <h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-3 flex items-center gap-2">
          <ExternalLink class="w-3.5 h-3.5" /> {{ $t('bindings.available') }}
        </h3>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <button
            v-for="provider in unboundProviders"
            :key="provider.id"
            @click="bindProvider(provider.id)"
            class="border border-border rounded-xl p-4 flex items-center gap-3.5 hover:bg-muted/50 transition-colors text-left"
          >
            <div class="w-10 h-10 rounded-lg bg-muted flex items-center justify-center shrink-0">
              <img
                v-if="provider.iconUrl"
                :src="provider.iconUrl"
                :alt="provider.name"
                class="w-5 h-5 object-contain"
              />
              <span v-else-if="isGoogleProvider(provider.id)" v-html="GOOGLE_SVG" />
              <svg v-else-if="getProviderIcon(provider.id)?.path" class="w-5 h-5" viewBox="0 0 24 24" fill="none">
                <path :d="getProviderIcon(provider.id)!.path" :fill="getProviderIcon(provider.id)!.color" />
              </svg>
              <span v-else class="text-sm font-semibold text-muted-foreground">
                {{ providerInitial(provider.name) }}
              </span>
            </div>
            <div>
              <div class="text-sm font-medium">{{ provider.name }}</div>
              <div class="text-xs text-muted-foreground">{{ $t('bindings.clickToConnect') }}</div>
            </div>
          </button>
        </div>
      </div>

      <div v-if="socialBindingDisabled" class="text-sm text-muted-foreground text-center py-6">
        {{ $t('bindings.bindingDisabled') }}
      </div>

      <div v-else-if="!hasEnabledProviders" class="text-sm text-muted-foreground text-center py-6">
        {{ $t('bindings.noAvailableProviders') }}
      </div>

      <div v-else-if="!unboundProviders.length && boundProviders.length" class="text-sm text-muted-foreground text-center py-6">
        {{ $t('bindings.allConnected') }}
      </div>
    </template>

    <!-- Unbind Confirm Modal -->
    <div v-if="showUnbindModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showUnbindModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <div class="flex items-center justify-between mb-2">
          <h2 class="text-lg font-semibold">{{ $t('bindings.unbind') }}</h2>
          <button @click="showUnbindModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('bindings.unbindConfirm') }}</p>
        <div class="flex justify-end gap-2">
          <button @click="showUnbindModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
          <button @click="doUnbind" class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors">{{ $t('confirm') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>
