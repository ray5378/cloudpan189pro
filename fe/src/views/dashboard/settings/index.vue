<template>
  <div class="settings-page">
    <div class="page-title">站点信息</div>
    <section class="settings-section">
      <div class="setting-item">
        <div class="item-left">
          <div class="item-title">网站名称</div>
          <div class="item-desc">用于站点标题展示</div>
        </div>
        <div class="item-right">
          <div class="right-inline">
            <n-input
              v-model:value="form.title"
              placeholder="请输入网站名称"
              clearable
              maxlength="50"
              show-count
              style="width: 320px"
              @keyup.enter="handleSaveTitle"
            />
            <n-button
              type="primary"
              size="small"
              :loading="savingTitle"
              :disabled="!isTitleChanged"
              @click="handleSaveTitle"
            >
              保存
            </n-button>
          </div>
        </div>
      </div>

      <div class="setting-item">
        <div class="item-left">
          <div class="item-title">基础 URL</div>
          <div class="item-desc">用于生成外部链接等场景</div>
        </div>
        <div class="item-right">
          <div class="right-inline">
            <n-button size="small" @click="autoDetectBaseURL">自动获取</n-button>
            <n-input
              v-model:value="form.baseURL"
              placeholder="http://example.com"
              clearable
              style="width: 320px"
            />
            <n-button
              size="small"
              type="primary"
              :loading="savingBaseURL"
              :disabled="!isBaseURLChanged"
              @click="handleSaveBaseURL"
            >
              保存
            </n-button>
          </div>
        </div>
      </div>

      <div class="setting-item">
        <div class="item-left">
          <div class="item-title">系统运行时间</div>
          <div class="item-desc">服务已运行时长</div>
        </div>
        <div class="item-right">
          <n-text>{{ systemInfo.runTimeHuman || '-' }}</n-text>
        </div>
      </div>
    </section>

    <div class="page-title">系统设置</div>
    <section class="settings-section">
      <!-- External API -->
      <div class="setting-item">
        <div class="item-left">
          <div class="item-title">External API Key</div>
          <div class="item-desc">用于 /api/external/create-storage 鉴权（Header: X-API-Key 或 Body: apiKey/api-key）。</div>
        </div>
        <div class="item-right">
          <div class="right-inline">
            <n-input
              v-model:value="additionForm.externalApiKey"
              :type="showExternalApiKey ? 'text' : 'password'"
              placeholder="请输入 External API Key"
              clearable
              style="width: 320px"
            />
            <n-button size="small" @click="showExternalApiKey = !showExternalApiKey">
              {{ showExternalApiKey ? '隐藏' : '显示' }}
            </n-button>
            <n-button size="small" @click="generateExternalApiKey">生成</n-button>
            <n-button
              size="small"
              type="primary"
              :loading="savingExternalApiKey"
              @click="handleSaveExternalApiKey"
            >
              保存
            </n-button>
          </div>
        </div>
      </div>

      <div class="setting-item">
        <div class="item-left">
          <div class="item-title">默认云盘令牌（tianyi）</div>
          <div class="item-desc">外部创建挂载时默认绑定的 tokenId（可在请求体 tokenId 覆盖）。</div>
        </div>
        <div class="item-right">
          <div class="right-inline">
            <n-select
              v-model:value="additionForm.defaultTokenId"
              :options="cloudTokenOptions"
              :loading="loadingCloudTokens"
              clearable
              placeholder="请选择默认令牌"
              style="width: 320px"
            />
            <n-button
              size="small"
              type="primary"
              :loading="savingDefaultToken"
              @click="handleSaveDefaultTokenId"
            >
              保存
            </n-button>
          </div>
        </div>
      </div>

      <div class="setting-item">
        <div class="item-left">
          <div class="item-title">外部创建默认自动刷新</div>
          <div class="item-desc">通过 external 接口创建挂载时默认的自动刷新策略。</div>
        </div>
        <div class="item-right">
          <div class="right-inline">
            <n-switch v-model:value="additionForm.externalAutoRefreshEnabled" />
            <n-input-number
              v-model:value="additionForm.externalRefreshIntervalMin"
              :min="30"
              :max="1440"
              :step="30"
              placeholder="间隔(分钟)"
              style="width: 140px"
            />
            <n-input-number
              v-model:value="additionForm.externalAutoRefreshDays"
              :min="1"
              :max="365"
              :step="1"
              placeholder="天数"
              style="width: 120px"
            />
            <n-button
              size="small"
              type="primary"
              :loading="savingExternalAutoRefresh"
              @click="handleSaveExternalAutoRefresh"
            >
              保存
            </n-button>
          </div>
        </div>
      </div>

      <n-divider />

      <!-- 用户认证 -->
      <div class="setting-item">
        <div class="item-left">
          <div class="item-title">用户认证</div>
          <div class="item-desc">开启后访问 WebDAV 需要用户登录认证</div>
        </div>
        <div class="item-right">
          <n-switch
            v-model:value="enableAuth"
            :loading="savingEnableAuth"
            @update:value="handleToggleEnableAuth"
          />
        </div>
      </div>

      <!-- 本地代理 -->
      <div class="setting-item">
        <div class="item-left">
          <div class="item-title">本地代理</div>
          <div class="item-desc">
            系统默认通过 302 跳转方式提供资源，开启后服务器代理获取资源再转发给用户
          </div>
        </div>
        <div class="item-right">
          <n-switch
            v-model:value="additionForm.localProxy"
            :loading="savingLocalProxy"
            @update:value="handleToggleLocalProxy"
          />
        </div>
      </div>

      <!-- 多线程流式下载 -->
      <div class="setting-item">
        <div class="item-left">
          <div class="item-title">多线程流式下载</div>
          <div class="item-desc">
            通过多连接分片下载大幅提升下载速度，为本地代理的增强版本。优先级大于本地代理（如果同时开启）
          </div>
        </div>
        <div class="item-right">
          <n-switch
            v-model:value="additionForm.multipleStream"
            :loading="savingMultipleStream"
            @update:value="handleToggleMultipleStream"
          />
        </div>
      </div>

      <!-- 多线程数量 -->
      <div v-if="additionForm.multipleStream" class="setting-item">
        <div class="item-left">
          <div class="item-title">多线程数量</div>
          <div class="item-desc">建议 4-8 线程，过多可能受限于网络或服务端限制</div>
        </div>
        <div class="item-right">
          <div class="right-inline">
            <n-slider
              v-model:value="additionForm.multipleStreamThreadCount"
              :min="1"
              :max="64"
              :step="1"
              :marks="threadCountMarks"
              :format-tooltip="formatThreadTooltip"
              style="width: 340px"
              @change="handleThreadCountChange"
            />
            <n-button
              size="small"
              type="primary"
              :loading="savingThreadCount"
              @click="handleSaveThreadCount"
            >
              保存
            </n-button>
          </div>
        </div>
      </div>

      <!-- 分片大小 -->
      <div v-if="additionForm.multipleStream" class="setting-item">
        <div class="item-left">
          <div class="item-title">分片大小</div>
          <div class="item-desc">步长 512 KiB，推荐 ≥ 1 MB。分片越大，对网络稳定性要求越高</div>
        </div>
        <div class="item-right">
          <div class="right-inline">
            <n-slider
              v-model:value="additionForm.multipleStreamChunkSize"
              :min="1048576"
              :max="67108864"
              :step="524288"
              :marks="chunkSizeMarks"
              :format-tooltip="formatChunkTooltip"
              style="width: 340px"
              @change="handleChunkSizeChange"
            />
            <n-button
              size="small"
              type="primary"
              :loading="savingChunkSize"
              @click="handleSaveChunkSize"
            >
              保存
            </n-button>
          </div>
        </div>
      </div>

      <!-- 任务线程数 -->
      <div v-if="additionForm.multipleStream" class="setting-item">
        <div class="item-left">
          <div class="item-title">任务线程数</div>
          <div class="item-desc">同时执行的下载任务数，建议按设备性能与带宽适度调整</div>
        </div>
        <div class="item-right">
          <div class="right-inline">
            <n-slider
              v-model:value="additionForm.taskThreadCount"
              :min="1"
              :max="32"
              :step="1"
              :marks="taskThreadMarks"
              :format-tooltip="formatTaskThreadTooltip"
              style="width: 340px"
              @change="handleTaskThreadChange"
            />
            <n-button
              size="small"
              type="primary"
              :loading="savingTaskThreads"
              @click="handleSaveTaskThreads"
            >
              保存
            </n-button>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watchEffect, onMounted, computed } from 'vue'
