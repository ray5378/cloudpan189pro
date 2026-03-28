declare namespace Enums {
  // 登录日志来源（method）
  type LoginMethod = 'web' | 'api' | 'app' | 'cli'
  // 登录日志事件类型（event）
  type LoginEvent = 'login' | 'refresh_token'
  // 登录日志状态（status）
  type LoginStatus = 'success' | 'failed' | 'blocked'

  // 媒体配置文件冲突策略
  type MediaFileConflictPolicy = 'skip' | 'replace'
}
