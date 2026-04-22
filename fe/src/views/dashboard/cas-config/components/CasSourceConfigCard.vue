<template>
  <!--
    CAS 最终目录卡片
    规则：这里只负责最终目录相关 UI 展示与输入，不吸纳“恢复流程”“缓存管理逻辑实现”“跨卡片默认值同步”等职责。
    如果这里继续长胖，请优先拆更小的局部组件，而不是把逻辑堆回 index.vue。
  -->
  <n-card title="CAS 最终目录" size="small">
    <n-space vertical size="small">
      <n-text depth="3">这里配置的是 CAS 播放恢复时的最终落盘目录。订阅 `.cas` 会自动保存到本地 `/local_cas`，不再需要在这里手工配置本地来源目录。</n-text>
      <n-text depth="3">当目标类型为家庭云盘目录时，可通过“CAS指定恢复位置”固定家庭空间 ID，避免恢复时在多个家庭间跳来跳去。</n-text>

      <n-form :model="sourceForm" label-placement="left" label-width="140px">
        <n-grid :cols="24" :x-gap="16" :y-gap="8">
          <n-grid-item :span="6">
            <n-form-item label="启用 CAS 最终目录">
              <n-switch v-model:value="sourceForm.enabled" />
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="6">
            <n-form-item label="自动归集订阅 .cas">
              <n-switch v-model:value="sourceForm.autoCollectEnabled" />
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="6">
            <n-form-item label="保留订阅路径结构">
              <n-switch v-model:value="sourceForm.preservePath" />
            </n-form-item>
          </n-grid-item>
        </n-grid>

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
  NSwitch,
  NText,
} from 'naive-ui'

defineProps<{
  sourceForm: any
  cloudTokenOptions: Array<{ label: string; value: number }>
  sourceTypeOptions: Array<{ label: string; value: string }>
  familyOptions: Array<{ label: string; value: string }>
  cloudTokenLoading: boolean
  familyLoading: boolean
  sourceLoading: boolean
  sourcePathLabel: string
  currentFolderIdLabel: string
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
