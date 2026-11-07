import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { setUnauthorizedHandler } from './api/http'
import { useAuthStore } from './stores/auth'
import "./assets/scss/_main.scss";

const app = createApp(App)
app.use(createPinia())
app.use(router)

// Any 401 on a non-auth endpoint kicks the user back to /login.
setUnauthorizedHandler(() => {
  const auth = useAuthStore()
  auth.clear()
  const current = router.currentRoute.value
  const isAuthPage = current.name === 'login' || current.name === 'register'
  if (!isAuthPage) {
    router.push({ name: 'login', query: { redirect: current.fullPath } })
  }
})

app.mount('#app')