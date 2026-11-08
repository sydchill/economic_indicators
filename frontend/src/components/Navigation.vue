<template>
  <nav class="navbar" role="navigation" aria-label="main navigation">
    <div class="navbar-brand">
      <RouterLink :to="{ name: 'dashboard' }" class="navbar-item logo">
        Macro Sentiment
      </RouterLink>

      <a
        role="button"
        class="navbar-burger"
        :class="{ 'is-active': menuOpen }"
        aria-label="menu"
        :aria-expanded="menuOpen"
        @click="menuOpen = !menuOpen"
      >
        <span aria-hidden="true"></span>
        <span aria-hidden="true"></span>
        <span aria-hidden="true"></span>
      </a>
    </div>

    <div class="navbar-menu" :class="{ 'is-active': menuOpen }">
      <div class="navbar-start">
        <RouterLink :to="{ name: 'dashboard' }" class="navbar-item nav-link">
          Dashboard
        </RouterLink>
        <RouterLink :to="{ name: 'scores' }" class="navbar-item nav-link">
          Asset Scores
        </RouterLink>
        <RouterLink :to="{ name: 'setups' }" class="navbar-item nav-link">
          Trade Setups
        </RouterLink>
      </div>
      <div class="navbar-end">
        <template v-if="auth.user">
          <div class="navbar-item">
            <span class="user-email">{{ auth.user.email }}</span>
          </div>
          <div class="navbar-item">
            <button class="button is-small is-light" @click="onLogout">Sign out</button>
          </div>
        </template>
        <template v-else>
          <div class="navbar-item">
            <RouterLink :to="{ name: 'login' }" class="button is-small is-light">
              Sign in
            </RouterLink>
          </div>
        </template>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const menuOpen = ref(false)

async function onLogout() {
  menuOpen.value = false
  await auth.logout()
  router.replace({ name: 'login' })
}
</script>

<style lang="scss" scoped>
nav.navbar {
  background-color: var(--surface-2);
  min-height: 3.25rem; // 52px — Bulma default, close to your 50px
}

// Brand + right-side items are always white on the accent background
.navbar-item,
.navbar-burger {
  color: #fff;
}

.navbar-item:hover,
.navbar-item:focus {
  background-color: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.logo {
  font-family: var(--font);
  font-weight: 600;
  font-size: 1rem;
}

.user-email {
  color: rgba(255, 255, 255, 0.8);
  font-size: 0.8125rem;
}

// When the burger is opened on mobile, the dropdown keeps the accent background
@media (max-width: 1023px) {
  .navbar-menu {
    background-color: var(--accent);
  }
}
</style>