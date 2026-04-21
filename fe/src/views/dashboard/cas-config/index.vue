<template>
  <div class="cas-config-page">
    <n-space vertical size="large">
      <n-alert type="info" :bordered="false" title="CAS 配置说明">
        <n-space vertical size="small">
          <n-text>本页面只承载 CAS 相关配置与手动恢复入口，不改动原有 STRM / 媒体页面。</n-text>
          <n-text>账号密码继续复用原项目 cloud token / mountpoint 体系，这里不重复配置账号密码。</n-text>
          <n-text>当前仅支持 reference-backed 组合：person → person、family → family、family → person。</n-text>
        </n-space>
      </n-alert>

      <n-grid :cols="24" :x-gap="16" :y-gap="16">
        <n-grid-item :span="24">
          <n-card title="默认配置" size="small">
            <n-form :model="defaultForm" label-placement="left" label-width="140px">
              <n-grid :cols="24" :x-gap="16">
                <n-grid-item :span="8">
                  <n-form-item label="默认上传路线">
                    <n-select
                      v-model:value="defaultForm.uploadRoute"
                      :options="uploadRouteOptions"
                      placeholder="选择默认上传路线"
                    />
                  </n-form-item>
                </n-grid-item>
                <n-grid-item :span="8">
                  <n-form-item label="默认最终目录类型">
                    <n-select
                      v-model:value="defaultForm.destinationType"
                      :options="defaultDestinationOptions"
                      placeholder="选择默认目录类型"
                    />
                  </n-form-item>
                </n-grid-item>
                <n-grid-item :span="8">
                  <n-form-item label="默认目标目录 ID">
                    <n-input v-model:value="defaultForm.targetFolderId" placeholder="例如 -11 或目录ID" />
                  </n-form-item>
                </n-grid-item>
              </n-grid>

              <n-grid :cols="24" :x-gap="16">
                <n-grid-item :span="8">
                  <n-form-item label="默认输入模式">
                    <n-select
                      v-model:value="defaultForm.inputMode"
                      :options="inputModeOptions"
                      placeholder="选择默认输入模式"
                    />
                  </n-form-item>
                </n-grid-item>
              </n-grid>

              <n-space justify="end">
                <n-button @click="resetDefaults">恢复默认</n-button>
                <n-button type="primary" @click="saveDefaults">保存默认配置</n-button>
              </n-space>
            </n-form>
          </n-card>
        </n-grid-item>

        <n-grid-item :span="24">
          <n-card title="CAS 手动恢复" size="small">
            <n-form :model="restoreForm" label-placement="left" label-width="140px">
              <n-grid :cols="24" :x-gap="16">
                <n-grid-item :span="8">
                  <n-form-item label="输入模式">
                    <n-select
                      v-model:value="restoreForm.inputMode"
                      :options="inputModeOptions"
                      placeholder="选择输入模式"
                    />
                  </n-form-item>
                </n-grid-item>
                <n-grid-item :span="8">
                  <n-form-item label="上传路线">
                    <n-select
                      v-model:value="restoreForm.uploadRoute"
                      :options="uploadRouteOptions"
                      placeholder="选择上传路线"
                    />
                  </n-form-item>
                </n-grid-item>
                <n-grid-item :span="8">
                  <n-form-item label="最终目录类型">
                    <n-select
                      v-model:value="restoreForm.destinationType"
                      :options="destinationTypeOptions"
                      placeholder="选择最终目录类型"
                    />
                  </n-form-item>
                </n-grid-item>
              </n-grid>

              <n-grid :cols="24" :x-gap="16">
                <n-grid-item :span="12">
                  <n-form-item label="目标目录 ID">
                    <n-input
                      v-model:value="restoreForm.targetFolderId"
                      placeholder="最终目录 ID，例如 -11 或个人目录ID"
                    />
                  </n-form-item>
                </n-grid-item>
              </n-grid>

              <template v-if="restoreForm.inputMode === 'virtualId'">
                <n-grid :cols="24" :x-gap="16">
                  <n-grid-item :span="12">
                    <n-form-item label="CAS Virtual ID">
                      <n-input-number
                        v-model:value="restoreForm.casVirtualId"
                        clearable
                        placeholder="例如 1001"
                        style="width: 100%"
                      />
                    </n-form-item>
                  </n-grid-item>
                </n-grid>
              </template>

              <template v-else-if="restoreForm.inputMode === 'path'">
                <n-grid :cols="24" :x-gap="16">
                  <n-grid-item :span="16">
                    <n-form-item label="CAS 路径">
                      <n-input v-model:value="restoreForm.casPath" placeholder="例如 /电影库/movie.cas" />
                    </n-form-item>
                  </n-grid-item>
                </n-grid>
              </template>

              <template v-else>
                <n-grid :cols="24" :x-gap="16">
                  <n-grid-item :span="8">
                    <n-form-item label="Storage ID">
                      <n-input-number v-model:value="restoreForm.storageId" clearable style="width: 100%" />
                    </n-form-item>
                  </n-grid-item>
                  <n-grid-item :span="8">
                    <n-form-item label="MountPoint ID">
                      <n-input-number v-model:value="restoreForm.mountPointId" clearable style="width: 100%" />
                    </n-form-item>
                  </n-grid-item>
                  <n-grid-item :span="8">
                    <n-form-item label="CAS Virtual ID">
                      <n-input-number v-model:value="restoreForm.casVirtualId" clearable style="width: 100%" />
                    </n-form-item>
                  </n-grid-item>
                </n-grid>
                <n-grid :cols="24" :x-gap="16">
                  <n-grid-item :span="8">
                    <n-form-item label="CAS File ID">
                      <n-input v-model:value="restoreForm.casFileId" placeholder="云端 CAS file id" />
                    </n-form-item>
                  </n-grid-item>
                  <n-grid-item :span="8">
                    <n-form-item label="CAS File Name">
                      <n-input v-model:value="restoreForm.casFileName" placeholder="例如 movie.cas" />
                    </n-form-item>
                  </n-grid-item>
                  <n-grid-item :span="8">
                    <n-form-item label="CAS 路径（可选）">
                      <n-input v-model:value="restoreForm.casPath" placeholder="例如 /电影库/movie.cas" />
                    </n-form-item>
                  </n-grid-item>
                </n-grid>
              </template>

              <n-space justify="end">
                <n-button @click="applyDefaultsToRestore">恢复默认配置</n-button>
                <n-button @click="resetRestore">重置本次输入</n-button>
                <n-button type="primary" :loading="restoring" @click="handleRestore">开始恢复</n-button>
              </n-space>
            </n-form>
          </n-card>
        </n-grid-item>

        <n-grid-item :span="12">
          <n-card title="请求预览" size="small">
            <pre class="result-pre">{{ requestPreview }}</pre>
          </n-card>
        </n-grid-item>

        <n-grid-item :span="12">
          <n-card title="链路说明" size="small">
            <n-space vertical size="small">
              <n-text>reference-backed 支持组合：</n-text>
              <n-ul>
                <n-li>person → person</n-li>
                <n-li>family → family</n-li>
                <n-li>family → person</n-li>
              </n-ul>
              <n-text>当前不支持：</n-text>
              <n-ul>
                <n-li>person → family（暂无可直接照搬的 reference-backed 主链）</n-li>
              </n-ul>
              <n-text depth="3">targetFolderId 只表示最终目录 ID，不表示上传路线。</n-text>
              <n-text depth="3">账号密码继续复用原项目 cloud token，不在本页面单独配置。</n-text>
            </n-space>
          </n-card>
        </n-grid-item>

        <n-grid-item :span="24" v-if="resultText">
          <n-card title="恢复结果" size="small">
            <pre class="result-pre">{{ resultText }}</pre>
          </n-card>
        </n-grid-item>
      </n-grid>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
  NText,
  NUl,
  NLi,
  useMessage,
} from 'naive-ui'
import type { RestoreCasRequest, CasUploadRoute, CasDestinationType } from '@/api/media'
import { restoreCas } from '@/api/media'

