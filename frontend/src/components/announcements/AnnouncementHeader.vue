<script setup lang="ts">
import { ref } from 'vue'
import { useResizeObserver } from '@vueuse/core'
import AnnouncementBanner from './AnnouncementBanner.vue'

const header = ref<HTMLElement | null>(null)
const extraHeight = ref(0)
const emit = defineEmits<{ resize: [height: number] }>()
useResizeObserver(header, ([entry]) => {
  const height = entry?.contentRect.height || 64
  extraHeight.value = Math.max(0, height - 64)
  emit('resize', height)
})
</script>

<template>
  <div ref="header" class="fixed inset-x-0 top-0 z-50 border-b border-border bg-white/95 backdrop-blur-xl">
    <AnnouncementBanner />
    <slot />
  </div>
  <div aria-hidden="true" :style="{ height: `${extraHeight}px` }" />
</template>
