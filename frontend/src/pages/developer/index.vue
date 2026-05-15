<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { Plus, Loader2, AppWindow } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const { t } = useI18n()
const router = useRouter()

interface App {
  id: string
  client_id: string
  client_name: string
  description: string
  logo_url: string
  is_active: boolean
  created_at: string
}

const apps = ref<App[]>([])
const loading = ref(false)
const error = ref('')

onMounted(fetchApps)

async function fetchApps() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<App[]>('/developer/apps')
    apps.value = res.data ?? []
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function formatDate(d: string) {
  return new Date(d).toLocaleDateString()
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-lg font-semibold">{{ $t('developer.title') }}</h2>
      <router-link
        to="/developer/create"
        class="bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors flex items-center gap-2"
      >
        <Plus class="w-4 h-4" /> {{ $t('developer.createApp') }}
      </router-link>
    </div>

    <!-- Error -->
    <div v-if="error" class="mb-4 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
      {{ error }}
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <!-- Empty state -->
    <div v-else-if="apps.length === 0" class="flex flex-col items-center justify-center py-16 text-muted-foreground">
      <AppWindow class="w-10 h-10 mb-3 opacity-40" />
      <p class="text-sm">{{ $t('developer.noApps') }}</p>
      <router-link
        to="/developer/create"
        class="mt-4 bg-foreground text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-foreground/90 transition-colors flex items-center gap-2"
      >
        <Plus class="w-4 h-4" /> {{ $t('developer.createApp') }}
      </router-link>
    </div>

    <!-- App cards -->
    <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <router-link
        v-for="app in apps"
        :key="app.id"
        :to="`/developer/apps/${app.id}`"
        class="border border-border rounded-xl p-5 hover:border-foreground/20 hover:shadow-sm transition-all group"
      >
        <div class="flex items-start gap-3 mb-3">
          <img
            v-if="app.logo_url"
            :src="app.logo_url"
            :alt="app.client_name"
            class="w-10 h-10 rounded-lg object-cover border border-border shrink-0"
          />
          <div
            v-else
            class="w-10 h-10 rounded-lg bg-muted flex items-center justify-center shrink-0"
          >
            <AppWindow class="w-5 h-5 text-muted-foreground" />
          </div>
          <div class="min-w-0 flex-1">
            <h3 class="font-medium text-sm truncate group-hover:text-foreground transition-colors">{{ app.client_name }}</h3>
            <p class="text-xs text-muted-foreground font-mono truncate">{{ app.client_id }}</p>
          </div>
        </div>
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <span>{{ formatDate(app.created_at) }}</span>
          <span
            class="px-2 py-0.5 rounded-full text-xs font-medium"
            :class="app.is_active ? 'bg-green-50 text-green-700' : 'bg-muted text-muted-foreground'"
          >
            {{ app.is_active ? 'active' : 'inactive' }}
          </span>
        </div>
      </router-link>
    </div>
  </div>
</template>
