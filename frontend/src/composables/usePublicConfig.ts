import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

export interface EnabledProvider {
  name: string
  display_name: string
  type?: string
  icon_url?: string
  login_enabled: boolean
  register_enabled: boolean
}

export interface PublicSettings {
  site_url: string
  github_url: string
  contact_info: string
  registration_enabled: boolean
  registration_email_verification_required: boolean
  password_login_enabled: boolean
  social_login_enabled: boolean
  social_register_enabled: boolean
  social_binding_enabled: boolean
  captcha_enabled: boolean
  captcha_provider: string
  captcha_site_key: string
  turnstile_site_key: string
  developer_min_trust_level: number
  passkey_enabled: boolean
  risk_report_email_notification_enabled: boolean
  version: string
}

const PROVIDER_ICONS: Record<string, { path: string; color: string }> = {
  github: {
    path: 'M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 12 6.844a9.59 9.59 0 0 1 2.504.337c1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.02 10.02 0 0 0 22 12.017C22 6.484 17.522 2 12 2z',
    color: '#24292f',
  },
  google: {
    path: '',
    color: '#4285F4',
  },
  gitlab: {
    path: 'M23.955 13.587l-1.342-4.135-2.664-8.189a.455.455 0 00-.867 0L16.418 9.45H7.582L4.918 1.263a.455.455 0 00-.867 0L1.386 9.45.045 13.587a.924.924 0 00.331 1.023L12 23.054l11.624-8.443a.92.92 0 00.331-1.024',
    color: '#fc6d26',
  },
  gitee: {
    path: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4.95 7.8h-3.27c-.26 0-.47.21-.47.47v.63c0 .26.21.47.47.47h2.17v2.17c0 .26-.21.47-.47.47H9.62a.47.47 0 01-.47-.47V9.37c0-.26.21-.47.47-.47h7.33c.26 0 .47-.21.47-.47v-.63c0-.26-.21-.47-.47-.47H9.14C8.05 7.33 7.17 8.21 7.17 9.3v5.56c0 1.09.88 1.97 1.97 1.97h5.72c1.09 0 1.97-.88 1.97-1.97v-3.1c0-1.09-.88-1.97-1.88-1.97z',
    color: '#c71d23',
  },
  linuxdo: {
    path: '',
    color: '#1f2937',
  },
  discord: {
    path: 'M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.041-.106 13.107 13.107 0 01-1.872-.892.077.077 0 01-.008-.128 10.2 10.2 0 00.372-.292.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.01c.12.098.246.198.373.292a.077.077 0 01-.006.127 12.299 12.299 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.839 19.839 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03z',
    color: '#5865f2',
  },
  telegram: {
    path: 'M11.944 0A12 12 0 000 12a12 12 0 0012 12 12 12 0 0012-12A12 12 0 0012 0a12 12 0 00-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 01.171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.479.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z',
    color: '#26a5e4',
  },
  microsoft: {
    path: 'M1 1h10v10H1zM13 1h10v10H13zM1 13h10v10H1zM13 13h10v10H13z',
    color: '#00a4ef',
  },
  apple: {
    path: 'M17.05 20.28c-.98.95-2.05.88-3.08.4-1.09-.5-2.08-.48-3.24 0-1.44.62-2.2.44-3.06-.4C2.79 15.25 3.51 7.59 9.05 7.31c1.35.07 2.29.74 3.08.8 1.18-.24 2.31-.93 3.57-.84 1.51.12 2.65.72 3.4 1.8-3.12 1.87-2.38 5.98.48 7.13-.57 1.5-1.31 2.99-2.53 4.09zM12.03 7.25c-.15-2.23 1.66-4.07 3.74-4.25.32 2.32-1.55 4.3-3.74 4.25z',
    color: '#000000',
  },
  qq: {
    path: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm3.22 14.34c-.35.14-.65-.12-.95-.32-.59.4-1.22.62-1.92.66h-.7c-.7-.04-1.33-.26-1.92-.66-.3.2-.6.46-.95.32-.4-.16-.27-.73-.15-1.11.06-.2.14-.37.22-.52-.36-.5-.58-1.03-.58-1.51 0-2.09 1.94-3.28 3.73-3.28h.6c1.79 0 3.73 1.19 3.73 3.28 0 .48-.22 1.01-.58 1.51.08.15.16.32.22.52.12.38.25.95-.15 1.11-.17.07-.35.07-.6 0z',
    color: '#12b7f5',
  },
  wechat: {
    path: 'M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 01.213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 00.167-.054l1.903-1.114a.864.864 0 01.717-.098 10.16 10.16 0 002.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348z',
    color: '#07c160',
  },
}

const GOOGLE_SVG = `<svg class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none">
  <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
  <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
  <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18A10.96 10.96 0 0 0 1 12c0 1.77.42 3.45 1.18 4.93l3.66-2.84z" fill="#FBBC05"/>
  <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
</svg>`

export function getProviderIcon(name: string) {
  return PROVIDER_ICONS[name]
}

export function isGoogleProvider(name: string) {
  return name === 'google'
}

export { GOOGLE_SVG }

export function usePublicConfig() {
  const providers = ref<EnabledProvider[]>([])
  const settings = ref<PublicSettings>({
    site_url: window.location.origin,
    github_url: 'https://github.com/Luotianyi-0712/OpenOIDC',
    contact_info: '',
    registration_enabled: true,
    registration_email_verification_required: true,
    password_login_enabled: true,
    social_login_enabled: true,
    social_register_enabled: true,
    social_binding_enabled: true,
    captcha_enabled: true,
    captcha_provider: 'turnstile',
    captcha_site_key: '',
    turnstile_site_key: '',
    developer_min_trust_level: 1,
    passkey_enabled: true,
    risk_report_email_notification_enabled: true,
    version: '1.0.0',
  })
  const loaded = ref(false)

  async function load() {
    try {
      const [provRes, setRes] = await Promise.all([
        api.get<EnabledProvider[]>('/social/providers'),
        api.get<Record<string, string>>('/settings/public'),
      ])
      providers.value = (provRes.data ?? []).filter(p => p.name !== 'phone')
      if (setRes.data) {
        const d = setRes.data
        settings.value = {
          site_url: (d.site_url || window.location.origin).replace(/\/+$/, ''),
          github_url: d.github_url || 'https://github.com/Luotianyi-0712/OpenOIDC',
          contact_info: d.contact_info || '',
          registration_enabled: d.registration_enabled !== 'false',
          registration_email_verification_required: d.registration_email_verification_required !== 'false',
          password_login_enabled: d.password_login_enabled !== 'false',
          social_login_enabled: d.social_login_enabled !== 'false',
          social_register_enabled: d.social_register_enabled !== 'false',
          social_binding_enabled: d.social_binding_enabled !== 'false',
          captcha_enabled: d.captcha_enabled !== 'false',
          captcha_provider: d.captcha_provider || 'turnstile',
          captcha_site_key: d.captcha_site_key || d.turnstile_site_key || '',
          turnstile_site_key: d.turnstile_site_key || '',
          developer_min_trust_level: parseInt(d.developer_min_trust_level) || 1,
          passkey_enabled: d.passkey_enabled !== 'false',
          risk_report_email_notification_enabled: d.risk_report_email_notification_enabled !== 'false',
          version: d.version || '1.0.0',
        }
      }
    } catch {
      // fallback to defaults
    } finally {
      loaded.value = true
    }
  }

  onMounted(load)

  return { providers, settings, loaded }
}
