<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AlertTriangle, ShieldX, XCircle } from 'lucide-vue-next'

const route = useRoute()
const { t } = useI18n()

const errorType = ref('')
const appName = ref('')

onMounted(() => {
  errorType.value = (route.query.type as string) || 'unknown'
  appName.value = (route.query.app as string) || ''
})

function getErrorInfo() {
  switch (errorType.value) {
    case 'app_disabled':
      return {
        icon: ShieldX,
        title: t('error.appDisabled.title'),
        description: t('error.appDisabled.description', { app: appName.value }),
        color: 'text-amber-600'
      }
    case 'app_not_found':
      return {
        icon: XCircle,
        title: t('error.appNotFound.title'),
        description: t('error.appNotFound.description'),
        color: 'text-red-600'
      }
    default:
      return {
        icon: AlertTriangle,
        title: t('error.unknown.title'),
        description: t('error.unknown.description'),
        color: 'text-gray-600'
      }
  }
}

function goHome() {
  window.location.href = '/'
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background px-4">
    <div class="max-w-md w-full text-center">
      <component
        :is="getErrorInfo().icon"
        :class="['w-16 h-16 mx-auto mb-6', getErrorInfo().color]"
      />
      <h1 class="text-2xl font-bold mb-3">{{ getErrorInfo().title }}</h1>
      <p class="text-muted-foreground mb-8">{{ getErrorInfo().description }}</p>
      <button
        @click="goHome"
        class="px-6 py-2.5 bg-foreground text-white rounded-lg hover:bg-foreground/90 transition-colors font-medium"
      >
        {{ $t('error.backToHome') }}
      </button>
    </div>
  </div>
</template>
