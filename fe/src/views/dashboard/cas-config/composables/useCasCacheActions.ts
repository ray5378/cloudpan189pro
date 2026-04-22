import { ref } from 'vue'
import { useDialog, useMessage } from 'naive-ui'
import { clearCasTargetCache, rebuildCasTargetCache } from '@/api/setting'

export const useCasCacheActions = () => {
  const dialog = useDialog()
  const message = useMessage()
  const clearingCache = ref(false)
  const rebuildingCache = ref(false)

  const handleClearCasTargetCache = async () => {
    clearingCache.value = true
    try {
      const res = await clearCasTargetCache()
      message.success(`CAS 缓存表已清空，共删除 ${res.data?.deleted ?? 0} 条`)
    } catch (err: any) {
      message.error(err?.message || '清空 CAS 缓存表失败')
    } finally {
      clearingCache.value = false
    }
  }

  const confirmClearCasTargetCache = () => {
    dialog.warning({
      title: '确认清空缓存表？',
      content: '这会清空本地 CAS 目录缓存表，后续需要重新扫描/重建缓存。',
      positiveText: '确认清空',
      negativeText: '取消',
      onPositiveClick: () => handleClearCasTargetCache(),
    })
  }

  const handleRebuildCasTargetCache = async () => {
    rebuildingCache.value = true
    try {
      const res = await rebuildCasTargetCache()
      message.success(`CAS 缓存表已重建：刷新目录 ${res.data?.dirCount ?? 0} 个，写入条目 ${res.data?.itemCount ?? 0} 条`)
    } catch (err: any) {
      message.error(err?.message || '重建 CAS 缓存表失败')
    } finally {
      rebuildingCache.value = false
    }
  }

  const confirmRebuildCasTargetCache = () => {
    dialog.warning({
      title: '确认重建缓存表？',
      content: '这会只读扫描当前 CAS 目标目录和已缓存过的目标目录，并重建本地缓存；不会创建新的云盘目录。',
      positiveText: '开始重建',
      negativeText: '取消',
      onPositiveClick: () => handleRebuildCasTargetCache(),
    })
  }

  return {
    clearingCache,
    rebuildingCache,
    confirmClearCasTargetCache,
    confirmRebuildCasTargetCache,
  }
}
