import { computed, inject, onBeforeUnmount, onMounted, provide, readonly, ref, shallowRef, watch, type InjectionKey } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { Announcement } from '@/types/announcement'

const key: InjectionKey<ReturnType<typeof createAnnouncements>> = Symbol('announcements')

function createAnnouncements() {
  const auth = useAuthStore()
  const items = ref<Announcement[]>([])
  const loading = shallowRef(false)
  const error = shallowRef('')
  const open = shallowRef(false)
  const selectedID = shallowRef<string | null>(null)
  const read = ref<Record<string, string>>({})
  const dismissed = ref<Record<string, string>>({})
  const storageKey = computed(() => `oidc.announcements.${auth.user?.id || 'guest'}`)
  const notifications = computed(() => items.value.filter(item => item.display_mode !== 'banner'))
  const banners = computed(() => items.value.filter(item => item.display_mode !== 'modal' && dismissed.value[item.id] !== item.revision))
  const selected = computed(() => items.value.find(item => item.id === selectedID.value) || null)
  const unreadCount = computed(() => notifications.value.filter(item => read.value[item.id] !== item.revision).length)
  let trigger: HTMLElement | null = null
  let interval: ReturnType<typeof setInterval> | undefined
  let requestID = 0
  let disposed = false

  function loadState() {
    try {
      const state = JSON.parse(localStorage.getItem(storageKey.value) || '{}')
      read.value = validRecord(state?.read)
      dismissed.value = validRecord(state?.dismissed)
    } catch {
      read.value = {}
      dismissed.value = {}
    }
  }

  function persist() {
    try {
      localStorage.setItem(storageKey.value, JSON.stringify({ read: read.value, dismissed: dismissed.value }))
    } catch { /* Keep the current session usable when storage is unavailable. */ }
  }

  function isUnread(item: Announcement) {
    return read.value[item.id] !== item.revision
  }

  function markRead(item: Announcement) {
    read.value = { ...read.value, [item.id]: item.revision }
    persist()
  }

  function markAllRead() {
    read.value = { ...read.value, ...Object.fromEntries(notifications.value.map(item => [item.id, item.revision])) }
    persist()
  }

  function dismiss(item: Announcement) {
    if (!item.dismissible) return
    dismissed.value = { ...dismissed.value, [item.id]: item.revision }
    persist()
  }

  function show(item?: Announcement) {
    if (!open.value) trigger = document.activeElement instanceof HTMLElement ? document.activeElement : null
    selectedID.value = item?.id ?? null
    if (item) markRead(item)
    open.value = true
  }

  function restoreFocus(event: Event) {
    event.preventDefault()
    if (trigger?.isConnected) trigger.focus()
  }

  async function refresh(force = false) {
    if (loading.value && !force) return
    const current = ++requestID
    loading.value = true
    try {
      const res = await api.get<Announcement[]>('/announcements')
      if (disposed || current !== requestID) return
      items.value = res.data || []
      error.value = ''
      if (selectedID.value && !selected.value) selectedID.value = null
    } catch (e) {
      if (!disposed && current === requestID) error.value = e instanceof Error ? e.message : String(e)
    } finally {
      if (!disposed && current === requestID) loading.value = false
    }
  }

  function storageChanged(event: StorageEvent) {
    if (event.key === storageKey.value || event.key === null) loadState()
  }

  function visibilityChanged() {
    if (document.visibilityState === 'visible') void refresh()
  }

  watch(storageKey, () => {
    open.value = false
    selectedID.value = null
    loadState()
  }, { immediate: true })

  onMounted(() => {
    void refresh()
    interval = setInterval(visibilityChanged, 60000)
    window.addEventListener('storage', storageChanged)
    document.addEventListener('visibilitychange', visibilityChanged)
  })
  onBeforeUnmount(() => {
    disposed = true
    clearInterval(interval)
    window.removeEventListener('storage', storageChanged)
    document.removeEventListener('visibilitychange', visibilityChanged)
  })

  return { items: readonly(items), notifications, banners, selected, unreadCount, loading, error, open, selectedID, isUnread, markAllRead, dismiss, show, refresh, restoreFocus }
}

function validRecord(value: unknown): Record<string, string> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  return Object.fromEntries(Object.entries(value).filter((entry): entry is [string, string] => typeof entry[1] === 'string').slice(-1000))
}

export function provideAnnouncements() {
  const context = createAnnouncements()
  provide(key, context)
  return context
}

export function useAnnouncements() {
  const context = inject(key)
  if (!context) throw new Error('Announcement context is missing')
  return context
}
