<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1>Create account</h1>
      <p class="muted">Currency Signal</p>

      <form @submit.prevent="onSubmit">
        <label>
          Display name
          <input
            v-model="displayName"
            type="text"
            autocomplete="name"
            :disabled="auth.loading"
          />
        </label>

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
            autocomplete="new-password"
            required
            :disabled="auth.loading"
          />
          <span class="hint">12+ characters, at least one letter and one digit</span>
        </label>

        <label>
          Confirm password
          <input
            v-model="confirm"
            type="password"
            autocomplete="new-password"
            required
            :disabled="auth.loading"
          />
        </label>

        <p v-if="error" class="error">{{ error }}</p>

        <button type="submit" :disabled="auth.loading">
          {{ auth.loading ? 'Creating account…' : 'Create account' }}
        </button>
      </form>

      <p class="footnote">
        Already have an account?
        <RouterLink :to="{ name: 'login' }">Sign in</RouterLink>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const displayName = ref('')
const email = ref('')
const password = ref('')
const confirm = ref('')
const error = ref('')

function validate(): string | null {
  if (password.value.length < 12) return 'Password must be at least 12 characters'
  if (password.value.length > 128) return 'Password must be 128 characters or fewer'
  if (!/[A-Za-z]/.test(password.value)) return 'Password must contain at least one letter'
  if (!/\d/.test(password.value)) return 'Password must contain at least one digit'
  if (password.value !== confirm.value) return 'Passwords do not match'
  return null
}

async function onSubmit() {
  error.value = ''
  const validationError = validate()
  if (validationError) {
    error.value = validationError
    return
  }
  try {
    await auth.register(email.value, password.value, displayName.value)
    router.replace({ name: 'dashboard' })
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? 'Registration failed'
  }
}
</script>

<style scoped>
.hint {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-muted);
}
</style>