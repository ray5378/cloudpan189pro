/**
 * osType 显示名称映射工具
 */
import { DocumentTextOutline, FolderOutline, PersonOutline, PeopleOutline } from '@vicons/ionicons5'

// osType 显示名称映射
const osTypeMap: Record<string, string> = {
  subscribe: '订阅号',
  subscribe_share_folder: '订阅分享',
  share_folder: '分享',
  person_folder: '个人',
  family_folder: '家庭',
}

// osType 常量定义
export const OS_TYPES = {
  SUBSCRIBE: 'subscribe',
  SUBSCRIBE_SHARE_FOLDER: 'subscribe_share_folder',
  SHARE_FOLDER: 'share_folder',
  PERSON_FOLDER: 'person_folder',
  FAMILY_FOLDER: 'family_folder',
} as const

// 挂载类型配置（用于新增挂载弹窗）
export const mountTypeConfigs = [
  {
    value: OS_TYPES.SUBSCRIBE,
    label: '订阅号',
    description: '天翼云盘资源分享的订阅号',
    icon: DocumentTextOutline,
    color: '#1976d2',
  },
  // {
  //   value: OS_TYPES.SUBSCRIBE_SHARE_FOLDER,
  //   label: '订阅分享文件夹',
  //   description: '订阅天翼云盘分享的文件夹',
  //   icon: ShareSocialOutline,
  //   color: '#7b1fa2'
  // },
  {
    value: OS_TYPES.SHARE_FOLDER,
    label: '文件分享',
    description: '挂载天翼云盘文件分享',
    icon: FolderOutline,
    color: '#f57c00',
  },
  {
    value: OS_TYPES.PERSON_FOLDER,
    label: '个人文件夹',
    description: '挂载个人天翼云盘文件夹',
    icon: PersonOutline,
    color: '#1976d2',
  },
  {
    value: OS_TYPES.FAMILY_FOLDER,
    label: '家庭云文件夹',
    description: '挂载家庭云盘文件夹',
    icon: PeopleOutline,
    color: '#c2185b',
  },
]

/**
 * 获取 osType 的显示名称
 * @param osType - 原始的 osType 值
 * @returns 对应的中文显示名称，如果没有匹配则返回原值
 */
export const getOsTypeDisplayName = (osType: string): string => {
  return osTypeMap[osType] || osType
}

/**
 * 获取所有支持的 osType 选项
 * @returns osType 选项数组
 */
export const getOsTypeOptions = () => {
  return Object.entries(osTypeMap).map(([value, label]) => ({
    value,
    label,
  }))
}

/**
 * 获取 osType 对应的自定义颜色
 * @param osType - osType 值
 * @returns 颜色配置对象
 */
export const getOsTypeColor = (osType: string) => {
  const colorMap: Record<string, { color: string; textColor: string }> = {
    subscribe: { color: '#e3f2fd', textColor: '#1976d2' }, // 蓝色系
    subscribe_share_folder: { color: '#f3e5f5', textColor: '#7b1fa2' }, // 紫色系
    share_folder: { color: '#fff3e0', textColor: '#f57c00' }, // 橙色系
    person_folder: { color: '#e3f2fd', textColor: '#1976d2' }, // 蓝色系
    family_folder: { color: '#fce4ec', textColor: '#c2185b' }, // 粉色系
  }
  return colorMap[osType] || { color: '#f5f5f5', textColor: '#666666' }
}
