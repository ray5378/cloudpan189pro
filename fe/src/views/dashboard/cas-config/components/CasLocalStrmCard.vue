<template>
  <n-card title="CAS本地STRM" size="small">
    <n-space vertical size="small">
      <n-text depth="3">这个模块用于给手动保存到 <code>/local_cas</code> 目录中的 <code>.cas</code> 文件生成对应的 <code>.strm</code> 文件。</n-text>
      <n-text depth="3">系统订阅流程自动下载到 <code>/local_cas</code> 的 CAS 会自动生成 STRM；这里主要覆盖手动放入本地目录的 CAS。</n-text>

      <n-form :model="sourceForm" label-placement="left" label-width="140px">
        <n-grid :cols="24" :x-gap="16" :y-gap="8">
          <n-grid-item :span="6">
            <n-form-item label="自动扫描本地CAS">
              <n-switch v-model:value="sourceForm.localCasAutoScanEnabled" />
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="6">
            <n-form-item label="扫描间隔(分钟)">
              <n-input-number
                v-model:value="sourceForm.localCasAutoScanIntervalMin"
                :min="1"
                :max="1440"
                :disabled="!sourceForm.localCasAutoScanEnabled"
                style="width: 100%"
              />
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="12">
            <n-form-item label="本地STRM操作">
              <n-space>
                <n-button @click="$emit('save-settings')">保存设置</n-button>
                <n-button type="primary" :loading="manualScanning" @click="$emit('manual-scan')">立即扫描并生成STRM</n-button>
              </n-space>
            </n-form-item>
          </n-grid-item>
          <n-grid-item :span="24">
            <n-form-item label="恢复文件到期清理">
              <n-space>
                <n-button type="warning" :loading="manualFallbackRecycleRunning" @click="$emit('manual-fallback-recycle')">立即触发一次到期清理</n-button>
                <n-text depth="3">立即执行与原自动清理相同的恢复记录回收链：按数据库中的到期恢复记录删除文件、清理空目录并处理回收站。</n-text>
              </n-space>
            </n-form-item>
          </n-grid-item>
        </n-grid>
      </n-form>

      <n-text depth="3">先点“保存设置”保存自动扫描开关和扫描间隔；“立即扫描并生成STRM”用于手动触发一次本地扫描。</n-text>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import {
  NButton,
  NCard,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NInputNumber,
  NSpace,
  NSwitch,
  NText,
} from 'naive-ui'

defineProps<{
  sourceForm: any
  manualScanning: boolean
  manualFallbackRecycleRunning: boolean
}>()

defineEmits<{
  (e: 'save-settings'): void
  (e: 'manual-scan'): void
  (e: 'manual-fallback-recycle'): void
}>()
</script>
