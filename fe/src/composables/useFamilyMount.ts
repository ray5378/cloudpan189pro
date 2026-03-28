import { h } from 'vue'
import { useModal } from 'naive-ui'
import { FamilyMountModal } from '@/components/storage'

export function useFamilyMount() {
  const modal = useModal()

  const show = (): Promise<{ success: boolean }> => {
    return new Promise((resolve) => {
      const modalInstance = modal.create({
        title: '家庭文件夹挂载',
        preset: 'dialog',
        style: {
          width: 'auto',
          maxWidth: '1200px',
        },
        content: () =>
          h(FamilyMountModal, {
            onConfirm: (payload) => {
              resolve(payload)
              modalInstance.destroy()
            },
            onCancel: () => {
              resolve({ success: false })
              modalInstance.destroy()
            },
          }),
        action: () => null, // 动作按钮已在内容组件中处理
        closable: true,
        maskClosable: false,
      })
    })
  }

  return {
    show,
  }
}
