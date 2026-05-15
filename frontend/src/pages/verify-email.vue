<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'
import { Loader2, CheckCircle, XCircle } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const status = ref<'loading' | 'success' | 'error'>('loading')
const errorMsg = ref('')

onMounted(async () => {
  const token = route.query.token as string
  if (!token) {
    status.value = 'error'
    errorMsg.value = t('verifyEmail.noToken')
    return
  }
  try {
    await api.post('/auth/verify-email', { token })
    status.value = 'success'
  } catch (e: any) {
    status.value = 'error'
    errorMsg.value = e.message || t('verifyEmail.failed')
  }
})

function goLogin() {
  router.push('/login')
}

function goHome() {
  router.push('/me')
}
</script>

<template>
  <div class="w-full max-w-sm text-center">
    <div v-if="status === 'loading'" class="flex flex-col items-center gap-3">
      <Loader2 class="w-8 h-8 animate-spin text-muted-foreground" />
      <p class="text-sm text-muted-foreground">{{ $t('verifyEmail.verifying') }}</p>
    </div>

    <div v-else-if="status === 'success'" class="flex flex-col items-center gap-4">
      <CheckCircle class="w-12 h-12 text-success" />
      <h2 class="text-lg font-semibold">{{ $t('verifyEmail.successTitle') }}</h2>
      <p class="text-sm text-muted-foreground">{{ $t('verifyEmail.successDesc') }}</p>
      <button @click="goHome" class="bg-foreground text-white px-6 py-2.5 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors mt-2">
        {{ $t('verifyEmail.goHome') }}
      </button>
    </div>

    <div v-else class="flex flex-col items-center gap-4">
      <XCircle class="w-12 h-12 text-destructive" />
      <h2 class="text-lg font-semibold">{{ $t('verifyEmail.failedTitle') }}</h2>
      <p class="text-sm text-muted-foreground">{{ errorMsg }}</p>
      <button @click="goLogin" class="bg-foreground text-white px-6 py-2.5 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors mt-2">
        {{ $t('verifyEmail.goLogin') }}
      </button>
    </div>
  </div>
</template>
