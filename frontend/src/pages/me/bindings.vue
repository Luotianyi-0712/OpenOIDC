<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'
import { useI18n } from 'vue-i18n'
import { useToastStore } from '@/stores/toast'
import {
  Loader2,
  Github,
  Chrome,
  Gitlab,
  MessageCircle,
  Apple,
  Gamepad2,
  Building2,
  Send,
  Link2,
  Unlink,
  ExternalLink,
  X,
} from 'lucide-vue-next'

interface Binding {
  id: string
  provider: string
  provider_uid: string
  provider_name: string
  bound_at: string
}

const PROVIDERS = [
  { id: 'github', name: 'GitHub', icon: Github },
  { id: 'google', name: 'Google', icon: Chrome },
  { id: 'gitlab', name: 'GitLab', icon: Gitlab },
  { id: 'gitee', name: 'Gitee', icon: ExternalLink },
  { id: 'discord', name: 'Discord', icon: Gamepad2 },
  { id: 'microsoft', name: 'Microsoft', icon: Building2 },
  { id: 'qq', name: 'QQ', icon: MessageCircle },
  { id: 'wechat', name: 'WeChat', icon: MessageCircle },
  { id: 'telegram', name: 'Telegram', icon: Send },
  { id: 'apple', name: 'Apple', icon: Apple },
] as const

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const toastStore = useToastStore()

const bindings = ref<Binding[]>([])
const loading = ref(true)
const error = ref('')
const unbindingProvider = ref<string | null>(null)

const showUnbindModal = ref(false)
const unbindTarget = ref<string | null>(null)
const unbindTargetName = ref('')

onMounted(() => {
  const result = route.query.result as string
  const errorParam = route.query.error as string
  if (result === 'bind_success') {
    toastStore.success(t('bindings.bindSuccess'))
  } else if (errorParam) {
    toastStore.error(t(`bindings.errors.${errorParam}`, errorParam))
  }
  if (result || errorParam) {
    router.replace({ path: route.path })
  }
  fetchBindings()
})

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

const boundProviders = computed(() => {
  const boundIds = new Set(bindings.value.map((b) => b.provider))
  return PROVIDERS.filter((p) => boundIds.has(p.id)).map((p) => ({
    ...p,
    binding: bindings.value.find((b) => b.provider === p.id)!,
  }))
})

const unboundProviders = computed(() => {
  const boundIds = new Set(bindings.value.map((b) => b.provider))
  return PROVIDERS.filter((p) => !boundIds.has(p.id))
})

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

    <div v-if="loading" class="flex items-center gap-2 text-sm text-muted-foreground py-12 justify-center">
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
            class="border border-border rounded-xl p-5 flex items-center justify-between"
          >
            <div class="flex items-center gap-4">
              <div class="w-10 h-10 rounded-lg bg-muted flex items-center justify-center">
                <component :is="item.icon" class="w-5 h-5" />
              </div>
              <div>
                <div class="text-sm font-medium">{{ item.name }}</div>
                <div class="text-xs text-muted-foreground mt-0.5">
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
              class="px-3 py-1.5 text-xs font-medium border rounded-lg transition-colors flex items-center gap-1.5 text-destructive border-destructive/30 hover:bg-destructive/5 disabled:opacity-50"
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
              <component :is="provider.icon" class="w-5 h-5 text-muted-foreground" />
            </div>
            <div>
              <div class="text-sm font-medium">{{ provider.name }}</div>
              <div class="text-xs text-muted-foreground">{{ $t('bindings.clickToConnect') }}</div>
            </div>
          </button>
        </div>
      </div>

      <div v-if="!unboundProviders.length && boundProviders.length" class="text-sm text-muted-foreground text-center py-6">
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
