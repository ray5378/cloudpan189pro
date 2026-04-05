import { api, type ApiResponse } from '@/utils/api'

// ===== 登录日志相关接口 =====

// 登录日志列表查询参数
export interface LoginLogListQuery {
  currentPage?: number // 当前页码，默认为1
  pageSize?: number // 每页大小，默认为10
  noPaginate?: boolean // 是否不分页，默认false
  userId?: number // 用户ID
  username?: string // 用户名
  addr?: string // 客户端地址或IP
  method?: Enums.LoginMethod // 事件来源：web/api/app/cli
  event?: Enums.LoginEvent // 事件类型：login/refresh_token
  status?: Enums.LoginStatus // 状态：success/failed/blocked
  beginAt?: string // 开始时间(ISO8601)
  endAt?: string // 结束时间(ISO8601)
}

// ===== 登录日志接口 =====

// 获取登录日志列表
export const getLoginLogList = (
  params?: LoginLogListQuery
): Promise<ApiResponse<Models.PaginationResponse<Models.LoginLog>>> => {
  return api.get('/login_log/list', { params }).then((res) => res.data)
}

// 清理登录日志
export const cleanupLoginLogs = (): Promise<ApiResponse<{ deleted: number; retentionDays: number }>> => {
  return api.post('/login_log/cleanup').then((res) => res.data)
}
