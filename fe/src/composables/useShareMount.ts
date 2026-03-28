import { h } from 'vue'
import { useModal } from 'naive-ui'
import { ShareMountModal } from '@/components/storage'

export function useShareMount() {
  const modal = useModal()

  const show = (): Promise<{ success: boolean }> => {
    return new Promise((resolve) => {
      const modalInstance = modal.create({
        title: '文件分享挂载',
        preset: 'dialog',
        style: {
          width: 'auto',
          maxWidth: '1200px',
        },
        content: () =>
          h(ShareMountModal, {
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
