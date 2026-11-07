import axios from 'axios'

const TOKEN_KEY = 'session_token'

export function getStoredToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setStoredToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

export const http = axios.create({
  headers: { 'Content-Type': 'application/json' },
})

// Inject the bearer token on every outgoing request.
http.interceptors.request.use((config) => {
  const token = getStoredToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Auth endpoints legitimately return 401 (bad credentials). Those must not
// trigger the global "session expired" handler or we'd redirect in a loop.
const AUTH_ENDPOINTS = [
  '/api/auth/login',
  '/api/auth/register',
  '/api/auth/google',
  '/api/auth/reset-password',
]

let onUnauthorized: (() => void) | null = null
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn
}

http.interceptors.response.use(
  (r) => r,
  (error) => {
    const url: string = error.config?.url ?? ''
    const isAuthEndpoint = AUTH_ENDPOINTS.some((p) => url.startsWith(p))
    if (error.response?.status === 401 && !isAuthEndpoint) {
      setStoredToken(null)
      onUnauthorized?.()
    }
    return Promise.reject(error)
  },
)