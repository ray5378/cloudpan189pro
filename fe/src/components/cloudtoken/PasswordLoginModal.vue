<template>
  <n-modal
    v-model:show="showModal"
    preset="dialog"
    :title="updateMode ? '更新密码令牌' : '密码登录'"
    :mask-closable="false"
  >
    <div class="password-login-container">
      <!-- 登录表单 -->
      <n-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-placement="left"
        label-width="80px"
        autocomplete="off"
      >
        <!-- 隐藏的假输入框，用于欺骗浏览器自动填充 -->
        <input type="text" style="display: none" autocomplete="username" />
        <input type="password" style="display: none" autocomplete="current-password" />

        <n-form-item label="用户名" path="username">
          <n-input
            v-model:value="form.username"
            :placeholder="
              updateMode ? '请输入天翼云盘用户名（可选，不填则保持原有）' : '请输入天翼云盘用户名'
            "
            clearable
            autocomplete="nope"
            name="fake-username"
            @keyup.enter="handleLogin"
          />
        </n-form-item>

        <n-form-item label="密码" path="password">
          <n-input
            v-model:value="form.password"
            type="password"
            :placeholder="
              updateMode ? '请输入天翼云盘密码（可选，不填则保持原有）' : '请输入天翼云盘密码'
            "
            show-password-on="click"
            clearable
            autocomplete="nope"
            name="fake-password"
            @keyup.enter="handleLogin"
          />
        </n-form-item>

        <n-form-item label="令牌名称" path="name">
          <n-input
            v-model:value="form.name"
            placeholder="请输入令牌名称（可选）"
            clearable
            @keyup.enter="handleLogin"
          />
        </n-form-item>
      </n-form>

      <!-- 登录说明 -->
      <div class="login-instructions">
        <n-alert type="info" title="登录说明">
          <ul>
            <li>请输入您的天翼云盘账号和密码</li>
            <li>令牌名称为可选项，用于标识此令牌</li>
            <li>登录成功后将自动获取访问令牌</li>
            <li>请确保账号密码正确，避免多次失败导致账号锁定</li>
          </ul>
        </n-alert>
      </div>

      <!-- 状态提示 -->
      <div v-if="statusMessage" class="status-message">
        <n-alert :type="statusType" :title="statusMessage" />
      </div>
    </div>

    <template #action>
      <n-space>
        <n-button @click="handleCancel">取消</n-button>
        <n-button type="primary" @click="handleLogin" :loading="loading"> 登录 </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, nextTick } from 'vue'
import {
  NModal,
  NForm,
  NFormItem,
  NInput,
  NButton,
  NSpace,
  NAlert,
  useMessage,
  type FormInst,
  type FormRules,
} from 'naive-ui'
import { usernameLogin } from '@/api/cloudtoken'

// Props
interface Props {
  show: boolean
  updateMode?: boolean // 是否为更新模式
  tokenData?: { id: number; [key: string]: unknown } | null // 更新时的令牌数据
}

// Emits
interface Emits {
  (e: 'update:show', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 响应式数据
const loading = ref(false)
const statusMessage = ref('')
const statusType = ref<'success' | 'info' | 'warning' | 'error'>('info')
const formRef = ref<FormInst>()

// 表单数据
const form = reactive({
  username: '',
  password: '',
  name: '',
})

// 表单验证规则
const rules = computed<FormRules>(() => {
  if (props.updateMode) {
    // 更新模式下，用户名和密码都是可选的
    return {
      username: [{ min: 1, max: 50, message: '用户名长度应在1-50个字符之间', trigger: 'blur' }],
      password: [{ min: 1, max: 100, message: '密码长度应在1-100个字符之间', trigger: 'blur' }],
      name: [{ max: 50, message: '令牌名称长度不能超过50个字符', trigger: 'blur' }],
    }
  } else {
    // 添加模式下，用户名和密码都是必填的
    return {
      username: [
        { required: true, message: '请输入用户名', trigger: 'blur' },
        { min: 1, max: 50, message: '用户名长度应在1-50个字符之间', trigger: 'blur' },
      ],
      password: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 1, max: 100, message: '密码长度应在1-100个字符之间', trigger: 'blur' },
      ],
      name: [{ max: 50, message: '令牌名称长度不能超过50个字符', trigger: 'blur' }],
    }
  }
})

// 消息提示
const message = useMessage()

// 计算属性
const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value),
})

// 监听弹窗显示状态
watch(showModal, (newVal) => {
  if (newVal) {
    // 弹窗打开时重置状态
    resetState()
    // 延迟清空输入框，防止自动填充
    nextTick(() => {
      setTimeout(() => {
        form.username = ''
        form.password = ''
        form.name = ''
      }, 100)
    })
  }
})

// 重置状态
const resetState = () => {
  form.username = ''
  form.password = ''
  form.name = ''
  statusMessage.value = ''
  loading.value = false

  // 清除表单验证状态
  formRef.value?.restoreValidation()
}

// 处理登录
const handleLogin = () => {
  formRef.value?.validate((errors) => {
    if (!errors) {
      performLogin()
    }
  })
}

// 执行登录
const performLogin = () => {
  loading.value = true
  statusMessage.value = ''

  const loginData = {
    username: form.username,
    password: form.password,
    name: form.name || undefined,
    ...(props.updateMode && props.tokenData?.id ? { id: props.tokenData.id } : {}),
  }

  usernameLogin(loginData)
    .then((response) => {
      if (response.code === 200) {
        // 登录成功
        statusMessage.value = '登录成功！'
        statusType.value = 'success'
        message.success('密码登录成功')

        // 延迟关闭弹窗并触发成功回调
        setTimeout(() => {
          showModal.value = false
          emit('success')
        }, 1500)
      } else {
        // 登录失败
        statusMessage.value = response.msg || '登录失败，请检查用户名和密码'
        statusType.value = 'error'
        message.error(response.msg || '登录失败')
      }
    })
    .catch((error) => {
      console.error('密码登录失败:', error)
      statusMessage.value = '登录失败，请检查网络连接或稍后重试'
      statusType.value = 'error'
      message.error('登录失败')
    })
    .finally(() => {
      loading.value = false
    })
}

// 取消操作
const handleCancel = () => {
  showModal.value = false
}
</script>

<style scoped>
.password-login-container {
  padding: 20px;
  min-height: 300px;
}

.login-instructions {
  margin: 20px 0;
}

.login-instructions ul {
  margin: 8px 0 0;
  padding-left: 20px;
}

.login-instructions li {
  margin-bottom: 4px;
  color: #666;
  font-size: 14px;
  line-height: 1.5;
}

.status-message {
  margin-top: 16px;
}
</style>