type InputMode = 'virtualId' | 'path' | 'explicit'

interface CasDefaults {
  uploadRoute: CasUploadRoute
  destinationType: CasDestinationType
  targetFolderId: string
  inputMode: InputMode
}

interface CasRestoreForm extends RestoreCasRequest {
  inputMode: InputMode
}

const DEFAULTS_KEY = 'cas.config.defaults'
const message = useMessage()
const restoring = ref(false)
const resultText = ref('')

const uploadRouteOptions = [
  { label: 'family（家庭路线，默认）', value: 'family' },
  { label: 'person（个人路线）', value: 'person' },
]

const inputModeOptions = [
  { label: '最简 ID 模式', value: 'virtualId' },
  { label: '路径模式', value: 'path' },
  { label: '显式模式', value: 'explicit' },
]

const emptyDefaults = (): CasDefaults => ({
  uploadRoute: 'family',
  destinationType: 'family',
  targetFolderId: '',
  inputMode: 'virtualId',
})

const defaultForm = reactive<CasDefaults>(emptyDefaults())
const restoreForm = reactive<CasRestoreForm>({
  ...emptyDefaults(),
  casVirtualId: undefined,
  casPath: '',
  storageId: undefined,
  mountPointId: undefined,
  casFileId: '',
  casFileName: '',
})

const defaultDestinationOptions = computed(() => {
  if (defaultForm.uploadRoute === 'person') {
    return [{ label: 'person（个人目录）', value: 'person' }]
  }
  return [
    { label: 'person（个人目录）', value: 'person' },
    { label: 'family（家庭目录）', value: 'family' },
  ]
})

