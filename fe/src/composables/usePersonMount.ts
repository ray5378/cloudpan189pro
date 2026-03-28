import { h } from 'vue'
import { useModal } from 'naive-ui'
import { PersonMountModal } from '@/components/storage'

export function usePersonMount() {
  const modal = useModal()

  const show = (): Promise<{ success: boolean }> => {
    return new Promise((resolve) => {
      const modalInstance = modal.create({
        title: '个人文件夹挂载',
        preset: 'dialog',
        style: {
          width: 'auto',
          maxWidth: '1200px',
        },
        content: () =>
          h(PersonMountModal, {
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
