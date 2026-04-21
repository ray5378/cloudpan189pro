<template>
  <div class="cas-config-page">
    <n-space vertical size="large">
      <n-card title="CAS配置" size="small">
        <n-space vertical size="small">
          <n-text depth="3">
            这里用于配置并手动触发 CAS 恢复。天翼云盘账号密码继续复用原项目已有账号，不在这里单独配置。
          </n-text>
          <n-alert type="info" :bordered="false">
            当前仅支持 reference-backed 组合：person → person、family → family、family → person。
            person → family 因缺少参考主链，当前前后端都会拒绝。
          </n-alert>
        </n-space>
      </n-card>

      <n-grid :cols="24" :x-gap="16" :y-gap="16">
        <n-grid-item :span="12">
          <n-card title="默认配置" size="small">
            <n-form :model="defaultsForm" label-placement="left" label-width="130px">
              <n-form-item label="默认上传路线">
                <n-select
                  v-model:value="defaultsForm.uploadRoute"
                  :options="uploadRouteOptions"
                  placeholder="选择默认上传路线"
                />
              </n-form-item>

              <n-form-item label="默认最终目录类型">
                <n-select
                  v-model:value="defaultsForm.destinationType"
                  :options="defaultsDestinationOptions"
                  placeholder="选择默认目录类型"
                />
              </n-form-item>

              <n-form-item label="默认目标目录 ID">
                <n-input
                  v-model:value="defaultsForm.targetFolderId"
                  placeholder="例如 -11 或具体目录 ID"
                  clearable
                />
              </n-form-item>

              <n-space justify="end">
                <n-button @click="resetDefaults">恢复默认</n-button>
                <n-button type="primary" @click="saveDefaults">保存默认配置</n-button>
              </n-space>
            </n-form>
          </n-card>
        </n-grid-item>

        <n-grid-item :span="12">
          <n-card title="链路说明" size="small">
            <n-descriptions bordered size="small" :column="1" label-placement="left">
              <n-descriptions-item label="person → person">
                /person/initMultiUpload → /person/checkTransSecond → /person/commitMultiUploadFile
              </n-descriptions-item>
              <n-descriptions-item label="family → family">
                /family/initMultiUpload → /family/checkTransSecond → /family/commitMultiUploadFile
              </n-descriptions-item>
              <n-descriptions-item label="family → person">
                family rapid upload → COPY → checkBatchTask → DELETE
              </n-descriptions-item>
              <n-descriptions-item label="账号来源">
                复用 cloudpan189pro 里挂载点对应的天翼云盘账号，不单独配置账号密码
              </n-descriptions-item>
            </n-descriptions>
          </n-card>
        </n-grid-item>

        <n-grid-item :span="24">
          <n-card title="恢复参数" size="small">
            <n-form :model="restoreForm" label-placement="left" label-width="140px">
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
                <n-grid-item :span="12">
                  <n-form-item label="CAS 路径">
                    <n-input
                      v-model:value="restoreForm.casPath"
                      placeholder="例如 /电影库/movie.cas"
                      clearable
                    />
                  </n-form-item>
                </n-grid-item>

                <n-grid-item :span="12">
                  <n-form-item label="上传路线">
                    <n-select
                      v-model:value="restoreForm.uploadRoute"
                      :options="uploadRouteOptions"
                      placeholder="选择上传路线"
                    />
                  </n-form-item>
                </n-grid-item>
                <n-grid-item :span="12">
                  <n-form-item label="最终目录类型">
                    <n-select
                      v-model:value="restoreForm.destinationType"
                      :options="destinationTypeOptions"
                      placeholder="选择最终目录类型"
                    />
                  </n-form-item>
                </n-grid-item>

                <n-grid-item :span="12">
                  <n-form-item label="目标目录 ID">
                    <n-input
                      v-model:value="restoreForm.targetFolderId"
                      placeholder="例如 -11 或具体目录 ID"
                      clearable
                    />
                  </n-form-item>
                </n-grid-item>
                <n-grid-item :span="12">
                  <n-form-item label="storageId（可选）">
                    <n-input-number
                      v-model:value="restoreForm.storageId"
                      clearable
                      placeholder="显式模式可填"
                      style="width: 100%"
                    />
                  </n-form-item>
                </n-grid-item>

                <n-grid-item :span="12">
                  <n-form-item label="mountPointId（可选）">
                    <n-input-number
                      v-model:value="restoreForm.mountPointId"
                      clearable
                      placeholder="显式模式可填"
                      style="width: 100%"
                    />
                  </n-form-item>
                </n-grid-item>
                <n-grid-item :span="12">
                  <n-form-item label="casFileId（可选）">
                    <n-input
                      v-model:value="restoreForm.casFileId"
                      placeholder="显式模式可填"
                      clearable
                    />
                  </n-form-item>
                </n-grid-item>

                <n-grid-item :span="24">
                  <n-form-item label="casFileName（可选）">
                    <n-input
                      v-model:value="restoreForm.casFileName"
                      placeholder="显式模式可填，例如 movie.cas"
                      clearable
                    />
                  </n-form-item>
                </n-grid-item>
              </n-grid>

              <n-space justify="end">
                <n-button @click="applyDefaultsToRestore">套用默认配置</n-button>
                <n-button @click="resetRestoreForm">重置</n-button>
                <n-button type="primary" :loading="restoringCas" @click="handleRestoreCas">
                  开始恢复
                </n-button>
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
          <n-card title="恢复结果" size="small">
            <pre class="result-pre">{{ restoreResultText || '暂无结果' }}</pre>
          </n-card>
        </n-grid-item>
      </n-grid>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
  NText,
  useMessage,
} from 'naive-ui'
import type {
  CasDestinationType,
  CasUploadRoute,
  RestoreCasRequest,
} from '@/api/media'
import { restoreCas } from '@/api/media'

