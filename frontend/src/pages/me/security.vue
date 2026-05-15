<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { Loader2, Shield, ShieldCheck, ShieldAlert, Check, X, ArrowUp, Link2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface MissingCondition {
  provider: string
  min_binding_days: number
  is_bound: boolean
  bound_days: number
}

interface NextLevel {
  level: number
  rule_name: string
  missing: MissingCondition[]
}

interface Binding {
  provider: string
  bound_at: string
}

interface LevelInfo {
  level: number
  max_level: number
  bindings: Binding[]
  next_level?: NextLevel
}

const auth = useAuthStore()
const info = ref<LevelInfo | null>(null)
const loading = ref(true)
const error = ref('')

onMounted(fetchSecurity)

async function fetchSecurity() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<LevelInfo>('/me/security-level')
    info.value = res.data || null
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

const maxLevel = computed(() => info.value?.max_level || 1)
const percentage = computed(() => {
  if (!info.value) return 0
  return Math.min(100, Math.round((info.value.level / maxLevel.value) * 100))
})

const levelColor = computed(() => {
  if (!info.value) return 'text-muted-foreground'
  const p = percentage.value
  if (p >= 80) return 'text-success'
  if (p >= 40) return 'text-brand'
  return 'text-destructive'
})

const levelBg = computed(() => {
  if (!info.value) return 'bg-muted'
  const p = percentage.value
  if (p >= 80) return 'bg-success/10'
  if (p >= 40) return 'bg-brand/10'
  return 'bg-destructive/10'
})

const ShieldIcon = computed(() => {
  if (!info.value) return Shield
  const p = percentage.value
  if (p >= 80) return ShieldCheck
  if (p >= 40) return Shield
  return ShieldAlert
})

const levelLabel = computed(() => {
  if (!info.value) return ''
  const p = percentage.value
  if (p >= 80) return t('security.strong')
  if (p >= 40) return t('security.moderate')
  return t('security.weak')
})

function providerLabel(provider: string): string {
  const key = `adminRules.providerOptions.${provider}`
  const translated = t(key)
  return translated !== key ? translated : provider
}
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-lg font-semibold">{{ $t('security.title') }}</h2>
      <p class="text-sm text-muted-foreground mt-0.5">{{ $t('security.desc') }}</p>
    </div>

    <div v-if="loading" class="flex items-center gap-2 text-sm text-muted-foreground py-12 justify-center">
      <Loader2 class="w-4 h-4 animate-spin" /> {{ $t('security.loadingSecurity') }}
    </div>

    <div v-else-if="error" class="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <template v-else-if="info">
      <!-- Level Badge Card -->
      <div class="border border-border rounded-xl p-8 mb-6">
        <div class="flex items-center gap-6">
          <div class="w-20 h-20 rounded-2xl flex items-center justify-center shrink-0" :class="levelBg">
            <component :is="ShieldIcon" class="w-10 h-10" :class="levelColor" />
          </div>
          <div class="flex-1">
            <div class="flex items-baseline gap-3 mb-2">
              <span class="text-4xl font-bold tabular-nums" :class="levelColor">{{ info.level }}</span>
              <span class="text-lg text-muted-foreground font-medium">/ {{ maxLevel }}</span>
            </div>
            <div class="text-sm font-medium" :class="levelColor">{{ levelLabel }}</div>
            <div class="mt-3 w-full h-2 rounded-full bg-muted overflow-hidden">
              <div class="h-full rounded-full transition-all duration-500" :class="{ 'bg-success': percentage >= 80, 'bg-brand': percentage >= 40 && percentage < 80, 'bg-destructive': percentage < 40 }" :style="{ width: percentage + '%' }" />
            </div>
          </div>
        </div>
      </div>

      <!-- Current Bindings -->
      <div v-if="info.bindings && info.bindings.length" class="border border-border rounded-xl p-6 mb-6">
        <h3 class="text-sm font-medium mb-3 flex items-center gap-2">
          <Link2 class="w-4 h-4 text-muted-foreground" /> {{ $t('security.currentBindings') }}
        </h3>
        <div class="flex flex-wrap gap-2">
          <span v-for="b in info.bindings" :key="b.provider" class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-green-50 text-green-700 text-xs font-medium">
            <Check class="w-3 h-3" /> {{ providerLabel(b.provider) }}
          </span>
        </div>
      </div>

      <!-- Next Level Requirements -->
      <div v-if="info.next_level" class="border border-border rounded-xl p-6 mb-6">
        <div class="flex items-center gap-2 mb-3">
          <ArrowUp class="w-4 h-4 text-brand" />
          <h3 class="text-sm font-medium">{{ $t('security.nextLevelTitle', { level: info.next_level.level }) }}</h3>
        </div>
        <p class="text-xs text-muted-foreground mb-4">{{ $t('security.nextLevelDesc', { name: info.next_level.rule_name }) }}</p>
        <div class="space-y-2">
          <div v-for="cond in info.next_level.missing" :key="cond.provider" class="flex items-center gap-3 px-4 py-3 rounded-lg" :class="cond.is_bound ? 'bg-green-50' : 'bg-muted/50'">
            <div class="w-6 h-6 rounded-full flex items-center justify-center shrink-0" :class="cond.is_bound ? 'bg-green-100 text-green-600' : 'bg-muted text-muted-foreground'">
              <Check v-if="cond.is_bound" class="w-3.5 h-3.5" />
              <X v-else class="w-3.5 h-3.5" />
            </div>
            <div class="flex-1">
              <div class="text-sm font-medium" :class="cond.is_bound ? 'text-green-700' : 'text-foreground'">
                {{ $t('security.bindProvider', { provider: providerLabel(cond.provider) }) }}
              </div>
              <div v-if="cond.min_binding_days > 0" class="text-xs text-muted-foreground">
                <template v-if="cond.is_bound">
                  {{ $t('security.boundDays', { current: cond.bound_days, required: cond.min_binding_days }) }}
                </template>
                <template v-else>
                  {{ $t('security.requireDays', { days: cond.min_binding_days }) }}
                </template>
              </div>
            </div>
            <span class="text-xs font-medium px-2 py-0.5 rounded-full" :class="cond.is_bound && (cond.min_binding_days <= 0 || cond.bound_days >= cond.min_binding_days) ? 'bg-green-100 text-green-700' : 'bg-muted text-muted-foreground'">
              {{ cond.is_bound && (cond.min_binding_days <= 0 || cond.bound_days >= cond.min_binding_days) ? $t('security.completed') : $t('security.incomplete') }}
            </span>
          </div>
        </div>
        <router-link to="/me/bindings" class="inline-flex items-center gap-1.5 mt-4 text-sm text-brand hover:underline">
          <Link2 class="w-3.5 h-3.5" /> {{ $t('security.goBindings') }}
        </router-link>
      </div>

      <!-- Already at max or no rules -->
      <div v-else-if="info.level >= maxLevel" class="border border-border rounded-xl p-6 mb-6 text-center">
        <ShieldCheck class="w-8 h-8 text-success mx-auto mb-2" />
        <p class="text-sm font-medium">{{ $t('security.maxReached') }}</p>
      </div>

      <!-- Account Info -->
      <div class="border border-border rounded-xl p-6">
        <h3 class="text-sm font-medium mb-3">{{ $t('security.accountDetails') }}</h3>
        <div class="space-y-2.5 text-sm">
          <div class="flex justify-between py-2 border-b border-border">
            <span class="text-muted-foreground">{{ $t('security.emailVerified') }}</span>
            <span :class="auth.user?.email_verified ? 'text-success' : 'text-muted-foreground'">
              {{ auth.user?.email_verified ? $t('yes') : $t('no') }}
            </span>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <span class="text-muted-foreground">{{ $t('security.accountStatus') }}</span>
            <span class="font-medium">{{ auth.user?.status }}</span>
          </div>
          <div class="flex justify-between py-2 border-b border-border">
            <span class="text-muted-foreground">{{ $t('security.securityLevel') }}</span>
            <span class="font-medium" :class="levelColor">{{ info.level }} / {{ maxLevel }}</span>
          </div>
          <div class="flex justify-between py-2">
            <span class="text-muted-foreground">{{ $t('security.accountCreated') }}</span>
            <span>{{ auth.user?.created_at ? new Date(auth.user.created_at).toLocaleDateString('zh-CN') : '-' }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
