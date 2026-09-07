<script setup lang="ts">
import ToastContainer from '@/components/ToastContainer.vue'
import AnnouncementDialog from '@/components/announcements/AnnouncementDialog.vue'
import { provideAnnouncements } from '@/composables/useAnnouncements'

provideAnnouncements()
</script>

<template>
  <RouterView />
  <AnnouncementDialog />
  <ToastContainer />
</template>
