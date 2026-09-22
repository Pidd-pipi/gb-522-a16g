import { storeToRefs } from 'pinia'
import { useAuthStore } from '@/stores/auth'

export function useAuth() {
  const store = useAuthStore(); const { user, authenticated } = storeToRefs(store)
  return { user, authenticated, login: store.login, logout: store.logout, canAnalyze: () => store.hasRole('analyst', 'admin'), canReview: () => store.hasRole('reviewer', 'admin'), hasRole: store.hasRole }
}
