<script setup lang="ts">
import { onMounted } from 'vue'
import { Loader2, Plus } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import ProviderCard from '@/components/admin/providers/ProviderCard.vue'
import ProviderFormModal from '@/components/admin/providers/ProviderFormModal.vue'
import { useAdminProviders, type Provider, type ProviderFormKey } from '@/composables/useAdminProviders'
import { useToastStore } from '@/stores/toast'

const { t } = useI18n()
const toast = useToastStore()
const adminProviders = useAdminProviders()

onMounted(() => {
  void adminProviders.fetchProviders()
})

function updateField(key: ProviderFormKey, value: string | boolean) {
  adminProviders.form[key] = value
}

async function saveProvider() {
  try {
    const creating = adminProviders.isCreate.value
    await adminProviders.saveProvider()
    toast.success(creating ? t('adminProviders.createProviderSuccess') : t('adminProviders.saveSuccess'))
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('adminProviders.saveFailed'))
  }
}

async function deleteProvider(provider: Provider) {
  if (!adminProviders.isCustomProvider(provider)) return
  if (!window.confirm(t('adminProviders.deleteProviderConfirm', { name: adminProviders.providerDisplayName(provider) }))) return

  try {
    await adminProviders.deleteProvider(provider)
    toast.success(t('adminProviders.deleteProviderSuccess'))
  } catch (e) {
    toast.error(e instanceof Error ? e.message : t('adminProviders.deleteProviderFailed'))
  }
}
</script>

<template>
  <div>
    <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h2 class="text-lg font-semibold">{{ t('adminProviders.title') }}</h2>
        <p class="text-sm text-muted-foreground mt-1">{{ t('adminProviders.subtitle') }}</p>
      </div>
      <button
        type="button"
        class="inline-flex items-center justify-center gap-2 rounded-full bg-foreground px-4 py-2 text-sm font-medium text-white hover:bg-foreground/90 transition-colors"
        @click="adminProviders.openCreate"
      >
        <Plus class="w-4 h-4" />
        {{ t('adminProviders.addThirdParty') }}
      </button>
    </div>

    <div v-if="adminProviders.error.value" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ adminProviders.error.value }}
    </div>

    <div v-if="adminProviders.loading.value" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" />
      {{ t('loading') }}
    </div>

    <div v-else-if="adminProviders.providers.value.length" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      <ProviderCard
        v-for="provider in adminProviders.providers.value"
        :key="provider.provider"
        :provider="provider"
        :meta="adminProviders.getProviderMeta(provider.provider)"
        :display-name="adminProviders.providerDisplayName(provider)"
        :is-custom="adminProviders.isCustomProvider(provider)"
        :deleting="adminProviders.deleting.value === provider.provider"
        @edit="adminProviders.openEdit"
        @delete="deleteProvider"
      />
    </div>

    <div v-else class="rounded-xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
      {{ t('adminProviders.noProviders') }}
    </div>

    <ProviderFormModal
      v-if="adminProviders.showModal.value"
      :mode="adminProviders.mode.value"
      :provider="adminProviders.activeProvider.value"
      :meta="adminProviders.activeMeta.value"
      :display-name="adminProviders.activeProvider.value ? adminProviders.providerDisplayName(adminProviders.activeProvider.value) : t('adminProviders.addProvider')"
      :form="adminProviders.form"
      :fields="adminProviders.activeFields.value"
      :oauth2-fields="adminProviders.oauth2Fields"
      :is-custom="adminProviders.activeIsCustom.value"
      :callback-url="adminProviders.activeCallbackPath.value"
      :base-url="adminProviders.baseUrl.value"
      :saving="adminProviders.saving.value"
      :deleting="!!adminProviders.deleting.value"
      @close="adminProviders.closeModal"
      @submit="saveProvider"
      @delete="deleteProvider"
      @update-field="updateField"
    />
  </div>
</template>