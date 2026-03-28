<template>
  <n-modal v-model:show="showModal" preset="dialog" title="添加用户组">
    <n-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-placement="left"
      label-width="auto"
      require-mark-placement="right-hanging"
    >
      <n-form-item label="用户组名称" path="name">
        <n-input
          v-model:value="formData.name"
          placeholder="请输入用户组名称"
          maxlength="255"
          show-count
        />
      </n-form-item>
    </n-form>

    <template #action>
      <n-space>
        <n-button @click="handleCancel">取消</n-button>
        <n-button type="primary" :loading="loading" @click="handleSubmit">确定</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import {
  NModal,
  NForm,
  NFormItem,
  NInput,
  NButton,
  NSpace,
  useMessage,
  type FormInst,
} from 'naive-ui'
import { addUserGroup } from '@/api/usergroup'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'update:show', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 表单引用
const formRef = ref<FormInst | null>(null)

// 弹窗显示状态
const showModal = ref(false)

// 加载状态
const loading = ref(false)

// 消息提示
const message = useMessage()

// 表单数据
const formData = reactive({
  name: '',
})

// 表单验证规则
const rules = {
  name: [
    { required: true, message: '请输入用户组名称', trigger: 'blur' },
    { min: 1, max: 255, message: '用户组名称长度应在1-255位之间', trigger: 'blur' },
  ],
}

// 监听 props.show 变化
watch(
  () => props.show,
  (newVal) => {
    showModal.value = newVal
    if (newVal) {
      // 重置表单
      resetForm()
    }
  }
)

// 监听 showModal 变化
watch(showModal, (newVal) => {
  emit('update:show', newVal)
})

// 重置表单
const resetForm = () => {
  formData.name = ''
  formRef.value?.restoreValidation()
}

// 取消
const handleCancel = () => {
  showModal.value = false
}

// 提交
const handleSubmit = () => {
  formRef.value?.validate((errors) => {
    if (!errors) {
      loading.value = true

      addUserGroup({
        name: formData.name,
      })
        .then((response) => {
          if (response.code === 200) {
            message.success('添加用户组成功')
            showModal.value = false
            emit('success')
          } else {
            message.error(response.msg || '添加用户组失败')
          }
        })
        .catch((error) => {
          console.error('添加用户组失败:', error)
          message.error('添加用户组失败')
        })
        .finally(() => {
          loading.value = false
        })
    }
  })
}
</script>
