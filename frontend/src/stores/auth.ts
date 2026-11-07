import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as authApi from '@/api/auth'
import { getStoredToken, setStoredToken } from '@/api/http'
import type { User } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const ready = ref(false)
  const loading = ref(false)
  const isAuthenticated = computed(() => user.value !== null)

  let bootstrapPromise: Promise<void> | null = null

  async function doBootstrap() {
    const token = getStoredToken()
    if (!token) {
      ready.value = true
      return
    }
    try {
      user.value = await authApi.me()
    } catch {
      user.value = null
      setStoredToken(null)
    } finally {
      ready.value = true
    }
  }

  async function bootstrap() {
    if (ready.value) return
    if (bootstrapPromise) return bootstrapPromise
    bootstrapPromise = doBootstrap()
    try {
      await bootstrapPromise
    } finally {
      bootstrapPromise = null
    }
  }

  async function login(email: string, password: string) {
    loading.value = true
    try {
      const res = await authApi.login(email, password)
      setStoredToken(res.session_token)
      user.value = res.user
    } finally {
      loading.value = false
    }
  }

  async function register(email: string, password: string, displayName: string) {
    loading.value = true
    try {
      await authApi.register(email, password, displayName)
      // Register returns no session token; auto-login with the same creds.
      const res = await authApi.login(email, password)
      setStoredToken(res.session_token)
      user.value = res.user
    } finally {
      loading.value = false
    }
  }

  async function loginWithGoogle(idToken: string) {
    loading.value = true
    try {
      const res = await authApi.loginWithGoogle(idToken)
      setStoredToken(res.session_token)
      user.value = res.user
    } finally {
      loading.value = false
    }
  }

  async function requestPasswordReset(email: string) {
    loading.value = true
    try {
      await authApi.forgotPassword(email)
    } finally {
      loading.value = false
    }
  }

  async function resetPassword(token: string, password: string) {
    loading.value = true
    try {
      await authApi.resetPassword(token, password)
    } finally {
      loading.value = false
    }
  }

  async function logout() {
    try {
      await authApi.logout()
    } catch {
      // Session may already be revoked
    } finally {
      setStoredToken(null)
      user.value = null
    }
  }

  function clear() {
    setStoredToken(null)
    user.value = null
  }

  return {
    user,
    ready,
    loading,
    isAuthenticated,
    bootstrap,
    login,
    register,
    loginWithGoogle,
    requestPasswordReset,
    resetPassword,
    logout,
    clear,
  }
})