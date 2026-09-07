<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogTitle, DialogClose } from 'radix-vue'
import { Eye, Loader2, Pencil, Save, X } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import AnnouncementMarkdown from '@/components/announcements/AnnouncementMarkdown.vue'
import { announcementInput } from '@/composables/useAdminAnnouncements'
import type { Announcement, AnnouncementInput, AnnouncementMode } from '@/types/announcement'

const props = defineProps<{ item?: Announcement; saving: boolean; error: string }>()
const emit = defineEmits<{ close: []; save: [input: AnnouncementInput] }>()
const { t } = useI18n()
const form = reactive(announcementInput(props.item))
const tab = ref<'write' | 'preview'>('write')
const modes: AnnouncementMode[] = ['banner', 'modal', 'both']
const contentBytes = computed(() => new TextEncoder().encode(form.content).length)
const invalid = computed(() => !form.title.trim() || !form.content.trim() || contentBytes.value > 65536)
function close(value: boolean) { if (!value && !props.saving) emit('close') }
function submit() { if (!invalid.value && !props.saving) emit('save', { ...form }) }
</script>

<template>
  <DialogRoot :open="true" @update:open="close">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-[80] bg-black/30" />
      <DialogContent class="fixed left-1/2 top-1/2 z-[81] flex max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] max-w-2xl -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-lg border border-border bg-white shadow-xl focus:outline-none" :aria-describedby="undefined">
        <div class="flex items-center justify-between border-b border-border px-5 py-4">
          <DialogTitle class="text-base font-semibold">{{ t(item ? 'announcements.edit' : 'announcements.create') }}</DialogTitle>
          <DialogClose :disabled="saving" class="flex h-8 w-8 items-center justify-center rounded-md hover:bg-muted disabled:opacity-50" :title="t('close')" :aria-label="t('close')"><X class="h-4 w-4" /></DialogClose>
        </div>
        <form class="flex min-h-0 flex-1 flex-col" @submit.prevent="submit">
          <div class="space-y-5 overflow-y-auto p-5">
            <div>
              <label for="announcement-title" class="mb-1.5 block text-sm font-medium">{{ t('announcements.subject') }}</label>
              <input id="announcement-title" v-model="form.title" required maxlength="160" :disabled="saving" class="w-full rounded-md border border-border px-3 py-2 text-sm focus:outline-2 focus:outline-sky-600" />
            </div>
            <fieldset :disabled="saving">
              <legend class="mb-2 text-sm font-medium">{{ t('announcements.displayMode') }}</legend>
              <div class="inline-flex max-w-full rounded-md border border-border p-1" role="group" :aria-label="t('announcements.displayMode')">
                <button v-for="mode in modes" :key="mode" type="button" :aria-pressed="form.display_mode === mode" class="rounded px-3 py-1.5 text-sm font-medium transition-colors" :class="form.display_mode === mode ? 'bg-foreground text-white' : 'text-muted-foreground hover:bg-muted'" @click="form.display_mode = mode">{{ t(`announcements.modes.${mode}`) }}</button>
              </div>
              <div v-if="form.display_mode !== 'modal'" class="mt-3 flex flex-wrap gap-x-6 gap-y-3">
                <label class="flex items-center gap-2 text-sm"><input v-model="form.dismissible" type="checkbox" class="h-4 w-4 accent-sky-700" />{{ t('announcements.dismissible') }}</label>
                <label class="flex items-center gap-2 text-sm"><input v-model="form.scrolling" type="checkbox" class="h-4 w-4 accent-sky-700" />{{ t('announcements.scrolling') }}</label>
              </div>
            </fieldset>
            <div>
              <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
                <label for="announcement-content" class="text-sm font-medium">{{ t('announcements.content') }}</label>
                <div class="flex gap-1" role="tablist" :aria-label="t('announcements.content')">
                  <button id="announcement-write-tab" type="button" role="tab" aria-controls="announcement-content-panel" :aria-selected="tab === 'write'" class="flex items-center gap-1.5 rounded px-2 py-1 text-xs" :class="tab === 'write' ? 'bg-muted text-foreground' : 'text-muted-foreground'" @click="tab = 'write'"><Pencil class="h-3.5 w-3.5" />{{ t('announcements.write') }}</button>
                  <button id="announcement-preview-tab" type="button" role="tab" aria-controls="announcement-content-panel" :aria-selected="tab === 'preview'" class="flex items-center gap-1.5 rounded px-2 py-1 text-xs" :class="tab === 'preview' ? 'bg-muted text-foreground' : 'text-muted-foreground'" @click="tab = 'preview'"><Eye class="h-3.5 w-3.5" />{{ t('announcements.preview') }}</button>
                </div>
              </div>
              <div id="announcement-content-panel" role="tabpanel" :aria-labelledby="tab === 'write' ? 'announcement-write-tab' : 'announcement-preview-tab'">
                <textarea v-show="tab === 'write'" id="announcement-content" v-model="form.content" :disabled="saving" class="block h-60 w-full resize-y rounded-md border border-border p-3 font-mono text-sm leading-6 focus:outline-2 focus:outline-sky-600" />
                <div v-if="tab === 'preview'" class="h-60 overflow-y-auto rounded-md border border-border p-3"><AnnouncementMarkdown :content="form.content" /></div>
              </div>
              <p v-if="contentBytes > 65536" class="mt-1 text-xs text-destructive">{{ t('announcements.contentTooLong') }}</p>
            </div>
            <label class="flex items-center gap-2 text-sm font-medium"><input v-model="form.is_published" type="checkbox" :disabled="saving" class="h-4 w-4 accent-sky-700" />{{ t('announcements.published') }}</label>
            <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
          </div>
          <div class="flex justify-end gap-2 border-t border-border px-5 py-4">
            <button type="button" :disabled="saving" class="rounded-md border border-border px-4 py-2 text-sm hover:bg-muted disabled:opacity-50" @click="emit('close')">{{ t('cancel') }}</button>
            <button type="submit" :disabled="saving || invalid" class="inline-flex items-center gap-2 rounded-md bg-foreground px-4 py-2 text-sm font-medium text-white hover:bg-foreground/90 disabled:opacity-50"><Loader2 v-if="saving" class="h-4 w-4 animate-spin" /><Save v-else class="h-4 w-4" />{{ t('save') }}</button>
          </div>
        </form>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
