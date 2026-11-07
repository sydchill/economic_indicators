export interface User {
  id: string
  email: string
  email_verified: boolean
  status?: string
  created_at?: string
  display_name?: string | null
  avatar_url?: string | null
}

export interface LoginResponse {
  session_token: string
  expires_at: string
  user: User
}

export interface RegisterResponse {
  user_id: string
  email: string
}