import { NInput, NButton, NText, useMessage, NSlider, NSwitch, NSelect, NInputNumber, NDivider } from 'naive-ui'
import { useSystemStore } from '@/stores'
import {
  modifySystemTitle,
  modifySystemBaseURL,
  getSystemInfo,
  getSettingAddition,
  modifySettingAddition,
  toggleSystemEnableAuth,
} from '@/api/setting'
import { getCloudTokenList } from '@/api/cloudtoken'
import { formatFileSize } from '@/utils/format'

const message = useMessage()

const systemStore = useSystemStore()
const systemInfo = systemStore.get()

// 表单状态
const form = reactive({
  title: systemInfo.title || '',
  baseURL: systemInfo.baseURL || '',
})

watchEffect(() => {
  form.title = systemInfo.title || ''
  form.baseURL = systemInfo.baseURL || ''
})

// ===== 用户认证开关 =====
const enableAuth = ref<boolean>(systemInfo.enableAuth || false)
const savingEnableAuth = ref(false)
const handleToggleEnableAuth = (val: boolean) => {
  savingEnableAuth.value = true
  toggleSystemEnableAuth(val)
    .then((res) => {
      if (res.code === 200) {
        message.success('设置已保存')
        // 刷新系统信息
        systemStore
          .refresh()
          .then(() => {})
          .catch(() => {})
          .finally(() => {})
      } else {
        message.error(res.msg || '保存失败')
        enableAuth.value = !val // 失败时回滚显示
      }
    })
    .catch((err) => {
      message.error(err instanceof Error ? err.message : '网络错误')
      enableAuth.value = !val
    })
    .finally(() => {
      savingEnableAuth.value = false
    })
}

