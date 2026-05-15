import { ref, onMounted, onUnmounted, watch } from 'vue'

declare global {
  interface Window {
    turnstile?: {
      render: (container: string | HTMLElement, options: any) => string
      reset: (widgetId: string) => void
      remove: (widgetId: string) => void
    }
    onTurnstileLoad?: () => void
  }
}

export function useTurnstile(siteKey: () => string) {
  const token = ref('')
  const widgetId = ref<string | null>(null)
  const containerId = 'turnstile-container-' + Math.random().toString(36).slice(2, 8)
  let scriptLoaded = false

  function loadScript() {
    if (scriptLoaded || document.querySelector('script[src*="turnstile"]')) {
      scriptLoaded = true
      return
    }
    const script = document.createElement('script')
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?onload=onTurnstileLoad'
    script.async = true
    document.head.appendChild(script)
    scriptLoaded = true
  }

  function renderWidget() {
    const key = siteKey()
    if (!key || !window.turnstile) return
    const container = document.getElementById(containerId)
    if (!container) return
    if (widgetId.value) {
      window.turnstile.remove(widgetId.value)
    }
    widgetId.value = window.turnstile.render(container, {
      sitekey: key,
      callback: (t: string) => { token.value = t },
      'expired-callback': () => { token.value = '' },
    })
  }

  function reset() {
    token.value = ''
    if (widgetId.value && window.turnstile) {
      window.turnstile.reset(widgetId.value)
    }
  }

  onMounted(() => {
    const key = siteKey()
    if (!key) return
    loadScript()
    window.onTurnstileLoad = renderWidget
    // If script already loaded
    if (window.turnstile) renderWidget()
  })

  onUnmounted(() => {
    if (widgetId.value && window.turnstile) {
      window.turnstile.remove(widgetId.value)
    }
  })

  return { token, containerId, reset, renderWidget }
}
