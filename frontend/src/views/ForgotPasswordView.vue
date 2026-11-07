<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1>Forgot password</h1>
      <p class="muted">Enter your email and we'll send a reset link.</p>

      <form v-if="!sent" @submit.prevent="onSubmit">
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

        <p v-if="error" class="error">{{ error }}</p>

        <button type="submit" :disabled="auth.loading">
          {{ auth.loading ? 'Sending…' : 'Send reset link' }}
        </button>
      </form>

      <div v-else class="success">
        <p>
          If an account exists for <strong>{{ email }}</strong>, a reset link is on its way.
        </p>
        <p class="muted small">
          The link expires in 30 minutes. Check your spam folder if you don't see it.
        </p>
      </div>

      <p class="footnote">
        <RouterLink :to="{ name: 'login' }">Back to sign in</RouterLink>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const email = ref('')
const error = ref('')
const sent = ref(false)

async function onSubmit() {
  error.value = ''
  try {
    await auth.requestPasswordReset(email.value)
    sent.value = true
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? 'Something went wrong. Try again.'
  }
}
</script>

<style scoped>
.success {
  padding: 16px;
  border-radius: 8px;
  background: var(--surface-2);
}

.success .small {
  font-size: 12px;
}
</style>