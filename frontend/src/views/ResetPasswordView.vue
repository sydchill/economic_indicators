<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1>Reset password</h1>

      <template v-if="!token">
        <p class="muted">No reset token found in the URL.</p>
        <p class="error">
          Open the reset link from your email, or
          <RouterLink :to="{ name: 'forgot-password' }">request a new one</RouterLink>.
        </p>
      </template>

      <template v-else-if="done">
        <p class="muted">
          Password updated. You've been signed out of all sessions.
        </p>
        <RouterLink :to="{ name: 'login' }" class="button-link">Sign in</RouterLink>
      </template>

      <template v-else>
        <p class="muted">Choose a new password.</p>

        <form @submit.prevent="onSubmit">
          <label>
            New password
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
            Confirm new password
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
            {{ auth.loading ? 'Updating…' : 'Update password' }}
          </button>
        </form>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()

const token = computed(() => (route.query.token as string) || '')
const password = ref('')
const confirm = ref('')
const error = ref('')
const done = ref(false)

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
    await auth.resetPassword(token.value, password.value)
    // The backend revokes all sessions; clear any stored token locally too.
    auth.clear()
    done.value = true
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? 'Reset failed. The link may have expired.'
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

.button-link {
  display: inline-block;
  margin-top: 12px;
  padding: 10px 16px;
  background: var(--accent);
  color: #fff;
  border-radius: 6px;
  text-decoration: none;
  font-weight: 500;
}

.button-link:hover {
  background: var(--accent-hover);
}
</style>