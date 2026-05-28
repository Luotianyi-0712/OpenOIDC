<script setup lang="ts">
import { computed } from 'vue'
import { MinusCircle, Plus } from 'lucide-vue-next'
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

type ConditionItem = {
  condition?: RuleCondition
  group?: ConditionGroup
}

type ConditionGroup = {
  operator: RuleOperator
  items: ConditionItem[]
}

type ConditionOption = {
  type: ConditionType
  valueType: ValueType
  operatorType?: 'number' | 'string' | 'bool'
  needsProvider?: boolean
  needsField?: boolean
  supportsAnyProvider?: boolean
}

type FieldOption = {
  value: string
  labelKey: string
  type: 'number' | 'string' | 'bool' | 'time'
}

const props = defineProps<{
  item: ConditionItem
  index: number
  depth: number
  conditionOptions: ConditionOption[]
  providerOptions: string[]
}>()

const emit = defineEmits<{
  remove: []
  update: [item: ConditionItem]
}>()

const isGroup = computed(() => !!props.item.group)

function toggleType() {
  if (isGroup.value) {
    // Convert group to condition
    emit('update', { condition: defaultCondition() })
  } else {
    // Convert condition to group
    emit('update', {
      group: {
        operator: 'AND',
        items: [{ condition: defaultCondition() }],
      },
    })
  }
}

function defaultCondition(): RuleCondition {
  return {
    type: 'provider_bound',
    provider: '',
    operator: 'eq',
    value: '',
  }
}

function addItemToGroup() {
  if (props.item.group) {
    const newGroup = { ...props.item.group }
    newGroup.items = [...newGroup.items, { condition: defaultCondition() }]
    emit('update', { group: newGroup })
  }
}

function addGroupToGroup() {
  if (props.item.group) {
    const newGroup = { ...props.item.group }
    newGroup.items = [
      ...newGroup.items,
      {
        group: {
          operator: 'AND',
          items: [{ condition: defaultCondition() }],
        },
      },
    ]
    emit('update', { group: newGroup })
  }
}

function removeGroupItem(index: number) {
  if (props.item.group) {
    const newGroup = { ...props.item.group }
    newGroup.items = newGroup.items.filter((_, i) => i !== index)
    emit('update', { group: newGroup })
  }
}

function updateGroupItem(index: number, newItem: ConditionItem) {
  if (props.item.group) {
    const newGroup = { ...props.item.group }
    newGroup.items = [...newGroup.items]
    newGroup.items[index] = newItem
    emit('update', { group: newGroup })
  }
}

function updateGroupOperator(op: RuleOperator) {
  if (props.item.group) {
    emit('update', {
      group: {
        ...props.item.group,
        operator: op,
      },
    })
  }
}

function updateCondition(updates: Partial<RuleCondition>) {
  if (props.item.condition) {
    emit('update', {
      condition: {
        ...props.item.condition,
        ...updates,
      },
    })
  }
}

function conditionOption(type?: ConditionType): ConditionOption {
  return props.conditionOptions.find(o => o.type === type) || props.conditionOptions[0]
}

function onTypeChange(newType: ConditionType) {
  const option = conditionOption(newType)
  updateCondition({
    type: newType,
    provider: option.needsProvider ? '' : undefined,
    field: option.needsField ? '' : undefined,
    operator: option.operatorType === 'number' ? 'gte' : option.operatorType === 'bool' ? 'eq' : 'eq',
    value: option.valueType === 'number' ? 0 : option.valueType === 'bool' ? true : '',
    values: option.valueType === 'domains' ? [] : undefined,
  })
}

function onProviderChange() {
  if (props.item.condition?.type === 'provider_raw_number' ||
      props.item.condition?.type === 'provider_raw_string' ||
      props.item.condition?.type === 'provider_raw_bool') {
    updateCondition({ field: '' })
  }
}

function operatorOptions(cond?: RuleCondition): string[] {
  const option = conditionOption(cond?.type)
  if (option.operatorType === 'number') return ['eq', 'ne', 'gt', 'gte', 'lt', 'lte']
  if (option.operatorType === 'string') return ['eq', 'ne', 'contains', 'starts_with', 'ends_with']
  if (option.operatorType === 'bool') return ['eq', 'ne']
  return []
}