// ===== 附加设置表单 =====
const originalAddition = ref<Models.SettingAddition | null>(null)
const additionForm = reactive<Models.SettingAddition>({
  localProxy: false,
  multipleStream: false,
  multipleStreamThreadCount: 4,
  multipleStreamChunkSize: 4 * 1024 * 1024, // 4 MiB
  taskThreadCount: 1,

  externalApiKey: '',
  defaultTokenId: undefined,
  externalAutoRefreshEnabled: true,
  externalRefreshIntervalMin: 60,
  externalAutoRefreshDays: 60,
})

// 初始化完成标记，防止初始渲染触发自动保存
const additionLoaded = ref(false)

// 单字段保存的 loading 状态
const savingLocalProxy = ref(false)
const savingMultipleStream = ref(false)
const savingThreadCount = ref(false)
const savingChunkSize = ref(false)
const savingTaskThreads = ref(false)

// ===== External API / Default Token / Auto refresh defaults =====
const savingExternalApiKey = ref(false)
const showExternalApiKey = ref(false)
const cloudTokenOptions = ref<Array<{ label: string; value: number }>>([])
const loadingCloudTokens = ref(false)
const savingDefaultToken = ref(false)
const savingExternalAutoRefresh = ref(false)

const generateExternalApiKey = () => {
  // 32 chars strong random (URL-safe-ish + symbols)
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*_-'
  let out = ''
  const buf = new Uint32Array(32)
  crypto.getRandomValues(buf)
  for (let i = 0; i < buf.length; i++) {
    out += chars[buf[i] % chars.length]
  }
  additionForm.externalApiKey = out
}

const handleSaveExternalApiKey = () => {
  const key = (additionForm.externalApiKey || '').trim()
  savingExternalApiKey.value = true
  modifySettingAddition({ externalApiKey: key })
    .then((res) => {
      if (res.code === 200) {
        message.success('已保存')
        originalAddition.value = { ...additionForm }
      } else {
        message.error(res.msg || '保存失败')
      }
    })
    .catch((err) => {
      message.error(err instanceof Error ? err.message : '网络错误')
    })
    .finally(() => {
      savingExternalApiKey.value = false
    })
}

const handleSaveDefaultTokenId = () => {
  savingDefaultToken.value = true
  modifySettingAddition({ defaultTokenId: additionForm.defaultTokenId as number })
    .then((res) => {
      if (res.code === 200) {
        message.success('已保存')
        originalAddition.value = { ...additionForm }
      } else {
        message.error(res.msg || '保存失败')
      }
    })
    .catch((err) => {
      message.error(err instanceof Error ? err.message : '网络错误')
    })
    .finally(() => {
      savingDefaultToken.value = false
    })
}

const handleSaveExternalAutoRefresh = () => {
  savingExternalAutoRefresh.value = true
  modifySettingAddition({
    externalAutoRefreshEnabled: !!additionForm.externalAutoRefreshEnabled,
    externalRefreshIntervalMin: Number(additionForm.externalRefreshIntervalMin || 60),
    externalAutoRefreshDays: Number(additionForm.externalAutoRefreshDays || 60),
  })
    .then((res) => {
      if (res.code === 200) {
        message.success('已保存')
        originalAddition.value = { ...additionForm }
      } else {
        message.error(res.msg || '保存失败')
      }
    })
    .catch((err) => {
      message.error(err instanceof Error ? err.message : '网络错误')
    })
    .finally(() => {
      savingExternalAutoRefresh.value = false
    })
}

