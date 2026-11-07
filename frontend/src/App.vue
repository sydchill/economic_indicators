<template>
  <div v-if="!auth.ready" class="splash">Loading…</div>
  <RouterView v-else />
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

onMounted(() => {
  if (!auth.ready) auth.bootstrap()
})
</script>

<style scoped>
.splash {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  color: var(--text-muted);
  font-size: 14px;
}
</style>