function fieldOptions(cond?: RuleCondition): FieldOption[] {
  const provider = cond?.provider?.toLowerCase()
  if (!provider) return []

  const commonFields: FieldOption[] = [
    { value: 'id', labelKey: 'id', type: 'string' },
    { value: 'login', labelKey: 'login', type: 'string' },
    { value: 'name', labelKey: 'name', type: 'string' },
    { value: 'email', labelKey: 'email', type: 'string' },
    { value: 'avatar_url', labelKey: 'avatar_url', type: 'string' },
    { value: 'created_at', labelKey: 'created_at', type: 'time' },
  ]

  if (provider === 'github') {
    return [
      ...commonFields,
      { value: 'followers', labelKey: 'followers', type: 'number' },
      { value: 'following', labelKey: 'following', type: 'number' },
      { value: 'public_repos', labelKey: 'public_repos', type: 'number' },
      { value: 'public_gists', labelKey: 'public_gists', type: 'number' },
    ]
  }

  return commonFields
}

function valuesText(cond?: RuleCondition): string {
  return cond?.values?.join(', ') || ''
}

function handleValuesInput(event: Event) {
  const input = event.target as HTMLInputElement
  const values = input.value.split(',').map(v => v.trim()).filter(Boolean)
  updateCondition({ values })
}
</script>