const loadCloudTokens = () => {
  loadingCloudTokens.value = true
  getCloudTokenList({ noPaginate: true })
    .then((res) => {
      const list = res?.data?.data || []
      cloudTokenOptions.value = list.map((t) => ({
        label: `${t.name || t.username || 'Token'}(#${t.id})`,
        value: t.id,
      }))
      if (cloudTokenOptions.value.length === 0) {
        // avoid "no data" confusion
        cloudTokenOptions.value = []
      }
    })
    .catch(() => {})
    .finally(() => {
      loadingCloudTokens.value = false
    })
}

// 通用保存函数：仅提交传入字段
const saveAdditionField = (payload: Record<string, unknown>, setLoading: (v: boolean) => void) => {
  if (!additionLoaded.value) return
  setLoading(true)
  modifySettingAddition(payload)
    .then((res) => {
      if (res.code === 200) {
        message.success('已保存')
        originalAddition.value = { ...additionForm }
      } else {
        message.error(res.msg || '保存失败')
      }
    })
    .catch((err) => {
      message.error(err instanceof Error ? err.message : '网络错误')
    })
    .finally(() => {
      setLoading(false)
    })
}

// 开机关联保存
const handleToggleLocalProxy = (val: boolean) => {
  saveAdditionField({ localProxy: val }, (v) => (savingLocalProxy.value = v))
}
const handleToggleMultipleStream = (val: boolean) => {
  saveAdditionField({ multipleStream: val }, (v) => (savingMultipleStream.value = v))
}

// Slider change事件处理
const handleThreadCountChange = () => {
  // 当slider值改变时自动保存
  if (additionLoaded.value) {
    handleSaveThreadCount()
  }
}
const handleChunkSizeChange = () => {
  // 当slider值改变时自动保存
  if (additionLoaded.value) {
    handleSaveChunkSize()
  }
}
const handleTaskThreadChange = () => {
  // 当slider值改变时自动保存
  if (additionLoaded.value) {
    handleSaveTaskThreads()
  }
}

// 数值保存函数
const handleSaveThreadCount = () => {
  saveAdditionField(
    { multipleStreamThreadCount: additionForm.multipleStreamThreadCount },
    (v) => (savingThreadCount.value = v)
  )
}
const handleSaveChunkSize = () => {
  saveAdditionField(
    { multipleStreamChunkSize: additionForm.multipleStreamChunkSize },
    (v) => (savingChunkSize.value = v)
  )
}
const handleSaveTaskThreads = () => {
  saveAdditionField(
    { taskThreadCount: additionForm.taskThreadCount },
    (v) => (savingTaskThreads.value = v)
  )
}

// Slider 标记
const threadCountMarks: Record<number, string> = {
  1: '1',
  4: '4',
  8: '8',
  16: '16',
  32: '32',
  64: '64',
}
const formatThreadTooltip = (val: number) => `${val} 线程`

const chunkSizeMarks: Record<number, string> = {
  1048576: '1 MB',
  4194304: '4 MB',
  8388608: '8 MB',
  16777216: '16 MB',
  33554432: '32 MB',
  67108864: '64 MB',
}
const formatChunkTooltip = (val: number) => formatFileSize(val)

const taskThreadMarks: Record<number, string> = {
  1: '1',
  4: '4',
  8: '8',
  16: '16',
  24: '24',
  32: '32',
}
const formatTaskThreadTooltip = (val: number) => `${val} 任务`

// 标题/URL修改
const savingTitle = ref(false)
const savingBaseURL = ref(false)
const isTitleChanged = computed(() => (form.title || '').trim() !== (systemInfo.title || '').trim())
const isBaseURLChanged = computed(
  () => (form.baseURL || '').trim() !== (systemInfo.baseURL || '').trim()
)

const handleSaveTitle = () => {
  const newTitle = (form.title || '').trim()
  if (newTitle.length === 0) {
    message.warning('请输入网站名称')
    return
  }
  savingTitle.value = true
  modifySystemTitle(newTitle)
    .then((res) => {
      if (res.code === 200) {
        message.success('网站名称已更新')
        systemStore
          .refresh()
          .then(() => {})
          .catch(() => {})
          .finally(() => {})
      } else {
        message.error(res.msg || '更新失败')
      }
    })
    .catch((err) => {
      message.error(err instanceof Error ? err.message : '网络错误')
    })
    .finally(() => {
      savingTitle.value = false
    })
}

