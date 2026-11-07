<template>
  <div>
    <AppNavbar />

    <section class="section">
      <div class="container">

        <h2 class="title is-3">Welcome back</h2>

        <p v-if="auth.user" class="subtitle is-6 has-text-grey">
          Signed in as <strong class="has-text-white">{{ auth.user.email }}</strong>
          <span v-if="!auth.user.email_verified" class="tag is-danger is-light ml-2">
            email not verified
          </span>
        </p>

        <div class="box mt-5">
          <h3 class="title is-5">Dashboard coming next</h3>
          <p class="has-text-grey">
            The Asset Scores and Setups screens will live here.
          </p>
        </div>

      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import AppNavbar from '../components/Navigation.vue'

const auth = useAuthStore()
</script>