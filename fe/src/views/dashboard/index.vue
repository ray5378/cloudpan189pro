<template>
  <div class="dashboard">
    <n-grid :cols="24" :x-gap="16" :y-gap="16">
      <!-- 个人信息展示区 -->
      <n-grid-item :span="12">
        <n-card title="个人信息" class="info-card">
          <n-descriptions
            :column="1"
            label-placement="left"
            label-style="width: 120px; font-weight: 500;"
          >
            <n-descriptions-item label="用户名">
              <n-text strong>{{ userInfo.username || '-' }}</n-text>
            </n-descriptions-item>
            <n-descriptions-item label="状态">
              <n-tag :type="getUserStatusType(userInfo.status)" size="small">
                {{ getUserStatusText(userInfo.status) }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="用户组">
              <n-text>{{ userInfo.groupName || '-' }}</n-text>
            </n-descriptions-item>
            <n-descriptions-item label="管理员权限">
              <n-tag :type="userInfo.isAdmin ? 'success' : 'default'" size="small">
                {{ userInfo.isAdmin ? '是' : '否' }}
              </n-tag>
            </n-descriptions-item>
          </n-descriptions>
        </n-card>
      </n-grid-item>

      <!-- 系统信息展示区 -->
      <n-grid-item :span="12">
        <n-card title="系统信息" class="info-card">
          <n-descriptions
            :column="1"
            label-placement="left"
            label-style="width: 120px; font-weight: 500;"
          >
            <n-descriptions-item label="站点名称">
              <n-text strong>{{ systemInfo.title || '-' }}</n-text>
            </n-descriptions-item>
            <n-descriptions-item label="站点URL">
              <n-text>{{ systemInfo.baseURL || '-' }}</n-text>
            </n-descriptions-item>
            <n-descriptions-item label="运行时间">
              <n-text>{{ systemInfo.runTimeHuman || '-' }}</n-text>
            </n-descriptions-item>
            <n-descriptions-item label="当前版本">
              <n-tag type="info" size="small">{{ appVersion }}</n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="WebDAV认证">
              <n-tag :type="systemInfo.enableAuth ? 'success' : 'warning'" size="small">
                {{ systemInfo.enableAuth ? '需要认证' : '无需认证' }}
              </n-tag>
            </n-descriptions-item>
          </n-descriptions>
        </n-card>
      </n-grid-item>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
import { NGrid, NGridItem, NCard, NDescriptions, NDescriptionsItem, NTag, NText } from 'naive-ui'
import { useSystemStore, useUserStore } from '@/stores'

const userStore = useUserStore()
const systemStore = useSystemStore()
const appVersion = (import.meta.env.VITE_APP_VERSION as string) || 'unknown'

// 用户信息
const userInfo = userStore.get()

// 系统信息
const systemInfo = systemStore.get()

// 获取用户状态类型
const getUserStatusType = (status: number) => {
  switch (status) {
    case 1:
      return 'success'
    case 2:
      return 'warning'
    case 0:
    default:
      return 'error'
  }
}

// 获取用户状态文本
const getUserStatusText = (status: number) => {
  switch (status) {
    case 1:
      return '正常'
    case 2:
      return '受限'
    case 0:
    default:
      return '禁用'
  }
}
</script>

<style scoped>
.dashboard {
  padding: 0;
  background: var(--n-color-target);
}

.info-card {
  height: 280px;
  background: var(--n-card-color);
  border: 1px solid var(--n-border-color);
  border-radius: 12px;
  box-shadow: 0 2px 8px rgb(0 0 0 / 6%);
}

.info-card :deep(.n-card__content) {
  height: calc(100% - 50px);
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.info-card :deep(.n-descriptions) {
  margin: 0;
}

.info-card :deep(.n-descriptions-item) {
  margin-bottom: 16px;
}

.info-card :deep(.n-descriptions-item:last-child) {
  margin-bottom: 0;
}

.info-card :deep(.n-descriptions-item__label) {
  color: var(--n-text-color-2);
}

.info-card :deep(.n-descriptions-item__content) {
  color: var(--n-text-color);
}

/* 响应式设计 */
@media (width <= 1200px) {
  .info-card {
    height: auto;
    min-height: 250px;
  }
}

@media (width <= 768px) {
  .info-card {
    height: auto;
    min-height: 220px;
  }

  .info-card :deep(.n-descriptions-item) {
    margin-bottom: 12px;
  }
}
</style>
