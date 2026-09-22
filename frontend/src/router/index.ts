import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: () => import('@/pages/LoginPage.vue'), meta: { public: true } },
    { path: '/', redirect: '/routes' },
    { path: '/routes', name: 'routes', component: () => import('@/pages/RoutesPage.vue') },
    { path: '/traces', name: 'traces', component: () => import('@/pages/TracesPage.vue') },
    { path: '/events', name: 'events', component: () => import('@/pages/EventsPage.vue') },
    { path: '/cases', name: 'cases', component: () => import('@/pages/CasesPage.vue') },
    { path: '/audit', name: 'audit', component: () => import('@/pages/AuditPage.vue'), meta: { roles: ['reviewer', 'admin'] } },
    { path: '/:pathMatch(.*)*', redirect: '/routes' },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.authenticated) return { name: 'login', query: { redirect: to.fullPath } }
  if (to.name === 'login' && auth.authenticated) return { name: 'routes' }
  const roles = to.meta.roles as string[] | undefined
  if (roles && !roles.includes(auth.user?.role ?? '')) return { name: 'routes' }
  return true
})

export default router
