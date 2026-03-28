<template>
  <n-modal v-model:show="visible" preset="dialog" title="绑定用户组">
    <div style="margin: 20px 0">
      <n-alert type="info" style="margin-bottom: 20px">
        正在为用户 <strong>{{ userInfo?.username }}</strong> 绑定用户组
      </n-alert>
    </div>

    <n-form
      ref="formRef"
      :model="form"
      :rules="formRules"
      label-placement="left"
      label-width="80px"
      style="margin-top: 20px"
    >
      <n-form-item label="用户组" path="groupId">
        <n-select
          v-model:value="form.groupId"
          placeholder="请选择用户组"
          :options="groupOptions"
          :loading="groupLoading"
          clearable
        />
      </n-form-item>
    </n-form>

    <template #action>
      <n-space>
        <n-button @click="handleCancel">取消</n-button>
        <n-button type="primary" :loading="loading" @click="handleConfirm"> 确认绑定 </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import {
  NModal,
  NForm,
  NFormItem,
  NSelect,
  NButton,
  NSpace,
  NAlert,
  type FormInst,
  type FormRules,
  type SelectOption,
  useMessage,
} from 'naive-ui'
import { bindUserGroup, type BindGroupRequest } from '@/api/user'
import { getUserGroupList } from '@/api/usergroup'

interface Props {
  show: boolean
  userInfo?: Models.UserInfo | null
}

interface Emits {
  (e: 'update:show', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 消息提示
const message = useMessage()

// 控制弹窗显示
const visible = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value),
})

// 表单相关
const loading = ref(false)
const groupLoading = ref(false)
const formRef = ref<FormInst | null>(null)
const form = reactive({
  groupId: undefined as number | undefined,
})

// 用户组选项
const groupOptions = ref<SelectOption[]>([])

// 表单验证规则
const formRules: FormRules = {
  groupId: [{ required: true, message: '请选择用户组', trigger: 'change', type: 'number' }],
}

// 获取用户组列表
const fetchUserGroups = () => {
  groupLoading.value = true

  getUserGroupList({ noPaginate: true })
    .then((response) => {
      if (response.code === 200 && response.data) {
        // 添加默认用户组选项
        const options: SelectOption[] = [
          {
            label: '默认用户组',
            value: 0,
          },
        ]

        // 添加其他用户组选项
        if (response.data.data) {
          response.data.data.forEach((group) => {
            options.push({
              label: group.name,
              value: group.id,
            })
          })
        }

        groupOptions.value = options
      }
    })
    .catch((error) => {
      console.error('获取用户组列表失败:', error)
      message.error('获取用户组列表失败')
    })
    .finally(() => {
      groupLoading.value = false
    })
}

// 重置表单
const resetForm = () => {
  form.groupId = props.userInfo?.groupId || undefined
  formRef.value?.restoreValidation()
}

// 监听弹窗显示状态，显示时重置表单并获取用户组列表
watch(
  () => props.show,
  (newVal) => {
    if (newVal) {
      resetForm()
      fetchUserGroups()
    }
  }
)

// 取消操作
const handleCancel = () => {
  visible.value = false
}

// 确认绑定用户组
const handleConfirm = () => {
  if (!formRef.value || !props.userInfo) return

  // 验证表单
  formRef.value
    .validate()
    .then(() => {
      loading.value = true

      const requestData: BindGroupRequest = {
        userId: props.userInfo!.id,
        groupId: form.groupId,
      }

      // 调用绑定用户组API
      return bindUserGroup(requestData)
    })
    .then((response) => {
      if (response.code === 200) {
        message.success('用户组绑定成功')
        visible.value = false
        emit('success')
      } else {
        message.error(response.msg || '用户组绑定失败')
      }
    })
    .catch((error) => {
      console.error('绑定用户组失败:', error)
      if (error?.response?.data?.msg) {
        message.error(error.response.data.msg)
      } else {
        message.error('绑定用户组失败，请稍后重试')
      }
    })
    .finally(() => {
      loading.value = false
    })
}

// 组件挂载时获取用户组列表
onMounted(() => {
  fetchUserGroups()
})
</script>
