<template>
  <div v-if="enabled" ref="btnRef" class="google-btn" />
  <p v-else class="help has-text-grey has-text-centered">
    Google Sign-In is not configured
  </p>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'

const props = defineProps<{
  onCredential: (idToken: string) => void | Promise<void>
}>()

const btnRef = ref<HTMLElement>()
const enabled = !!import.meta.env.VITE_GOOGLE_CLIENT_ID

declare global {
  interface Window {
    google?: any
  }
}

let scriptPromise: Promise<void> | null = null

function loadGisScript(): Promise<void> {
  if (window.google?.accounts?.id) return Promise.resolve()
  if (scriptPromise) return scriptPromise
  scriptPromise = new Promise((resolve, reject) => {
    const s = document.createElement('script')
    s.src = 'https://accounts.google.com/gsi/client'
    s.async = true
    s.defer = true
    s.onload = () => resolve()
    s.onerror = () => reject(new Error('failed to load Google Identity Services'))
    document.head.appendChild(s)
  })
  return scriptPromise
}

onMounted(async () => {
  if (!enabled) return
  try {
    await loadGisScript()
    window.google.accounts.id.initialize({
      client_id: import.meta.env.VITE_GOOGLE_CLIENT_ID,
      callback: (response: { credential: string }) => {
        void props.onCredential(response.credential)
      },
    })
    window.google.accounts.id.renderButton(btnRef.value, {
      theme: 'filled_black',
      size: 'large',
      width: 320,
      text: 'continue_with',
      shape: 'rectangular',
    })
  } catch (e) {
    console.error('[GoogleButton]', e)
  }
})
</script>

<style scoped>
.google-btn {
  display: flex;
  justify-content: center;
}
</style>