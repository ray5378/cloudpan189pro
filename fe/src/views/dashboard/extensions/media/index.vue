<template>
  <n-spin :show="loading">
    <div class="media-settings-page">
      <template v-if="!initialized">
        <n-alert type="warning" title="初始化 STRM 能力" :bordered="false">
          <n-space vertical size="small">
            <n-text>媒体服务尚未初始化，请先完成初始化以启用 STRM 生成能力</n-text>
            <n-space>
              <n-button type="primary" @click="openInitModal">开始初始化</n-button>
            </n-space>
          </n-space>
        </n-alert>
      </template>

      <template v-else>
        <n-space vertical size="large">
          <n-descriptions
            bordered
            size="small"
            :column="1"
            label-placement="left"
            :label-style="{ width: '220px' }"
          >
            <n-descriptions-item label="启用媒体服务">
              <n-switch
                :value="!!config?.enable"
                :loading="savingEnable"
                @update:value="handleToggleEnable"
              />
              <div class="desc-sub">
                开启后，当入库文件时，会自动生成 STRM 文件，删除时也会自动删除相关文件
              </div>
            </n-descriptions-item>

            <n-descriptions-item label="存储根路径">
              <n-text>{{ config?.storagePath || '-' }}</n-text>
              <div class="desc-sub">STRM 与相关文件输出的根目录</div>
            </n-descriptions-item>

            <n-descriptions-item label="自动清理空文件夹">
              <n-tag :type="config?.autoClean ? 'success' : 'default'">
                {{ config?.autoClean ? '已启用' : '未启用' }}
              </n-tag>
              <div class="desc-sub">启用后，生成或删除文件后将自动清理空文件夹</div>
            </n-descriptions-item>

            <n-descriptions-item label="冲突策略">
              <n-text>{{
                config?.conflictPolicy === 'replace' ? 'replace（替换）' : 'skip（跳过）'
              }}</n-text>
              <div class="desc-sub">处理已存在的目标文件时的策略（跳过或替换）</div>
            </n-descriptions-item>

            <n-descriptions-item label="媒体基础 URL">
              <n-text>{{ config?.baseURL || '-' }}</n-text>
              <div class="desc-sub">用于生成外部链接等场景</div>
            </n-descriptions-item>

            <n-descriptions-item label="包括的后缀格式">
              <n-text>{{ config?.includedSuffixes?.join(', ') || '-' }}</n-text>
              <div class="desc-sub">仅支持这些后缀的文件生成 STRM</div>
            </n-descriptions-item>
          </n-descriptions>

          <n-space justify="space-between" align="center">
            <n-space>
              <n-popover trigger="hover" placement="top">
                <template #trigger>
                  <n-button
                    size="small"
                    type="error"
                    :loading="clearingMedia"
                    @click="handleClearMedia"
                  >
                    清理媒体文件
                  </n-button>
                </template>
                <n-text style="font-size: 12px">
                  警告：会把整个媒体目录文件全部清空，包括自己创建的文件，请谨慎操作
                </n-text>
              </n-popover>
              <n-popover trigger="hover" placement="top">
                <template #trigger>
                  <n-button
                    size="small"
                    type="warning"
                    :loading="rebuildingStrm"
                    @click="handleRebuildStrm"
                  >
                    重建strm文件
                  </n-button>
                </template>
                <n-text style="font-size: 12px">
                  只会给还没有创建strm的创建，已创建的不会影响。如需重新构建，请先清空再重建
                </n-text>
              </n-popover>
            </n-space>
            <n-space>
              <n-button size="small" type="primary" @click="openEditModal">编辑配置</n-button>
              <n-button size="small" type="info" @click="reload">刷新配置</n-button>
            </n-space>
          </n-space>

          <n-card title="CAS 恢复测试" size="small" :bordered="true">
            <n-space vertical size="small">
              <n-alert type="info" :bordered="false">
                当前前端入口只暴露已经按参考实现收口的组合：person → person、family → family、family → person。
                person → family 因缺少 reference-backed 主链，前后端都会拒绝。
              </n-alert>

              <n-form :model="restoreForm" label-placement="left" label-width="120px">
                <n-form-item label="CAS Virtual ID">
                  <n-input-number
                    v-model:value="restoreForm.casVirtualId"
                    clearable
                    placeholder="例如 1001"
                    style="width: 100%"
                  />
                </n-form-item>

                <n-form-item label="CAS 路径">
                  <n-input
                    v-model:value="restoreForm.casPath"
                    placeholder="例如 /电影库/movie.cas"
                    clearable
                  />
                </n-form-item>

                <n-form-item label="上传路线">
                  <n-select
                    v-model:value="restoreForm.uploadRoute"
                    :options="uploadRouteOptions"
                    placeholder="选择上传路线"
                  />
                </n-form-item>

                <n-form-item label="最终目录类型">
                  <n-select
                    v-model:value="restoreForm.destinationType"
                    :options="destinationTypeOptions"
                    placeholder="选择最终目录类型"
                  />
                </n-form-item>

                <n-form-item label="目标目录 ID">
                  <n-input
                    v-model:value="restoreForm.targetFolderId"
                    placeholder="个人目录 ID 或家庭目录 ID，例如 -11"
                    clearable
                  />
                </n-form-item>

                <n-space justify="end">
                  <n-button @click="resetRestoreForm">重置</n-button>
                  <n-button type="primary" :loading="restoringCas" @click="handleRestoreCas">
                    开始恢复
                  </n-button>
                </n-space>
              </n-form>

              <n-alert v-if="restoreResultText" type="success" :bordered="false" title="恢复结果">
                <pre class="result-pre">{{ restoreResultText }}</pre>
              </n-alert>
            </n-space>
          </n-card>
        </n-space>
      </template>

      <n-modal
        v-model:show="showInitModal"
        preset="card"
        title="初始化媒体配置"
        style="width: 640px"
      >
        <n-form :model="initForm" label-placement="left" label-width="130px">
          <n-form-item label="是否启用">
            <n-switch v-model:value="initForm.enable" />
          </n-form-item>

          <n-form-item label="存储根路径">
            <n-input v-model:value="initForm.storagePath" placeholder="/opt/media" clearable />
          </n-form-item>

          <n-form-item label="自动清理空文件夹">
            <n-switch v-model:value="initForm.autoClean" />
          </n-form-item>

          <n-form-item label="冲突策略">
            <n-select
              v-model:value="initForm.conflictPolicy"
              :options="conflictPolicyOptions"
              placeholder="请选择冲突策略"
            />
          </n-form-item>

          <n-form-item label="媒体基础 URL">
            <n-space>
              <n-input
                v-model:value="initForm.baseURL"
                placeholder="http://localhost:12395"
                clearable
              />
              <n-button size="small" @click="autoDetectBaseURL">自动获取</n-button>
            </n-space>
          </n-form-item>

          <n-form-item label="包括的后缀格式">
            <n-dynamic-tags
              v-model:value="initForm.includedSuffixes"
              input-placeholder="添加后缀..."
              @create="handleSuffixCreate"
            />
            <div class="desc-sub">输入以 . 开头的后缀名后按回车添加，留空将支持所有类型。</div>
          </n-form-item>

          <n-space justify="end">
            <n-button @click="showInitModal = false">取消</n-button>
            <n-button type="primary" @click="handleInit">完成初始化</n-button>
          </n-space>
        </n-form>
      </n-modal>

      <n-modal v-model:show="showEditModal" preset="card" title="编辑媒体配置" style="width: 680px">
        <n-form :model="editForm" label-placement="left" label-width="130px">
          <n-form-item label="存储根路径">
            <n-input v-model:value="editForm.storagePath" placeholder="/opt/media" clearable />
          </n-form-item>

          <n-form-item label="自动清理空文件夹">
            <n-switch v-model:value="editForm.autoClean" />
          </n-form-item>

          <n-form-item label="冲突策略">
            <n-select
              v-model:value="editForm.conflictPolicy"
              :options="conflictPolicyOptions"
              placeholder="请选择冲突策略"
            />
          </n-form-item>

          <n-form-item label="媒体基础 URL">
            <n-space>
              <n-input
                v-model:value="editForm.baseURL"
                placeholder="http://localhost:12395"
                clearable
              />
              <n-button size="small" @click="autoDetectConfigBaseURL">自动获取</n-button>
            </n-space>
          </n-form-item>

          <n-form-item label="包括的后缀格式">
            <n-dynamic-tags
              v-model:value="editForm.includedSuffixes"
              input-placeholder="添加后缀..."
              @create="handleSuffixCreate"
            />
            <div class="desc-sub">输入以 . 开头的后缀名后按回车添加，留空将支持所有类型。</div>
          </n-form-item>

          <n-space justify="end">
            <n-button @click="showEditModal = false">取消</n-button>
            <n-button type="primary" @click="handleSaveEdit">保存</n-button>
          </n-space>
        </n-form>
      </n-modal>
    </div>
  </n-spin>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import {
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NSwitch,
  NSpace,
  NButton,
  NText,
  NModal,
  NSpin,
  NDescriptions,
  NDescriptionsItem,
  NTag,
  NAlert,
  NDynamicTags,
  NCard,
  useMessage,
  useDialog,
} from 'naive-ui'
import type {
  ConfigInitRequest,
  ConfigUpdateRequest,
  RestoreCasRequest,
  CasUploadRoute,
  CasDestinationType,
} from '@/api/media'
import {
  getMediaConfigInfo,
  initMediaConfig,
  toggleMediaConfig,
  updateMediaConfig,
  clearMediaFiles,
  rebuildStrmFiles,
  restoreCas,
} from '@/api/media'

