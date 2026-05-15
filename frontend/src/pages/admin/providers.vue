<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '@/api/client'
import { Loader2, X, Copy, Check, ExternalLink } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface Provider {
  provider: string
  display_name: string
  enabled: boolean
  client_id?: string
  has_secret: boolean
  app_id?: string
  has_app_secret?: boolean
  team_id?: string
  key_id?: string
  has_private_key?: boolean
  base_url?: string
  tenant_id?: string
  sort_order: number
}

interface FieldDef {
  key: string
  label: string
  type: 'text' | 'password' | 'textarea'
  placeholder?: string
}

interface ProviderMeta {
  label: string
  color: string
  icon: string
  fields: FieldDef[]
  callbackPath: string
  docUrl?: string
}

const providerMeta: Record<string, ProviderMeta> = {
  github: {
    label: 'GitHub',
    color: '#24292e',
    icon: 'M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z',
    fields: [
      { key: 'client_id', label: 'Client ID', type: 'text', placeholder: 'Iv1.xxxxxxxxxx' },
      { key: 'client_secret', label: 'Client Secret', type: 'password' },
    ],
    callbackPath: '/api/v1/social/github/callback',
  },
  google: {
    label: 'Google',
    color: '#4285f4',
    icon: 'M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z',
    fields: [
      { key: 'client_id', label: 'Client ID', type: 'text', placeholder: 'xxxx.apps.googleusercontent.com' },
      { key: 'client_secret', label: 'Client Secret', type: 'password' },
    ],
    callbackPath: '/api/v1/social/google/callback',
  },
  gitlab: {
    label: 'GitLab',
    color: '#fc6d26',
    icon: 'M23.955 13.587l-1.342-4.135-2.664-8.189a.455.455 0 00-.867 0L16.418 9.45H7.582L4.918 1.263a.455.455 0 00-.867 0L1.386 9.45.045 13.587a.924.924 0 00.331 1.023L12 23.054l11.624-8.443a.92.92 0 00.331-1.024',
    fields: [
      { key: 'base_url', label: 'Instance URL', type: 'text', placeholder: 'https://gitlab.com' },
      { key: 'client_id', label: 'Application ID', type: 'text' },
      { key: 'client_secret', label: 'Secret', type: 'password' },
    ],
    callbackPath: '/api/v1/social/gitlab/callback',
  },
  gitee: {
    label: 'Gitee',
    color: '#c71d23',
    icon: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4.95 7.8h-3.27c-.26 0-.47.21-.47.47v.63c0 .26.21.47.47.47h2.17v2.17c0 .26-.21.47-.47.47H9.62a.47.47 0 01-.47-.47V9.37c0-.26.21-.47.47-.47h7.33c.26 0 .47-.21.47-.47v-.63c0-.26-.21-.47-.47-.47H9.14C8.05 7.33 7.17 8.21 7.17 9.3v5.56c0 1.09.88 1.97 1.97 1.97h5.72c1.09 0 1.97-.88 1.97-1.97v-3.1c0-1.09-.88-1.97-1.88-1.97z',
    fields: [
      { key: 'client_id', label: 'Client ID', type: 'text' },
      { key: 'client_secret', label: 'Client Secret', type: 'password' },
    ],
    callbackPath: '/api/v1/social/gitee/callback',
  },
  discord: {
    label: 'Discord',
    color: '#5865f2',
    icon: 'M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.041-.106 13.107 13.107 0 01-1.872-.892.077.077 0 01-.008-.128 10.2 10.2 0 00.372-.292.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.01c.12.098.246.198.373.292a.077.077 0 01-.006.127 12.299 12.299 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.839 19.839 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03z',
    fields: [
      { key: 'client_id', label: 'Client ID', type: 'text' },
      { key: 'client_secret', label: 'Client Secret', type: 'password' },
    ],
    callbackPath: '/api/v1/social/discord/callback',
  },
  telegram: {
    label: 'Telegram',
    color: '#26a5e4',
    icon: 'M11.944 0A12 12 0 000 12a12 12 0 0012 12 12 12 0 0012-12A12 12 0 0012 0a12 12 0 00-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 01.171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.479.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z',
    fields: [
      { key: 'client_secret', label: 'Bot Token', type: 'password', placeholder: '123456:ABC-DEF...' },
    ],
    callbackPath: '/api/v1/social/telegram/callback',
  },
  microsoft: {
    label: 'Microsoft',
    color: '#00a4ef',
    icon: 'M1 1h10v10H1zM13 1h10v10H13zM1 13h10v10H1zM13 13h10v10H13z',
    fields: [
      { key: 'tenant_id', label: 'Tenant ID', type: 'text', placeholder: 'common' },
      { key: 'client_id', label: 'Application (client) ID', type: 'text' },
      { key: 'client_secret', label: 'Client Secret', type: 'password' },
    ],
    callbackPath: '/api/v1/social/microsoft/callback',
  },
  apple: {
    label: 'Apple',
    color: '#000000',
    icon: 'M17.05 20.28c-.98.95-2.05.88-3.08.4-1.09-.5-2.08-.48-3.24 0-1.44.62-2.2.44-3.06-.4C2.79 15.25 3.51 7.59 9.05 7.31c1.35.07 2.29.74 3.08.8 1.18-.24 2.31-.93 3.57-.84 1.51.12 2.65.72 3.4 1.8-3.12 1.87-2.38 5.98.48 7.13-.57 1.5-1.31 2.99-2.53 4.09zM12.03 7.25c-.15-2.23 1.66-4.07 3.74-4.25.32 2.32-1.55 4.3-3.74 4.25z',
    fields: [
      { key: 'client_id', label: 'Service ID', type: 'text' },
      { key: 'team_id', label: 'Team ID', type: 'text' },
      { key: 'key_id', label: 'Key ID', type: 'text' },
      { key: 'private_key', label: 'Private Key (.p8)', type: 'textarea', placeholder: '-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----' },
    ],
    callbackPath: '/api/v1/social/apple/callback',
  },
  qq: {
    label: 'QQ',
    color: '#12b7f5',
    icon: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm3.22 14.34c-.35.14-.65-.12-.95-.32-.59.4-1.22.62-1.92.66h-.7c-.7-.04-1.33-.26-1.92-.66-.3.2-.6.46-.95.32-.4-.16-.27-.73-.15-1.11.06-.2.14-.37.22-.52-.36-.5-.58-1.03-.58-1.51 0-2.09 1.94-3.28 3.73-3.28h.6c1.79 0 3.73 1.19 3.73 3.28 0 .48-.22 1.01-.58 1.51.08.15.16.32.22.52.12.38.25.95-.15 1.11-.17.07-.35.07-.6 0z',
    fields: [
      { key: 'app_id', label: 'APP ID', type: 'text' },
      { key: 'app_secret', label: 'APP Key', type: 'password' },
    ],
    callbackPath: '/api/v1/social/qq/callback',
  },
  wechat: {
    label: 'WeChat',
    color: '#07c160',
    icon: 'M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 01.213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 00.167-.054l1.903-1.114a.864.864 0 01.717-.098 10.16 10.16 0 002.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178A1.17 1.17 0 014.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178 1.17 1.17 0 01-1.162-1.178c0-.651.52-1.18 1.162-1.18zm3.453 2.862c-3.837 0-6.95 2.708-6.95 6.048 0 3.342 3.113 6.05 6.95 6.05.604 0 1.192-.075 1.763-.22a.72.72 0 01.597.082l1.583.928a.27.27 0 00.139.045c.133 0 .241-.108.241-.243a.39.39 0 00-.04-.176l-.323-1.233a.492.492 0 01.177-.554c1.524-1.12 2.495-2.779 2.495-4.629 0-3.34-3.113-6.048-6.632-6.048zm-2.497 3.023a.97.97 0 01.968.983.97.97 0 01-.968.983.97.97 0 01-.967-.983.97.97 0 01.967-.983zm4.994 0a.97.97 0 01.968.983.97.97 0 01-.968.983.97.97 0 01-.967-.983.97.97 0 01.967-.983z',
    fields: [
      { key: 'app_id', label: 'AppID', type: 'text' },
      { key: 'app_secret', label: 'AppSecret', type: 'password' },
    ],
    callbackPath: '/api/v1/social/wechat/callback',
  },
  phone: {
    label: 'Phone',
    color: '#16a34a',
    icon: 'M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07 19.5 19.5 0 01-6-6 19.79 19.79 0 01-3.07-8.67A2 2 0 014.11 2h3a2 2 0 012 1.72 12.84 12.84 0 00.7 2.81 2 2 0 01-.45 2.11L8.09 9.91a16 16 0 006 6l1.27-1.27a2 2 0 012.11-.45 12.84 12.84 0 002.81.7A2 2 0 0122 16.92z',
    fields: [
      { key: 'client_id', label: 'Access Key', type: 'text' },
      { key: 'client_secret', label: 'Access Secret', type: 'password' },
    ],
    callbackPath: '',
  },
}

