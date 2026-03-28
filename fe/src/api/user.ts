import { api, type ApiResponse } from '@/utils/api'

// 基础权限接口（需要登录）
export interface ModifyOwnPasswordRequest {
  oldPassword: string
  password: string
}

// 管理员权限接口
export interface AddUserRequest {
  username: string
  password: string
}

export interface AddUserResponse {
  id: number
}

export interface UpdateUserRequest {
  id: number
  status?: 1 | 2
}

export interface ToggleStatusRequest {
  id: number
  status: 1 | 2
}

export interface ModifyPasswordRequest {
  id: number
  password: string
}

export interface DeleteUserRequest {
  id: number
}

export interface BindGroupRequest {
  userId: number
  groupId?: number // 0表示默认用户组
}

export interface BindGroupResponse {
  userId: number
  groupId: number
  groupName: string
}

export interface UserListQuery {
  currentPage?: number
  pageSize?: number
  noPaginate?: boolean
  username?: string
}

// ===== 基础权限接口 =====

// 获取当前用户信息
export const getUserInfo = (): Promise<ApiResponse<Models.UserInfo>> => {
  return api.get('/user/info').then((res) => res.data)
}

// 修改当前用户密码
export const modifyOwnPassword = (data: ModifyOwnPasswordRequest): Promise<ApiResponse> => {
  return api.post('/user/modify_own_pass', data).then((res) => res.data)
}

// ===== 管理员权限接口 =====

// 添加用户
export const addUser = (data: AddUserRequest): Promise<ApiResponse<AddUserResponse>> => {
  return api.post('/user/add', data).then((res) => res.data)
}

// 删除用户
export const deleteUser = (data: DeleteUserRequest): Promise<ApiResponse> => {
  return api.post('/user/del', data).then((res) => res.data)
}

// 更新用户信息
export const updateUser = (data: UpdateUserRequest): Promise<ApiResponse> => {
  return api.post('/user/update', data).then((res) => res.data)
}

// 获取用户列表
export const getUserList = (
  params?: UserListQuery
): Promise<ApiResponse<Models.PaginationResponse<Models.UserInfo>>> => {
  return api.get('/user/list', { params }).then((res) => res.data)
}

// 修改用户密码
export const modifyUserPassword = (data: ModifyPasswordRequest): Promise<ApiResponse> => {
  return api.post('/user/modify_pass', data).then((res) => res.data)
}

// 绑定用户到用户组
export const bindUserGroup = (data: BindGroupRequest): Promise<ApiResponse<BindGroupResponse>> => {
  return api.post('/user/bind_group', data).then((res) => res.data)
}

// 切换用户状态
export const toggleUserStatus = (data: ToggleStatusRequest): Promise<ApiResponse> => {
  return api.post('/user/toggle_status', data).then((res) => res.data)
}
