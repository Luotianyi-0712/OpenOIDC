<script setup lang="ts">
import { Trash2 } from 'lucide-vue-next'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Provider, ProviderMeta } from '@/composables/useAdminProviders'

const props = defineProps<{
  provider: Provider
  meta: ProviderMeta
  displayName: string
  isCustom: boolean
  deleting: boolean
}>()

const emit = defineEmits<{
  edit: [provider: Provider]
  delete: [provider: Provider]
}>()

const { t } = useI18n()

const configuredLabel = computed(() => {
  if (props.provider.has_secret || props.provider.has_app_secret || props.provider.has_private_key) {
    return t('adminProviders.secretConfigured')
  }
  return t('adminProviders.secretNotSet')
})

const providerHint = computed(() => props.provider.client_id || props.provider.app_id || props.provider.provider)
const hasIconImage = computed(() => props.isCustom && !!props.provider.icon_url)
</script>

<template>
  <div
    class="group relative border rounded-xl p-5 hover:shadow-sm transition-all cursor-pointer"
    :class="provider.enabled ? 'border-green-200 bg-green-50/30' : 'border-border bg-white'"
    @click="emit('edit', provider)"
  >
    <div class="flex items-start gap-3 mb-3">
      <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 overflow-hidden" :style="{ backgroundColor: meta.color + '15' }">
        <img v-if="hasIconImage" :src="provider.icon_url" :alt="displayName" class="w-full h-full object-cover" />
        <svg v-else-if="meta.icon" class="w-5 h-5" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <path :d="meta.icon" :fill="meta.color" />
        </svg>
        <span v-else class="text-sm font-semibold" :style="{ color: meta.color }">
          {{ displayName.slice(0, 1).toUpperCase() }}
        </span>
      </div>

      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2 pr-8">
          <h3 class="font-medium text-sm truncate">{{ displayName }}</h3>
          <span
            class="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-medium shrink-0"
            :class="provider.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'"
          >
            {{ provider.enabled ? t('adminProviders.enabled') : t('adminProviders.disabled') }}
          </span>
          <span v-if="isCustom" class="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-blue-50 text-blue-700 shrink-0">
            {{ t('adminProviders.customOAuth2') }}
          </span>
        </div>
        <p class="text-xs text-muted-foreground mt-0.5 truncate">{{ providerHint }}</p>
      </div>

      <button
        v-if="isCustom"
        type="button"
        class="absolute top-4 right-4 p-1.5 rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-50"
        :disabled="deleting"
        :title="t('adminProviders.deleteProvider')"
        @click.stop="emit('delete', provider)"
      >
        <Trash2 class="w-4 h-4" />
      </button>
    </div>

    <div class="flex items-center justify-between text-xs text-muted-foreground">
      <span :class="provider.has_secret || provider.has_app_secret || provider.has_private_key ? 'text-green-600' : 'text-amber-500'">
        {{ configuredLabel }}
      </span>
      <span class="text-muted-foreground/60 group-hover:text-foreground transition-colors">
        {{ t('adminProviders.configure') }} &rarr;
      </span>
    </div>
  </div>
</template>