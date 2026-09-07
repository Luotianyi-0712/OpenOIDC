<script setup lang="ts">
import { ref } from 'vue'
import { Archive, Loader2, Megaphone, Pencil, Plus, RefreshCw, Send, Trash2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useAdminAnnouncements } from '@/composables/useAdminAnnouncements'
import { useToastStore } from '@/stores/toast'
import type { Announcement, AnnouncementInput } from '@/types/announcement'
import AnnouncementEditor from './AnnouncementEditor.vue'

const { t, locale } = useI18n()
const toast = useToastStore()
const { items, loading, saving, busyID, error, editorOpen, editing, edit, load, save, togglePublished, remove } = useAdminAnnouncements()
const editorError = ref('')
function openEditor(item?: Announcement) { editorError.value = ''; edit(item) }
function date(value: string) { return new Date(value).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US') }
async function submit(input: AnnouncementInput) {
  editorError.value = ''
  try { await save(input); toast.success(t('announcements.saved')) }
  catch (e) { editorError.value = e instanceof Error ? e.message : String(e) }
}
async function publish(item: Announcement) {
  try { await togglePublished(item); toast.success(t('announcements.saved')) }
  catch (e) { toast.error(e instanceof Error ? e.message : String(e)) }
}
async function deleteItem(item: Announcement) {
  if (!window.confirm(t('announcements.deleteConfirm', { title: item.title }))) return
  try { await remove(item); toast.success(t('announcements.deleted')) }
  catch (e) { toast.error(e instanceof Error ? e.message : String(e)) }
}
</script>

<template>
  <section>
    <div class="mb-5 flex items-center justify-between gap-3">
      <h2 class="text-lg font-semibold">{{ t('announcements.title') }}</h2>
      <div class="flex items-center gap-2">
        <button type="button" :disabled="loading" class="admin-announcement-icon" :title="t('refresh')" :aria-label="t('refresh')" @click="load"><RefreshCw class="h-4 w-4" :class="{ 'animate-spin': loading }" /></button>
        <button type="button" class="flex items-center gap-2 rounded-md bg-foreground px-3 py-2 text-sm font-medium text-white" @click="openEditor()"><Plus class="h-4 w-4" />{{ t('announcements.create') }}</button>
      </div>
    </div>
    <div v-if="error" role="alert" class="mb-4 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{{ error }}</div>
    <div v-if="loading" class="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground"><Loader2 class="h-4 w-4 animate-spin" />{{ t('loading') }}</div>
    <div v-else-if="!items.length && !error" class="border-y border-border py-16 text-center text-sm text-muted-foreground"><Megaphone class="mx-auto mb-3 h-6 w-6" />{{ t('announcements.emptyAdmin') }}</div>
    <div v-else-if="items.length" class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead class="border-y border-border bg-muted/40 text-xs text-muted-foreground"><tr><th class="px-3 py-3 font-medium">{{ t('announcements.subject') }}</th><th class="px-3 py-3 font-medium">{{ t('announcements.status') }}</th><th class="hidden px-3 py-3 font-medium sm:table-cell">{{ t('announcements.displayMode') }}</th><th class="hidden px-3 py-3 font-medium md:table-cell">{{ t('announcements.updatedAt') }}</th><th class="px-3 py-3 text-right font-medium">{{ t('announcements.actions') }}</th></tr></thead>
        <tbody class="divide-y divide-border">
          <tr v-for="item in items" :key="item.id" class="hover:bg-muted/20">
            <td class="max-w-xs break-words px-3 py-4 font-medium"><button type="button" class="text-left hover:underline" @click="openEditor(item)">{{ item.title }}</button></td>
            <td class="px-3 py-4"><span class="inline-flex items-center gap-1.5 whitespace-nowrap text-xs" :class="item.is_published ? 'text-emerald-700' : 'text-muted-foreground'"><span class="h-1.5 w-1.5 rounded-full" :class="item.is_published ? 'bg-emerald-500' : 'bg-zinc-400'" />{{ t(item.is_published ? 'announcements.published' : 'announcements.draft') }}</span></td>
            <td class="hidden whitespace-nowrap px-3 py-4 text-muted-foreground sm:table-cell">{{ t(`announcements.modes.${item.display_mode}`) }}</td>
            <td class="hidden whitespace-nowrap px-3 py-4 text-xs text-muted-foreground md:table-cell">{{ date(item.updated_at) }}</td>
            <td class="px-2 py-3"><div class="flex justify-end gap-1">
              <button type="button" :disabled="!!busyID" class="admin-announcement-icon" :title="t(item.is_published ? 'announcements.unpublish' : 'announcements.publish')" :aria-label="t(item.is_published ? 'announcements.unpublish' : 'announcements.publish')" @click="publish(item)"><Loader2 v-if="busyID === item.id" class="h-4 w-4 animate-spin" /><Archive v-else-if="item.is_published" class="h-4 w-4" /><Send v-else class="h-4 w-4" /></button>
              <button type="button" :disabled="!!busyID" class="admin-announcement-icon" :title="t('edit')" :aria-label="t('edit')" @click="openEditor(item)"><Pencil class="h-4 w-4" /></button>
              <button type="button" :disabled="!!busyID" class="admin-announcement-icon text-destructive" :title="t('delete')" :aria-label="t('delete')" @click="deleteItem(item)"><Trash2 class="h-4 w-4" /></button>
            </div></td>
          </tr>
        </tbody>
      </table>
    </div>
    <AnnouncementEditor v-if="editorOpen" :item="editing" :saving="saving" :error="editorError" @close="editorOpen = false" @save="submit" />
  </section>
</template>

<style scoped>
.admin-announcement-icon { display: inline-flex; width: 32px; height: 32px; align-items: center; justify-content: center; flex-shrink: 0; border-radius: 4px; }
.admin-announcement-icon:hover { background: #f4f4f5; }
.admin-announcement-icon:disabled { opacity: 0.5; }
</style>
