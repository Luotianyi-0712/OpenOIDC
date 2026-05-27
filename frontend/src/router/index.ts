import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: () => import('@/layouts/DefaultLayout.vue'),
      children: [
        { path: '', component: () => import('@/pages/index.vue') },
        { path: 'features', component: () => import('@/pages/features.vue') },
        { path: 'docs', component: () => import('@/pages/docs.vue') },
        { path: 'privacy', component: () => import('@/pages/privacy.vue') },
        { path: 'terms', component: () => import('@/pages/terms.vue') },
      ],
    },
    {
      path: '/',
      component: () => import('@/layouts/AuthLayout.vue'),
      children: [
        { path: 'login', component: () => import('@/pages/login.vue') },
        { path: 'register', component: () => import('@/pages/register.vue') },
        { path: 'forgot-password', component: () => import('@/pages/forgot-password.vue') },
        { path: 'reset-password', component: () => import('@/pages/reset-password.vue') },
        { path: 'authorize', component: () => import('@/pages/authorize.vue') },
        { path: 'verify-email', component: () => import('@/pages/verify-email.vue') },
      ],
    },
    {
      path: '/me',
      component: () => import('@/layouts/DashboardLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', component: () => import('@/pages/me/index.vue') },
        { path: 'sessions', component: () => import('@/pages/me/sessions.vue') },
        { path: 'bindings', component: () => import('@/pages/me/bindings.vue') },
        { path: 'security', component: () => import('@/pages/me/security.vue') },
        { path: 'authorized', component: () => import('@/pages/me/authorized.vue') },
        { path: 'activity', component: () => import('@/pages/me/activity.vue') },
      ],
    },
    {
      path: '/developer',
      component: () => import('@/layouts/DeveloperLayout.vue'),
      meta: { requiresAuth: true, requiresDeveloper: true },
      children: [
        { path: '', component: () => import('@/pages/developer/index.vue') },
        { path: 'create', component: () => import('@/pages/developer/create.vue') },
        { path: 'apps/:id', component: () => import('@/pages/developer/apps/[id].vue') },
        { path: 'apps/:id/users', component: () => import('@/pages/developer/apps/[id]/users.vue') },
      ],
    },
    {
      path: '/admin',
      component: () => import('@/layouts/AdminLayout.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
      children: [
        { path: '', component: () => import('@/pages/admin/index.vue') },
        { path: 'users', component: () => import('@/pages/admin/users.vue') },
        { path: 'clients', component: () => import('@/pages/admin/clients.vue') },
        { path: 'security-rules', component: () => import('@/pages/admin/security-rules.vue') },
        { path: 'providers', component: () => import('@/pages/admin/providers.vue') },
        { path: 'settings', component: () => import('@/pages/admin/settings.vue') },
        { path: 'audit', component: () => import('@/pages/admin/audit.vue') },
        { path: 'keys', component: () => import('@/pages/admin/keys.vue') },
        { path: 'risk', component: () => import('@/pages/admin/risk.vue') },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      component: () => import('@/pages/not-found.vue'),
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (auth.loading) {
    await Promise.all([auth.fetchUser(), auth.fetchPublicSettings()])
  }
  if (to.meta.requiresAuth && !auth.isLoggedIn) return { path: '/login', query: { return_to: to.fullPath } }
  if (auth.isLoggedIn) await auth.fetchDeveloperStatus()
  if (to.meta.requiresAdmin && !auth.isAdmin) return '/'
  if (to.meta.requiresDeveloper && !auth.isLoggedIn) return { path: '/login', query: { return_to: to.fullPath } }
})

export default router
