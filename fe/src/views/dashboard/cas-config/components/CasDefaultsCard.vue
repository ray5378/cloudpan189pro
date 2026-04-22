<template>
  <n-card title="默认恢复配置" size="small">
    <n-form :model="model" label-placement="left" label-width="140px">
      <n-grid :cols="24" :x-gap="16">
        <n-grid-item :span="8">
          <n-form-item label="默认上传路线">
            <n-select v-model:value="model.uploadRoute" :options="uploadRouteOptions" placeholder="选择默认上传路线" />
          </n-form-item>
        </n-grid-item>
        <n-grid-item :span="8">
          <n-form-item label="默认最终目录类型">
            <n-select v-model:value="model.destinationType" :options="defaultDestinationOptions" placeholder="选择默认目录类型" />
          </n-form-item>
        </n-grid-item>
        <n-grid-item :span="8">
          <n-form-item label="默认目标目录 ID">
            <n-input v-model:value="model.targetFolderId" placeholder="例如 -11 或目录ID" />
          </n-form-item>
        </n-grid-item>
      </n-grid>

      <n-grid :cols="24" :x-gap="16">
        <n-grid-item :span="8">
          <n-form-item label="默认输入模式">
            <n-select v-model:value="model.inputMode" :options="inputModeOptions" placeholder="选择默认输入模式" />
          </n-form-item>
        </n-grid-item>
      </n-grid>

      <n-space justify="end">
        <n-button @click="$emit('reset')">恢复默认</n-button>
        <n-button type="primary" @click="$emit('save')">保存默认配置</n-button>
      </n-space>
    </n-form>
  </n-card>
</template>

<script setup lang="ts">
import { NButton, NCard, NForm, NFormItem, NGrid, NGridItem, NInput, NSelect, NSpace } from 'naive-ui'

defineProps<{
  model: any
  uploadRouteOptions: Array<{ label: string; value: string }>
  defaultDestinationOptions: Array<{ label: string; value: string }>
  inputModeOptions: Array<{ label: string; value: string }>
}>()

defineEmits<{
  (e: 'reset'): void
  (e: 'save'): void
}>()
</script>
