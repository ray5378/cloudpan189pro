import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { localStg } from '@/utils/storage'
import type { StorageType } from '@/types/global'

const DEFAULT_SETTINGS: StorageType.PageAutoRefreshSetting = {
  autoRefreshEnabled: true, // 默认开启
  refreshInterval: 30, // 默认30秒
}

export const usePageAutoRefreshStore = defineStore('pageAutoRefresh', () => {
  // 从localStg读取设置
  const loadSettings = (): StorageType.PageAutoRefreshSetting => {
    const saved = localStg.get('pageAutoRefreshSetting')
    if (saved) {
      return {
        autoRefreshEnabled: saved.autoRefreshEnabled ?? DEFAULT_SETTINGS.autoRefreshEnabled,
        refreshInterval: saved.refreshInterval ?? DEFAULT_SETTINGS.refreshInterval,
      }
    }
    return { ...DEFAULT_SETTINGS }
  }

  // 保存设置到localStg
  const saveSettings = (settings: StorageType.PageAutoRefreshSetting) => {
    localStg.set('pageAutoRefreshSetting', settings)
  }

  // 响应式状态
  const autoRefreshEnabled = ref(DEFAULT_SETTINGS.autoRefreshEnabled)
  const refreshInterval = ref(DEFAULT_SETTINGS.refreshInterval)

  // 加载设置
  const load = () => {
    const settings = loadSettings()
    autoRefreshEnabled.value = settings.autoRefreshEnabled
    refreshInterval.value = settings.refreshInterval
  }

  // 更新设置
  const updateSettings = (settings: Partial<StorageType.PageAutoRefreshSetting>) => {
    if (settings.autoRefreshEnabled !== undefined) {
      autoRefreshEnabled.value = settings.autoRefreshEnabled
    }
    if (settings.refreshInterval !== undefined) {
      refreshInterval.value = settings.refreshInterval
    }
  }

  // 重置为默认设置
  const resetSettings = () => {
    autoRefreshEnabled.value = DEFAULT_SETTINGS.autoRefreshEnabled
    refreshInterval.value = DEFAULT_SETTINGS.refreshInterval
    saveSettings(DEFAULT_SETTINGS)
  }

  // 获取当前设置
  const getCurrentSettings = (): StorageType.PageAutoRefreshSetting => ({
    autoRefreshEnabled: autoRefreshEnabled.value,
    refreshInterval: refreshInterval.value,
  })

  // 监听变化并自动保存
  watch(
    [autoRefreshEnabled, refreshInterval],
    () => {
      saveSettings(getCurrentSettings())
    },
    { deep: true }
  )

  // 初始化时加载设置
  load()

  return {
    // 状态
    autoRefreshEnabled,
    refreshInterval,

    // 方法
    load,
    updateSettings,
    resetSettings,
    getCurrentSettings,

    // 常量
    DEFAULT_SETTINGS,
  }
})
