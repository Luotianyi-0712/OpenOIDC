import { ref, onMounted, onUnmounted, watch } from 'vue'

declare global {
  interface Window {
    turnstile?: {
      render: (container: string | HTMLElement, options: any) => string
      reset: (widgetId: string) => void
      remove: (widgetId: string) => void
    }
    hcaptcha?: {
      render: (container: string | HTMLElement, options: any) => string
      reset: (widgetId: string) => void
      remove: (widgetId: string) => void
    }
    onCaptchaLoad?: () => void
  }
}

export type CaptchaProvider = 'turnstile' | 'hcaptcha'

export function useCaptcha(provider: () => string, siteKey: () => string) {
  const token = ref('')
  const widgetId = ref<string | null>(null)
  const containerId = 'captcha-container-' + Math.random().toString(36).slice(2, 8)
  let loadedScript = ''

  function normalizedProvider(): CaptchaProvider {
    return provider() === 'hcaptcha' ? 'hcaptcha' : 'turnstile'
  }

  function api() {
    return normalizedProvider() === 'hcaptcha' ? window.hcaptcha : window.turnstile
  }

  function scriptSrc() {
    return normalizedProvider() === 'hcaptcha'
      ? 'https://js.hcaptcha.com/1/api.js?onload=onCaptchaLoad&render=explicit'
      : 'https://challenges.cloudflare.com/turnstile/v0/api.js?onload=onCaptchaLoad'
  }

  function loadScript() {
    const src = scriptSrc()
    if (loadedScript === src || document.querySelector(`script[src="${src}"]`)) {
      loadedScript = src
      return
    }
    const script = document.createElement('script')
    script.src = src
    script.async = true
    document.head.appendChild(script)
    loadedScript = src
  }

  function renderWidget() {
    const key = siteKey()
    const captcha = api()
    if (!key || !captcha) return
    const container = document.getElementById(containerId)
    if (!container) return
    if (widgetId.value) captcha.remove(widgetId.value)
    token.value = ''
    widgetId.value = captcha.render(container, {
      sitekey: key,
      callback: (t: string) => { token.value = t },
      'expired-callback': () => { token.value = '' },
      'error-callback': () => { token.value = '' },
    })
  }

  function reset() {
    token.value = ''
    const captcha = api()
    if (widgetId.value && captcha) captcha.reset(widgetId.value)
  }

  function ensureRendered() {
    if (!siteKey()) return
    loadScript()
    window.onCaptchaLoad = renderWidget
    if (api()) renderWidget()
  }

  onMounted(ensureRendered)
  watch([() => provider(), () => siteKey()], ensureRendered)

  onUnmounted(() => {
    const captcha = api()
    if (widgetId.value && captcha) captcha.remove(widgetId.value)
  })

  return { token, containerId, reset, renderWidget }
}
