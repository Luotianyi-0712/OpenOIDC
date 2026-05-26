<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api } from '@/api/client'
import { Pencil, Loader2, X, Mail, Save, Eye, EyeOff } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { usePublicConfig } from '@/composables/usePublicConfig'

const { t } = useI18n()
const auth = useAuthStore()
const { settings: publicSettings, loaded: publicConfigLoaded } = usePublicConfig()

interface Setting {
  key: string
  value: string
  description: string
}

interface VersionCheckInfo {
  current_version: string
  latest_version: string
  update_available: boolean
  release_url: string
  checked_at: string
  source: string
}

const BOOL_SETTINGS = new Set([
  'registration_enabled',
  'registration_email_verification_required',
  'password_login_enabled',
  'email_verification_required',
  'client_email_verification_default',
  'allow_public_clients',
  'captcha_enabled',
  'turnstile_enabled',
  'social_login_enabled',
  'social_register_enabled',
  'social_binding_enabled',
  'passkey_enabled',
  'password_require_upper',
  'password_require_lower',
  'password_require_digit',
  'password_require_symbol',
])

const SMTP_KEYS = ['smtp_host', 'smtp_port', 'smtp_username', 'smtp_password', 'smtp_from']
const DEDICATED_SETTING_KEYS = new Set([
  ...SMTP_KEYS,
  'site_url',
  'allowed_email_domains',
  'captcha_provider',
  'captcha_site_key',
  'captcha_secret_key',
  'turnstile_site_key',
  'turnstile_secret_key',
  'developer_min_trust_level',
])

function tOrFallback(path: string, fallback: string) {
  const translated = t(path)
  return translated === path ? fallback : translated
}

function settingLabel(key: string) {
  return tOrFallback(`adminSettings.labels.${key}`, key)
}

function settingDescription(setting: Setting) {
  return tOrFallback(`adminSettings.descriptions.${setting.key}`, '—')
}

function aliasRuleTypeLabel(type: string) {
  return tOrFallback(`adminSettings.aliasRuleTypes.${type}`, type)
}

const settings = ref<Setting[]>([])
const loading = ref(false)
const error = ref('')
const success = ref('')
const checkingVersion = ref(false)
const versionCheck = ref<VersionCheckInfo | null>(null)

const showModal = ref(false)
const editingKey = ref('')
const saving = ref(false)
const form = ref({ value: '', description: '' })
const editingDescription = computed(() => tOrFallback(`adminSettings.descriptions.${editingKey.value}`, editingKey.value))

// Site URL
const siteURL = ref(window.location.origin)
const siteSaving = ref(false)

// SMTP form
const smtpForm = ref({ host: '', port: '465', username: '', password: '', from: '' })
const showSmtpPassword = ref(false)
const smtpSaving = ref(false)

// Captcha
const captchaForm = ref({ provider: 'turnstile', siteKey: '', secretKey: '' })
const showCaptchaSecret = ref(false)
const captchaSaving = ref(false)

// Email domain whitelist
const domainWhitelist = ref('')
const domainSaving = ref(false)

// Developer console
const developerMinLevel = ref(1)
const developerSaving = ref(false)

// Alias restrictions
interface AliasRestriction {
  id: string
  pattern: string
  restriction_type: string
  reason: string
  created_at: string
}
const aliases = ref<AliasRestriction[]>([])
const newAlias = ref({ pattern: '', restriction_type: 'blocked', reason: '' })
const aliasAdding = ref(false)

function isBoolSetting(key: string) {
  return BOOL_SETTINGS.has(key)
}

function isDedicatedSetting(key: string) {
  return DEDICATED_SETTING_KEYS.has(key) || key.startsWith('developer_')
}

const generalSettings = computed(() => settings.value.filter(s => !isDedicatedSetting(s.key)))

async function toggleBool(setting: Setting) {
  const newVal = setting.value === 'true' ? 'false' : 'true'
  try {
    await api.put(`/admin/settings/${setting.key}`, { value: newVal, description: setting.description })
    setting.value = newVal
  } catch (e: any) {
    error.value = e.message
  }
}

onMounted(() => {
  fetchSettings()
  fetchAliases()
})

