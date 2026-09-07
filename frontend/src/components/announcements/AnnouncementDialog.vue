<script setup lang="ts">
import { ArrowLeft, Bell, CheckCheck, ChevronRight, Loader2, RefreshCw, X } from 'lucide-vue-next'
import { DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogTitle, DialogClose } from 'radix-vue'
import { useI18n } from 'vue-i18n'
import { useAnnouncements } from '@/composables/useAnnouncements'
import AnnouncementMarkdown from './AnnouncementMarkdown.vue'

const { t, locale } = useI18n()
const { notifications, selected, selectedID, open, unreadCount, loading, error, show, isUnread, markAllRead, refresh, restoreFocus } = useAnnouncements()
function date(value: string) { return new Date(value).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US') }
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[70] bg-black/20" />
      <DialogContent class="announcement-window fixed right-3 top-20 z-[71] flex max-h-[calc(100dvh-6rem)] w-[calc(100%-1.5rem)] max-w-lg flex-col overflow-hidden rounded-lg border border-border bg-white shadow-xl focus:outline-none sm:right-6" :aria-describedby="undefined" @close-auto-focus="restoreFocus">
        <div class="flex min-h-14 items-center gap-2 border-b border-border px-4 py-3">
          <button v-if="selected" class="notification-icon" type="button" :title="t('announcements.back')" :aria-label="t('announcements.back')" @click="selectedID = null"><ArrowLeft class="h-4 w-4" /></button>
          <Bell v-else class="h-4 w-4 shrink-0 text-muted-foreground" />
          <DialogTitle class="min-w-0 flex-1 text-sm font-semibold">{{ t('announcements.notifications') }}</DialogTitle>
          <button v-if="!selected && unreadCount" class="notification-icon" type="button" :title="t('announcements.markAllRead')" :aria-label="t('announcements.markAllRead')" @click="markAllRead"><CheckCheck class="h-4 w-4" /></button>
          <DialogClose class="notification-icon" :title="t('announcements.close')" :aria-label="t('announcements.close')"><X class="h-4 w-4" /></DialogClose>
        </div>
        <div class="min-h-0 overflow-y-auto overscroll-contain">
          <div v-if="error" role="alert" class="flex items-center justify-between gap-3 border-b border-red-100 bg-red-50 px-4 py-3 text-xs text-red-700">
            <span>{{ t('announcements.loadError') }}</span>
            <button class="notification-icon" type="button" :title="t('announcements.retry')" :aria-label="t('announcements.retry')" @click="refresh(true)"><RefreshCw class="h-4 w-4" /></button>
          </div>
          <article v-if="selected" class="p-5">
            <h2 class="break-words text-lg font-semibold leading-7">{{ selected.title }}</h2>
            <time class="mb-5 mt-1 block text-xs text-muted-foreground" :datetime="selected.updated_at">{{ date(selected.updated_at) }}</time>
            <AnnouncementMarkdown :content="selected.content" />
          </article>
          <div v-else-if="loading && !notifications.length" class="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground"><Loader2 class="h-4 w-4 animate-spin" />{{ t('loading') }}</div>
          <div v-else-if="!notifications.length && !error" class="py-12 text-center text-sm text-muted-foreground">{{ t('announcements.empty') }}</div>
          <ul v-else class="divide-y divide-border">
            <li v-for="item in notifications" :key="item.id">
              <button type="button" class="flex w-full items-center gap-3 px-5 py-4 text-left hover:bg-muted/50" @click="show(item)">
                <span class="h-2 w-2 shrink-0 rounded-full" :class="isUnread(item) ? 'bg-red-500' : 'bg-transparent'" :aria-label="isUnread(item) ? t('announcements.unread') : undefined" />
                <span class="min-w-0 flex-1"><span class="block break-words text-sm" :class="isUnread(item) ? 'font-semibold' : 'font-medium'">{{ item.title }}</span><time class="mt-1 block text-xs text-muted-foreground" :datetime="item.updated_at">{{ date(item.updated_at) }}</time></span>
                <ChevronRight class="h-4 w-4 shrink-0 text-muted-foreground" />
              </button>
            </li>
          </ul>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style scoped>
.notification-icon { display: inline-flex; width: 28px; height: 28px; align-items: center; justify-content: center; flex-shrink: 0; border-radius: 4px; }
.notification-icon:hover { background: #f4f4f5; }
</style>
