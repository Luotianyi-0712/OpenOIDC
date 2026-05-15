<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

const { t } = useI18n()
</script>

<template>
  <div class="min-h-screen flex items-center justify-center px-6">
    <div class="text-center">
      <p class="text-7xl font-bold tracking-tight">404</p>
      <p class="text-muted-foreground mt-3 text-[0.9375rem]">{{ $t('notFound.message') }}</p>
      <RouterLink to="/" class="inline-block mt-6 bg-foreground text-white rounded-full px-5 py-2.5 text-sm font-medium hover:bg-foreground/90 transition-colors">
        {{ $t('notFound.home') }}
      </RouterLink>
    </div>
  </div>
</template>
