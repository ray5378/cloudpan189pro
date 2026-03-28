/**
 * 时间相关工具函数
 */

import dayjs from 'dayjs'

/**
 * 格式化剩余时间
 * @param expiresIn 过期时间戳（毫秒）
 * @returns 格式化后的剩余时间字符串
 */
export const formatRemainingTime = (expiresIn: number | null | undefined): string => {
  if (!expiresIn) {
    return '永久有效'
  }

  const now = Date.now()
  const remainingTime = expiresIn - now

  if (remainingTime <= 0) {
    return '已过期'
  }

  const days = Math.floor(remainingTime / (1000 * 60 * 60 * 24))
  const hours = Math.floor((remainingTime % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  const minutes = Math.floor((remainingTime % (1000 * 60 * 60)) / (1000 * 60))
  const seconds = Math.floor((remainingTime % (1000 * 60)) / 1000)

  let result = ''
  if (days > 0) result += `${days}天`
  if (hours > 0) result += `${hours}时`
  if (minutes > 0) result += `${minutes}分`
  if (seconds > 0) result += `${seconds}秒`

  return result || '即将过期'
}

/**
 * 格式化时间戳为本地时间字符串
 * @param timestamp 时间戳（毫秒）
 * @returns 格式化后的时间字符串
 */
export const formatDateTime = (timestamp: number | string): string => {
  return dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss')
}