const autoDetectBaseURL = () => {
  form.baseURL = window.location.origin
}
const handleSaveBaseURL = () => {
  const newBaseURL = (form.baseURL || '').trim()
  if (newBaseURL.length === 0) {
    message.warning('请输入基础 URL')
    return
  }
  if (!/^https?:\/\/.+/.test(newBaseURL)) {
    message.warning('基础 URL 必须以 http:// 或 https:// 开头')
    return
  }
  savingBaseURL.value = true
  modifySystemBaseURL(newBaseURL)
    .then((res) => {
      if (res.code === 200) {
        message.success('基础 URL 已更新')
        systemStore
          .refresh()
          .then(() => {})
          .catch(() => {})
          .finally(() => {})
      } else {
        message.error(res.msg || '更新失败')
      }
    })
    .catch((err) => {
      message.error(err instanceof Error ? err.message : '网络错误')
    })
    .finally(() => {
      savingBaseURL.value = false
    })
}

// 初始化
onMounted(() => {
  systemStore
    .refresh()
    .then((res) => {
      enableAuth.value = systemStore.get().enableAuth || false
      if (res?.code !== 200) {
        getSystemInfo()
          .then((r) => {
            if (r.data) {
              systemStore.load()
              enableAuth.value = r.data.enableAuth || false
            }
          })
          .catch(() => {})
          .finally(() => {})
      }
    })
    .catch(() => {})
    .finally(() => {})

  getSettingAddition()
    .then((res) => {
      if (res.code === 200 && res.data) {
        originalAddition.value = { ...res.data }
        additionForm.localProxy = !!res.data.localProxy
        additionForm.multipleStream = !!res.data.multipleStream
        additionForm.multipleStreamThreadCount = res.data.multipleStreamThreadCount ?? 4
        additionForm.multipleStreamChunkSize = res.data.multipleStreamChunkSize ?? 4 * 1024 * 1024
        additionForm.taskThreadCount = res.data.taskThreadCount ?? 1

        additionForm.externalApiKey = (res.data.externalApiKey as string) || ''
        additionForm.defaultTokenId = (res.data.defaultTokenId as number) || undefined
        additionForm.externalAutoRefreshEnabled =
          (res.data.externalAutoRefreshEnabled ?? true) as boolean
        additionForm.externalRefreshIntervalMin =
          res.data.externalRefreshIntervalMin ?? 60
        additionForm.externalAutoRefreshDays =
          res.data.externalAutoRefreshDays ?? 60
      } else {
        message.error(res.msg || '获取附加设置失败')
      }
    })
    .catch(() => {})
    .finally(() => {
      additionLoaded.value = true
    })

  loadCloudTokens()
})
</script>

<style scoped>
.settings-page {
  padding: 16px 24px 32px;
  background: var(--n-color-target);
}

/* 扁平化页面标题 */
.page-title {
  font-size: 18px;
  font-weight: 600;
  margin: 8px 0 12px;
  color: var(--n-text-color);
}

/* 区块容器（无卡片边框与重阴影，更贴近参考） */
.settings-section {
  background: var(--n-card-color);
  border: 1px solid var(--n-border-color);
  border-radius: 10px;
  box-shadow: 0 1px 6px rgb(0 0 0 / 8%);
  margin-bottom: 20px;
  padding: 8px 16px;
}

/* 单行项目 */
.setting-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 4px;
  border-bottom: 1px solid var(--n-border-color);
}

.setting-item:last-child {
  border-bottom: none;
}

/* 左侧文案 */
.item-left {
  flex: 0 0 560px;
  display: flex;
  flex-direction: column;
}

.item-title {
  font-weight: 600;
  color: var(--n-text-color);
}

.item-desc {
  color: var(--n-text-color-3);
  font-size: 12px;
  margin-top: 6px;
  line-height: 1.6;
}

/* 右侧控件区 */
.item-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.right-inline {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Switch/Slider 细节优化 */
:deep(.n-switch) {
  box-shadow: 0 1px 4px rgb(0 0 0 / 10%);
}

:deep(.n-slider-rail) {
  height: 6px;
}

:deep(.n-slider-handle) {
  width: 12px;
  height: 12px;
  box-shadow: 0 2px 8px rgb(0 0 0 / 12%);
}

:deep(.n-slider-marks) {
  font-size: 12px;
  color: var(--n-text-color-3);
}

/* 响应式 */
@media (width <= 1024px) {
  .item-left {
    flex: 1 1 auto;
  }
}

@media (width <= 768px) {
  .settings-section {
    padding: 8px 12px;
    border-radius: 8px;
  }

  .setting-item {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
  }

  .item-right {
    justify-content: flex-start;
  }
}
</style>
