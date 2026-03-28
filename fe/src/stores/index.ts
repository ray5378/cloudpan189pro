import { createPinia } from 'pinia'

// 创建 pinia 实例
export const pinia = createPinia()

// 导出所有 stores
export { useAuthStore } from './modules/auth'
export { useSystemStore } from './modules/system'
export { useThemeStore } from './modules/theme'
export { useUserStore } from './modules/user'

// 默认导出 pinia 实例
export default pinia
