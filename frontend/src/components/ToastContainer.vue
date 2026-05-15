<script setup lang="ts">
import { useToastStore } from '@/stores/toast'
import { X, CheckCircle2, AlertCircle, Info } from 'lucide-vue-next'

const toast = useToastStore()

const iconMap = {
  success: CheckCircle2,
  error: AlertCircle,
  info: Info,
} as const

const colorMap = {
  success: 'border-green-200 bg-green-50 text-green-800',
  error: 'border-destructive/30 bg-destructive/5 text-destructive',
  info: 'border-blue-200 bg-blue-50 text-blue-800',
} as const
</script>

<template>
  <Teleport to="body">
    <div v-if="toast.toasts.length" class="fixed top-6 left-1/2 -translate-x-1/2 z-[100] flex flex-col gap-2 w-96 max-w-[calc(100vw-2rem)] pointer-events-none">
      <div
        v-for="t in toast.toasts"
        :key="t.id"
        class="pointer-events-auto border rounded-lg px-4 py-3 text-sm shadow-lg flex items-start gap-3"
        :class="colorMap[t.type]"
      >
        <component :is="iconMap[t.type]" class="w-4 h-4 mt-0.5 shrink-0" />
        <span class="flex-1">{{ t.message }}</span>
        <button @click="toast.remove(t.id)" class="shrink-0 opacity-60 hover:opacity-100 transition-opacity">
          <X class="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  </Teleport>
</template>