const message = useMessage()

const DEFAULTS_KEY = 'cas.config.defaults'

const uploadRouteOptions = [
  { label: 'family（家庭路线，默认）', value: 'family' },
  { label: 'person（个人路线）', value: 'person' },
]

const defaultsForm = reactive({
  uploadRoute: 'family' as CasUploadRoute,
  destinationType: 'family' as CasDestinationType,
  targetFolderId: '',
})

const restoreForm = reactive<RestoreCasRequest>({
  casVirtualId: undefined,
  casPath: '',
  uploadRoute: 'family',
  destinationType: 'family',
  targetFolderId: '',
  storageId: undefined,
  mountPointId: undefined,
  casFileId: '',
  casFileName: '',
})

const restoringCas = ref(false)
const restoreResultText = ref('')

const defaultsDestinationOptions = computed(() => {
  if (defaultsForm.uploadRoute === 'person') {
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

const requestPreview = computed(() => {
  const payload = buildRestorePayload()
  return JSON.stringify(payload, null, 2)
})

const loadDefaults = () => {
  try {
    const raw = localStorage.getItem(DEFAULTS_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw)
    defaultsForm.uploadRoute = parsed.uploadRoute || 'family'
    defaultsForm.destinationType = parsed.destinationType || 'family'
    defaultsForm.targetFolderId = parsed.targetFolderId || ''
    applyDefaultsToRestore()
  } catch {
    // ignore invalid local config
  }
}

const saveDefaults = () => {
  if (defaultsForm.uploadRoute === 'person' && defaultsForm.destinationType === 'family') {
    message.warning('默认配置当前不支持 person → family')
    return
  }
  localStorage.setItem(DEFAULTS_KEY, JSON.stringify(defaultsForm))
  message.success('默认配置已保存')
}

const resetDefaults = () => {
  defaultsForm.uploadRoute = 'family'
  defaultsForm.destinationType = 'family'
  defaultsForm.targetFolderId = ''
  localStorage.removeItem(DEFAULTS_KEY)
  message.success('已恢复默认配置')
}

const applyDefaultsToRestore = () => {
  restoreForm.uploadRoute = defaultsForm.uploadRoute
  restoreForm.destinationType = defaultsForm.destinationType
  restoreForm.targetFolderId = defaultsForm.targetFolderId
}

const resetRestoreForm = () => {
  restoreForm.casVirtualId = undefined
  restoreForm.casPath = ''
  restoreForm.uploadRoute = defaultsForm.uploadRoute
  restoreForm.destinationType = defaultsForm.destinationType
  restoreForm.targetFolderId = defaultsForm.targetFolderId
  restoreForm.storageId = undefined
  restoreForm.mountPointId = undefined
  restoreForm.casFileId = ''
  restoreForm.casFileName = ''
  restoreResultText.value = ''
}

const buildRestorePayload = (): RestoreCasRequest => {
  const payload: RestoreCasRequest = {
    uploadRoute: restoreForm.uploadRoute,
    destinationType: restoreForm.destinationType,
    targetFolderId: (restoreForm.targetFolderId || '').trim(),
  }
  if (restoreForm.casVirtualId) payload.casVirtualId = restoreForm.casVirtualId
  if ((restoreForm.casPath || '').trim()) payload.casPath = restoreForm.casPath?.trim()
  if (restoreForm.storageId) payload.storageId = restoreForm.storageId
  if (restoreForm.mountPointId) payload.mountPointId = restoreForm.mountPointId
  if ((restoreForm.casFileId || '').trim()) payload.casFileId = restoreForm.casFileId?.trim()
  if ((restoreForm.casFileName || '').trim()) payload.casFileName = restoreForm.casFileName?.trim()
  return payload
}

const handleRestoreCas = () => {
  if (!restoreForm.casVirtualId && !(restoreForm.casPath || '').trim()) {
    message.warning('请填写 casVirtualId 或 casPath 其中一个')
    return
  }
  if (!(restoreForm.targetFolderId || '').trim()) {
    message.warning('请填写目标目录 ID')
    return
  }
  if (restoreForm.uploadRoute === 'person' && restoreForm.destinationType === 'family') {
    message.warning('当前前后端都只支持 reference-backed 组合，person → family 暂不支持')
    return
  }

  const payload = buildRestorePayload()
  restoringCas.value = true
  restoreCas(payload)
    .then((res) => {
      restoreResultText.value = JSON.stringify(res.data || {}, null, 2)
      message.success(res.msg || 'CAS 恢复请求成功')
    })
    .catch((err) => {
      message.error(err?.message || 'CAS 恢复失败')
    })
    .finally(() => {
      restoringCas.value = false
    })
}

loadDefaults()
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
