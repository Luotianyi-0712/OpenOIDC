<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useResizeObserver } from '@vueuse/core'
import { ArrowUpRight, ChevronLeft, ChevronRight, Megaphone, Pause, Play, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useAnnouncements } from '@/composables/useAnnouncements'

const { t } = useI18n()
const { banners, dismiss, show } = useAnnouncements()
const index = ref(0)
const paused = ref(false)
const container = ref<HTMLElement | null>(null)
const title = ref<HTMLElement | null>(null)
const distance = ref(0)
const item = computed(() => banners.value[index.value] || banners.value[0])
const duration = computed(() => `${Math.max(12, distance.value / 35)}s`)
function measure() {
  distance.value = Math.max(0, (title.value?.scrollWidth || 0) - (container.value?.clientWidth || 0))
}
useResizeObserver(container, measure)
watch(() => item.value?.revision, () => { paused.value = false; measure() }, { flush: 'post' })
watch(() => banners.value.length, length => { if (index.value >= length) index.value = 0 })
function step(delta: number) { index.value = (index.value + delta + banners.value.length) % banners.value.length }
</script>

<template>
  <section v-if="item" class="announcement-banner border-b border-sky-200 bg-sky-50 text-sky-950" :aria-label="t('announcements.title')" data-testid="announcement-banner">
    <div class="mx-auto flex min-h-10 max-w-[1200px] items-center gap-2 px-4 sm:px-6 md:px-10">
      <Megaphone class="h-4 w-4 shrink-0 text-sky-700" aria-hidden="true" />
      <button ref="container" type="button" class="banner-title min-w-0 flex-1 overflow-hidden py-2 text-left text-sm font-medium" :class="{ 'is-scrolling': item.scrolling && distance > 0, 'is-paused': paused }" :style="{ '--scroll-distance': `-${distance}px`, '--scroll-duration': duration }" :title="item.title" @click="show(item)">
        <span ref="title" class="banner-text" :class="item.scrolling ? 'whitespace-nowrap' : 'block break-words'">{{ item.title }}</span>
      </button>
      <button type="button" class="banner-icon" :title="t('announcements.view')" :aria-label="t('announcements.view')" @click="show(item)"><ArrowUpRight class="h-4 w-4" /></button>
      <button v-if="item.scrolling" type="button" class="banner-icon" :title="t(paused ? 'announcements.resume' : 'announcements.pause')" :aria-label="t(paused ? 'announcements.resume' : 'announcements.pause')" @click="paused = !paused"><Play v-if="paused" class="h-3.5 w-3.5" /><Pause v-else class="h-3.5 w-3.5" /></button>
      <template v-if="banners.length > 1">
        <button type="button" class="banner-icon" :title="t('announcements.previous')" :aria-label="t('announcements.previous')" @click="step(-1)"><ChevronLeft class="h-4 w-4" /></button>
        <span class="text-xs tabular-nums">{{ index + 1 }}/{{ banners.length }}</span>
        <button type="button" class="banner-icon" :title="t('announcements.next')" :aria-label="t('announcements.next')" @click="step(1)"><ChevronRight class="h-4 w-4" /></button>
      </template>
      <button v-if="item.dismissible" type="button" class="banner-icon" :title="t('announcements.dismiss')" :aria-label="t('announcements.dismiss')" @click="dismiss(item)"><X class="h-4 w-4" /></button>
    </div>
  </section>
</template>

<style scoped>
.banner-icon { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 32px; flex-shrink: 0; border-radius: 4px; }
.banner-icon:hover { background: #e0f2fe; }
.banner-text { display: block; width: fit-content; }
.is-scrolling .banner-text { animation: announcement-scroll var(--scroll-duration) linear infinite alternate; }
.is-paused .banner-text, .announcement-banner:hover .banner-text, .announcement-banner:focus-within .banner-text { animation-play-state: paused; }
@keyframes announcement-scroll { 0%, 15% { transform: translateX(0); } 85%, 100% { transform: translateX(var(--scroll-distance)); } }
@media (prefers-reduced-motion: reduce) { .is-scrolling .banner-text { animation: none; } }
</style>