const destinationTypeOptions = computed(() => {
  if (restoreForm.uploadRoute === 'person') {
    return [{ label: 'person（个人目录）', value: 'person' }]
  }
  return [
    { label: 'person（个人目录）', value: 'person' },
    { label: 'family（家庭目录）', value: 'family' },
  ]
})

watch(
  () => defaultForm.uploadRoute,
  (val) => {
    if (val === 'person') {
      defaultForm.destinationType = 'person'
    }
  }
)

watch(
  () => restoreForm.uploadRoute,
  (val) => {
    if (val === 'person') {
      restoreForm.destinationType = 'person'
    }
  }
)

const loadDefaults = () => {
  try {
    const raw = localStorage.getItem(DEFAULTS_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw) as Partial<CasDefaults>
    Object.assign(defaultForm, emptyDefaults(), parsed)
  } catch {
    // ignore broken local config
  }
}

const saveDefaults = () => {
  localStorage.setItem(DEFAULTS_KEY, JSON.stringify(defaultForm))
  message.success('CAS 默认配置已保存')
}

const resetDefaults = () => {
  Object.assign(defaultForm, emptyDefaults())
  localStorage.setItem(DEFAULTS_KEY, JSON.stringify(defaultForm))
  message.success('已恢复默认配置')
}

const applyDefaultsToRestore = () => {
  restoreForm.uploadRoute = defaultForm.uploadRoute
  restoreForm.destinationType = defaultForm.destinationType
  restoreForm.targetFolderId = defaultForm.targetFolderId
  restoreForm.inputMode = defaultForm.inputMode
  message.success('已应用默认配置')
}

const resetRestore = () => {
  Object.assign(restoreForm, {
    ...defaultForm,
    casVirtualId: undefined,
    casPath: '',
    storageId: undefined,
    mountPointId: undefined,
    casFileId: '',
    casFileName: '',
  })
  resultText.value = ''
}

const buildRequestPayload = (): RestoreCasRequest => {
  const payload: RestoreCasRequest = {
    uploadRoute: restoreForm.uploadRoute,
    destinationType: restoreForm.destinationType,
    targetFolderId: (restoreForm.targetFolderId || '').trim(),
  }

  if (restoreForm.inputMode === 'virtualId') {
    if (restoreForm.casVirtualId) payload.casVirtualId = restoreForm.casVirtualId
  } else if (restoreForm.inputMode === 'path') {
    if ((restoreForm.casPath || '').trim()) payload.casPath = restoreForm.casPath?.trim()
  } else {
    if (restoreForm.storageId) payload.storageId = restoreForm.storageId
    if (restoreForm.mountPointId) payload.mountPointId = restoreForm.mountPointId
    if ((restoreForm.casFileId || '').trim()) payload.casFileId = restoreForm.casFileId?.trim()
    if ((restoreForm.casFileName || '').trim()) payload.casFileName = restoreForm.casFileName?.trim()
    if (restoreForm.casVirtualId) payload.casVirtualId = restoreForm.casVirtualId
    if ((restoreForm.casPath || '').trim()) payload.casPath = restoreForm.casPath?.trim()
  }

  return payload
}

const requestPreview = computed(() => JSON.stringify(buildRequestPayload(), null, 2))

const validateBeforeSubmit = (): boolean => {
  if (!(restoreForm.targetFolderId || '').trim()) {
    message.warning('请填写目标目录 ID')
    return false
  }
  if (restoreForm.uploadRoute === 'person' && restoreForm.destinationType === 'family') {
    message.warning('当前仅支持 reference-backed 组合，person → family 暂未实现')
    return false
  }

  if (restoreForm.inputMode === 'virtualId' && !restoreForm.casVirtualId) {
    message.warning('最简 ID 模式下请填写 casVirtualId')
    return false
  }
  if (restoreForm.inputMode === 'path' && !(restoreForm.casPath || '').trim()) {
    message.warning('路径模式下请填写 casPath')
    return false
  }
  if (restoreForm.inputMode === 'explicit') {
    const hasLocator = !!restoreForm.casVirtualId || !!(restoreForm.casPath || '').trim()
    if (!hasLocator) {
      message.warning('显式模式至少仍需提供 casVirtualId 或 casPath 其中一个用于定位 CAS')
      return false
    }
  }
  return true
}

const handleRestore = () => {
  if (!validateBeforeSubmit()) return
  const payload = buildRequestPayload()
  restoring.value = true
  restoreCas(payload)
    .then((res) => {
      resultText.value = JSON.stringify(res.data || {}, null, 2)
      message.success(res.msg || 'CAS 恢复请求成功')
    })
    .catch((err) => {
      message.error(err?.message || 'CAS 恢复失败')
    })
    .finally(() => {
      restoring.value = false
    })
}

onMounted(() => {
  loadDefaults()
  applyDefaultsToRestore()
})
</script>

<style scoped>
.cas-config-page {
  padding: 24px;
}

.result-pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
  line-height: 1.6;
}
</style>
