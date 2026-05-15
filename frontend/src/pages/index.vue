<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import {
  ArrowRight,
  KeyRound,
  Users,
  ShieldCheck,
  ScanEye,
  MonitorSmartphone,
  ScrollText,
  Terminal,
  Layers,
  Zap,
  Fingerprint,
} from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const stats = computed(() => [
  { value: '10+', label: t('landing.stats.providers') },
  { value: '5', label: t('landing.stats.levels') },
  { value: 'OAuth2', label: t('landing.stats.protocol') },
  { value: '<50ms', label: t('landing.stats.latency') },
])

const features = computed(() => [
  { icon: KeyRound, title: t('landing.features.oidc'), desc: t('landing.features.oidcDesc') },
  { icon: Users, title: t('landing.features.social'), desc: t('landing.features.socialDesc') },
  { icon: ShieldCheck, title: t('landing.features.security'), desc: t('landing.features.securityDesc') },
  { icon: ScanEye, title: t('landing.features.access'), desc: t('landing.features.accessDesc') },
  { icon: MonitorSmartphone, title: t('landing.features.session'), desc: t('landing.features.sessionDesc') },
  { icon: ScrollText, title: t('landing.features.audit'), desc: t('landing.features.auditDesc') },
])

const steps = computed(() => [
  {
    icon: Terminal,
    label: t('landing.steps.step1'),
    title: t('landing.steps.step1Title'),
    desc: t('landing.steps.step1Desc'),
  },
  {
    icon: Layers,
    label: t('landing.steps.step2'),
    title: t('landing.steps.step2Title'),
    desc: t('landing.steps.step2Desc'),
  },
  {
    icon: Zap,
    label: t('landing.steps.step3'),
    title: t('landing.steps.step3Title'),
    desc: t('landing.steps.step3Desc'),
  },
])

const providers = [
  { name: 'GitHub', color: '#181717', svg: 'M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z' },
  { name: 'Google', color: '#4285F4', svg: 'M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18A10.96 10.96 0 001 12c0 1.77.42 3.45 1.18 4.93l3.66-2.84z M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z' },
  { name: 'WeChat', color: '#07C160', svg: 'M9.5 4C5.36 4 2 6.69 2 10c0 1.89 1.08 3.56 2.78 4.66L4 17l2.5-1.18C7.47 16.27 8.47 16.5 9.5 16.5c.17 0 .34 0 .5-.02C9.68 15.83 9.5 15.17 9.5 14.5 9.5 10.91 12.86 8 17 8c.34 0 .68.02 1 .06C17.44 5.72 13.78 4 9.5 4zM7 9a1 1 0 110-2 1 1 0 010 2zm5 0a1 1 0 110-2 1 1 0 010 2zm5 5.5c0 2.49-2.69 4.5-6 4.5-.83 0-1.62-.13-2.34-.36L6 20l1.34-1.34C5.89 17.53 5 16.08 5 14.5 5 12.01 7.69 10 11 10s6 2.01 6 4.5zM14 13a.75.75 0 110-1.5.75.75 0 010 1.5zm4 0a.75.75 0 110-1.5.75.75 0 010 1.5z' },
  { name: 'QQ', color: '#12B7F5', svg: 'M12 2C7.58 2 4 5.58 4 10c0 1.5.42 2.9 1.14 4.1-.36 1.07-.83 2.1-1.14 2.9.97-.14 1.88-.38 2.7-.7C8.26 17.37 10.04 18 12 18s3.74-.63 5.3-1.7c.82.32 1.73.56 2.7.7-.31-.8-.78-1.83-1.14-2.9C19.58 12.9 20 11.5 20 10c0-4.42-3.58-8-8-8zm-3 9a1.5 1.5 0 110-3 1.5 1.5 0 010 3zm6 0a1.5 1.5 0 110-3 1.5 1.5 0 010 3z' },
  { name: 'GitLab', color: '#FC6D26', svg: 'M22.65 14.39L12 22.13 1.35 14.39a.84.84 0 01-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 014.82 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.49h8.1l2.44-7.51A.42.42 0 0118.6 2a.43.43 0 01.58 0 .42.42 0 01.11.18l2.44 7.51L23 13.45a.84.84 0 01-.35.94z' },
  { name: 'Discord', color: '#5865F2', svg: 'M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.041-.106 13.107 13.107 0 01-1.872-.892.077.077 0 01-.008-.128 10.2 10.2 0 00.372-.292.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.01c.12.098.246.198.373.292a.077.077 0 01-.006.127 12.299 12.299 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.839 19.839 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z' },
  { name: 'Microsoft', color: '#00A4EF', svg: 'M3 3h8.5v8.5H3V3zm9.5 0H21v8.5h-8.5V3zM3 12.5h8.5V21H3v-8.5zm9.5 0H21V21h-8.5v-8.5z' },
  { name: 'Apple', color: '#000000', svg: 'M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.8-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83zM13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z' },
  { name: 'Telegram', color: '#26A5E4', svg: 'M11.944 0A12 12 0 000 12a12 12 0 0012 12 12 12 0 0012-12A12 12 0 0012 0a12 12 0 00-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 01.171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.479.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z' },
]

