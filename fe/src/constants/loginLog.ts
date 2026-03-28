/**
 * 登录日志（LoginLog）相关常量
 * 对应后端枚举：Enums.LoginEvent / Enums.LoginStatus
 */

// 事件
export const LOGIN_EVENTS = {
  LOGIN: 'login',
  REFRESH_TOKEN: 'refresh_token',
} as const
export type LoginEvent = (typeof LOGIN_EVENTS)[keyof typeof LOGIN_EVENTS]

// 事件选项（下拉）
export const LOGIN_EVENT_OPTIONS: { label: string; value: LoginEvent }[] = [
  { label: '登录', value: LOGIN_EVENTS.LOGIN },
  { label: '刷新令牌', value: LOGIN_EVENTS.REFRESH_TOKEN },
]

// 事件显示文本映射
export const LOGIN_EVENT_TEXT_MAP: Record<LoginEvent, string> = {
  [LOGIN_EVENTS.LOGIN]: '登录',
  [LOGIN_EVENTS.REFRESH_TOKEN]: '刷新令牌',
}

// 状态
export const LOGIN_STATUS = {
  SUCCESS: 'success',
  FAILED: 'failed',
  BLOCKED: 'blocked',
} as const
export type LoginStatus = (typeof LOGIN_STATUS)[keyof typeof LOGIN_STATUS]

// 状态选项（下拉）
export const LOGIN_STATUS_OPTIONS: { label: string; value: LoginStatus }[] = [
  { label: '成功', value: LOGIN_STATUS.SUCCESS },
  { label: '失败', value: LOGIN_STATUS.FAILED },
  { label: '拦截', value: LOGIN_STATUS.BLOCKED },
]

// 状态显示文本映射
export const LOGIN_STATUS_TEXT_MAP: Record<LoginStatus, string> = {
  [LOGIN_STATUS.SUCCESS]: '成功',
  [LOGIN_STATUS.FAILED]: '失败',
  [LOGIN_STATUS.BLOCKED]: '拦截',
}

// 状态对应的 Naive UI Tag 类型映射
export const LOGIN_STATUS_TAG_MAP: Record<
  LoginStatus,
  'default' | 'success' | 'error' | 'warning' | 'info'
> = {
  [LOGIN_STATUS.SUCCESS]: 'success',
  [LOGIN_STATUS.FAILED]: 'error',
  [LOGIN_STATUS.BLOCKED]: 'warning',
}
