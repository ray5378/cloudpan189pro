<template>
  <div class="extensions-page">
    <n-tabs type="line" :value="activeTab" @update:value="handleTabChange">
      <n-tab name="strm">STRM 生成</n-tab>
    </n-tabs>

    <!-- 子路由内容渲染 -->
    <div class="content">
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NTabs, NTab } from 'naive-ui'

const router = useRouter()
const route = useRoute()

const activeTab = ref<string>('strm')

// 根据当前路由初始化/同步选中的 Tab
watch(
  () => route.name,
  (name) => {
    if (name === 'ExtensionsMedia') {
      activeTab.value = 'strm'
    }
  },
  { immediate: true }
)

// 切换 Tab 时进行路由跳转
const handleTabChange = (name: 'strm') => {
  if (name === 'strm') {
    router.push({ name: 'ExtensionsMedia' })
  }
}
</script>

<style scoped>
.extensions-page {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.content {
  background: var(--n-card-color);
  border-radius: 6px;
  padding: 0;
}
</style>