const message = useMessage()
const dialog = useDialog()

const loading = ref<boolean>(true)
const initialized = ref<boolean>(true)
const config = ref<Models.MediaConfig | undefined>(undefined)
const showInitModal = ref(false)
const showEditModal = ref(false)
const savingEnable = ref(false)
const clearingMedia = ref(false)
const rebuildingStrm = ref(false)
const restoringCas = ref(false)
const restoreResultText = ref('')

const editForm = reactive<ConfigUpdateRequest>({
  storagePath: '',
  autoClean: false,
  conflictPolicy: 'skip',
  baseURL: '',
  includedSuffixes: [],
})

const initForm = reactive<ConfigInitRequest>({
  enable: true,
  storagePath: '',
  autoClean: true,
  conflictPolicy: 'skip',
  baseURL: '',
  includedSuffixes: [],
})

const restoreForm = reactive<RestoreCasRequest>({
  casVirtualId: undefined,
  casPath: '',
  uploadRoute: 'family',
  destinationType: 'family',
  targetFolderId: '',
})

const conflictPolicyOptions = [
  { label: 'skip（跳过）', value: 'skip' },
  { label: 'replace（替换）', value: 'replace' },
]

const uploadRouteOptions = [
  { label: 'family（家庭路线，默认）', value: 'family' },
  { label: 'person（个人路线）', value: 'person' },
]

