<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '@/api/client'
import { Plus, Pencil, Trash2, Loader2, RefreshCw, X, MinusCircle } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

type RuleOperator = 'AND' | 'OR'
type ConditionType =
  | 'provider_bound'
  | 'binding_age_days'
  | 'provider_account_age_days'
  | 'provider_email_verified'
  | 'provider_email_domain'
  | 'provider_raw_number'
  | 'provider_raw_string'
  | 'provider_raw_bool'
  | 'user_email_domain'
  | 'user_created_age_days'
  | 'user_has_verified_email'

type ValueType = 'none' | 'number' | 'string' | 'bool' | 'domains'

type RuleCondition = {
  type?: ConditionType
  provider?: string
  field?: string
  operator?: string
  value?: string | number | boolean
  values?: string[]
  min_days?: number
  min_binding_days?: number
}

interface SecurityRule {
  id: string
  name: string
  description: string
  level: number
  priority: number
  conditions: {
    operator: RuleOperator
    rules: RuleCondition[]
  }
  is_active: boolean
  created_at: string
}

type FieldOption = {
  value: string
  labelKey: string
  type: 'number' | 'string' | 'bool' | 'time'
}

type ConditionOption = {
  type: ConditionType
  valueType: ValueType
  operatorType?: 'number' | 'string' | 'bool'
  needsProvider?: boolean
  needsField?: boolean
  supportsAnyProvider?: boolean
}

const rules = ref<SecurityRule[]>([])
const loading = ref(false)
const error = ref('')
const success = ref('')

const showModal = ref(false)
const isCreate = ref(false)
const editingRule = ref<SecurityRule | null>(null)
const saving = ref(false)
const form = ref({
  name: '',
  description: '',
  level: 1,
  priority: 0,
  operator: 'AND' as RuleOperator,
  conditions: [] as RuleCondition[],
  is_active: true,
})

const showDeleteModal = ref(false)
const deletingRule = ref<SecurityRule | null>(null)
const deleting = ref(false)

const recomputing = ref(false)

const providerOptions = [
  'github', 'google', 'gitlab', 'gitee', 'linuxdo', 'discord',
  'telegram', 'microsoft', 'apple', 'qq', 'wechat', 'phone',
]

const conditionOptions: ConditionOption[] = [
  { type: 'provider_bound', valueType: 'none', needsProvider: true, supportsAnyProvider: true },
  { type: 'binding_age_days', valueType: 'number', operatorType: 'number', needsProvider: true, supportsAnyProvider: true },
  { type: 'provider_account_age_days', valueType: 'number', operatorType: 'number', needsProvider: true, needsField: true },
  { type: 'provider_email_verified', valueType: 'bool', operatorType: 'bool', needsProvider: true, supportsAnyProvider: true },
  { type: 'provider_email_domain', valueType: 'domains', needsProvider: true, supportsAnyProvider: true },
  { type: 'provider_raw_number', valueType: 'number', operatorType: 'number', needsProvider: true, needsField: true },
  { type: 'provider_raw_string', valueType: 'string', operatorType: 'string', needsProvider: true, needsField: true },
  { type: 'provider_raw_bool', valueType: 'bool', operatorType: 'bool', needsProvider: true, needsField: true },
  { type: 'user_email_domain', valueType: 'domains' },
  { type: 'user_created_age_days', valueType: 'number', operatorType: 'number' },
  { type: 'user_has_verified_email', valueType: 'bool', operatorType: 'bool' },
]

const numberOperators = ['gte', 'gt', 'lte', 'lt', 'eq', 'neq']
const stringOperators = ['eq', 'neq', 'contains', 'prefix', 'suffix', 'regex', 'in']
const boolOperators = ['eq', 'neq']

const emailDomainField: FieldOption = { value: 'email_domain', labelKey: 'emailDomain', type: 'string' }
const emailVerifiedField: FieldOption = { value: 'email_verified', labelKey: 'emailVerified', type: 'bool' }

const commonProviderFields: FieldOption[] = [
  emailDomainField,
  emailVerifiedField,
]

