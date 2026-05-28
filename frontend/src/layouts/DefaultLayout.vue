<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Fingerprint, Menu, X, ChevronDown, Github, Mail } from 'lucide-vue-next'
import { ref, reactive, onMounted } from 'vue'
import { setLocale, currentLocale } from '@/i18n'
import { usePublicConfig } from '@/composables/usePublicConfig'

const auth = useAuthStore()
const { settings } = usePublicConfig()
const mobileOpen = ref(false)
const locale = ref(currentLocale())

onMounted(() => {
  if (auth.isLoggedIn && !auth.developerStatus) {
    auth.fetchDeveloperStatus()
  }
})

// Desktop dropdown state — tracks hover timeout per menu
const activeDropdown = ref<string | null>(null)
let hoverTimeout: ReturnType<typeof setTimeout> | null = null

function openDropdown(key: string) {
  if (hoverTimeout) { clearTimeout(hoverTimeout); hoverTimeout = null }
  activeDropdown.value = key
}

function closeDropdown() {
  hoverTimeout = setTimeout(() => { activeDropdown.value = null }, 150)
}

// Mobile accordion state
const mobileExpanded = reactive<Record<string, boolean>>({
  product: false,
  developers: false,
  links: false,
  about: false,
})

function toggleMobileSection(key: string) {
  mobileExpanded[key] = !mobileExpanded[key]
}

function toggleLocale() {
  const next = locale.value === 'zh' ? 'en' : 'zh'
  setLocale(next)
  locale.value = next
}
</script>