const destinationTypeOptions = computed(() => {
  const uploadRoute = restoreForm.uploadRoute as CasUploadRoute
  const options: { label: string; value: CasDestinationType }[] = [
    { label: 'person（个人目录）', value: 'person' },
    { label: 'family（家庭目录）', value: 'family' },
  ]
  if (uploadRoute === 'person') {
    return options.filter((item) => item.value === 'person')
  }
  return options
})

const defaultIncludedSuffixes = [
  'mp4', 'mkv', 'avi', 'mov', 'wmv', 'flv', 'webm', 'm4v', 'mpg', 'mpeg', 'm2v', 'm4p', 'm4b',
  'ts', 'mts', 'm2ts', 'm2t', 'mxf', 'dv', 'dvr-ms', 'asf', '3gp', '3g2', 'f4v', 'f4p', 'f4a',
  'f4b', 'vob', 'ogv', 'ogg', 'divx', 'xvid', 'rm', 'rmvb', 'dat', 'nsv', 'qt', 'amv', 'mpv',
  'm1v', 'svi', 'viv', 'fli', 'flc',
].map((s) => `.${s}`)

const reload = () => {
  loading.value = true
  getMediaConfigInfo()
    .then((res) => {
      if (res.data) {
        initialized.value = !!res.data.initialized
        config.value = res.data.config
        Object.assign(editForm, res.data.config)
      } else {
        message.error(res.msg || '获取媒体配置失败')
      }
    })
    .catch((err) => {
      message.error(err?.message || '获取媒体配置失败')
    })
    .finally(() => {
      loading.value = false
    })
}

const handleInit = () => {
  if (!initForm.storagePath) {
    message.warning('请填写存储根路径')
    return
  }
  if (!initForm.baseURL) {
    message.warning('请填写媒体基础URL')
    return
  }
  initForm.includedSuffixes = normalizeSuffixes(initForm.includedSuffixes || [])
  initMediaConfig(initForm)
    .then(() => {
      message.success('初始化成功')
      showInitModal.value = false
      reload()
    })
    .catch((err) => {
      message.error(err?.message || '初始化媒体配置失败')
    })
}

