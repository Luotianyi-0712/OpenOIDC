import { createI18n } from 'vue-i18n'
import en from './en'
import zh from './zh'

const savedLocale = typeof localStorage !== 'undefined' ? localStorage.getItem('locale') : null

const i18n = createI18n({
  legacy: false,
  locale: savedLocale || 'zh',
  fallbackLocale: 'en',
  messages: { en, zh },
})

export default i18n

export function setLocale(locale: 'en' | 'zh') {
  ;(i18n.global.locale as any).value = locale
  localStorage.setItem('locale', locale)
}

export function currentLocale(): string {
  return (i18n.global.locale as any).value
}
