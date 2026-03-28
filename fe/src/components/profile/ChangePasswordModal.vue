<template>
  <n-modal v-model:show="showModal" preset="dialog" title="修改密码">
    <n-form
      ref="formRef"
      :model="passwordForm"
      :rules="passwordRules"
      label-placement="left"
      label-width="120px"
      require-mark-placement="right-hanging"
    >
      <n-form-item label="当前密码" path="oldPassword">
        <n-input
          v-model:value="passwordForm.oldPassword"
          type="password"
          placeholder="请输入当前密码"
          show-password-on="mousedown"
          maxlength="50"
        />
      </n-form-item>
      <n-form-item label="新密码" path="password">
        <n-input
          v-model:value="passwordForm.password"
          type="password"
          placeholder="请输入新密码"
          show-password-on="mousedown"
          maxlength="50"
        />
      </n-form-item>
      <n-form-item label="确认新密码" path="confirmPassword">
        <n-input
          v-model:value="passwordForm.confirmPassword"
          type="password"
          placeholder="请再次输入新密码"
          show-password-on="mousedown"
          maxlength="50"
        />
      </n-form-item>
    </n-form>
    <template #action>
      <n-space>
        <n-button @click="handleCancel">取消</n-button>
        <n-button type="primary" :loading="loading" @click="handleSubmit"> 确认修改 </n-button>
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
  type FormRules,
} from 'naive-ui'
import { modifyOwnPassword } from '@/api/user'
import type { ModifyOwnPasswordRequest } from '@/api/user'
import { useAuthStore } from '@/stores'
import { useRouter } from 'vue-router'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'update:show', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const message = useMessage()
const authStore = useAuthStore()
const router = useRouter()

// 控制弹窗显示
const showModal = ref(false)

// 表单引用
const formRef = ref<FormInst | null>(null)

// 加载状态
const loading = ref(false)

// 密码表单
const passwordForm = reactive({
  oldPassword: '',
  password: '',
  confirmPassword: '',
})

// 表单验证规则
const passwordRules: FormRules = {
  oldPassword: [
    {
      required: true,
      message: '请输入当前密码',
      trigger: ['input', 'blur'],
    },
  ],
  password: [
    {
      required: true,
      message: '请输入新密码',
      trigger: ['input', 'blur'],
    },
    {
      min: 6,
      message: '密码长度不能少于6位',
      trigger: ['input', 'blur'],
    },
  ],
  confirmPassword: [
    {
      required: true,
      message: '请再次输入新密码',
      trigger: ['input', 'blur'],
    },
    {
      validator: (_, value) => {
        if (value !== passwordForm.password) {
          return new Error('两次输入的密码不一致')
        }
        return true
      },
      trigger: ['input', 'blur'],
    },
  ],
}

// 监听props变化
watch(
  () => props.show,
  (newVal) => {
    showModal.value = newVal
    if (newVal) {
      // 打开弹窗时重置表单
      resetForm()
    }
  },
  { immediate: true }
)

// 监听弹窗状态变化
watch(showModal, (newVal) => {
  emit('update:show', newVal)
})

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate()
  loading.value = true

  const data: ModifyOwnPasswordRequest = {
    oldPassword: passwordForm.oldPassword,
    password: passwordForm.password,
  }

  modifyOwnPassword(data)
    .then((response) => {
      if (response.code === 200) {
        message.success('密码修改成功，请重新登录')
        showModal.value = false
        emit('success')

        // 延迟1秒后退出登录，让用户看到成功提示
        setTimeout(() => {
          authStore.logout()
          router.push('/@login')
        }, 1000)
      } else {
        message.error(response.msg || '密码修改失败')
      }
    })
    .catch((error) => {
      console.error('修改密码失败:', error)
      message.error('密码修改失败')
    })
    .finally(() => {
      loading.value = false
    })
}

// 取消操作
const handleCancel = () => {
  showModal.value = false
}

// 重置表单
const resetForm = () => {
  passwordForm.oldPassword = ''
  passwordForm.password = ''
  passwordForm.confirmPassword = ''
  formRef.value?.restoreValidation()
}
</script>
