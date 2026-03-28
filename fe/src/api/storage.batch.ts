import { api, type ApiResponse } from '@/utils/api'

export interface BatchToggleAutoRefreshRequest {
  ids: number[]
  enableAutoRefresh: boolean
  autoRefreshDays?: number
  refreshInterval?: number
  refreshBeginAt?: string
  enableDeepRefresh?: boolean
}
export interface BatchRefreshRequest { ids: number[]; deep?: boolean }
export interface BatchModifyTokenRequest { ids: number[]; tokenId: number }

export const batchToggleAutoRefreshApi = (
  data: BatchToggleAutoRefreshRequest,
): Promise<ApiResponse<{ successCount: number; failCount: number }>> => {
  return api.post('/storage/batch_toggle_auto_refresh', data).then((res) => res.data)
}
export const batchRefreshApi = (
  data: BatchRefreshRequest,
): Promise<ApiResponse<{ successCount: number; failCount: number }>> => {
  return api.post('/storage/batch_refresh', data).then((res) => res.data)
}
export const batchModifyTokenApi = (
  data: BatchModifyTokenRequest,
): Promise<ApiResponse<{ successCount: number; failCount: number; failIds: number[] }>> => {
  return api.post('/storage/batch_modify_token', data).then((res) => res.data)
}
