<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { RefreshCw, Loader2, X, AlertTriangle } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface SigningKey {
  id: string
  key_id: string
  algorithm: string
  is_current: boolean
  created_at: string
}

const keys = ref<SigningKey[]>([])
const loading = ref(false)
const error = ref('')
const rotating = ref(false)
const showRotateModal = ref(false)

onMounted(fetchKeys)

async function fetchKeys() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<SigningKey[]>('/admin/keys')
    keys.value = res.data ?? []
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function doRotate() {
  showRotateModal.value = false
  rotating.value = true
  error.value = ''
  try {
    await api.post('/admin/keys/rotate')
    await fetchKeys()
  } catch (e: any) {
    error.value = e.message
  } finally {
    rotating.value = false
  }
}

function formatDate(d: string) {
  return new Date(d).toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit',
  })
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-lg font-semibold">{{ $t('adminKeys.title') }}</h2>
      <button
        @click="showRotateModal = true"
        :disabled="rotating"
        class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors disabled:opacity-50 flex items-center gap-2"
      >
        <RefreshCw class="w-4 h-4" :class="rotating ? 'animate-spin' : ''" />
        {{ $t('adminKeys.rotate') }}
      </button>
    </div>

    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <div v-if="loading && keys.length === 0" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <div v-else class="border border-border rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-muted/50 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
          <tr>
            <th class="px-4 py-3">{{ $t('adminKeys.keyId') }}</th>
            <th class="px-4 py-3">{{ $t('adminKeys.algorithm') }}</th>
            <th class="px-4 py-3">{{ $t('adminUsers.status') }}</th>
            <th class="px-4 py-3">{{ $t('adminKeys.created') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-if="keys.length === 0">
            <td colspan="4" class="px-4 py-8 text-center text-muted-foreground">{{ $t('adminKeys.noKeys') }}</td>
          </tr>
          <tr v-for="key in keys" :key="key.id" class="hover:bg-muted/30 transition-colors">
            <td class="px-4 py-3 font-mono text-xs">{{ key.key_id }}</td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted">
                {{ key.algorithm }}
              </span>
            </td>
            <td class="px-4 py-3">
              <span
                class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium"
                :class="key.is_current ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'"
              >
                {{ key.is_current ? $t('adminKeys.current') : $t('adminKeys.retired') }}
              </span>
            </td>
            <td class="px-4 py-3 text-muted-foreground">{{ formatDate(key.created_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Rotate Confirm Modal -->
    <div v-if="showRotateModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" @click.self="showRotateModal = false">
      <div class="bg-white rounded-xl shadow-lg w-full max-w-md mx-4 p-6">
        <div class="flex items-start gap-3 mb-4">
          <AlertTriangle class="w-5 h-5 text-yellow-600 mt-0.5" />
          <div>
            <h2 class="text-lg font-semibold">{{ $t('adminKeys.rotate') }}</h2>
            <p class="text-sm text-muted-foreground mt-1">{{ $t('adminKeys.rotateConfirm') }}</p>
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <button @click="showRotateModal = false" class="px-4 py-2 text-sm font-medium rounded-lg hover:bg-muted transition-colors">{{ $t('cancel') }}</button>
          <button @click="doRotate" class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors">{{ $t('confirm') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>
