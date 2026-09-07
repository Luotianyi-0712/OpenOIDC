import { onMounted, ref, shallowRef } from 'vue'
import { api } from '@/api/client'
import { useAnnouncements } from '@/composables/useAnnouncements'
import type { Announcement, AnnouncementInput } from '@/types/announcement'

export function announcementInput(item?: Announcement): AnnouncementInput {
  return {
    title: item?.title || '', content: item?.content || '', display_mode: item?.display_mode || 'both',
    dismissible: item?.dismissible ?? true, scrolling: item?.scrolling ?? false, is_published: item?.is_published ?? false,
  }
}

export function useAdminAnnouncements() {
  const items = ref<Announcement[]>([])
  const loading = shallowRef(true)
  const saving = shallowRef(false)
  const busyID = shallowRef('')
  const error = shallowRef('')
  const editorOpen = shallowRef(false)
  const editing = ref<Announcement | undefined>()
  const { refresh } = useAnnouncements()

  async function load() {
    loading.value = true
    error.value = ''
    try {
      const res = await api.get<Announcement[]>('/admin/announcements')
      items.value = res.data || []
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally { loading.value = false }
  }

  function edit(item?: Announcement) {
    editing.value = item
    editorOpen.value = true
  }

  async function save(input: AnnouncementInput) {
    saving.value = true
    try {
      if (editing.value) await api.put(`/admin/announcements/${editing.value.id}`, input)
      else await api.post('/admin/announcements', input)
      editorOpen.value = false
      await load()
      await refresh(true)
    } finally { saving.value = false }
  }

  async function togglePublished(item: Announcement) {
    busyID.value = item.id
    try {
      await api.put(`/admin/announcements/${item.id}`, { ...announcementInput(item), is_published: !item.is_published })
      await load()
      await refresh(true)
    } finally { busyID.value = '' }
  }

  async function remove(item: Announcement) {
    busyID.value = item.id
    try {
      await api.del(`/admin/announcements/${item.id}`)
      await load()
      await refresh(true)
    } finally { busyID.value = '' }
  }

  onMounted(load)
  return { items, loading, saving, busyID, error, editorOpen, editing, edit, load, save, togglePublished, remove }
}
