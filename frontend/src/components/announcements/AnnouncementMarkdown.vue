<script setup lang="ts">
import { computed } from 'vue'
import { renderAnnouncementMarkdown } from '@/utils/announcementMarkdown'

const props = defineProps<{ content: string }>()
const html = computed(() => renderAnnouncementMarkdown(props.content))
</script>

<template>
  <div class="announcement-markdown" v-html="html" />
</template>

<style scoped>
.announcement-markdown { min-width: 0; font-size: 14px; line-height: 1.7; overflow-wrap: anywhere; letter-spacing: 0; }
.announcement-markdown :deep(p), .announcement-markdown :deep(ul), .announcement-markdown :deep(ol), .announcement-markdown :deep(blockquote), .announcement-markdown :deep(pre), .announcement-markdown :deep(table) { margin: 0.75rem 0; }
.announcement-markdown :deep(h1), .announcement-markdown :deep(h2), .announcement-markdown :deep(h3), .announcement-markdown :deep(h4), .announcement-markdown :deep(h5), .announcement-markdown :deep(h6) { font-weight: 600; line-height: 1.4; margin: 1rem 0 0.5rem; }
.announcement-markdown :deep(h1) { font-size: 22px; }
.announcement-markdown :deep(h2) { font-size: 19px; }
.announcement-markdown :deep(h3) { font-size: 16px; }
.announcement-markdown :deep(ul) { list-style: disc; padding-left: 1.4rem; }
.announcement-markdown :deep(ol) { list-style: decimal; padding-left: 1.4rem; }
.announcement-markdown :deep(a) { color: #0369a1; text-decoration: underline; text-underline-offset: 3px; }
.announcement-markdown :deep(blockquote) { padding-left: 1rem; border-left: 3px solid #cbd5e1; color: #52525b; }
.announcement-markdown :deep(code) { font-size: 12px; background: #f4f4f5; padding: 2px 4px; border-radius: 3px; }
.announcement-markdown :deep(pre) { max-width: 100%; overflow-x: auto; background: #f4f4f5; padding: 12px; border-radius: 6px; }
.announcement-markdown :deep(pre code) { padding: 0; background: none; }
.announcement-markdown :deep(table) { display: block; max-width: 100%; overflow-x: auto; border-collapse: collapse; }
.announcement-markdown :deep(th), .announcement-markdown :deep(td) { border: 1px solid #d4d4d8; padding: 6px 10px; }
.announcement-markdown :deep(th) { background: #f4f4f5; font-weight: 600; }
.announcement-markdown :deep(img) { display: block; max-width: 100%; height: auto; border-radius: 4px; }
.announcement-markdown :deep(hr) { margin: 1rem 0; border-color: #e4e4e7; }
.announcement-markdown :deep(input) { pointer-events: none; }
.announcement-markdown :deep(> :first-child) { margin-top: 0; }
.announcement-markdown :deep(> :last-child) { margin-bottom: 0; }
</style>
