import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  login as loginApi,
  refreshToken as refreshTokenApi,
  type LoginRequest,
  type LoginResponse,
} from '@/api/auth'
import type { ApiResponse } from '@/utils/api'
import { localStg } from '@/utils/storage'
import { useUserStore } from '../user'

// Token刷新提前时间（5分钟）
const TOKEN_REFRESH_BUFFER = 5 * 60 * 1000
// Token自动刷新阈值（60分钟）
const TOKEN_AUTO_REFRESH_THRESHOLD = 60 * 60 * 1000

export const useAuthStore = defineStore('auth', () => {
  const userStore = useUserStore()

  const accessToken = ref<string>('')
  const refreshToken = ref<string>('')
  const expireTime = ref<number>(0)
  const isLogin = computed(() => !!accessToken.value && expireTime.value > Date.now())
  // 需要刷新token（距离过期时间小于60分钟）
  const requireRefreshToken = computed(
    () => expireTime.value - Date.now() < TOKEN_AUTO_REFRESH_THRESHOLD
  )

  const loading = ref<boolean>(false)

  const refreshLock = ref<boolean>(false)

  accessToken.value = localStg.get('token') || ''
  refreshToken.value = localStg.get('refreshToken') || ''
  expireTime.value = localStg.get('expireTime') || 0

  const login = (loginData: LoginRequest): Promise<ApiResponse<LoginResponse>> => {
    loading.value = true

    return loginApi(loginData)
      .then((res) => {
        if (res.data) {
          storeWithUser(res.data)
        }

        return res
      })
      .finally(() => {
        loading.value = false
      })
  }

  const logout = () => {
    userStore.clear()
    clear()
  }

  const doRefreshToken = () => {
    if (!requireRefreshToken.value) {
      return
    }

    if (refreshLock.value) {
      return
    }

    refreshLock.value = true

    refreshTokenApi({
      refreshToken: refreshToken.value,
    })
      .then((res) => {
        if (res.data) {
          storeWithUser(res.data)
        }
      })
      .finally(() => {
        refreshLock.value = false
      })
  }

  const storeWithUser = (loginResponse: LoginResponse) => {
    store(loginResponse)
    userStore.store(loginResponse.user)
  }

  const store = (data: { accessToken: string; refreshToken: string; expiresIn: number }) => {
    expireTime.value = Date.now() + data.expiresIn * 1000 - TOKEN_REFRESH_BUFFER
    accessToken.value = data.accessToken
    refreshToken.value = data.refreshToken

    localStg.set('token', accessToken.value)
    localStg.set('refreshToken', refreshToken.value)
    localStg.set('expireTime', expireTime.value)
  }

  const clear = () => {
    store({
      accessToken: '',
      refreshToken: '',
      expiresIn: 0,
    })
  }

  const getToken = () => {
    return accessToken.value
  }

  return {
    // 状态
    // user,
    // refreshTokenValue,
    loading,
    requireRefreshToken,

    // 计算属性
    // isAdmin,
    // username,
    // userId,

    // 方法
    // fetchUserInfo,
    // tryRefreshToken,
    // updateUserInfo,

    doRefreshToken,
    login,
    logout,
    getToken,

    isLogin,
  }
})
