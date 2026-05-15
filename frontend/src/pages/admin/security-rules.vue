<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { Plus, Pencil, Trash2, Loader2, RefreshCw, X, MinusCircle } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface RuleCondition {
  provider: string
  min_binding_days: number
}

interface SecurityRule {
  id: string
  name: string
  description: string
  level: number
  priority: number
  conditions: {
    operator: 'AND' | 'OR'
    rules: RuleCondition[]
  }
  is_active: boolean
  created_at: string
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
  operator: 'AND' as 'AND' | 'OR',
  conditions: [] as RuleCondition[],
  is_active: true,
})

const showDeleteModal = ref(false)
const deletingRule = ref<SecurityRule | null>(null)
const deleting = ref(false)

const recomputing = ref(false)

const providerOptions = [
  'github', 'google', 'gitlab', 'gitee', 'discord',
  'telegram', 'microsoft', 'apple', 'qq', 'wechat', 'phone',
]

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
  form.value = { name: '', description: '', level: 1, priority: 0, operator: 'AND', conditions: [{ provider: 'github', min_binding_days: 0 }], is_active: true }
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
    conditions: rule.conditions?.rules?.length ? [...rule.conditions.rules] : [{ provider: 'github', min_binding_days: 0 }],
    is_active: rule.is_active,
  }
  showModal.value = true
}

function addCondition() {
  form.value.conditions.push({ provider: 'github', min_binding_days: 0 })
}

function removeCondition(index: number) {
  form.value.conditions.splice(index, 1)
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
        rules: form.value.conditions,
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
  const op = rule.conditions.operator === 'OR' ? ` ${t('adminRules.conditionOr')} ` : ` ${t('adminRules.conditionAnd')} `
  return rule.conditions.rules.map(c => {
    const pLabel = t(`adminRules.providerOptions.${c.provider}`)
    return c.min_binding_days > 0 ? `${pLabel} ≥${c.min_binding_days}${t('adminRules.days')}` : pLabel
  }).join(op)
}
</script>

<template>
  <div>
    <div class="flex items-start justify-between gap-4 mb-6">
      <div>
        <h2 class="text-lg font-semibold">{{ $t('adminRules.title') }}</h2>
        <p class="text-sm text-muted-foreground mt-1">{{ $t('adminRules.subtitle') }}</p>
      </div>
      <div class="flex items-center gap-2">
        <button @click="recomputeAll" :disabled="recomputing" class="border border-border px-4 py-2 rounded-full text-sm font-medium hover:bg-muted transition-colors flex items-center gap-2 disabled:opacity-50">
          <RefreshCw class="w-4 h-4" :class="recomputing ? 'animate-spin' : ''" /> {{ $t('adminRules.recompute') }}
        </button>
        <button @click="openCreate" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors flex items-center gap-2">
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

    <div v-else class="border border-border rounded-xl overflow-hidden">
      <table class="w-full text-sm">
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
            <td class="px-4 py-3 text-muted-foreground text-xs max-w-64 truncate">{{ conditionSummary(rule) }}</td>
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

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4 p-6 max-h-[90vh] overflow-y-auto">
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
          <div class="grid grid-cols-2 gap-4">
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

          <!-- Conditions Builder -->
          <div class="border border-border rounded-lg p-4">
            <div class="flex items-center justify-between mb-3">
              <label class="text-sm font-medium">{{ $t('adminRules.conditions') }}</label>
              <select v-model="form.operator" class="text-xs px-2 py-1 border border-border rounded-lg">
                <option value="AND">{{ $t('adminRules.conditionAnd') }}</option>
                <option value="OR">{{ $t('adminRules.conditionOr') }}</option>
              </select>
            </div>
            <div v-if="form.conditions.length === 0" class="text-xs text-muted-foreground text-center py-3">{{ $t('adminRules.noConditions') }}</div>
            <div v-for="(cond, i) in form.conditions" :key="i" class="flex items-center gap-2 mb-2">
              <select v-model="cond.provider" class="flex-1 px-2 py-1.5 border border-border rounded-lg text-sm">
                <option v-for="p in providerOptions" :key="p" :value="p">{{ $t(`adminRules.providerOptions.${p}`) }}</option>
              </select>
              <div class="flex items-center gap-1">
                <span class="text-xs text-muted-foreground whitespace-nowrap">≥</span>
                <input v-model.number="cond.min_binding_days" type="number" min="0" class="w-16 px-2 py-1.5 border border-border rounded-lg text-sm" />
                <span class="text-xs text-muted-foreground whitespace-nowrap">{{ $t('adminRules.days') }}</span>
              </div>
              <button type="button" @click="removeCondition(i)" class="text-destructive hover:text-destructive/80">
                <MinusCircle class="w-4 h-4" />
              </button>
            </div>
            <button type="button" @click="addCondition" class="text-xs text-foreground/70 hover:text-foreground mt-1 flex items-center gap-1">
              <Plus class="w-3 h-3" /> {{ $t('adminRules.addCondition') }}
            </button>
          </div>

          <label class="flex items-center gap-2 text-sm font-medium cursor-pointer">
            <input type="checkbox" v-model="form.is_active" class="rounded border-border" /> {{ $t('adminRules.active') }}
          </label>

          <div class="flex justify-end gap-2 mt-2">
            <button type="button" @click="showModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
            <button type="submit" :disabled="saving" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2">
              <Loader2 v-if="saving" class="w-4 h-4 animate-spin" /> {{ isCreate ? $t('adminRules.createRule') : $t('save') }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Modal -->
    <div v-if="showDeleteModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showDeleteModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
        <h2 class="text-lg font-semibold mb-2">{{ $t('adminRules.deleteRule') }}</h2>
        <p class="text-sm text-muted-foreground mb-5">{{ $t('adminRules.deleteConfirm') }}</p>
        <div class="flex justify-end gap-2">
          <button @click="showDeleteModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
          <button @click="deleteRule" :disabled="deleting" class="bg-destructive text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-destructive/90 transition-colors disabled:opacity-50 flex items-center gap-2">
            <Loader2 v-if="deleting" class="w-4 h-4 animate-spin" /> {{ $t('delete') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