<template>
  <div
    :class="[
      'rounded-lg border p-3 mb-3',
      isGroup ? 'border-blue-300 bg-blue-50/30' : 'border-border/70 bg-muted/10'
    ]"
    :style="{ marginLeft: `${depth * 12}px` }"
  >
    <!-- Group Header -->
    <div v-if="isGroup && item.group" class="mb-3">
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center gap-2">
          <span class="text-xs font-medium text-blue-700">{{ $t('adminRules.conditionGroup') }}</span>
          <select
            :value="item.group.operator"
            @change="updateGroupOperator(($event.target as HTMLSelectElement).value as RuleOperator)"
            class="text-xs px-2 py-1 border border-blue-300 rounded-lg bg-white"
          >
            <option value="AND">{{ $t('adminRules.conditionAnd') }}</option>
            <option value="OR">{{ $t('adminRules.conditionOr') }}</option>
          </select>
        </div>
        <div class="flex items-center gap-1">
          <button
            type="button"
            @click="toggleType"
            class="text-xs px-2 py-1 text-blue-600 hover:bg-blue-100 rounded transition-colors"
          >
            {{ $t('adminRules.convertToCondition') }}
          </button>
          <button
            type="button"
            @click="emit('remove')"
            class="text-destructive hover:bg-destructive/10 p-1 rounded transition-colors"
          >
            <MinusCircle class="w-4 h-4" />
          </button>
        </div>
      </div>

      <!-- Nested Items -->
      <div class="space-y-2">
        <ConditionItemEditor
          v-for="(subItem, i) in item.group.items"
          :key="i"
          :item="subItem"
          :index="i"
          :depth="depth + 1"
          :condition-options="conditionOptions"
          :provider-options="providerOptions"
          @remove="removeGroupItem(i)"
          @update="updateGroupItem(i, $event)"
        />
      </div>

      <!-- Add Buttons -->
      <div class="flex gap-2 mt-2">
        <button
          type="button"
          @click="addItemToGroup"
          class="text-xs px-3 py-1.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-1"
        >
          <Plus class="w-3 h-3" /> {{ $t('adminRules.addCondition') }}
        </button>
        <button
          type="button"
          @click="addGroupToGroup"
          class="text-xs px-3 py-1.5 border border-blue-300 text-blue-700 rounded-lg hover:bg-blue-50 transition-colors flex items-center gap-1"
        >
          <Plus class="w-3 h-3" /> {{ $t('adminRules.addGroup') }}
        </button>
      </div>
    </div>

    <!-- Single Condition -->
    <div v-else-if="item.condition">
      <div class="flex items-center justify-between mb-2">
        <span class="text-xs font-medium text-muted-foreground">{{ $t('adminRules.condition') }}</span>
        <div class="flex items-center gap-1">
          <button
            type="button"
            @click="toggleType"
            class="text-xs px-2 py-1 text-blue-600 hover:bg-blue-100 rounded transition-colors"
          >
            {{ $t('adminRules.convertToGroup') }}
          </button>
          <button
            type="button"
            @click="emit('remove')"
            class="text-destructive hover:bg-destructive/10 p-1 rounded transition-colors"
          >
            <MinusCircle class="w-4 h-4" />
          </button>
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
          <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.conditionType') }}</label>
          <select
            :value="item.condition.type"
            @change="onTypeChange(($event.target as HTMLSelectElement).value as ConditionType)"
            class="w-full px-2 py-1.5 border border-border rounded-lg text-sm"
          >
            <option v-for="option in conditionOptions" :key="option.type" :value="option.type">
              {{ $t(`adminRules.conditionTypes.${option.type}`) }}
            </option>
          </select>
        </div>

        <div v-if="conditionOption(item.condition.type).needsProvider">
          <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.conditionProvider') }}</label>
          <input
            :value="item.condition.provider"
            @input="updateCondition({ provider: ($event.target as HTMLInputElement).value })"
            @change="onProviderChange"
            :list="`provider-options-${index}-${depth}`"
            :placeholder="conditionOption(item.condition.type).supportsAnyProvider ? $t('adminRules.anyProviderPlaceholder') : $t('adminRules.providerPlaceholder')"
            class="w-full px-2 py-1.5 border border-border rounded-lg text-sm"
          />
          <datalist :id="`provider-options-${index}-${depth}`">
            <option v-for="p in providerOptions" :key="p" :value="p">
              {{ $t(`adminRules.providerOptions.${p}`) }}
            </option>
          </datalist>
        </div>

        <div v-if="conditionOption(item.condition.type).needsField">
          <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.field') }}</label>
          <input
            :value="item.condition.field"
            @input="updateCondition({ field: ($event.target as HTMLInputElement).value })"
            :list="`field-options-${index}-${depth}`"
            :placeholder="$t('adminRules.fieldPlaceholder')"
            class="w-full px-2 py-1.5 border border-border rounded-lg text-sm"
          />
          <datalist :id="`field-options-${index}-${depth}`">
            <option v-for="field in fieldOptions(item.condition)" :key="field.value" :value="field.value">
              {{ $t(`adminRules.fieldOptions.${field.labelKey}`) }}
            </option>
          </datalist>
        </div>

        <div v-if="operatorOptions(item.condition).length">
          <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.operator') }}</label>
          <select
            :value="item.condition.operator"
            @change="updateCondition({ operator: ($event.target as HTMLSelectElement).value })"
            class="w-full px-2 py-1.5 border border-border rounded-lg text-sm"
          >
            <option v-for="op in operatorOptions(item.condition)" :key="op" :value="op">
              {{ $t(`adminRules.operatorOptions.${op}`) }}
            </option>
          </select>
        </div>

        <div v-if="conditionOption(item.condition.type).valueType === 'number'">
          <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.value') }}</label>
          <input
            :value="item.condition.value"
            @input="updateCondition({ value: Number(($event.target as HTMLInputElement).value) })"
            type="number"
            min="0"
            class="w-full px-2 py-1.5 border border-border rounded-lg text-sm"
          />
        </div>

        <div v-else-if="conditionOption(item.condition.type).valueType === 'string'">
          <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.value') }}</label>
          <input
            :value="item.condition.value"
            @input="updateCondition({ value: ($event.target as HTMLInputElement).value })"
            type="text"
            class="w-full px-2 py-1.5 border border-border rounded-lg text-sm"
          />
        </div>

        <div v-else-if="conditionOption(item.condition.type).valueType === 'domains'">
          <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.domains') }}</label>
          <input
            :value="valuesText(item.condition)"
            @input="handleValuesInput"
            type="text"
            :placeholder="$t('adminRules.domainsPlaceholder')"
            class="w-full px-2 py-1.5 border border-border rounded-lg text-sm"
          />
        </div>

        <div v-else-if="conditionOption(item.condition.type).valueType === 'bool'">
          <label class="block text-xs text-muted-foreground mb-1">{{ $t('adminRules.value') }}</label>
          <select
            :value="item.condition.value"
            @change="updateCondition({ value: ($event.target as HTMLSelectElement).value === 'true' })"
            class="w-full px-2 py-1.5 border border-border rounded-lg text-sm"
          >
            <option :value="true">{{ $t('yes') }}</option>
            <option :value="false">{{ $t('no') }}</option>
          </select>
        </div>
      </div>
    </div>
  </div>
</template>
