import { useCaptcha } from './useCaptcha'

export function useTurnstile(siteKey: () => string) {
  return useCaptcha(() => 'turnstile', siteKey)
}