const autoDetectBaseURL = () => {
  initForm.baseURL = window.location.origin
}
const autoDetectConfigBaseURL = () => {
  editForm.baseURL = window.location.origin
}

const normalizeSuffixes = (arr: string[]): string[] => {
  const out: string[] = []
  const seen = new Set<string>()
  for (const raw of arr || []) {
    const s = (raw || '').trim().toLowerCase()
    if (!s || !s.startsWith('.')) continue
    if (!seen.has(s)) {
      seen.add(s)
      out.push(s)
    }
  }
  return out
}

const handleSuffixCreate = (label: string): string => {
  let s = (label || '').trim()
  if (!s.startsWith('.')) {
    s = `.${s}`
  }
  return s.toLowerCase()
}

const openInitModal = () => {
  if (!initForm.baseURL) {
    initForm.baseURL = window.location.origin
  }
  if (!initForm.includedSuffixes || initForm.includedSuffixes.length === 0) {
    initForm.includedSuffixes = [...defaultIncludedSuffixes]
  }
  showInitModal.value = true
}

const openEditModal = () => {
  if (config.value) {
    Object.assign(editForm, config.value)
  }
  showEditModal.value = true
}

const handleSaveEdit = () => {
  const v = (editForm.baseURL || '').trim()
  if (v.length === 0) {
    message.warning('请输入基础 URL')
    return
  }
  if (!/^https?:\/\/.+/.test(v)) {
    message.warning('基础 URL 必须以 http:// 或 https:// 开头')
    return
  }
  editForm.includedSuffixes = normalizeSuffixes(editForm.includedSuffixes || [])
  updateMediaConfig(editForm)
    .then(() => {
      message.success('更新媒体配置成功')
      showEditModal.value = false
      reload()
    })
    .catch((err) => {
      message.error(err?.message || '更新媒体配置失败')
    })
}

const handleToggleEnable = (val: boolean) => {
  savingEnable.value = true
  toggleMediaConfig({ enable: val })
    .then(() => {
      message.success(val ? '已启用媒体配置' : '已禁用媒体配置')
      reload()
    })
    .catch((err) => {
      message.error(err?.message || '切换媒体配置启用状态失败')
    })
    .finally(() => {
      savingEnable.value = false
    })
}

const handleClearMedia = () => {
  dialog.warning({
    title: '确认清理媒体文件',
    content: '此操作将清空整个媒体目录，包括所有文件以及自己创建的文件，请确认是否继续？',
    positiveText: '确认清理',
    negativeText: '取消',
    onPositiveClick: () => {
      clearingMedia.value = true
      clearMediaFiles()
        .then(() => {
          message.success('清理任务已提交，请稍后查看效果')
        })
        .catch((err) => {
          message.error(err?.message || '清理媒体文件失败')
        })
        .finally(() => {
          clearingMedia.value = false
        })
    },
  })
}

const handleRebuildStrm = () => {
  dialog.info({
    title: '确认重建strm文件',
    content: '确认重新生成strm文件吗？只会给还没有创建strm的创建，已创建的不会影响。',
    positiveText: '确认重建',
    negativeText: '取消',
    onPositiveClick: () => {
      rebuildingStrm.value = true
      rebuildStrmFiles()
        .then(() => {
          message.success('重建任务已提交，将扫描所有挂载点并重新生成strm文件')
        })
        .catch((err) => {
          message.error(err?.message || '重建strm文件失败')
        })
        .finally(() => {
          rebuildingStrm.value = false
        })
    },
  })
}

const resetRestoreForm = () => {
  restoreForm.casVirtualId = undefined
  restoreForm.casPath = ''
  restoreForm.uploadRoute = 'family'
  restoreForm.destinationType = 'family'
  restoreForm.targetFolderId = ''
  restoreResultText.value = ''
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

  const payload: RestoreCasRequest = {
    destinationType: restoreForm.destinationType,
    targetFolderId: restoreForm.targetFolderId.trim(),
    uploadRoute: restoreForm.uploadRoute,
  }
  if (restoreForm.casVirtualId) {
    payload.casVirtualId = restoreForm.casVirtualId
  }
  if ((restoreForm.casPath || '').trim()) {
    payload.casPath = restoreForm.casPath?.trim()
  }

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

onMounted(reload)
</script>

<style scoped>
.desc-sub {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--n-text-color-3);
}

.result-pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
  line-height: 1.6;
}
</style>