const providerFields: Record<string, FieldOption[]> = {
  github: [
    { value: 'created_at', labelKey: 'githubCreatedAt', type: 'time' },
    { value: 'followers', labelKey: 'githubFollowers', type: 'number' },
    { value: 'following', labelKey: 'githubFollowing', type: 'number' },
    { value: 'public_repos', labelKey: 'githubPublicRepos', type: 'number' },
    { value: 'public_gists', labelKey: 'githubPublicGists', type: 'number' },
    ...commonProviderFields,
  ],
  google: [
    emailVerifiedField,
    emailDomainField,
    { value: 'hd', labelKey: 'googleHostedDomain', type: 'string' },
  ],
  discord: [
    emailVerifiedField,
    emailDomainField,
    { value: 'mfa_enabled', labelKey: 'discordMfaEnabled', type: 'bool' },
    { value: 'public_flags', labelKey: 'discordPublicFlags', type: 'number' },
  ],
  gitlab: [
    { value: 'created_at', labelKey: 'createdAt', type: 'time' },
    { value: 'username', labelKey: 'username', type: 'string' },
    ...commonProviderFields,
  ],
  gitee: [
    { value: 'created_at', labelKey: 'createdAt', type: 'time' },
    { value: 'login', labelKey: 'username', type: 'string' },
    { value: 'name', labelKey: 'displayName', type: 'string' },
    emailDomainField,
  ],
  linuxdo: [
    { value: 'username', labelKey: 'username', type: 'string' },
    { value: 'name', labelKey: 'displayName', type: 'string' },
    ...commonProviderFields,
  ],
  microsoft: [
    emailDomainField,
    { value: 'tenant', labelKey: 'microsoftTenant', type: 'string' },
    { value: 'tid', labelKey: 'microsoftTenantId', type: 'string' },
    { value: 'userPrincipalName', labelKey: 'microsoftUserPrincipalName', type: 'string' },
  ],
  apple: [
    emailVerifiedField,
    emailDomainField,
  ],
  telegram: [
    { value: 'username', labelKey: 'username', type: 'string' },
    { value: 'first_name', labelKey: 'firstName', type: 'string' },
    { value: 'last_name', labelKey: 'lastName', type: 'string' },
    { value: 'auth_date', labelKey: 'telegramAuthDate', type: 'number' },
  ],
  qq: [
    { value: 'nickname', labelKey: 'nickname', type: 'string' },
  ],
  wechat: [
    { value: 'nickname', labelKey: 'nickname', type: 'string' },
    { value: 'country', labelKey: 'country', type: 'string' },
    { value: 'province', labelKey: 'province', type: 'string' },
    { value: 'city', labelKey: 'city', type: 'string' },
    { value: 'sex', labelKey: 'wechatSex', type: 'number' },
  ],
  phone: [
    { value: 'phone_number', labelKey: 'phoneNumber', type: 'string' },
  ],
}

const defaultCondition = (): RuleCondition => ({
  type: 'provider_bound',
  provider: 'github',
})

const hasConditions = computed(() => form.value.conditions.length > 0)

onMounted(fetchRules)

async function fetchRules() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<SecurityRule[]>('/admin/security-rules')
    rules.value = res.data ?? []
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function openCreate() {
  isCreate.value = true
  editingRule.value = null
  form.value = {
    name: '',
    description: '',
    level: 1,
    priority: 0,
    operator: 'AND',
    conditions: [defaultCondition()],
    is_active: true,
  }
  showModal.value = true
}

function openEdit(rule: SecurityRule) {
  isCreate.value = false
  editingRule.value = rule
  form.value = {
    name: rule.name,
    description: rule.description,
    level: rule.level,
    priority: rule.priority,
    operator: rule.conditions?.operator || 'AND',
    conditions: rule.conditions?.rules?.length ? rule.conditions.rules.map(normalizeConditionForEdit) : [defaultCondition()],
    is_active: rule.is_active,
  }
  showModal.value = true
}

function addCondition() {
  form.value.conditions.push(defaultCondition())
}

function removeCondition(index: number) {
  form.value.conditions.splice(index, 1)
}

function conditionOption(type?: string) {
  return conditionOptions.find(item => item.type === type) ?? conditionOptions[0]
}

