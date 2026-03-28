import { h, ref } from 'vue'
import { useModal, NButton } from 'naive-ui'
import { MountPointBindModal, type MountItem } from '@/components/storage'
import type { AddStorageResponse } from '@/api/storage'

export function useMountPointBind() {
  const modal = useModal()

  const show = (
    items: MountItem[],
    options?: { defaultCloudToken?: number }
  ): Promise<AddStorageResponse[]> => {
    return new Promise((resolve) => {
      const contentRef = ref<InstanceType<typeof MountPointBindModal> | null>(null)

      const modalInstance = modal.create({
        title: '挂载点绑定',
        preset: 'dialog',
        style: {
          width: '1000px',
        },
        content: () =>
          h(MountPointBindModal, {
            ref: contentRef,
            items,
            defaultCloudToken: options?.defaultCloudToken,
            onConfirm: (payload) => {
              resolve(payload)
              modalInstance.destroy()
            },
            onCancel: () => {
              modalInstance.destroy()
            },
          }),
        action: () =>
          h('div', { style: 'display: flex; gap: 8px; justify-content: flex-end' }, [
            h(
              NButton,
              {
                onClick: () => {
                  modalInstance.destroy()
                },
              },
              { default: () => '取消' }
            ),
            h(
              NButton,
              {
                type: 'primary',
                loading: contentRef.value?.state.submitLoading,
                onClick: () => {
                  contentRef.value?.handleConfirm()
                },
              },
              { default: () => '确认挂载' }
            ),
          ]),
        closable: true,
        maskClosable: false,
      })
    })
  }

  return {
    show,
  }
}
