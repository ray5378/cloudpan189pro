<template>
  <n-card title="CAS 最终目录" size="small">
    <n-space vertical size="small">
      <n-text depth="3">这里配置的是 CAS 播放恢复时的最终落盘目录。</n-text>
      <n-text depth="3">个人云盘目录配置绑定 family → person；家庭云盘目录配置绑定 family → family。</n-text>

      <n-form :model="sourceForm" label-placement="left" label-width="140px">
        <n-grid :cols="24" :x-gap="16" :y-gap="8">
          <n-grid-item :span="8">
            <n-form-item label="云盘账号">
              <n-select
                v-model:value="sourceForm.cloudToken"
                :options="cloudTokenOptions"
                placeholder="选择已添加的云盘账号"
                :loading="cloudTokenLoading"
                filterable
                clearable
              />
            </n-form-item>
          </n-grid-item>

          <n-grid-item :span="8">
            <n-form-item label="目标类型">
              <n-select v-model:value="sourceForm.sourceType" :options="sourceTypeOptions" placeholder="选择目标类型" />
            </n-form-item>
          </n-grid-item>

          <n-grid-item v-if="sourceForm.sourceType === 'family'" :span="8">
            <n-form-item label="CAS指定恢复位置">
              <n-select
                v-model:value="sourceForm.fixedFamilyId"
                :options="familyOptions"
                placeholder="选择固定家庭空间 ID"
                :loading="familyLoading"
                filterable
                clearable
              />
            </n-form-item>
          </n-grid-item>
        </n-grid>

        <n-grid :cols="24" :x-gap="16" :y-gap="8">
          <n-grid-item :span="8">
            <n-form-item label="恢复后文件留存时间">
              <n-select
                v-model:value="sourceForm.retentionHours"
                :options="retentionOptions"
                placeholder="选择留存时间"
              />
            </n-form-item>
          </n-grid-item>
        </n-grid>

        <n-grid :cols="24" :x-gap="16" :y-gap="8">
          <n-grid-item :span="12">
            <n-form-item label="当前最终目录">
              <n-input :value="sourcePathLabel" readonly />
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="12">
            <n-form-item label="当前最终目录 ID">
              <n-input :value="currentFolderIdLabel" readonly />
            </n-form-item>
          </n-grid-item>
        </n-grid>

        <n-grid v-if="sourceForm.sourceType === 'family'" :cols="24" :x-gap="16" :y-gap="8">
          <n-grid-item :span="12">
            <n-form-item label="家庭组对应目录 ID">
              <n-input :value="familyGroupFolderIdLabel" readonly placeholder="根据已选家庭组自动回显" />
            </n-form-item>
          </n-grid-item>
        </n-grid>

        <n-grid :cols="24" :x-gap="16" :y-gap="8">
          <n-grid-item :span="8">
            <n-form-item label="已保存最终目录 ID">
              <n-input :value="savedFolderIdLabel" readonly />
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="16">
            <n-form-item label="已保存最终目录路径">
              <n-input :value="savedCasPathLabel" readonly />
            </n-form-item>
          </n-grid-item>
        </n-grid>

        <n-space justify="space-between">
          <n-space>
            <n-button @click="$emit('load-root')">加载根目录</n-button>
            <n-button @click="$emit('go-parent')" :disabled="sourceFolderStack.length === 0">返回上级</n-button>
            <n-button type="primary" @click="$emit('save-source')">保存 CAS 最终目录</n-button>
          </n-space>
          <n-text depth="3">当前目录决定 `.strm` 播放时 CAS 恢复后的最终云盘落点；恢复仍优先使用本地 `/local_cas` 中的 `.cas` 文件。</n-text>
        </n-space>
      </n-form>

      <n-data-table :columns="sourceColumns" :data="sourceEntries" :loading="sourceLoading" :pagination="false" size="small" />
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import {
  NButton,
  NCard,
  NDataTable,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NInput,
  NSelect,
  NSpace,
  NText,
} from 'naive-ui'

defineProps<{
  sourceForm: any
  cloudTokenOptions: Array<{ label: string; value: number }>
  sourceTypeOptions: Array<{ label: string; value: string }>
  familyOptions: Array<{ label: string; value: string }>
  retentionOptions: Array<{ label: string; value: number }>
  cloudTokenLoading: boolean
  familyLoading: boolean
  sourceLoading: boolean
  sourcePathLabel: string
  currentFolderIdLabel: string
  familyGroupFolderIdLabel: string
  savedFolderIdLabel: string
  savedCasPathLabel: string
  sourceFolderStack: Array<{ id: string; name: string }>
  sourceColumns: any[]
  sourceEntries: any[]
}>()

defineEmits<{
  (e: 'load-root'): void
  (e: 'go-parent'): void
  (e: 'save-source'): void
}>()
</script>