function normalizeConditionForEdit(cond: RuleCondition): RuleCondition {
  const normalized: RuleCondition = { ...cond }
  if (!normalized.type) {
    normalized.type = (normalized.min_binding_days ?? 0) > 0 ? 'binding_age_days' : 'provider_bound'
  }
  if (normalized.type === 'binding_age_days') {
    normalized.min_days = normalized.min_days ?? normalized.min_binding_days ?? 0
  }
  if (normalized.type === 'provider_account_age_days') {
    normalized.field = normalized.field || 'created_at'
    normalized.min_days = normalized.min_days ?? Number(normalized.value ?? 0)
  }
  if (normalized.type === 'provider_email_verified' || normalized.type === 'provider_raw_bool' || normalized.type === 'user_has_verified_email') {
    normalized.value = normalized.value ?? true
    normalized.operator = normalized.operator || 'eq'
  }
  if (normalized.type === 'provider_email_domain' || normalized.type === 'user_email_domain') {
    normalized.values = normalized.values?.length ? normalized.values : valuesFromRaw(normalized.value)
  }
  ensureDefaults(normalized)
  return normalized
}

function ensureDefaults(cond: RuleCondition) {
  const option = conditionOption(cond.type)
  cond.type = option.type
  if (option.needsProvider && cond.provider === undefined) cond.provider = 'github'
  if (option.type === 'provider_account_age_days') cond.field = cond.field || 'created_at'
  if (option.needsField && !cond.field) cond.field = defaultFieldForCondition(cond)
  if (option.operatorType === 'number') cond.operator = cond.operator || 'gte'
  if (option.operatorType === 'string') cond.operator = cond.operator || 'eq'
  if (option.operatorType === 'bool') cond.operator = cond.operator || 'eq'
  if (option.valueType === 'number') cond.value = Number(cond.value ?? cond.min_days ?? cond.min_binding_days ?? 0)
  if (option.valueType === 'string') cond.value = String(cond.value ?? '')
  if (option.valueType === 'bool') cond.value = cond.value === undefined ? true : Boolean(cond.value)
  if (option.valueType === 'domains') cond.values = cond.values?.length ? cond.values : valuesFromRaw(cond.value)
}

function onTypeChange(cond: RuleCondition) {
  const provider = cond.provider
  const type = cond.type
  Object.keys(cond).forEach(key => delete (cond as any)[key])
  cond.type = type
  if (conditionOption(type).needsProvider) cond.provider = provider || 'github'
  ensureDefaults(cond)
}

function onProviderChange(cond: RuleCondition) {
  const option = conditionOption(cond.type)
  if (option.needsField && !fieldOptions(cond).some(field => field.value === cond.field)) {
    cond.field = defaultFieldForCondition(cond)
  }
}

function defaultFieldForCondition(cond: RuleCondition) {
  if (cond.type === 'provider_account_age_days') return 'created_at'
  const fields = fieldOptions(cond)
  const expectedType = cond.type === 'provider_raw_number' ? 'number' : cond.type === 'provider_raw_bool' ? 'bool' : 'string'
  return fields.find(field => field.type === expectedType)?.value || fields[0]?.value || ''
}

function fieldOptions(cond: RuleCondition) {
  const provider = cond.provider || 'github'
  const fields = providerFields[provider] || commonProviderFields
  if (cond.type === 'provider_account_age_days') {
    return fields.filter(field => field.type === 'time')
  }
  if (cond.type === 'provider_raw_number') return fields.filter(field => field.type === 'number')
  if (cond.type === 'provider_raw_bool') return fields.filter(field => field.type === 'bool')
  if (cond.type === 'provider_raw_string') return fields.filter(field => field.type === 'string')
  return fields
}

function operatorOptions(cond: RuleCondition) {
  const option = conditionOption(cond.type)
  if (option.operatorType === 'number') return numberOperators
  if (option.operatorType === 'string') return stringOperators
  if (option.operatorType === 'bool') return boolOperators
  return []
}

function valuesFromRaw(value: unknown) {
  if (Array.isArray(value)) return value.map(String).filter(Boolean)
  if (typeof value === 'string') return value.split(',').map(v => v.trim()).filter(Boolean)
  return []
}

function valuesText(cond: RuleCondition) {
  return (cond.values || []).join(', ')
}

function setValuesText(cond: RuleCondition, value: string) {
  cond.values = value.split(',').map(v => v.trim()).filter(Boolean)
}

function handleValuesInput(cond: RuleCondition, event: Event) {
  setValuesText(cond, (event.target as HTMLInputElement).value)
}

