import { ref, onMounted, computed } from 'vue'
import { api } from '@/api/client'

export interface PasswordPolicy {
  min_length: number
  require_upper: boolean
  require_lower: boolean
  require_digit: boolean
  require_symbol: boolean
}

const defaultPolicy: PasswordPolicy = {
  min_length: 8,
  require_upper: false,
  require_lower: false,
  require_digit: false,
  require_symbol: false,
}

export function usePasswordPolicy() {
  const policy = ref<PasswordPolicy>({ ...defaultPolicy })
  const loaded = ref(false)

  async function fetchPolicy() {
    try {
      const res = await api.get<PasswordPolicy>('/settings/password-policy')
      if (res.data) {
        policy.value = res.data
      }
    } catch {
      // Use defaults
    } finally {
      loaded.value = true
    }
  }

  const hasRequirements = computed(() => {
    const p = policy.value
    return p.require_upper || p.require_lower || p.require_digit || p.require_symbol || p.min_length > 0
  })

  function validate(password: string): string[] {
    const errors: string[] = []
    const p = policy.value
    if (password.length < p.min_length) {
      errors.push(`min_length`)
    }
    if (p.require_upper && !/[A-Z]/.test(password)) {
      errors.push(`require_upper`)
    }
    if (p.require_lower && !/[a-z]/.test(password)) {
      errors.push(`require_lower`)
    }
    if (p.require_digit && !/[0-9]/.test(password)) {
      errors.push(`require_digit`)
    }
    if (p.require_symbol && !/[!@#$%^&*()\-_=+\[\]{};:,.<>/?\\|`~'"']/.test(password)) {
      errors.push(`require_symbol`)
    }
    return errors
  }

  onMounted(fetchPolicy)

  return { policy, loaded, hasRequirements, validate, fetchPolicy }
}
