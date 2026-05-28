import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

export interface Toast {
  id: string
  message: string
  type: ToastType
  duration: number
}

const toasts = ref<Toast[]>([])
let idCounter = 0

export function useToast() {
  function show(message: string, type: ToastType = 'info', duration = 3000) {
    const id = `toast-${++idCounter}`
    const toast: Toast = { id, message, type, duration }
    toasts.value.push(toast)

    if (duration > 0) {
      setTimeout(() => {
        remove(id)
      }, duration)
    }

    return id
  }

  function success(message: string, duration = 3000) {
    return show(message, 'success', duration)
  }

  function error(message: string, duration = 5000) {
    return show(message, 'error', duration)
  }

  function info(message: string, duration = 3000) {
    return show(message, 'info', duration)
  }

  function warning(message: string, duration = 4000) {
    return show(message, 'warning', duration)
  }

  function remove(id: string) {
    const index = toasts.value.findIndex(t => t.id === id)
    if (index > -1) {
      toasts.value.splice(index, 1)
    }
  }

  return {
    toasts,
    show,
    success,
    error,
    info,
    warning,
    remove,
  }
}
