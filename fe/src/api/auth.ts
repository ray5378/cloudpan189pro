import { api, type ApiResponse } from '@/utils/api'

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  accessToken: string
  refreshToken: string
  tokenType: string
  expiresIn: number
  user: Models.User
}

export interface RefreshRequest {
  refreshToken: string
}

export interface RefreshResponse {
  accessToken: string
  refreshToken: string
  tokenType: string
  expiresIn: number
  user: Models.User
}

// 用户登录
export const login = (data: LoginRequest): Promise<ApiResponse<LoginResponse>> => {
  return api.post('/user/login', data).then((res) => res.data)
}

// 刷新访问令牌
export const refreshToken = (data: RefreshRequest): Promise<ApiResponse<RefreshResponse>> => {
  return api.post('/user/refresh_token', data).then((res) => res.data)
}