const providers = ref<Provider[]>([])
const loading = ref(false)
const error = ref('')
const successMsg = ref('')

const showModal = ref(false)
const editingProvider = ref<Provider | null>(null)
const saving = ref(false)
const form = ref<Record<string, unknown>>({})
const copiedField = ref('')

const baseUrl = computed(() => window.location.origin)

onMounted(fetchProviders)

function getMeta(name: string): ProviderMeta {
  return providerMeta[name] ?? {
    label: name.charAt(0).toUpperCase() + name.slice(1),
    color: '#6b7280',
    icon: '',
    fields: [
      { key: 'client_id', label: 'Client ID', type: 'text' as const },
      { key: 'client_secret', label: 'Client Secret', type: 'password' as const },
    ],
    callbackPath: `/api/v1/social/${name}/callback`,
  }
}

async function fetchProviders() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<Provider[]>('/admin/providers')
    providers.value = res.data ?? []
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function openEdit(provider: Provider) {
  editingProvider.value = provider
  const meta = getMeta(provider.provider)
  const formData: Record<string, unknown> = { enabled: provider.enabled }
  for (const field of meta.fields) {
    if (field.type === 'text') {
      formData[field.key] = (provider as any)[field.key] ?? ''
    } else {
      formData[field.key] = ''
    }
  }
  form.value = formData
  showModal.value = true
}

