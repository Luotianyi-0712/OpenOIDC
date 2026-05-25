import { ref } from 'vue'
import { startRegistration, startAuthentication } from '@simplewebauthn/browser'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

export interface PasskeyCredential {
  id: string
  name: string
  last_used_at: string | null
  created_at: string
}

export function usePasskey() {
  const loading = ref(false)
  const error = ref('')
  const auth = useAuthStore()
  const router = useRouter()

  async function registerPasskey(name?: string) {
    loading.value = true
    error.value = ''
    try {
      const beginRes = await api.post<{ options: any; session_id: string }>('/me/passkeys/register/begin')
      const options = beginRes.data!.options
      const sessionId = beginRes.data!.session_id

      const attResp = await startRegistration({ optionsJSON: options.publicKey })

      await api.post('/me/passkeys/register/finish', attResp, {
        'X-Passkey-Session': sessionId,
      })

      // If a name was provided, rename it
      // The server returns the credential id in the finish response but we
      // just reload the list for simplicity.
      return true
    } catch (e: any) {
      if (e.name === 'NotAllowedError') {
        error.value = 'cancelled'
      } else {
        error.value = e.message || 'Registration failed'
      }
      return false
    } finally {
      loading.value = false
    }
  }

  async function loginWithPasskey(returnTo?: string) {
    loading.value = true
    error.value = ''
    try {
      const beginRes = await api.post<{ options: any; session_id: string }>('/auth/passkey/begin')
      const options = beginRes.data!.options
      const sessionId = beginRes.data!.session_id

      const assertionResp = await startAuthentication({ optionsJSON: options.publicKey })

      await api.post('/auth/passkey/finish', assertionResp, {
        'X-Passkey-Session': sessionId,
      })

      await auth.fetchUser()
      await auth.fetchDeveloperStatus()
      router.push(returnTo || '/me')
      return true
    } catch (e: any) {
      if (e.name === 'NotAllowedError') {
        error.value = 'cancelled'
      } else {
        error.value = e.message || 'Login failed'
      }
      return false
    } finally {
      loading.value = false
    }
  }

  async function listPasskeys() {
    const res = await api.get<PasskeyCredential[]>('/me/passkeys')
    return res.data || []
  }

  async function deletePasskey(id: string) {
    await api.del(`/me/passkeys/${id}`)
  }

  async function renamePasskey(id: string, name: string) {
    await api.put(`/me/passkeys/${id}`, { name })
  }

  return {
    loading,
    error,
    registerPasskey,
    loginWithPasskey,
    listPasskeys,
    deletePasskey,
    renamePasskey,
  }
}