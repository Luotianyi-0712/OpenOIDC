<script setup lang="ts">
import { Bell } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useAnnouncements } from '@/composables/useAnnouncements'

const { t } = useI18n()
const { unreadCount, show, refresh, open } = useAnnouncements()
function toggle() {
  if (open.value) open.value = false
  else {
    show()
    void refresh()
  }
}
</script>

<template>
  <button type="button" class="relative inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2" :aria-label="t('announcements.notifications')" :title="unreadCount ? t('announcements.unreadCount', { count: unreadCount }) : t('announcements.notifications')" :aria-expanded="open" aria-haspopup="dialog" data-testid="announcement-bell" @click="toggle">
    <Bell class="h-[18px] w-[18px]" />
    <span v-if="unreadCount" class="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-red-500 ring-2 ring-white" data-testid="announcement-unread" />
    <span v-if="unreadCount" class="sr-only">{{ t('announcements.unreadCount', { count: unreadCount }) }}</span>
  </button>
</template>
