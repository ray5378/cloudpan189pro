<template>
  <n-card title="链路说明" size="small">
    <n-space vertical size="small">
      <n-text>reference-backed 支持组合：</n-text>
      <ul class="plain-list">
        <li>person → person</li>
        <li>family → family</li>
        <li>family → person</li>
      </ul>
      <n-text>当前不支持：</n-text>
      <ul class="plain-list">
        <li>person → family（暂无可直接照搬的 reference-backed 主链）</li>
      </ul>
      <n-text depth="3">targetFolderId 只表示最终目录 ID，不表示上传路线。</n-text>
      <n-text depth="3">账号密码继续复用原项目 cloud token，不在本页面单独配置。</n-text>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { NCard, NSpace, NText } from 'naive-ui'
</script>

<style scoped>
.plain-list {
  margin: 0;
  padding-left: 20px;
}
</style>
