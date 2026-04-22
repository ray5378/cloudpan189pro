<template>
  <n-card title="CAS 手动恢复" size="small">
    <n-form :model="model" label-placement="left" label-width="140px">
      <n-grid :cols="24" :x-gap="16">
        <n-grid-item :span="8">
          <n-form-item label="输入模式">
            <n-select v-model:value="model.inputMode" :options="inputModeOptions" placeholder="选择输入模式" />
          </n-form-item>
        </n-grid-item>
        <n-grid-item :span="8">
          <n-form-item label="上传路线">
            <n-select v-model:value="model.uploadRoute" :options="uploadRouteOptions" placeholder="选择上传路线" />
          </n-form-item>
        </n-grid-item>
        <n-grid-item :span="8">
          <n-form-item label="最终目录类型">
            <n-select v-model:value="model.destinationType" :options="destinationTypeOptions" placeholder="选择最终目录类型" />
          </n-form-item>
        </n-grid-item>
      </n-grid>

      <n-grid :cols="24" :x-gap="16">
        <n-grid-item :span="12">
          <n-form-item label="目标目录 ID">
            <n-input v-model:value="model.targetFolderId" placeholder="最终目录 ID，例如 -11 或个人目录ID" />
          </n-form-item>
        </n-grid-item>
      </n-grid>

      <template v-if="model.inputMode === 'virtualId'">
        <n-grid :cols="24" :x-gap="16">
          <n-grid-item :span="12">
            <n-form-item label="CAS Virtual ID">
              <n-input-number v-model:value="model.casVirtualId" clearable placeholder="例如 1001" style="width: 100%" />
            </n-form-item>
          </n-grid-item>
        </n-grid>
      </template>

      <template v-else-if="model.inputMode === 'path'">
        <n-grid :cols="24" :x-gap="16">
          <n-grid-item :span="16">
            <n-form-item label="CAS 路径">
              <n-input v-model:value="model.casPath" placeholder="例如 /电影库/movie.cas" />
            </n-form-item>
          </n-grid-item>
        </n-grid>
      </template>

      <template v-else>
        <n-grid :cols="24" :x-gap="16">
          <n-grid-item :span="8">
            <n-form-item label="Storage ID">
              <n-input-number v-model:value="model.storageId" clearable style="width: 100%" />
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="8">
            <n-form-item label="MountPoint ID">
              <n-input-number v-model:value="model.mountPointId" clearable style="width: 100%" />
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="8">
            <n-form-item label="CAS Virtual ID">
              <n-input-number v-model:value="model.casVirtualId" clearable style="width: 100%" />
            </n-form-item>
          </n-grid-item>
        </n-grid>
        <n-grid :cols="24" :x-gap="16">
          <n-grid-item :span="8">
            <n-form-item label="CAS File ID">
              <n-input v-model:value="model.casFileId" placeholder="云端 CAS file id" />
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="8">
            <n-form-item label="CAS File Name">
              <n-input v-model:value="model.casFileName" placeholder="例如 movie.cas" />
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="8">
            <n-form-item label="CAS 路径（可选）">
              <n-input v-model:value="model.casPath" placeholder="例如 /电影库/movie.cas" />
            </n-form-item>
          </n-grid-item>
        </n-grid>
      </template>

      <n-space justify="end">
        <n-button @click="$emit('apply-defaults')">恢复默认配置</n-button>
        <n-button @click="$emit('reset')">重置本次输入</n-button>
        <n-button type="primary" :loading="restoring" @click="$emit('restore')">开始恢复</n-button>
      </n-space>
    </n-form>
  </n-card>
</template>

<script setup lang="ts">
import { NButton, NCard, NForm, NFormItem, NGrid, NGridItem, NInput, NInputNumber, NSelect, NSpace } from 'naive-ui'

defineProps<{
  model: any
  inputModeOptions: Array<{ label: string; value: string }>
  uploadRouteOptions: Array<{ label: string; value: string }>
  destinationTypeOptions: Array<{ label: string; value: string }>
  restoring: boolean
}>()

defineEmits<{
  (e: 'apply-defaults'): void
  (e: 'reset'): void
  (e: 'restore'): void
}>()
</script>
