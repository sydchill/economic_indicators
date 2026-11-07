import { http } from './http'
import type { LoginResponse, RegisterResponse, User } from '@/types'

export async function login(email: string, password: string): Promise<LoginResponse> {
  const { data } = await http.post<LoginResponse>('/api/auth/login', { email, password })
  return data
}

export async function register(
  email: string,
  password: string,
  displayName: string,
): Promise<RegisterResponse> {
  const { data } = await http.post<RegisterResponse>('/api/auth/register', {
    email,
    password,
    display_name: displayName,
  })
  return data
}

export async function loginWithGoogle(idToken: string): Promise<LoginResponse> {
  const { data } = await http.post<LoginResponse>('/api/auth/google', { id_token: idToken })
  return data
}

export async function logout(): Promise<void> {
  await http.post('/api/auth/logout')
}

export async function me(): Promise<User> {
  const { data } = await http.get<User>('/api/auth/me')
  return data
}

export async function forgotPassword(email: string): Promise<void> {
  await http.post('/api/auth/forgot-password', { email })
}

export async function resetPassword(token: string, password: string): Promise<void> {
  await http.post('/api/auth/reset-password', { token, password })
}