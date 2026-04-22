<template>
  <n-card title="恢复结果" size="small">
    <pre class="result-pre">{{ content }}</pre>
  </n-card>
</template>

<script setup lang="ts">
import { NCard } from 'naive-ui'
defineProps<{ content: string }>()
</script>

<style scoped>
.result-pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
  line-height: 1.6;
}
</style>
