<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { Fingerprint } from 'lucide-vue-next'
import { ref } from 'vue'
import { setLocale, currentLocale } from '@/i18n'

const locale = ref(currentLocale())

function toggleLocale() {
  const next = locale.value === 'zh' ? 'en' : 'zh'
  setLocale(next)
  locale.value = next
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
          <button @click="toggleLocale" class="text-sm text-muted-foreground hover:text-foreground transition-colors font-medium px-2 py-1 rounded border border-border">
            {{ locale === 'zh' ? 'EN' : '中文' }}
          </button>
          <slot name="nav-right" />
        </div>
      </div>
    </nav>
    <main class="min-h-screen flex items-center justify-center pt-24 pb-16 px-6">
      <RouterView />
    </main>
  </div>
</template>