<template>
  <div class="min-h-screen">
    <nav class="fixed top-0 inset-x-0 z-50 bg-white/85 backdrop-blur-xl border-b border-border">
      <div class="max-w-[1200px] mx-auto px-4 sm:px-6 md:px-10 h-16 flex items-center justify-between gap-3 relative">
        <!-- Brand -->
        <RouterLink to="/" class="flex items-center gap-2.5 font-bold text-lg tracking-tight shrink-0">
          <div class="w-7 h-7 bg-foreground rounded-md flex items-center justify-center text-white">
            <Fingerprint class="w-4 h-4" />
          </div>
          OIDC
        </RouterLink>

        <!-- Desktop nav dropdowns (centered) -->
        <ul class="hidden md:flex items-center gap-6 absolute left-1/2 -translate-x-1/2">
          <!-- Product -->
          <li class="relative" @mouseenter="openDropdown('product')" @mouseleave="closeDropdown()">
            <button class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors font-medium py-2">
              {{ $t('nav.product') }}
              <ChevronDown class="w-3.5 h-3.5" />
            </button>
            <div
              v-show="activeDropdown === 'product'"
              class="absolute top-full left-0 mt-1 min-w-[180px] bg-white border border-border rounded-lg shadow-lg py-1 z-50"
            >
              <RouterLink to="/features" class="block px-4 py-2.5 text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors">{{ $t('nav.features') }}</RouterLink>
              <RouterLink to="/docs" class="block px-4 py-2.5 text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors">{{ $t('nav.docs') }}</RouterLink>
            </div>
          </li>

          <!-- Developers -->
          <li class="relative" @mouseenter="openDropdown('developers')" @mouseleave="closeDropdown()">
            <button class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors font-medium py-2">
              {{ $t('nav.developers') }}
              <ChevronDown class="w-3.5 h-3.5" />
            </button>
            <div
              v-show="activeDropdown === 'developers'"
              class="absolute top-full left-0 mt-1 min-w-[180px] bg-white border border-border rounded-lg shadow-lg py-1 z-50"
            >
              <RouterLink to="/docs" class="block px-4 py-2.5 text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors">{{ $t('nav.apiDocs') }}</RouterLink>
              <a href="/.well-known/openid-configuration" target="_blank" rel="noopener noreferrer" class="block px-4 py-2.5 text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors">{{ $t('nav.openidConfig') }}</a>
            </div>
          </li>

          <!-- Links -->
          <li class="relative" @mouseenter="openDropdown('links')" @mouseleave="closeDropdown()">
            <button class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors font-medium py-2">
              {{ $t('nav.links') }}
              <ChevronDown class="w-3.5 h-3.5" />
            </button>
            <div
              v-show="activeDropdown === 'links'"
              class="absolute top-full left-0 mt-1 min-w-[180px] bg-white border border-border rounded-lg shadow-lg py-1 z-50"
            >
              <RouterLink to="/privacy" class="block px-4 py-2.5 text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors">{{ $t('nav.privacy') }}</RouterLink>
              <RouterLink to="/terms" class="block px-4 py-2.5 text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors">{{ $t('nav.terms') }}</RouterLink>
            </div>
          </li>

          <!-- About Us -->
          <li class="relative" @mouseenter="openDropdown('about')" @mouseleave="closeDropdown()">
            <button class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors font-medium py-2">
              {{ $t('landing.about.title') }}
              <ChevronDown class="w-3.5 h-3.5" />
            </button>
            <div
              v-show="activeDropdown === 'about'"
              class="absolute top-full left-0 mt-1 min-w-[200px] bg-white border border-border rounded-lg shadow-lg py-1 z-50"
            >
              <a :href="settings.github_url" target="_blank" rel="noopener noreferrer" class="flex items-center gap-2 px-4 py-2.5 text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors">
                <Github class="w-4 h-4" />
                {{ $t('landing.about.github') }}
              </a>
              <div v-if="settings.contact_info" class="flex items-center gap-2 px-4 py-2.5 text-sm text-muted-foreground">
                <Mail class="w-4 h-4" />
                <span class="break-all">{{ settings.contact_info }}</span>
              </div>
            </div>
          </li>
        </ul>

        <!-- Right-side items -->
        <div class="flex items-center gap-1.5 sm:gap-3 shrink-0">
          <template v-if="auth.isLoggedIn">
            <RouterLink to="/me" class="hidden sm:inline-flex text-sm text-muted-foreground hover:text-foreground transition-colors font-medium px-3 py-2">{{ $t('nav.account') }}</RouterLink>
            <RouterLink v-if="auth.canShowDeveloperConsole" to="/developer" class="hidden sm:inline-flex text-sm text-muted-foreground hover:text-foreground transition-colors font-medium px-2 lg:px-3 py-2">{{ $t('nav.developers') }}</RouterLink>
            <RouterLink v-if="auth.isAdmin" to="/admin" class="text-sm bg-foreground text-white px-3 sm:px-4 py-2 rounded-full font-medium hover:bg-foreground/90 transition-colors">{{ $t('nav.admin') }}</RouterLink>
          </template>
          <template v-else>
            <RouterLink to="/login" class="hidden sm:inline-flex text-sm text-muted-foreground hover:text-foreground transition-colors font-medium px-3 py-2">{{ $t('nav.login') }}</RouterLink>
            <RouterLink to="/register" class="hidden sm:inline-flex text-sm bg-foreground text-white px-4 py-2 rounded-full font-medium hover:bg-foreground/90 transition-colors">{{ $t('nav.register') }}</RouterLink>
          </template>
          <button @click="toggleLocale" class="text-sm text-muted-foreground hover:text-foreground transition-colors font-medium px-2 py-1 rounded border border-border">
            {{ locale === 'zh' ? 'EN' : '中文' }}
          </button>
          <button class="md:hidden p-2 rounded-md hover:bg-muted transition-colors" @click="mobileOpen = !mobileOpen">
            <X v-if="mobileOpen" class="w-5 h-5" />
            <Menu v-else class="w-5 h-5" />
          </button>
        </div>
      </div>
    </nav>

    <!-- Mobile menu overlay -->
    <div v-if="mobileOpen" class="fixed inset-0 top-16 bg-white z-40 overflow-y-auto md:hidden">
      <div class="p-6 space-y-0">
        <!-- Product -->
        <div class="border-b border-border/50">
          <button
            class="flex items-center justify-between w-full py-4 text-lg font-medium"
            @click="toggleMobileSection('product')"
          >
            {{ $t('nav.product') }}
            <ChevronDown class="w-5 h-5 transition-transform duration-200" :class="{ 'rotate-180': mobileExpanded.product }" />
          </button>
          <div v-show="mobileExpanded.product" class="pb-3 pl-4 space-y-1">
            <RouterLink to="/features" class="block py-2.5 text-base text-muted-foreground hover:text-foreground transition-colors" @click="mobileOpen = false">{{ $t('nav.features') }}</RouterLink>
            <RouterLink to="/docs" class="block py-2.5 text-base text-muted-foreground hover:text-foreground transition-colors" @click="mobileOpen = false">{{ $t('nav.docs') }}</RouterLink>
          </div>
        </div>

        <!-- Developers -->
        <div class="border-b border-border/50">
          <button
            class="flex items-center justify-between w-full py-4 text-lg font-medium"
            @click="toggleMobileSection('developers')"
          >
            {{ $t('nav.developers') }}
            <ChevronDown class="w-5 h-5 transition-transform duration-200" :class="{ 'rotate-180': mobileExpanded.developers }" />
          </button>
          <div v-show="mobileExpanded.developers" class="pb-3 pl-4 space-y-1">
            <RouterLink to="/docs" class="block py-2.5 text-base text-muted-foreground hover:text-foreground transition-colors" @click="mobileOpen = false">{{ $t('nav.apiDocs') }}</RouterLink>
            <a href="/.well-known/openid-configuration" target="_blank" rel="noopener noreferrer" class="block py-2.5 text-base text-muted-foreground hover:text-foreground transition-colors" @click="mobileOpen = false">{{ $t('nav.openidConfig') }}</a>
          </div>
        </div>

        <!-- Links -->
        <div class="border-b border-border/50">
          <button
            class="flex items-center justify-between w-full py-4 text-lg font-medium"
            @click="toggleMobileSection('links')"
          >
            {{ $t('nav.links') }}
            <ChevronDown class="w-5 h-5 transition-transform duration-200" :class="{ 'rotate-180': mobileExpanded.links }" />
          </button>
          <div v-show="mobileExpanded.links" class="pb-3 pl-4 space-y-1">
            <RouterLink to="/privacy" class="block py-2.5 text-base text-muted-foreground hover:text-foreground transition-colors" @click="mobileOpen = false">{{ $t('nav.privacy') }}</RouterLink>
            <RouterLink to="/terms" class="block py-2.5 text-base text-muted-foreground hover:text-foreground transition-colors" @click="mobileOpen = false">{{ $t('nav.terms') }}</RouterLink>
          </div>
        </div>

        <!-- About Us -->
        <div class="border-b border-border/50">
          <button
            class="flex items-center justify-between w-full py-4 text-lg font-medium"
            @click="toggleMobileSection('about')"
          >
            {{ $t('landing.about.title') }}
            <ChevronDown class="w-5 h-5 transition-transform duration-200" :class="{ 'rotate-180': mobileExpanded.about }" />
          </button>
          <div v-show="mobileExpanded.about" class="pb-3 pl-4 space-y-1">
            <a :href="settings.github_url" target="_blank" rel="noopener noreferrer" class="flex items-center gap-2 py-2.5 text-base text-muted-foreground hover:text-foreground transition-colors" @click="mobileOpen = false">
              <Github class="w-4 h-4" />
              {{ $t('landing.about.github') }}
            </a>
            <div v-if="settings.contact_info" class="flex items-center gap-2 py-2.5 text-base text-muted-foreground">
              <Mail class="w-4 h-4" />
              <span class="break-all">{{ settings.contact_info }}</span>
            </div>
          </div>
        </div>

        <!-- Auth links for mobile -->
        <div class="pt-4 space-y-1">
          <template v-if="auth.isLoggedIn">
            <RouterLink to="/me" class="block py-3 text-lg font-medium" @click="mobileOpen = false">{{ $t('nav.account') }}</RouterLink>
            <RouterLink v-if="auth.canShowDeveloperConsole" to="/developer" class="block py-3 text-lg font-medium" @click="mobileOpen = false">{{ $t('nav.developers') }}</RouterLink>
            <RouterLink v-if="auth.isAdmin" to="/admin" class="block py-3 text-lg font-medium" @click="mobileOpen = false">{{ $t('nav.admin') }}</RouterLink>
          </template>
          <template v-else>
            <RouterLink to="/login" class="block py-3 text-lg font-medium" @click="mobileOpen = false">{{ $t('nav.login') }}</RouterLink>
            <RouterLink to="/register" class="block py-3 text-lg font-medium" @click="mobileOpen = false">{{ $t('nav.register') }}</RouterLink>
          </template>
        </div>
      </div>
    </div>

    <RouterView />
  </div>
</template>
