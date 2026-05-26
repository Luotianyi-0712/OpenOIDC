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
  is_satisfied: boolean
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

onMounted(() => {
  fetchSecurity()
  fetchDeveloperAccess()
})

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

async function fetchDeveloperAccess() {
  if (!auth.user) {
    await auth.fetchUser()
  }
  await auth.fetchDeveloperStatus()
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

const developerMinLevel = computed(() => auth.developerStatus?.min_trust_level ?? auth.developerMinTrustLevel ?? 1)
const developerCurrentLevel = computed(() => auth.developerStatus?.current_trust_level ?? info.value?.level ?? auth.user?.security_level ?? 0)
const developerLevelGap = computed(() => Math.max(0, developerMinLevel.value - developerCurrentLevel.value))
const developerLevelMet = computed(() => developerCurrentLevel.value >= developerMinLevel.value)
const developerNeedsEmailVerification = computed(() => auth.developerStatus?.requires_email_verify ?? !auth.user?.email_verified)
const developerAccessGranted = computed(() => auth.developerStatus?.can_create ?? (developerLevelMet.value && !developerNeedsEmailVerification.value))
const developerAccessHint = computed(() => {
  if (developerAccessGranted.value) return t('security.developerAccessGranted')
  if (!developerLevelMet.value) return t('security.developerAccessNeedLevel', { count: developerLevelGap.value })
  if (developerNeedsEmailVerification.value) return t('security.developerAccessNeedEmail')
  return t('security.developerAccessNotReady')
})

function providerLabel(provider: string): string {
  const key = `adminRules.providerOptions.${provider}`
  const translated = t(key)
  return translated !== key ? translated : provider
}

function conditionState(cond: MissingCondition): 'completed' | 'partial' | 'incomplete' {
  if (cond.is_satisfied) return 'completed'
  if (cond.is_bound) return 'partial'
  return 'incomplete'
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

      <!-- Developer Access -->
      <div class="border border-border rounded-xl p-6 mb-6">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h3 class="text-sm font-medium mb-1">{{ $t('security.developerAccess') }}</h3>
            <p class="text-xs text-muted-foreground">{{ $t('security.developerAccessDesc') }}</p>
          </div>
          <span class="text-xs font-medium px-2.5 py-1 rounded-full whitespace-nowrap" :class="developerAccessGranted ? 'bg-green-100 text-green-700' : 'bg-muted text-muted-foreground'">
            {{ developerAccessGranted ? $t('security.developerAccessMet') : $t('security.developerAccessUnmet') }}
          </span>
        </div>
        <div class="grid gap-3 sm:grid-cols-3 mt-5">
          <div class="rounded-lg bg-muted/40 px-4 py-3">
            <div class="text-xs text-muted-foreground mb-1">{{ $t('security.developerCurrentLevel') }}</div>
            <div class="text-lg font-semibold tabular-nums">L{{ developerCurrentLevel }}</div>
          </div>
          <div class="rounded-lg bg-muted/40 px-4 py-3">
            <div class="text-xs text-muted-foreground mb-1">{{ $t('security.developerRequiredLevel') }}</div>
            <div class="text-lg font-semibold tabular-nums">L{{ developerMinLevel }}</div>
          </div>
          <div class="rounded-lg bg-muted/40 px-4 py-3">
            <div class="text-xs text-muted-foreground mb-1">{{ $t('security.emailVerified') }}</div>
            <div class="text-sm font-medium" :class="developerNeedsEmailVerification ? 'text-muted-foreground' : 'text-success'">
              {{ developerNeedsEmailVerification ? $t('security.incomplete') : $t('security.completed') }}
            </div>
          </div>
        </div>
        <p class="mt-4 text-sm" :class="developerAccessGranted ? 'text-success' : 'text-muted-foreground'">
          {{ developerAccessHint }}
        </p>
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
          <div
            v-for="cond in info.next_level.missing"
            :key="cond.provider"
            class="flex items-center gap-3 px-4 py-3 rounded-lg"
            :class="{
              'bg-green-50': conditionState(cond) === 'completed',
              'bg-amber-50': conditionState(cond) === 'partial',
              'bg-muted/50': conditionState(cond) === 'incomplete',
            }"
          >
            <div
              class="w-6 h-6 rounded-full flex items-center justify-center shrink-0"
              :class="{
                'bg-green-100 text-green-600': conditionState(cond) === 'completed',
                'bg-amber-100 text-amber-600': conditionState(cond) === 'partial',
                'bg-muted text-muted-foreground': conditionState(cond) === 'incomplete',
              }"
            >
              <Check v-if="conditionState(cond) !== 'incomplete'" class="w-3.5 h-3.5" />
              <X v-else class="w-3.5 h-3.5" />
            </div>
            <div class="flex-1">
              <div
                class="text-sm font-medium"
                :class="{
                  'text-green-700': conditionState(cond) === 'completed',
                  'text-amber-700': conditionState(cond) === 'partial',
                  'text-foreground': conditionState(cond) === 'incomplete',
                }"
              >
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
            <span
              class="text-xs font-medium px-2 py-0.5 rounded-full"
              :class="{
                'bg-green-100 text-green-700': conditionState(cond) === 'completed',
                'bg-amber-100 text-amber-700': conditionState(cond) === 'partial',
                'bg-muted text-muted-foreground': conditionState(cond) === 'incomplete',
              }"
            >
              {{ conditionState(cond) === 'completed' ? $t('security.completed') : conditionState(cond) === 'partial' ? $t('security.partial') : $t('security.incomplete') }}
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
    </template>
  </div>
</template>
