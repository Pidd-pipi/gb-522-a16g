import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { authApi } from '@/api/domain'
import type { User } from '@/types/domain'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('otdr_token') ?? '')
  const stored = localStorage.getItem('otdr_user')
  const user = ref<User | null>(stored ? JSON.parse(stored) : null)
  const authenticated = computed(() => Boolean(token.value && user.value))

  async function login(username: string, password: string) {
    const { data } = await authApi.login({ username, password })
    token.value = data.data.token
    user.value = data.data.user
    localStorage.setItem('otdr_token', token.value)
    localStorage.setItem('otdr_user', JSON.stringify(user.value))
  }
  function logout() {
    token.value = ''; user.value = null
    localStorage.removeItem('otdr_token'); localStorage.removeItem('otdr_user')
  }
  function hasRole(...roles: User['role'][]) { return Boolean(user.value && roles.includes(user.value.role)) }
  return { token, user, authenticated, login, logout, hasRole }
})
