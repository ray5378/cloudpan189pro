<template>
  <n-space>
    <n-button type="warning" :loading="clearingCache" @click="confirmClearCasTargetCache">清空缓存表</n-button>
    <n-button :loading="rebuildingCache" @click="confirmRebuildCasTargetCache">重建缓存表</n-button>
  </n-space>
</template>

<script setup lang="ts">
import { NButton, NSpace } from 'naive-ui'
import { useCasCacheActions } from '../composables/useCasCacheActions'

const {
  clearingCache,
  rebuildingCache,
  confirmClearCasTargetCache,
  confirmRebuildCasTargetCache,
} = useCasCacheActions()
</script>
