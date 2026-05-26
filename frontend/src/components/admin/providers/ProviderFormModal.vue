<script setup lang="ts">
import { computed, ref } from 'vue'
import { Check, Copy, Loader2, Trash2, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import type { FieldDef, Provider, ProviderForm, ProviderFormKey, ProviderMeta } from '@/composables/useAdminProviders'

const props = defineProps<{
  mode: 'create' | 'edit'
  provider: Provider | null
  meta: ProviderMeta
  displayName: string
  form: ProviderForm
  fields: FieldDef[]
  oauth2Fields: FieldDef[]
  isCustom: boolean
  callbackUrl: string
  baseUrl: string
  saving: boolean
  deleting: boolean
}>()

const emit = defineEmits<{
  close: []
  submit: []
  delete: [provider: Provider]
  'update-field': [key: ProviderFormKey, value: string | boolean]
}>()

const { t } = useI18n()
const copiedField = ref('')

const isCreate = computed(() => props.mode === 'create')
const title = computed(() => (isCreate.value ? t('adminProviders.addProvider') : props.displayName))
const providerKey = computed(() => (isCreate.value ? stringField('provider') : props.provider?.provider || ''))
const fullCallbackUrl = computed(() => (props.callbackUrl ? props.baseUrl + props.callbackUrl : ''))

const labelByKey: Partial<Record<ProviderFormKey, string>> = {
  provider: 'adminProviders.providerKey',
  display_name: 'adminProviders.displayName',
  client_id: 'adminProviders.clientId',
  client_secret: 'adminProviders.clientSecret',
  app_id: 'adminProviders.appId',
  app_secret: 'adminProviders.appSecret',
  team_id: 'adminProviders.teamId',
  key_id: 'adminProviders.keyId',
  private_key: 'adminProviders.privateKey',
  base_url: 'adminProviders.instanceUrl',
  tenant_id: 'adminProviders.tenantId',
  icon_url: 'adminProviders.iconUrl',
  authorization_endpoint: 'adminProviders.authorizationEndpoint',
  token_endpoint: 'adminProviders.tokenEndpoint',
  userinfo_endpoint: 'adminProviders.userinfoEndpoint',
  scopes: 'adminProviders.scopes',
  user_id_field: 'adminProviders.userIdField',
  email_field: 'adminProviders.emailField',
  name_field: 'adminProviders.nameField',
  avatar_field: 'adminProviders.avatarField',
}

function fieldLabel(field: FieldDef): string {
  const key = labelByKey[field.key]
  return key ? t(key) : field.label
}

function stringField(key: ProviderFormKey): string {
  const value = props.form[key]
  return typeof value === 'string' ? value : ''
}

function booleanField(key: ProviderFormKey): boolean {
  return props.form[key] === true
}

function updateStringField(key: ProviderFormKey, event: Event) {
  const target = event.target as HTMLInputElement | HTMLTextAreaElement | null
  emit('update-field', key, target?.value ?? '')
}

function updateBooleanField(key: ProviderFormKey, event: Event) {
  const target = event.target as HTMLInputElement | null
  emit('update-field', key, target?.checked ?? false)
}

function hasStoredSecret(field: FieldDef): boolean {
  if (!props.provider) return false
  if (field.key === 'client_secret') return props.provider.has_secret
  if (field.key === 'app_secret') return !!props.provider.has_app_secret
  if (field.key === 'private_key') return !!props.provider.has_private_key
  return false
}

async function copyText(text: string, field: string) {
  if (!text) return
  await navigator.clipboard.writeText(text)
  copiedField.value = field
  setTimeout(() => (copiedField.value = ''), 2000)
}
</script>

<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4" @click.self="emit('close')">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div class="sticky top-0 z-10 flex items-center justify-between gap-4 border-b bg-white px-6 py-4 rounded-t-xl">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <h2 class="text-lg font-semibold truncate">{{ title }}</h2>
              <span v-if="isCustom" class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700">
                {{ t('adminProviders.customOAuth2') }}
              </span>
            </div>
            <p v-if="providerKey" class="text-xs text-muted-foreground mt-0.5">{{ providerKey }}</p>
          </div>
          <button type="button" class="text-muted-foreground hover:text-foreground" @click="emit('close')">
            <X class="w-5 h-5" />
          </button>
        </div>

        <form class="flex flex-col gap-5 px-6 py-5" @submit.prevent="emit('submit')">
          <div class="grid gap-3 rounded-lg bg-muted/40 p-3 sm:grid-cols-3">
            <label class="flex items-center gap-3 cursor-pointer">
              <span class="relative inline-flex items-center">
                <input type="checkbox" :checked="booleanField('enabled')" class="sr-only peer" @change="updateBooleanField('enabled', $event)" />
                <span class="w-9 h-5 bg-gray-200 peer-focus:ring-2 peer-focus:ring-foreground/10 rounded-full peer peer-checked:bg-green-500 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full" />
              </span>
              <span class="text-sm font-medium">{{ t('adminProviders.enabled') }}</span>
            </label>
            <label class="flex items-center gap-3 cursor-pointer">
              <span class="relative inline-flex items-center">
                <input type="checkbox" :checked="booleanField('login_enabled')" class="sr-only peer" @change="updateBooleanField('login_enabled', $event)" />
                <span class="w-9 h-5 bg-gray-200 peer-focus:ring-2 peer-focus:ring-foreground/10 rounded-full peer peer-checked:bg-green-500 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full" />
              </span>
              <span class="text-sm font-medium">{{ t('adminProviders.loginEnabled') }}</span>
            </label>
            <label class="flex items-center gap-3 cursor-pointer">
              <span class="relative inline-flex items-center">
                <input type="checkbox" :checked="booleanField('register_enabled')" class="sr-only peer" @change="updateBooleanField('register_enabled', $event)" />
                <span class="w-9 h-5 bg-gray-200 peer-focus:ring-2 peer-focus:ring-foreground/10 rounded-full peer peer-checked:bg-green-500 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full" />
              </span>
              <span class="text-sm font-medium">{{ t('adminProviders.registerEnabled') }}</span>
            </label>
          </div>

          <div v-if="isCreate" class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ t('adminProviders.providerKey') }}</label>
              <input
                :value="stringField('provider')"
                type="text"
                class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
                placeholder="custom_example"
                @input="updateStringField('provider', $event)"
              />
              <p class="text-[11px] text-muted-foreground mt-1">{{ t('adminProviders.providerKeyHint') }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ t('adminProviders.displayName') }}</label>
              <input
                :value="stringField('display_name')"
                type="text"
                class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
                :placeholder="t('adminProviders.displayName')"
                @input="updateStringField('display_name', $event)"
              />
            </div>
          </div>

          <div v-else-if="isCustom">
            <label class="block text-sm font-medium mb-1.5">{{ t('adminProviders.displayName') }}</label>
            <input
              :value="stringField('display_name')"
              type="text"
              class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
              :placeholder="t('adminProviders.displayName')"
              @input="updateStringField('display_name', $event)"
            />
          </div>

          <div v-if="fullCallbackUrl">
            <label class="block text-xs font-medium text-muted-foreground mb-1">{{ t('adminProviders.callbackUrl') }}</label>
            <div class="flex items-center gap-2 bg-muted rounded-lg p-2.5">
              <code class="flex-1 text-xs font-mono break-all text-foreground">{{ fullCallbackUrl }}</code>
              <button type="button" class="shrink-0 p-1 rounded hover:bg-white transition-colors" @click="copyText(fullCallbackUrl, 'callback')">
                <Check v-if="copiedField === 'callback'" class="w-3.5 h-3.5 text-green-600" />
                <Copy v-else class="w-3.5 h-3.5 text-muted-foreground" />
              </button>
            </div>
            <p class="text-[11px] text-muted-foreground mt-1">{{ t('adminProviders.callbackHint') }}</p>
          </div>

          <section class="grid gap-4 md:grid-cols-2">
            <div v-for="field in fields" :key="field.key" :class="field.type === 'textarea' ? 'md:col-span-2' : ''">
              <label class="block text-sm font-medium mb-1.5">{{ fieldLabel(field) }}</label>
              <textarea
                v-if="field.type === 'textarea'"
                :value="stringField(field.key)"
                rows="4"
                class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10 resize-none"
                :placeholder="field.placeholder || fieldLabel(field)"
                @input="updateStringField(field.key, $event)"
              />
              <input
                v-else
                :value="stringField(field.key)"
                :type="field.type"
                class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
                :placeholder="field.placeholder || fieldLabel(field)"
                @input="updateStringField(field.key, $event)"
              />
              <p v-if="hasStoredSecret(field)" class="text-[11px] text-muted-foreground mt-1">
                {{ t('adminProviders.leaveBlankHint') }}
              </p>
            </div>
          </section>

          <section v-if="isCustom" class="border-t pt-5">
            <div class="mb-4">
              <h3 class="text-sm font-semibold">{{ t('adminProviders.oauth2Endpoints') }}</h3>
              <p class="text-xs text-muted-foreground mt-1">{{ t('adminProviders.oauth2Hint') }}</p>
            </div>
            <div class="grid gap-4 md:grid-cols-2">
              <div v-for="field in oauth2Fields" :key="field.key" :class="field.key.includes('endpoint') || field.key === 'scopes' ? 'md:col-span-2' : ''">
                <label class="block text-sm font-medium mb-1.5">{{ fieldLabel(field) }}</label>
                <input
                  :value="stringField(field.key)"
                  type="text"
                  class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
                  :placeholder="field.placeholder || fieldLabel(field)"
                  @input="updateStringField(field.key, $event)"
                />
                <p v-if="field.key === 'scopes'" class="text-[11px] text-muted-foreground mt-1">{{ t('adminProviders.scopesHint') }}</p>
              </div>
            </div>
          </section>

          <div class="flex items-center justify-between gap-2 border-t pt-4">
            <button
              v-if="isCustom && !isCreate && provider"
              type="button"
              :disabled="deleting || saving"
              class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-50"
              @click="emit('delete', provider)"
            >
              <Trash2 class="w-4 h-4" />
              {{ t('adminProviders.deleteProvider') }}
            </button>
            <span v-else />

            <div class="flex justify-end gap-2">
              <button type="button" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors" @click="emit('close')">
                {{ t('cancel') }}
              </button>
              <button type="submit" :disabled="saving" class="bg-foreground text-white px-5 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
                <Loader2 v-if="saving" class="w-4 h-4 animate-spin" />
                {{ isCreate ? t('adminProviders.addProvider') : t('save') }}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>