const footerCols = computed(() => [
  {
    heading: t('landing.footer.product'),
    links: [
      { label: t('landing.footer.features'), to: '/features' },
      { label: t('landing.footer.docs'), to: '/docs' },
    ],
  },
  {
    heading: t('landing.footer.developers'),
    links: [
      { label: t('landing.footer.docs'), to: '/docs' },
      { label: t('landing.footer.apiRef'), to: '/docs' },
    ],
  },
  {
    heading: t('landing.footer.links'),
    links: [
      { label: t('landing.footer.privacy'), to: '/privacy' },
      { label: t('landing.footer.terms'), to: '/terms' },
    ],
  },
])
</script>

<template>
  <div>
    <!-- Hero -->
    <section class="max-w-[1200px] mx-auto px-6 md:px-10 pt-36 pb-24 text-center">
      <div class="inline-flex items-center gap-2 px-4 py-1.5 rounded-full border border-border bg-muted text-xs font-medium text-muted-foreground mb-8">
        <span class="w-1.5 h-1.5 rounded-full bg-success" />
        {{ $t('landing.badge') }}
      </div>

      <h1 class="text-[clamp(2.5rem,6vw,4.5rem)] font-bold tracking-tight leading-[1.05] text-foreground mx-auto max-w-3xl">
        {{ $t('landing.heroTitle') }}
      </h1>

      <p class="text-muted-foreground text-lg max-w-xl mx-auto mt-5 leading-relaxed">
        {{ $t('landing.heroDesc') }}
      </p>

      <div class="flex items-center justify-center gap-3 mt-10">
        <RouterLink
          to="/register"
          class="inline-flex items-center gap-2 bg-foreground text-white px-6 py-2.5 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors"
        >
          {{ $t('landing.getStarted') }}
          <ArrowRight class="w-4 h-4" />
        </RouterLink>
        <RouterLink
          to="/docs"
          class="inline-flex items-center gap-2 border border-border px-6 py-2.5 rounded-full text-sm font-medium text-foreground hover:bg-muted transition-colors"
        >
          {{ $t('landing.discovery') }}
          <ArrowRight class="w-4 h-4" />
        </RouterLink>
      </div>
    </section>

    <!-- Stats -->
    <section class="max-w-[1200px] mx-auto px-6 md:px-10 pb-20">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-8 md:gap-4">
        <div v-for="s in stats" :key="s.label" class="text-center">
          <div class="text-3xl md:text-4xl font-bold tracking-tight text-foreground">{{ s.value }}</div>
          <div class="text-sm text-muted-foreground mt-1">{{ s.label }}</div>
        </div>
      </div>
    </section>

    <!-- Features -->
    <section id="features" class="max-w-[1200px] mx-auto px-6 md:px-10 py-20 border-t border-border">
      <div class="mb-14">
        <div class="font-mono text-xs font-semibold text-muted-foreground tracking-widest uppercase mb-3">01</div>
        <h2 class="text-[clamp(1.75rem,3.5vw,2.5rem)] font-bold tracking-tight text-foreground">{{ $t('landing.featuresSection') }}</h2>
      </div>

      <div class="rounded-xl overflow-hidden bg-border grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-px">
        <div
          v-for="f in features"
          :key="f.title"
          class="bg-background p-7 flex flex-col gap-4"
        >
          <div class="w-10 h-10 rounded-lg border border-border flex items-center justify-center text-foreground hover:bg-foreground hover:text-white transition-colors">
            <component :is="f.icon" class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-sm font-semibold text-foreground">{{ f.title }}</h3>
            <p class="text-sm text-muted-foreground mt-1 leading-relaxed">{{ f.desc }}</p>
          </div>
        </div>
      </div>
      <div class="mt-8 text-center">
        <RouterLink to="/features" class="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
          {{ $t('landing.viewAllFeatures') }}
          <ArrowRight class="w-4 h-4" />
        </RouterLink>
      </div>
    </section>

    <!-- Integration -->
    <section id="integration" class="max-w-[1200px] mx-auto px-6 md:px-10 py-20 border-t border-border">
      <div class="mb-14">
        <div class="font-mono text-xs font-semibold text-muted-foreground tracking-widest uppercase mb-3">02</div>
        <h2 class="text-[clamp(1.75rem,3.5vw,2.5rem)] font-bold tracking-tight text-foreground">{{ $t('landing.integrationSection') }}</h2>
      </div>

      <div class="space-y-0">
        <div
          v-for="step in steps"
          :key="step.label"
          class="border-t border-border py-8 grid grid-cols-1 md:grid-cols-[220px_1fr] gap-4 md:gap-12 items-start"
        >
          <div class="flex items-center gap-3 text-muted-foreground">
            <component :is="step.icon" class="w-4 h-4" />
            <span class="font-mono text-xs font-semibold tracking-widest uppercase">{{ step.label }}</span>
          </div>
          <div>
            <h3 class="text-base font-semibold text-foreground">{{ step.title }}</h3>
            <p class="text-sm text-muted-foreground mt-1 leading-relaxed">{{ step.desc }}</p>
          </div>
        </div>
      </div>

      <div class="mt-8 text-center">
        <RouterLink to="/docs" class="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors">
          {{ $t('landing.viewDocs') }}
          <ArrowRight class="w-4 h-4" />
        </RouterLink>
      </div>
    </section>

    <!-- Provider strip -->
    <section class="max-w-[1200px] mx-auto px-6 md:px-10 py-16">
      <div class="flex flex-wrap items-center justify-center gap-6 md:gap-8">
        <div
          v-for="p in providers"
          :key="p.name"
          class="flex items-center gap-2 text-muted-foreground/70 hover:text-foreground transition-colors"
        >
          <svg class="w-5 h-5 shrink-0" viewBox="0 0 24 24" fill="currentColor">
            <path :d="p.svg" />
          </svg>
          <span class="text-sm font-medium tracking-wide">{{ p.name }}</span>
        </div>
      </div>
    </section>

    <!-- CTA -->
    <section class="max-w-[1200px] mx-auto px-6 md:px-10 py-24 text-center border-t border-border">
      <h2 class="text-[clamp(1.75rem,3.5vw,2.5rem)] font-bold tracking-tight text-foreground">{{ $t('landing.cta.title') }}</h2>
      <p class="text-muted-foreground mt-3 max-w-md mx-auto">
        {{ $t('landing.cta.desc') }}
      </p>
      <RouterLink
        to="/register"
        class="inline-flex items-center gap-2 bg-foreground text-white px-6 py-2.5 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors mt-8"
      >
        {{ $t('landing.getStarted') }}
        <ArrowRight class="w-4 h-4" />
      </RouterLink>
    </section>

    <!-- Footer -->
    <footer class="border-t border-border">
      <div class="max-w-[1200px] mx-auto px-6 md:px-10 py-16">
        <div class="grid grid-cols-2 md:grid-cols-4 gap-10">
          <!-- Brand -->
          <div class="col-span-2 md:col-span-1">
            <div class="flex items-center gap-2.5 font-bold text-lg tracking-tight text-foreground">
              <div class="w-7 h-7 bg-foreground rounded-md flex items-center justify-center text-white">
                <Fingerprint class="w-4 h-4" />
              </div>
              OIDC
            </div>
            <p class="text-sm text-muted-foreground mt-3 leading-relaxed">
              {{ $t('landing.footer.desc') }}
            </p>
          </div>

          <!-- Link columns -->
          <div v-for="col in footerCols" :key="col.heading">
            <h4 class="text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-4">{{ col.heading }}</h4>
            <ul class="space-y-2.5">
              <li v-for="link in col.links" :key="link.label">
                <RouterLink :to="link.to" class="text-sm text-muted-foreground hover:text-foreground transition-colors">
                  {{ link.label }}
                </RouterLink>
              </li>
            </ul>
          </div>
        </div>

        <!-- Bottom bar -->
        <div class="border-t border-border mt-12 pt-6 flex flex-col sm:flex-row items-center justify-between gap-4">
          <span class="text-xs text-muted-foreground">{{ $t('landing.footer.rights', { year: new Date().getFullYear() }) }}</span>
          <div class="flex items-center gap-6">
            <RouterLink to="/privacy" class="text-xs text-muted-foreground hover:text-foreground transition-colors">{{ $t('landing.footer.privacy') }}</RouterLink>
            <RouterLink to="/terms" class="text-xs text-muted-foreground hover:text-foreground transition-colors">{{ $t('landing.footer.terms') }}</RouterLink>
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>
