import { h } from 'vue'
import { useModal } from 'naive-ui'
import { SubscribeMountModal } from '@/components/storage'

export function useSubscribeMount() {
  const modal = useModal()

  const show = (): Promise<{ success: boolean }> => {
    return new Promise((resolve) => {
      const modalInstance = modal.create({
        title: '订阅号资源挂载',
        preset: 'dialog',
        style: {
          width: 'auto',
          maxWidth: '1200px',
        },
        content: () =>
          h(SubscribeMountModal, {
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
