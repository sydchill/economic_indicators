<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1>Sign in</h1>
      <p class="muted">Currency Signal</p>

      <form @submit.prevent="onSubmit">
        <label>
          Email
          <input
            v-model="email"
            type="email"
            autocomplete="email"
            required
            :disabled="auth.loading"
          />
        </label>

        <label>
          Password
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            required
            :disabled="auth.loading"
          />
        </label>

        <p v-if="error" class="error">{{ error }}</p>

        <button type="submit" :disabled="auth.loading">
          {{ auth.loading ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>

      <template v-if="googleEnabled">
        <div class="divider">or</div>
        <GoogleButton :on-credential="handleGoogleCredential" />
      </template>

      <p class="footnote">
        <RouterLink :to="{ name: 'forgot-password' }">Forgot password?</RouterLink>
        ·
        <RouterLink :to="{ name: 'register' }">Create an account</RouterLink>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import GoogleButton from '@/components/GoogleButton.vue'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const email = ref('')
const password = ref('')
const error = ref('')
const googleEnabled = !!import.meta.env.VITE_GOOGLE_CLIENT_ID

async function onSubmit() {
  error.value = ''
  try {
    await auth.login(email.value, password.value)
    const redirect = (route.query.redirect as string) || '/dashboard'
    router.replace(redirect)
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? 'Sign in failed'
  }
}

async function handleGoogleCredential(idToken: string) {
  error.value = ''
  try {
    await auth.loginWithGoogle(idToken)
    const redirect = (route.query.redirect as string) || '/dashboard'
    router.replace(redirect)
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? 'Google sign-in failed'
  }
}
</script>