async function fetchSettings() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<Setting[]>('/admin/settings')
    settings.value = res.data ?? []
    // Populate domain whitelist
    for (const s of settings.value) {
      if (s.key === 'site_url') siteURL.value = s.value || window.location.origin
      if (s.key === 'smtp_host') smtpForm.value.host = s.value
      if (s.key === 'smtp_port') smtpForm.value.port = s.value || '465'
      if (s.key === 'smtp_username') smtpForm.value.username = s.value
      if (s.key === 'smtp_password') smtpForm.value.password = s.value
      if (s.key === 'smtp_from') smtpForm.value.from = s.value
      if (s.key === 'allowed_email_domains') domainWhitelist.value = s.value
      if (s.key === 'captcha_provider') captchaForm.value.provider = s.value || 'turnstile'
      if (s.key === 'captcha_site_key') captchaForm.value.siteKey = s.value
      if (s.key === 'captcha_secret_key') captchaForm.value.secretKey = s.value
      if (s.key === 'turnstile_site_key' && !captchaForm.value.siteKey) captchaForm.value.siteKey = s.value
      if (s.key === 'turnstile_secret_key' && !captchaForm.value.secretKey) captchaForm.value.secretKey = s.value
      if (s.key === 'developer_min_trust_level') developerMinLevel.value = parseInt(s.value) || 1
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function checkVersionUpdate() {
  checkingVersion.value = true
  error.value = ''
  try {
    const res = await api.get<VersionCheckInfo>('/admin/version/check')
    versionCheck.value = res.data || null
  } catch (e: any) {
    error.value = e.message
  } finally {
    checkingVersion.value = false
  }
}

function openEdit(setting: Setting) {
  editingKey.value = setting.key
  form.value = { value: setting.value, description: setting.description }
  showModal.value = true
}

async function saveSetting() {
  saving.value = true
  error.value = ''
  try {
    await api.put(`/admin/settings/${editingKey.value}`, form.value)
    showModal.value = false
    await fetchSettings()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

async function saveSiteURL() {
  siteSaving.value = true
  error.value = ''
  success.value = ''
  try {
    await api.put('/admin/settings/site_url', {
      value: siteURL.value,
      description: 'Public site URL for documentation examples and public callback displays',
    })
    await fetchSettings()
    await auth.fetchPublicSettings()
    success.value = t('adminSettings.siteSaved')
    setTimeout(() => (success.value = ''), 3000)
  } catch (e: any) {
    error.value = e.message
  } finally {
    siteSaving.value = false
  }
}

async function saveSmtp() {
  smtpSaving.value = true
  error.value = ''
  success.value = ''
  try {
    await api.put('/admin/settings/smtp_host', { value: smtpForm.value.host, description: 'SMTP server host' })
    await api.put('/admin/settings/smtp_port', { value: smtpForm.value.port, description: 'SMTP server port' })
    await api.put('/admin/settings/smtp_username', { value: smtpForm.value.username, description: 'SMTP username' })
    if (smtpForm.value.password) {
      await api.put('/admin/settings/smtp_password', { value: smtpForm.value.password, description: 'SMTP password' })
    }
    await api.put('/admin/settings/smtp_from', { value: smtpForm.value.from, description: 'SMTP sender address' })
    success.value = t('adminSettings.smtpSaved')
    setTimeout(() => (success.value = ''), 3000)
    await fetchSettings()
  } catch (e: any) {
    error.value = e.message
  } finally {
    smtpSaving.value = false
  }
}

async function saveCaptcha() {
  captchaSaving.value = true
  error.value = ''
  success.value = ''
  try {
    await api.put('/admin/settings/captcha_provider', { value: captchaForm.value.provider, description: 'Human verification provider: turnstile or hcaptcha' })
    await api.put('/admin/settings/captcha_site_key', { value: captchaForm.value.siteKey, description: 'Captcha frontend site key' })
    if (captchaForm.value.secretKey) {
      await api.put('/admin/settings/captcha_secret_key', { value: captchaForm.value.secretKey, description: 'Captcha backend verification secret' })
    }
    if (captchaForm.value.provider === 'turnstile') {
      await api.put('/admin/settings/turnstile_site_key', { value: captchaForm.value.siteKey, description: 'Cloudflare Turnstile site key' })
      if (captchaForm.value.secretKey) {
        await api.put('/admin/settings/turnstile_secret_key', { value: captchaForm.value.secretKey, description: 'Cloudflare Turnstile secret key' })
      }
    }
    success.value = t('adminSettings.captchaSaved')
    setTimeout(() => (success.value = ''), 3000)
  } catch (e: any) {
    error.value = e.message
  } finally {
    captchaSaving.value = false
  }
}

async function saveDomainWhitelist() {
  domainSaving.value = true
  error.value = ''
  success.value = ''
  try {
    await api.put('/admin/settings/allowed_email_domains', {
      value: domainWhitelist.value,
      description: 'Comma-separated list of allowed email domains for registration. Empty = allow all.',
    })
    success.value = t('adminSettings.domainSaved')
    setTimeout(() => (success.value = ''), 3000)
  } catch (e: any) {
    error.value = e.message
  } finally {
    domainSaving.value = false
  }
}

async function saveDeveloperLevel() {
  developerSaving.value = true
  error.value = ''
  success.value = ''
  try {
    await api.put('/admin/settings/developer_min_trust_level', {
      value: String(developerMinLevel.value),
      description: 'Minimum trust level required to access developer console',
    })
    await fetchSettings()
    await auth.fetchPublicSettings()
    if (auth.isLoggedIn) await auth.fetchDeveloperStatus()
    success.value = t('adminSettings.developerSaved')
    setTimeout(() => (success.value = ''), 3000)
  } catch (e: any) {
    error.value = e.message
  } finally {
    developerSaving.value = false
  }
}

async function fetchAliases() {
  try {
    const res = await api.get<AliasRestriction[]>('/admin/alias-restrictions')
    aliases.value = res.data ?? []
  } catch (e: any) {
    error.value = e.message
  }
}

async function addAlias() {
  if (!newAlias.value.pattern) return
  aliasAdding.value = true
  error.value = ''
  try {
    await api.post('/admin/alias-restrictions', newAlias.value)
    newAlias.value = { pattern: '', restriction_type: 'blocked', reason: '' }
    await fetchAliases()
  } catch (e: any) {
    error.value = e.message
  } finally {
    aliasAdding.value = false
  }
}

async function deleteAlias(id: string) {
  try {
    await api.del(`/admin/alias-restrictions/${id}`)
    aliases.value = aliases.value.filter(a => a.id !== id)
  } catch (e: any) {
    error.value = e.message
  }
}
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-lg font-semibold">{{ $t('adminSettings.title') }}</h2>
      <p class="text-sm text-muted-foreground mt-1">{{ $t('adminSettings.subtitle') }}</p>
    </div>

    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{{ error }}</div>
    <div v-if="success" class="mb-4 rounded-lg border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-700">{{ success }}</div>

    <div v-if="loading && settings.length === 0" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <template v-else>
      <div class="border border-border rounded-xl p-6 mb-8 bg-muted/20">
        <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 mb-4">
          <div>
            <h3 class="text-base font-semibold mb-2">{{ $t('adminSettings.versionTitle') }}</h3>
            <p class="text-sm text-muted-foreground">{{ $t('adminSettings.versionDesc') }}</p>
          </div>
          <button @click="checkVersionUpdate" :disabled="checkingVersion" class="px-4 py-2 rounded-full border border-border bg-white text-sm font-medium hover:bg-muted transition-colors disabled:opacity-50 flex items-center justify-center gap-2">
            <Loader2 v-if="checkingVersion" class="w-4 h-4 animate-spin" />
            {{ checkingVersion ? $t('adminSettings.checkingUpdate') : $t('adminSettings.checkUpdate') }}
          </button>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="flex items-center justify-between rounded-lg border border-border bg-white px-4 py-3">
            <span class="text-sm text-muted-foreground">{{ $t('adminSettings.currentVersion') }}</span>
            <code class="text-sm font-mono font-semibold">{{ publicConfigLoaded ? publicSettings.version : 'v1.10' }}</code>
          </div>
          <div v-if="versionCheck" class="flex items-center justify-between rounded-lg border border-border bg-white px-4 py-3">
            <span class="text-sm text-muted-foreground">{{ $t('adminSettings.latestVersion') }}</span>
            <code class="text-sm font-mono font-semibold">{{ versionCheck.latest_version }}</code>
          </div>
        </div>
        <div v-if="versionCheck" class="mt-4 rounded-lg px-4 py-3 text-sm" :class="versionCheck.update_available ? 'border border-amber-200 bg-amber-50 text-amber-800' : 'border border-green-200 bg-green-50 text-green-700'">
          <div class="font-medium">{{ versionCheck.update_available ? $t('adminSettings.updateAvailable') : $t('adminSettings.alreadyLatest') }}</div>
          <a v-if="versionCheck.release_url" :href="versionCheck.release_url" target="_blank" rel="noreferrer" class="inline-flex mt-1 underline underline-offset-2">
            {{ $t('adminSettings.releasePage') }}
          </a>
        </div>
      </div>

      <!-- Site URL -->
      <div class="border border-border rounded-xl p-6 mb-8">
        <h3 class="text-base font-semibold mb-2">{{ $t('adminSettings.siteTitle') }}</h3>
        <p class="text-sm text-muted-foreground mb-4">{{ $t('adminSettings.siteDesc') }}</p>
        <form @submit.prevent="saveSiteURL" class="flex flex-col gap-3">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.siteURL') }}</label>
            <input v-model="siteURL" type="url" :placeholder="$t('adminSettings.siteURLPlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10" />
            <p class="text-xs text-muted-foreground mt-1">{{ $t('adminSettings.siteURLHint') }}</p>
          </div>
          <div class="flex justify-end">
            <button type="submit" :disabled="siteSaving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="siteSaving" class="w-4 h-4 animate-spin" />
              <Save v-else class="w-4 h-4" />
              {{ $t('adminSettings.siteSave') }}
            </button>
          </div>
        </form>
      </div>

      <!-- General Settings -->
      <div class="border border-border rounded-xl overflow-hidden mb-8">
        <table class="w-full text-sm">
          <thead class="bg-muted/50 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
            <tr>
              <th class="px-4 py-3">{{ $t('adminSettings.key') }}</th>
              <th class="px-4 py-3">{{ $t('adminSettings.value') }}</th>
              <th class="px-4 py-3">{{ $t('adminRules.description') }}</th>
              <th class="px-4 py-3 w-20">{{ $t('actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-if="generalSettings.length === 0">
              <td colspan="4" class="px-4 py-8 text-center text-muted-foreground">{{ $t('adminSettings.noSettings') }}</td>
            </tr>
            <tr v-for="setting in generalSettings" :key="setting.key" class="hover:bg-muted/30 transition-colors">
              <td class="px-4 py-3 font-medium">
                <div>{{ settingLabel(setting.key) }}</div>
                <div class="font-mono text-[11px] text-muted-foreground mt-0.5">{{ setting.key }}</div>
              </td>
              <td class="px-4 py-3 max-w-64">
                <label v-if="isBoolSetting(setting.key)" class="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" :checked="setting.value === 'true'" @change="toggleBool(setting)" class="sr-only peer" />
                  <div class="w-9 h-5 bg-gray-200 peer-focus:ring-2 peer-focus:ring-foreground/10 rounded-full peer peer-checked:bg-green-500 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full"></div>
                </label>
                <code v-else class="text-xs bg-muted px-1.5 py-0.5 rounded break-all">{{ setting.value }}</code>
              </td>
              <td class="px-4 py-3 text-muted-foreground max-w-64 truncate">{{ settingDescription(setting) }}</td>
              <td class="px-4 py-3">
                <button v-if="!isBoolSetting(setting.key)" @click="openEdit(setting)" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <Pencil class="w-3 h-3" /> {{ $t('edit') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- SMTP Configuration -->
      <div class="border border-border rounded-xl p-6">
        <div class="flex items-center gap-2 mb-4">
          <Mail class="w-5 h-5 text-muted-foreground" />
          <h3 class="text-base font-semibold">{{ $t('adminSettings.smtpTitle') }}</h3>
        </div>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('adminSettings.smtpDesc') }}</p>
        <form @submit.prevent="saveSmtp" class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.smtpHost') }}</label>
            <input v-model="smtpForm.host" type="text" placeholder="smtp.example.com" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.smtpPort') }}</label>
            <input v-model="smtpForm.port" type="text" placeholder="465" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
            <p class="text-xs text-muted-foreground mt-1">{{ $t('adminSettings.smtpPortHint') }}</p>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.smtpUsername') }}</label>
            <input v-model="smtpForm.username" type="text" placeholder="noreply@example.com" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.smtpPassword') }}</label>
            <div class="relative">
              <input v-model="smtpForm.password" :type="showSmtpPassword ? 'text' : 'password'" :placeholder="$t('adminSettings.smtpPasswordPlaceholder')" class="w-full px-3 pr-10 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
              <button
                type="button"
                @click="showSmtpPassword = !showSmtpPassword"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
              >
                <EyeOff v-if="showSmtpPassword" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </button>
            </div>
          </div>
          <div class="md:col-span-2">
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.smtpFrom') }}</label>
            <input v-model="smtpForm.from" type="text" placeholder="OIDC Platform <noreply@example.com>" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
            <p class="text-xs text-muted-foreground mt-1">{{ $t('adminSettings.smtpFromHint') }}</p>
          </div>
          <div class="md:col-span-2 flex justify-end">
            <button type="submit" :disabled="smtpSaving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="smtpSaving" class="w-4 h-4 animate-spin" />
              <Save v-else class="w-4 h-4" />
              {{ $t('adminSettings.smtpSave') }}
            </button>
          </div>
        </form>
      </div>

      <!-- Captcha -->
      <div class="border border-border rounded-xl p-6 mt-8">
        <h3 class="text-base font-semibold mb-2">{{ $t('adminSettings.captchaTitle') }}</h3>
        <p class="text-sm text-muted-foreground mb-4">{{ $t('adminSettings.captchaDesc') }}</p>
        <form @submit.prevent="saveCaptcha" class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.captchaProvider') }}</label>
            <select v-model="captchaForm.provider" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10">
              <option value="turnstile">Cloudflare Turnstile</option>
              <option value="hcaptcha">hCaptcha</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.captchaSiteKey') }}</label>
            <input v-model="captchaForm.siteKey" type="text" placeholder="site key" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div class="md:col-span-2">
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.captchaSecretKey') }}</label>
            <div class="relative">
              <input v-model="captchaForm.secretKey" :type="showCaptchaSecret ? 'text' : 'password'" :placeholder="$t('adminSettings.smtpPasswordPlaceholder')" class="w-full px-3 pr-10 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
              <button
                type="button"
                @click="showCaptchaSecret = !showCaptchaSecret"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
              >
                <EyeOff v-if="showCaptchaSecret" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </button>
            </div>
          </div>
          <div class="md:col-span-2 flex justify-end">
            <button type="submit" :disabled="captchaSaving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="captchaSaving" class="w-4 h-4 animate-spin" />
              <Save v-else class="w-4 h-4" />
              {{ $t('save') }}
            </button>
          </div>
        </form>
      </div>

      <!-- Email Domain Whitelist -->
      <div class="border border-border rounded-xl p-6 mt-8">
        <h3 class="text-base font-semibold mb-2">{{ $t('adminSettings.domainTitle') }}</h3>
        <p class="text-sm text-muted-foreground mb-4">{{ $t('adminSettings.domainDesc') }}</p>
        <form @submit.prevent="saveDomainWhitelist" class="flex flex-col gap-3">
          <textarea v-model="domainWhitelist" rows="3" :placeholder="$t('adminSettings.domainPlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          <div class="flex justify-end">
            <button type="submit" :disabled="domainSaving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="domainSaving" class="w-4 h-4 animate-spin" />
              <Save v-else class="w-4 h-4" />
              {{ $t('save') }}
            </button>
          </div>
        </form>
      </div>

      <!-- Developer Console Access -->
      <div class="border border-border rounded-xl p-6 mt-8">
        <h3 class="text-base font-semibold mb-2">{{ $t('adminSettings.developerTitle') }}</h3>
        <p class="text-sm text-muted-foreground mb-4">{{ $t('adminSettings.developerDesc') }}</p>
        <form @submit.prevent="saveDeveloperLevel" class="flex items-end gap-4">
          <div class="w-32">
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.developerMinLevel') }}</label>
            <input v-model.number="developerMinLevel" type="number" min="0" max="10" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <button type="submit" :disabled="developerSaving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
            <Loader2 v-if="developerSaving" class="w-4 h-4 animate-spin" />
            <Save v-else class="w-4 h-4" />
            {{ $t('save') }}
          </button>
        </form>
      </div>

      <!-- Alias Restrictions -->
      <div class="border border-border rounded-xl p-6 mt-8">
        <h3 class="text-base font-semibold mb-2">{{ $t('adminSettings.aliasTitle') }}</h3>
        <p class="text-sm text-muted-foreground mb-4">{{ $t('adminSettings.aliasDesc') }}</p>
        <div v-if="aliases.length" class="mb-4 space-y-2">
          <div v-for="alias in aliases" :key="alias.id" class="flex items-center justify-between px-3 py-2 bg-muted/50 rounded-lg">
            <div>
              <code class="text-xs font-mono">{{ alias.pattern }}</code>
              <span class="text-xs text-muted-foreground ml-2">{{ aliasRuleTypeLabel(alias.restriction_type) }}</span>
              <span v-if="alias.reason" class="text-xs text-muted-foreground ml-2">— {{ alias.reason }}</span>
            </div>
            <button @click="deleteAlias(alias.id)" class="text-xs text-destructive hover:underline">{{ $t('delete') }}</button>
          </div>
        </div>
        <form @submit.prevent="addAlias" class="flex items-end gap-2">
          <div class="flex-1">
            <label class="block text-xs font-medium mb-1">{{ $t('adminSettings.aliasPattern') }}</label>
            <input v-model="newAlias.pattern" type="text" :placeholder="$t('adminSettings.aliasPatternPlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div class="w-36">
            <label class="block text-xs font-medium mb-1">{{ $t('adminSettings.aliasType') }}</label>
            <select v-model="newAlias.restriction_type" class="w-full px-3 py-2 border border-border rounded-lg text-sm">
              <option value="blocked">{{ $t('adminSettings.aliasBlocked') }}</option>
              <option value="reserved">{{ $t('adminSettings.aliasReserved') }}</option>
              <option value="regex_blocked">{{ $t('adminSettings.aliasRegex') }}</option>
            </select>
          </div>
          <div class="flex-1">
            <label class="block text-xs font-medium mb-1">{{ $t('adminSettings.aliasReason') }}</label>
            <input v-model="newAlias.reason" type="text" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <button type="submit" :disabled="aliasAdding || !newAlias.pattern" class="px-4 py-2 bg-foreground text-white rounded-full text-sm font-medium hover:bg-foreground/90 disabled:opacity-50">
            {{ $t('adminSettings.aliasAdd') }}
          </button>
        </form>
      </div>
    </template>

    <!-- Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4 p-6">
        <div class="flex items-center justify-between mb-5">
          <h2 class="text-lg font-semibold">{{ $t('edit') }}</h2>
          <button @click="showModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="saveSetting" class="flex flex-col gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.key') }}</label>
            <div class="px-3 py-2 bg-muted rounded-lg text-sm">
              <div class="font-medium">{{ settingLabel(editingKey) }}</div>
              <div class="font-mono text-xs text-muted-foreground mt-0.5">{{ editingKey }}</div>
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminSettings.value') }}</label>
            <textarea v-model="form.value" rows="3" class="w-full px-3 py-2 border border-border rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminRules.description') }}</label>
            <div class="px-3 py-2 bg-muted rounded-lg text-sm text-muted-foreground">{{ editingDescription }}</div>
          </div>
          <div class="flex justify-end gap-2 mt-2">
            <button type="button" @click="showModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
            <button type="submit" :disabled="saving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="saving" class="w-4 h-4 animate-spin" /> {{ $t('save') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
