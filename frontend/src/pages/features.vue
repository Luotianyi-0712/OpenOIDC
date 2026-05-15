<script setup lang="ts">
import {
  KeyRound,
  Users,
  ShieldCheck,
  ScanEye,
  MonitorSmartphone,
  ScrollText,
} from 'lucide-vue-next'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const featureCards = computed(() => [
  { icon: KeyRound, title: t('features.oidcTitle'), desc: t('features.oidcDesc') },
  { icon: Users, title: t('features.socialTitle'), desc: t('features.socialDesc') },
  { icon: ShieldCheck, title: t('features.securityTitle'), desc: t('features.securityDesc') },
  { icon: ScanEye, title: t('features.accessTitle'), desc: t('features.accessDesc') },
  { icon: MonitorSmartphone, title: t('features.sessionTitle'), desc: t('features.sessionDesc') },
  { icon: ScrollText, title: t('features.auditTitle'), desc: t('features.auditDesc') },
])
</script>

<template>
  <div>
    <!-- Hero -->
    <section class="max-w-[1200px] mx-auto px-6 md:px-10 pt-36 pb-20 text-center">
      <h1 class="text-[clamp(2.5rem,6vw,4.5rem)] font-bold tracking-tight leading-[1.05] text-foreground mx-auto max-w-3xl">
        {{ $t('features.title') }}
      </h1>
      <p class="text-muted-foreground text-lg max-w-2xl mx-auto mt-5 leading-relaxed">
        {{ $t('features.subtitle') }}
      </p>
    </section>

    <!-- Feature cards grid -->
    <section class="max-w-[1200px] mx-auto px-6 md:px-10 pb-32">
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        <div
          v-for="f in featureCards"
          :key="f.title"
          class="rounded-xl border border-border bg-muted/40 p-8 flex flex-col gap-5"
        >
          <div class="w-11 h-11 rounded-lg border border-border bg-background flex items-center justify-center text-foreground">
            <component :is="f.icon" class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-foreground">{{ f.title }}</h3>
            <p class="text-sm text-muted-foreground mt-2 leading-relaxed">{{ f.desc }}</p>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