function buildConditionPayload(cond: RuleCondition): RuleCondition {
  const copy: RuleCondition = { ...cond }
  ensureDefaults(copy)
  const option = conditionOption(copy.type)
  const payload: RuleCondition = { type: copy.type }

  if (option.needsProvider && copy.provider) payload.provider = copy.provider
  if (option.needsField && copy.field) payload.field = copy.field
  if (copy.operator) payload.operator = copy.operator

  if (copy.type === 'binding_age_days') {
    payload.min_days = Number(copy.value ?? copy.min_days ?? copy.min_binding_days ?? 0)
    payload.min_binding_days = payload.min_days
    return payload
  }
  if (copy.type === 'provider_account_age_days' || copy.type === 'user_created_age_days') {
    payload.min_days = Number(copy.value ?? copy.min_days ?? 0)
    return payload
  }
  if (option.valueType === 'number') payload.value = Number(copy.value ?? 0)
  if (option.valueType === 'string') payload.value = String(copy.value ?? '')
  if (option.valueType === 'bool') payload.value = Boolean(copy.value)
  if (option.valueType === 'domains') payload.values = copy.values || []
  if (option.valueType === 'none') delete payload.operator
  return payload
}

async function saveRule() {
  saving.value = true
  error.value = ''
  try {
    const payload = {
      name: form.value.name,
      description: form.value.description,
      level: form.value.level,
      priority: form.value.priority,
      conditions: {
        operator: form.value.operator,
        rules: form.value.conditions.map(buildConditionPayload),
      },
      is_active: form.value.is_active,
    }
    if (isCreate.value) {
      await api.post('/admin/security-rules', payload)
    } else if (editingRule.value) {
      await api.put(`/admin/security-rules/${editingRule.value.id}`, payload)
    }
    showModal.value = false
    await fetchRules()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

function confirmDelete(rule: SecurityRule) {
  deletingRule.value = rule
  showDeleteModal.value = true
}

async function deleteRule() {
  if (!deletingRule.value) return
  deleting.value = true
  error.value = ''
  try {
    await api.del(`/admin/security-rules/${deletingRule.value.id}`)
    showDeleteModal.value = false
    await fetchRules()
  } catch (e: any) {
    error.value = e.message
  } finally {
    deleting.value = false
  }
}

async function recomputeAll() {
  recomputing.value = true
  error.value = ''
  success.value = ''
  try {
    await api.post('/admin/security-rules/recompute')
    success.value = t('adminRules.recomputeSuccess')
    setTimeout(() => (success.value = ''), 3000)
  } catch (e: any) {
    error.value = e.message
  } finally {
    recomputing.value = false
  }
}

function conditionSummary(rule: SecurityRule) {
  if (!rule.conditions?.rules?.length) return '-'
  const op = rule.conditions.operator === 'OR' ? ` ${t('adminRules.conditionOrShort')} ` : ` ${t('adminRules.conditionAndShort')} `
  return rule.conditions.rules.map(conditionLabel).join(op)
}

function conditionLabel(condition: RuleCondition) {
  const cond = normalizeConditionForEdit(condition)
  const provider = providerLabel(cond.provider)
  const typeLabel = t(`adminRules.conditionTypes.${cond.type}`)
  if (cond.type === 'provider_bound') return cond.provider ? `${provider}` : t('adminRules.anyProvider')
  if (cond.type === 'binding_age_days') return `${provider} ${t('adminRules.boundDays')} ${operatorLabel(cond.operator)} ${cond.min_days ?? cond.min_binding_days ?? cond.value ?? 0}${t('adminRules.days')}`
  if (cond.type === 'provider_account_age_days') return `${provider} ${fieldLabel(cond)} ${operatorLabel(cond.operator)} ${cond.min_days ?? cond.value ?? 0}${t('adminRules.days')}`
  if (cond.type === 'provider_email_verified') return `${provider} ${typeLabel} ${operatorLabel(cond.operator)} ${formatValue(cond)}`
  if (cond.type === 'provider_email_domain' || cond.type === 'user_email_domain') return `${typeLabel}: ${(cond.values || []).join(', ')}`
  if (cond.type === 'user_created_age_days') return `${typeLabel} ${operatorLabel(cond.operator)} ${cond.min_days ?? cond.value ?? 0}${t('adminRules.days')}`
  if (cond.type === 'user_has_verified_email') return `${typeLabel} ${operatorLabel(cond.operator)} ${formatValue(cond)}`
  return `${provider} ${fieldLabel(cond)} ${operatorLabel(cond.operator)} ${formatValue(cond)}`
}

function providerLabel(provider?: string) {
  if (!provider) return t('adminRules.anyProvider')
  if (!providerOptions.includes(provider)) return provider
  return t(`adminRules.providerOptions.${provider}`)
}

function fieldLabel(cond: RuleCondition) {
  const field = fieldOptions(cond).find(item => item.value === cond.field)
  return field ? t(`adminRules.fieldOptions.${field.labelKey}`) : cond.field || '-'
}

function operatorLabel(operator?: string) {
  if (!operator) return ''
  return t(`adminRules.operatorOptions.${operator}`)
}

function formatValue(cond: RuleCondition) {
  if (Array.isArray(cond.values)) return cond.values.join(', ')
  if (typeof cond.value === 'boolean') return cond.value ? t('yes') : t('no')
  return cond.value ?? ''
}
</script>

<template>
  <div>
    <div class="flex flex-col gap-4 mb-6 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <h2 class="text-lg font-semibold">{{ $t('adminRules.title') }}</h2>
        <p class="text-sm text-muted-foreground mt-1">{{ $t('adminRules.subtitle') }}</p>
      </div>
      <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
        <button @click="recomputeAll" :disabled="recomputing" class="border border-border px-4 py-2 rounded-full text-sm font-medium hover:bg-muted transition-colors flex items-center justify-center gap-2 disabled:opacity-50">
          <RefreshCw class="w-4 h-4" :class="recomputing ? 'animate-spin' : ''" /> {{ $t('adminRules.recompute') }}
        </button>
        <button @click="openCreate" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors flex items-center justify-center gap-2">
          <Plus class="w-4 h-4" /> {{ $t('adminRules.createRule') }}
        </button>
      </div>
    </div>

    <div class="rounded-xl border border-border bg-muted/30 p-4 mb-4 text-sm text-muted-foreground">
      <div class="font-medium text-foreground mb-1">{{ $t('adminRules.howItWorks') }}</div>
      <p>{{ $t('adminRules.howItWorksDesc') }}</p>
    </div>

    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{{ error }}</div>
    <div v-if="success" class="mb-4 rounded-lg border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-700">{{ success }}</div>

    <div v-if="loading && rules.length === 0" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <div v-else-if="rules.length === 0" class="border border-dashed border-border rounded-xl py-12 text-center text-muted-foreground">
      {{ $t('adminRules.noRules') }}
    </div>

    <div v-else class="hidden md:block border border-border rounded-xl overflow-x-auto">
      <table class="w-full min-w-[860px] text-sm">
        <thead class="bg-muted/50 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
          <tr>
            <th class="px-4 py-3">{{ $t('adminRules.name') }}</th>
            <th class="px-4 py-3">{{ $t('adminRules.level') }}</th>
            <th class="px-4 py-3">{{ $t('adminRules.conditions') }}</th>
            <th class="px-4 py-3">{{ $t('adminRules.priority') }}</th>
            <th class="px-4 py-3">{{ $t('adminUsers.status') }}</th>
            <th class="px-4 py-3">{{ $t('actions') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="rule in rules" :key="rule.id" class="hover:bg-muted/30 transition-colors">
            <td class="px-4 py-3">
              <div class="font-medium">{{ rule.name }}</div>
              <div class="text-xs text-muted-foreground max-w-48 truncate">{{ rule.description }}</div>
            </td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-foreground/10">L{{ rule.level }}</span>
            </td>
            <td class="px-4 py-3 text-muted-foreground text-xs max-w-96 truncate">{{ conditionSummary(rule) }}</td>
            <td class="px-4 py-3 text-muted-foreground">{{ rule.priority }}</td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium" :class="rule.is_active ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'">
                {{ rule.is_active ? $t('adminProviders.enabled') : $t('adminProviders.disabled') }}
              </span>
            </td>
            <td class="px-4 py-3">
              <div class="flex items-center gap-1">
                <button @click="openEdit(rule)" class="text-xs font-medium px-2 py-1 rounded hover:bg-muted transition-colors flex items-center gap-1">
                  <Pencil class="w-3 h-3" /> {{ $t('edit') }}
                </button>
                <button @click="confirmDelete(rule)" class="text-xs font-medium px-2 py-1 rounded hover:bg-destructive/5 transition-colors text-destructive flex items-center gap-1">
                  <Trash2 class="w-3 h-3" /> {{ $t('delete') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="rules.length > 0" class="md:hidden space-y-3">
      <div v-for="rule in rules" :key="rule.id" class="border border-border rounded-xl p-4 bg-background space-y-3">
        <div>
          <div class="font-medium text-sm break-words">{{ rule.name }}</div>
          <div class="text-xs text-muted-foreground break-words">{{ rule.description }}</div>
        </div>
        <div class="flex flex-wrap gap-2">
          <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-foreground/10">L{{ rule.level }}</span>
          <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground">{{ $t('adminRules.priority') }} {{ rule.priority }}</span>
          <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium" :class="rule.is_active ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'">
            {{ rule.is_active ? $t('adminProviders.enabled') : $t('adminProviders.disabled') }}
          </span>
        </div>
        <div class="text-xs text-muted-foreground break-words"><span class="font-medium text-foreground">{{ $t('adminRules.conditions') }}：</span>{{ conditionSummary(rule) }}</div>
        <div class="grid grid-cols-2 gap-2">
          <button @click="openEdit(rule)" class="text-xs font-medium px-2 py-2 rounded border border-border hover:bg-muted transition-colors flex items-center justify-center gap-1">
            <Pencil class="w-3 h-3" /> {{ $t('edit') }}
          </button>
          <button @click="confirmDelete(rule)" class="text-xs font-medium px-2 py-2 rounded border border-destructive/30 hover:bg-destructive/5 transition-colors text-destructive flex items-center justify-center gap-1">
            <Trash2 class="w-3 h-3" /> {{ $t('delete') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-3xl mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-5">
          <h2 class="text-lg font-semibold">{{ isCreate ? $t('adminRules.createRule') : $t('adminRules.editRule') }}</h2>
          <button @click="showModal = false" class="text-muted-foreground hover:text-foreground"><X class="w-5 h-5" /></button>
        </div>
        <form @submit.prevent="saveRule" class="flex flex-col gap-4">
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminRules.name') }}</label>
            <input v-model="form.name" type="text" required :placeholder="$t('adminRules.namePlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1.5">{{ $t('adminRules.description') }}</label>
            <input v-model="form.description" type="text" :placeholder="$t('adminRules.descriptionPlaceholder')" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ $t('adminRules.level') }}</label>
              <input v-model.number="form.level" type="number" min="0" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
              <p class="text-xs text-muted-foreground mt-1">{{ $t('adminRules.levelHint') }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium mb-1.5">{{ $t('adminRules.priority') }}</label>
              <input v-model.number="form.priority" type="number" min="0" class="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10" />
              <p class="text-xs text-muted-foreground mt-1">{{ $t('adminRules.priorityHint') }}</p>
            </div>
          </div>

          <div class="border border-border rounded-lg p-4">
            <div class="flex items-center justify-between mb-3">
              <label class="text-sm font-medium">{{ $t('adminRules.conditions') }}</label>
              <select v-model="form.operator" class="text-xs px-2 py-1 border border-border rounded-lg">
                <option value="AND">{{ $t('adminRules.conditionAnd') }}</option>
                <option value="OR">{{ $t('adminRules.conditionOr') }}</option>
              </select>
            </div>
            <div v-if="!hasConditions" class="text-xs text-muted-foreground text-center py-3">{{ $t('adminRules.noConditions') }}</div>
            <div v-for="(cond, i) in form.conditions" :key="i" class="rounded-lg border border-border/70 p-3 mb-3 bg-muted/10">
              <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div>
                  <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.conditionType') }}</label>
                  <select v-model="cond.type" @change="onTypeChange(cond)" class="w-full px-2 py-1.5 border border-border rounded-lg text-sm">
                    <option v-for="option in conditionOptions" :key="option.type" :value="option.type">
                      {{ $t(`adminRules.conditionTypes.${option.type}`) }}
                    </option>
                  </select>
                </div>

                <div v-if="conditionOption(cond.type).needsProvider">
                  <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.conditionProvider') }}</label>
                  <input v-model="cond.provider" @change="onProviderChange(cond)" :list="`security-rule-provider-options-${i}`" :placeholder="conditionOption(cond.type).supportsAnyProvider ? $t('adminRules.anyProviderPlaceholder') : $t('adminRules.providerPlaceholder')" class="w-full px-2 py-1.5 border border-border rounded-lg text-sm" />
                  <datalist :id="`security-rule-provider-options-${i}`">
                    <option v-for="p in providerOptions" :key="p" :value="p">
                      {{ $t(`adminRules.providerOptions.${p}`) }}
                    </option>
                  </datalist>
                </div>

                <div v-if="conditionOption(cond.type).needsField">
                  <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.field') }}</label>
                  <input v-model="cond.field" :list="`security-rule-field-options-${i}`" :placeholder="$t('adminRules.fieldPlaceholder')" class="w-full px-2 py-1.5 border border-border rounded-lg text-sm" />
                  <datalist :id="`security-rule-field-options-${i}`">
                    <option v-for="field in fieldOptions(cond)" :key="field.value" :value="field.value">
                      {{ $t(`adminRules.fieldOptions.${field.labelKey}`) }}
                    </option>
                  </datalist>
                </div>

                <div v-if="operatorOptions(cond).length">
                  <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.operator') }}</label>
                  <select v-model="cond.operator" class="w-full px-2 py-1.5 border border-border rounded-lg text-sm">
                    <option v-for="op in operatorOptions(cond)" :key="op" :value="op">{{ $t(`adminRules.operatorOptions.${op}`) }}</option>
                  </select>
                </div>

                <div v-if="conditionOption(cond.type).valueType === 'number'">
                  <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.value') }}</label>
                  <input v-model.number="cond.value" type="number" min="0" class="w-full px-2 py-1.5 border border-border rounded-lg text-sm" />
                </div>

                <div v-else-if="conditionOption(cond.type).valueType === 'string'">
                  <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.value') }}</label>
                  <input v-model="cond.value" type="text" class="w-full px-2 py-1.5 border border-border rounded-lg text-sm" />
                </div>

                <div v-else-if="conditionOption(cond.type).valueType === 'domains'">
                  <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.domains') }}</label>
                  <input :value="valuesText(cond)" @input="handleValuesInput(cond, $event)" type="text" :placeholder="$t('adminRules.domainsPlaceholder')" class="w-full px-2 py-1.5 border border-border rounded-lg text-sm" />
                </div>

                <div v-else-if="conditionOption(cond.type).valueType === 'bool'">
                  <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.value') }}</label>
                  <select v-model="cond.value" class="w-full px-2 py-1.5 border border-border rounded-lg text-sm">
                    <option :value="true">{{ $t('yes') }}</option>
                    <option :value="false">{{ $t('no') }}</option>
                  </select>
                </div>
              </div>

              <div class="flex flex-col gap-2 mt-3 sm:flex-row sm:items-center sm:justify-between">
                <p class="text-xs text-muted-foreground break-words">{{ conditionLabel(cond) }}</p>
                <button type="button" @click="removeCondition(i)" class="text-destructive hover:text-destructive/80 flex items-center gap-1 text-xs shrink-0">
                  <MinusCircle class="w-4 h-4" /> {{ $t('adminRules.removeCondition') }}
                </button>
              </div>
            </div>
            <button type="button" @click="addCondition" class="text-xs text-foreground/70 hover:text-foreground mt-1 flex items-center gap-1">
              <Plus class="w-3 h-3" /> {{ $t('adminRules.addCondition') }}
            </button>
          </div>

          <label class="flex items-center gap-2 text-sm font-medium cursor-pointer">
            <input type="checkbox" v-model="form.is_active" class="rounded border-border" /> {{ $t('adminRules.active') }}
          </label>

          <div class="flex flex-col-reverse gap-2 mt-2 sm:flex-row sm:justify-end">
            <button type="button" @click="showModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors w-full sm:w-auto">{{ $t('cancel') }}</button>
            <button type="submit" :disabled="saving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2 w-full sm:w-auto">
              <Loader2 v-if="saving" class="w-4 h-4 animate-spin" /> {{ isCreate ? $t('adminRules.createRule') : $t('save') }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm px-4 py-4" @click.self="showDeleteModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm p-6 max-h-[90vh] overflow-y-auto">
        <h2 class="text-lg font-semibold mb-2">{{ $t('adminRules.deleteRule') }}</h2>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('adminRules.deleteConfirm') }}</p>
        <div class="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button @click="showDeleteModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors w-full sm:w-auto">{{ $t('cancel') }}</button>
          <button @click="deleteRule" :disabled="deleting" class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2 w-full sm:w-auto">
            <Loader2 v-if="deleting" class="w-4 h-4 animate-spin" /> {{ $t('delete') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>