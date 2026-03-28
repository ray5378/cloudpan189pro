import { api, type ApiResponse } from '@/utils/api'

// ===== 云盘令牌相关接口 =====

// 初始化二维码响应接口
export interface InitQrcodeResponse {
  uuid: string // 二维码UUID
}

// 检查二维码请求接口
export interface CheckQrcodeRequest {
  uuid: string // 二维码UUID
  id?: number // 云盘令牌ID，可选
}

// 用户名登录请求接口
export interface UsernameLoginRequest {
  username?: string // 用户名 添加时必填
  password?: string // 密码 添加时必填
  id?: number // 云盘令牌ID，可选
  name?: string // 令牌名称，可选
}

export interface UsernameLoginResponse {
  id: number // 云盘令牌ID
}

// 修改名称请求接口
export interface ModifyNameRequest {
  id: number // 云盘令牌ID
  name: string // 新名称
}

// 删除请求接口
export interface DeleteCloudTokenRequest {
  id: number // 云盘令牌ID
}

// 云盘令牌列表查询参数
export interface CloudTokenListQuery {
  currentPage?: number // 当前页码，默认为1
  pageSize?: number // 每页大小，默认为10
  noPaginate?: boolean // 是否不分页，默认false
  name?: string // 名称模糊搜索
}

// ===== 云盘令牌管理接口 =====

// 初始化二维码
export const initQrcode = (): Promise<ApiResponse<InitQrcodeResponse>> => {
  return api.post('/cloud_token/init_qrcode').then((res) => res.data)
}

// 检查二维码状态
export const checkQrcode = (data: CheckQrcodeRequest): Promise<ApiResponse> => {
  return api.post('/cloud_token/check_qrcode', data).then((res) => res.data)
}

// 用户名密码登录
export const usernameLogin = (
  data: UsernameLoginRequest
): Promise<ApiResponse<UsernameLoginResponse>> => {
  return api.post('/cloud_token/username_login', data).then((res) => res.data)
}

// 修改云盘令牌名称
export const modifyCloudTokenName = (data: ModifyNameRequest): Promise<ApiResponse> => {
  return api.post('/cloud_token/modify_name', data).then((res) => res.data)
}

// 删除云盘令牌
export const deleteCloudToken = (data: DeleteCloudTokenRequest): Promise<ApiResponse> => {
  return api.post('/cloud_token/delete', data).then((res) => res.data)
}

// 获取云盘令牌列表
export const getCloudTokenList = (
  params?: CloudTokenListQuery
): Promise<ApiResponse<Models.PaginationResponse<Models.CloudToken>>> => {
  return api.get('/cloud_token/list', { params }).then((res) => res.data)
}

// 查询云盘令牌详情
export const getCloudTokenById = (id: number): Promise<ApiResponse<Models.CloudToken>> => {
  return api.get(`/cloud_token/${id}`).then((res) => res.data)
}