async function saveProvider() {
  if (!editingProvider.value) return
  saving.value = true
  error.value = ''
  try {
    const payload: Record<string, unknown> = { enabled: form.value.enabled }
    const meta = getMeta(editingProvider.value.provider)
    for (const field of meta.fields) {
      const val = form.value[field.key]
      if (val && typeof val === 'string' && val.trim()) {
        payload[field.key] = val.trim()
      }
    }
    await api.put(`/admin/providers/${editingProvider.value.provider}`, payload)
    showModal.value = false
    successMsg.value = t('adminProviders.saveSuccess')
    setTimeout(() => (successMsg.value = ''), 3000)
    await fetchProviders()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

async function copyText(text: string, field: string) {
  await navigator.clipboard.writeText(text)
  copiedField.value = field
  setTimeout(() => (copiedField.value = ''), 2000)
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-6">
      <h2 class="text-lg font-semibold">{{ $t('adminProviders.title') }}</h2>
      <p class="text-sm text-muted-foreground mt-1">{{ $t('adminProviders.subtitle') }}</p>
    </div>

    <!-- Success -->
    <div v-if="successMsg" class="mb-4 rounded-lg border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-700">
      {{ successMsg }}
    </div>

    <!-- Error -->
    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <!-- Provider Cards -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <div
        v-for="provider in providers"
        :key="provider.provider"
        class="border rounded-xl p-5 hover:shadow-sm transition-all cursor-pointer group"
        :class="provider.enabled ? 'border-green-200 bg-green-50/30' : 'border-border'"
        @click="openEdit(provider)"
      >
        <div class="flex items-center gap-3 mb-3">
          <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0" :style="{ backgroundColor: getMeta(provider.provider).color + '15' }">
            <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none">
              <path :d="getMeta(provider.provider).icon" :fill="getMeta(provider.provider).color" />
            </svg>
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <h3 class="font-medium text-sm">{{ getMeta(provider.provider).label }}</h3>
              <span
                class="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-medium"
                :class="provider.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'"
              >
                {{ provider.enabled ? $t('adminProviders.enabled') : $t('adminProviders.disabled') }}
              </span>
            </div>
            <p class="text-xs text-muted-foreground mt-0.5">
              <template v-if="provider.client_id || provider.app_id">
                {{ provider.client_id || provider.app_id }}
              </template>
              <template v-else>
                {{ $t('adminProviders.secretNotSet') }}
              </template>
            </p>
          </div>
        </div>
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <span v-if="provider.has_secret || provider.has_app_secret" class="text-green-600">
            {{ $t('adminProviders.secretConfigured') }}
          </span>
          <span v-else class="text-amber-500">
            {{ $t('adminProviders.secretNotSet') }}
          </span>
          <span class="text-muted-foreground/60 group-hover:text-foreground transition-colors">
            {{ $t('adminProviders.configure') }} &rarr;
          </span>
        </div>
      </div>
    </div>

    <!-- Edit Modal -->
    <div v-if="showModal && editingProvider" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-5">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg flex items-center justify-center" :style="{ backgroundColor: getMeta(editingProvider.provider).color + '15' }">
              <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none">
                <path :d="getMeta(editingProvider.provider).icon" :fill="getMeta(editingProvider.provider).color" />
              </svg>
            </div>
            <div>
              <h2 class="text-lg font-semibold">{{ getMeta(editingProvider.provider).label }}</h2>
              <p class="text-xs text-muted-foreground">{{ editingProvider.provider }}</p>
            </div>
          </div>
          <button @click="showModal = false" class="text-muted-foreground hover:text-foreground">
            <X class="w-5 h-5" />
          </button>
        </div>

        <form @submit.prevent="saveProvider" class="flex flex-col gap-4">
          <!-- Enable toggle -->
          <div class="flex items-center gap-3 p-3 rounded-lg bg-muted/40">
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="(form as any).enabled" class="sr-only peer" />
              <div class="w-9 h-5 bg-gray-200 peer-focus:ring-2 peer-focus:ring-foreground/10 rounded-full peer peer-checked:bg-green-500 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full"></div>
            </label>
            <span class="text-sm font-medium">{{ $t('adminProviders.enabled') }}</span>
          </div>

          <!-- Callback URL -->
          <div v-if="getMeta(editingProvider.provider).callbackPath">
            <label class="block text-xs font-medium text-muted-foreground mb-1">{{ $t('adminProviders.callbackUrl') }}</label>
            <div class="flex items-center gap-2 bg-muted rounded-lg p-2.5">
              <code class="flex-1 text-xs font-mono break-all text-foreground">{{ baseUrl + getMeta(editingProvider.provider).callbackPath }}</code>
              <button type="button" @click.stop="copyText(baseUrl + getMeta(editingProvider.provider).callbackPath, 'callback')" class="shrink-0 p-1 rounded hover:bg-white transition-colors">
                <Check v-if="copiedField === 'callback'" class="w-3.5 h-3.5 text-green-600" />
                <Copy v-else class="w-3.5 h-3.5 text-muted-foreground" />
              </button>
            </div>
            <p class="text-[11px] text-muted-foreground mt-1">{{ $t('adminProviders.callbackHint') }}</p>
          </div>

          <!-- Dynamic fields -->
          <div v-for="field in getMeta(editingProvider.provider).fields" :key="field.key">
            <label class="block text-sm font-medium mb-1.5">{{ field.label }}</label>
            <textarea
              v-if="field.type === 'textarea'"
              v-model="(form as any)[field.key]"
              rows="4"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-none"
              :placeholder="field.placeholder || field.label"
            />
            <input
              v-else
              v-model="(form as any)[field.key]"
              :type="field.type"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
              :placeholder="field.placeholder || field.label"
            />
            <p v-if="field.type === 'password' || field.type === 'textarea'" class="text-[11px] text-muted-foreground mt-1">
              <template v-if="
                (field.key === 'client_secret' && editingProvider.has_secret) ||
                (field.key === 'app_secret' && editingProvider.has_app_secret) ||
                (field.key === 'private_key' && editingProvider.has_private_key)
              ">
                {{ $t('adminProviders.secretConfigured') }} — leave blank to keep current value.
              </template>
            </p>
          </div>

          <!-- Actions -->
          <div class="flex justify-end gap-2 mt-2">
            <button type="button" @click="showModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">
              {{ $t('cancel') }}
            </button>
            <button type="submit" :disabled="saving" class="bg-foreground text-white px-5 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="saving" class="w-4 h-4 animate-spin" />
              {{ $t('save') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
