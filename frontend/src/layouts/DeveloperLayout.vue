<script setup lang="ts">
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Fingerprint, AppWindow, ArrowLeft, LogOut } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { setLocale, currentLocale } from '@/i18n'

const { t } = useI18n()
const auth = useAuthStore()
const route = useRoute()
const locale = ref(currentLocale())

function toggleLocale() {
  const next = locale.value === 'zh' ? 'en' : 'zh'
  setLocale(next)
  locale.value = next
}

const nav = computed(() => [
  { to: '/developer', label: t('devNav.myApps'), icon: AppWindow },
])

function isActive(path: string) {
  return route.path === path
}
</script>

<template>
  <div class="min-h-screen">
    <nav class="fixed top-0 inset-x-0 z-50 bg-white/85 backdrop-blur-xl border-b border-border">
      <div class="max-w-[1200px] mx-auto px-6 md:px-10 h-16 flex items-center justify-between">
        <RouterLink to="/" class="flex items-center gap-2.5 font-bold text-lg tracking-tight">
          <div class="w-7 h-7 bg-foreground rounded-md flex items-center justify-center text-white">
            <Fingerprint class="w-4 h-4" />
          </div>
          OIDC
        </RouterLink>
        <div class="flex items-center gap-3">
          <RouterLink to="/me" class="text-sm text-muted-foreground hover:text-foreground transition-colors font-medium px-3 py-2 flex items-center gap-1.5">
            <ArrowLeft class="w-4 h-4" /> {{ $t('devNav.dashboard') }}
          </RouterLink>
          <button @click="toggleLocale" class="text-sm text-muted-foreground hover:text-foreground transition-colors font-medium px-2 py-1 rounded border border-border">
            {{ locale === 'zh' ? 'EN' : '中文' }}
          </button>
          <button @click="auth.logout().then(() => $router.push('/login'))" class="text-sm text-muted-foreground hover:text-foreground transition-colors font-medium px-3 py-2 flex items-center gap-1.5">
            <LogOut class="w-4 h-4" /> {{ $t('nav.logout') }}
          </button>
        </div>
      </div>
    </nav>

    <div class="max-w-[1200px] mx-auto px-6 md:px-10 pt-24 pb-16">
      <h1 class="text-2xl font-bold tracking-tight mb-6">{{ $t('devNav.title') }}</h1>
      <div class="flex gap-0 border-b border-border mb-8 overflow-x-auto">
        <RouterLink
          v-for="item in nav"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-2 px-4 py-3 text-sm font-medium whitespace-nowrap border-b-2 transition-colors"
          :class="isActive(item.to) ? 'border-foreground text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'"
        >
          <component :is="item.icon" class="w-4 h-4" />
          {{ item.label }}
        </RouterLink>
      </div>
      <RouterView />
    </div>
  </div>
</template>
