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
          <n-card title="CAS 来源目录" size="small">
            <n-space vertical size="small">
              <n-text depth="3">在这里选中的保存目录，天然等于 CAS访问路径，也等于 CAS归集路径；这里不再区分两套路径。</n-text>

              <n-form :model="sourceForm" label-placement="left" label-width="140px">
                <n-grid :cols="24" :x-gap="16" :y-gap="8">
                  <n-grid-item :span="6">
                    <n-form-item label="启用 CAS 目标目录">
                      <n-switch v-model:value="sourceForm.enabled" />
                    </n-form-item>
                  </n-grid-item>
                  <n-grid-item :span="6">
                    <n-form-item label="自动归集订阅 .cas">
                      <n-switch v-model:value="sourceForm.autoCollectEnabled" />
                    </n-form-item>
                  </n-grid-item>
                  <n-grid-item :span="6">
                    <n-form-item label="保留订阅路径结构">
                      <n-switch v-model:value="sourceForm.preservePath" />
                    </n-form-item>
                  </n-grid-item>
                </n-grid>

                <n-grid :cols="24" :x-gap="16" :y-gap="8">
                  <n-grid-item :span="8">
                    <n-form-item label="云盘账号">
                      <n-select
                        v-model:value="sourceForm.cloudToken"
                        :options="cloudTokenOptions"
                        placeholder="选择已添加的云盘账号"
                        :loading="cloudTokenLoading"
                        filterable
                        clearable
                      />
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item :span="8">
                    <n-form-item label="目标类型">
                      <n-select
                        v-model:value="sourceForm.sourceType"
                        :options="sourceTypeOptions"
                        placeholder="选择来源类型"
                      />
                    </n-form-item>
                  </n-grid-item>

                  <n-grid-item v-if="sourceForm.sourceType === 'family'" :span="8">
                    <n-form-item label="家庭组">
                      <n-select
                        v-model:value="sourceForm.familyId"
                        :options="familyOptions"
                        placeholder="选择家庭组"
                        :loading="familyLoading"
                        filterable
                        clearable
                      />
                    </n-form-item>
                  </n-grid-item>
                </n-grid>

                <n-grid :cols="24" :x-gap="16" :y-gap="8">
                  <n-grid-item :span="12">
                    <n-form-item label="当前目录">
                      <n-input :value="sourcePathLabel" readonly />
                    </n-form-item>
                  </n-grid-item>
                  <n-grid-item :span="12">
                    <n-form-item label="当前目录 ID">
                      <n-input :value="currentFolderIdLabel" readonly />
                    </n-form-item>
                  </n-grid-item>
                </n-grid>

                <n-grid :cols="24" :x-gap="16" :y-gap="8">
                  <n-grid-item :span="8">
                    <n-form-item label="已保存目录 ID">
                      <n-input :value="savedFolderIdLabel" readonly />
                    </n-form-item>
                  </n-grid-item>
                  <n-grid-item :span="16">
                    <n-form-item label="已保存 CAS归集路径">
                      <n-input :value="savedCasPathLabel" readonly />
                    </n-form-item>
                  </n-grid-item>
                </n-grid>

                <n-space justify="space-between">
                  <n-space>
                    <n-button @click="loadSourceRoot">加载根目录</n-button>
                    <n-button @click="goParentFolder" :disabled="sourceFolderStack.length === 0">返回上级</n-button>
                    <n-button type="primary" @click="saveSourceConfig">保存归集目录</n-button>
                  </n-space>
                  <n-text depth="3">当前目录就是 CAS归集路径，也是 CAS访问路径；保存后下方恢复会直接复用这条路径。</n-text>
                </n-space>
              </n-form>

              <n-data-table
                :columns="sourceColumns"
                :data="sourceEntries"
                :loading="sourceLoading"
                :pagination="false"
                size="small"
              />
            </n-space>
          </n-card>
        </n-grid-item>

        <n-grid-item :span="24">
          <n-card title="默认恢复配置" size="small">
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
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
  NDataTable,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
  NSwitch,
  NText,
  NUl,
  NLi,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import type { RestoreCasRequest, CasUploadRoute, CasDestinationType } from '@/api/media'
import { restoreCas } from '@/api/media'
import { getCloudTokenList } from '@/api/cloudtoken'
import { getSettingAddition, modifySettingAddition } from '@/api/setting'
import { getFamilyFiles, getFamilyList, getPersonFiles, type FileNode } from '@/api/storage/advance'

type InputMode = 'virtualId' | 'path' | 'explicit'
type SourceType = 'person' | 'family'

interface CasDefaults {
  uploadRoute: CasUploadRoute
  destinationType: CasDestinationType
  targetFolderId: string
  inputMode: InputMode
}

interface CasRestoreForm extends RestoreCasRequest {
  inputMode: InputMode
}

interface CasSourceForm {
  enabled: boolean
  autoCollectEnabled: boolean
  preservePath: boolean
  cloudToken?: number
  sourceType: SourceType
  familyId?: string
  parentId: string
  parentName: string
  casAccessPath: string
}

const DEFAULTS_KEY = 'cas.config.defaults'
const message = useMessage()
const restoring = ref(false)
const resultText = ref('')
const cloudTokenLoading = ref(false)
const familyLoading = ref(false)
const sourceLoading = ref(false)
const sourceEntries = ref<FileNode[]>([])
const cloudTokenOptions = ref<{ label: string; value: number }[]>([])
const familyOptions = ref<{ label: string; value: string }[]>([])
const sourceFolderStack = ref<Array<{ id: string; name: string }>>([])
const savedSourceFolderId = ref('')

const uploadRouteOptions = [
  { label: 'family（家庭路线，默认）', value: 'family' },
  { label: 'person（个人路线）', value: 'person' },
]

const sourceTypeOptions = [
  { label: '个人云盘目录', value: 'person' },
  { label: '家庭云盘目录', value: 'family' },
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

const emptySourceForm = (): CasSourceForm => ({
  enabled: false,
  autoCollectEnabled: false,
  preservePath: true,
  cloudToken: undefined,
  sourceType: 'person',
  familyId: undefined,
  parentId: '-11',
  parentName: '根目录',
  casAccessPath: '',
})

const defaultForm = reactive<CasDefaults>(emptyDefaults())
const sourceForm = reactive<CasSourceForm>(emptySourceForm())
const restoreForm = reactive<CasRestoreForm>({
  ...emptyDefaults(),
  casVirtualId: undefined,
  casPath: '',
  storageId: undefined,
  mountPointId: undefined,
  casFileId: '',
  casFileName: '',
})

const defaultParentId = computed(() => (sourceForm.sourceType === 'person' ? '-11' : ''))
const sourcePathLabel = computed(() => {
  const root = sourceForm.sourceType === 'person' ? '/个人云盘' : '/家庭云盘'
  if (sourceFolderStack.value.length === 0) {
    const currentId = sourceForm.parentId || defaultParentId.value || ''
    if (sourceForm.casAccessPath && currentId === savedSourceFolderId.value) {
      return sourceForm.casAccessPath
    }
    return root
  }
  return root + '/' + sourceFolderStack.value.map((item) => item.name).join('/')
})

const savedCasPathLabel = computed(() => sourceForm.casAccessPath || '未保存')
const currentFolderIdLabel = computed(() => sourceForm.parentId || defaultParentId.value || '')
const savedFolderIdLabel = computed(() => savedSourceFolderId.value || '未保存')

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

const sourceColumns: DataTableColumns<FileNode> = [
  {
    title: '名称',
    key: 'name',
  },
  {
    title: '类型',
    key: 'isFolder',
    render: (row) => (row.isFolder === 1 ? '目录' : '文件'),
  },
  {
    title: '操作',
    key: 'actions',
    render: (row) => {
      if (row.isFolder === 1) {
        return h(
          NButton,
          {
            size: 'small',
            onClick: () => enterFolder(row),
          },
          { default: () => '进入目录' }
        )
      }
      if ((row.name || '').toLowerCase().endsWith('.cas')) {
        return h(
          NButton,
          {
            size: 'small',
            type: 'primary',
            onClick: () => useCasFile(row),
          },
          { default: () => '使用这个 .cas' }
        )
      }
      return h('span', { style: 'color: var(--n-text-color-3); font-size: 12px;' }, '仅目录或 .cas 可操作')
    },
  },
]

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

watch(
  () => sourceForm.cloudToken,
  async (val) => {
    familyOptions.value = []
    sourceEntries.value = []
    sourceFolderStack.value = []
    sourceForm.parentId = defaultParentId.value
    sourceForm.parentName = '根目录'
    if (!val) return
    if (sourceForm.sourceType === 'family') {
      await loadFamilyList()
    }
  }
)

watch(
  () => sourceForm.sourceType,
  async (val) => {
    sourceEntries.value = []
    sourceFolderStack.value = []
    sourceForm.parentId = val === 'person' ? '-11' : ''
    sourceForm.parentName = '根目录'
    sourceForm.familyId = undefined
    familyOptions.value = []
    if (val === 'family' && sourceForm.cloudToken) {
      await loadFamilyList()
    }
  }
)

watch(
  () => sourceForm.familyId,
  () => {
    sourceEntries.value = []
    sourceFolderStack.value = []
    sourceForm.parentId = ''
    sourceForm.parentName = '根目录'
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
  message.success('CAS 默认恢复配置已保存')
}

const resetDefaults = () => {
  Object.assign(defaultForm, emptyDefaults())
  localStorage.setItem(DEFAULTS_KEY, JSON.stringify(defaultForm))
  message.success('已恢复默认恢复配置')
}

const applyDefaultsToRestore = () => {
  restoreForm.uploadRoute = defaultForm.uploadRoute
  restoreForm.destinationType = defaultForm.destinationType
  restoreForm.targetFolderId = defaultForm.targetFolderId
  restoreForm.inputMode = defaultForm.inputMode
  message.success('已应用默认恢复配置')
}

const resetRestore = () => {
  Object.assign(restoreForm, {
    ...defaultForm,
    casVirtualId: undefined,
    casPath: sourcePathLabel.value || '',
    storageId: undefined,
    mountPointId: undefined,
    casFileId: '',
    casFileName: '',
  })
  resultText.value = ''
}

const loadCloudTokens = async () => {
  cloudTokenLoading.value = true
  try {
    const res = await getCloudTokenList({ currentPage: 1, pageSize: 100, noPaginate: true })
    const list = res.data?.data || []
    cloudTokenOptions.value = list.map((item) => ({
      label: `${item.name || item.username || `Token ${item.id}`} (#${item.id})`,
      value: item.id,
    }))
  } finally {
    cloudTokenLoading.value = false
  }
}

const loadFamilyList = async () => {
  if (!sourceForm.cloudToken) return
  familyLoading.value = true
  try {
    const res = await getFamilyList({ cloudToken: sourceForm.cloudToken })
    familyOptions.value = (res.data?.familyInfoResp || []).map((item) => ({
      label: `${item.remarkName || item.familyId} (${item.familyId})`,
      value: item.familyId,
    }))
  } finally {
    familyLoading.value = false
  }
}

const loadSourceRoot = async () => {
  if (!sourceForm.cloudToken) {
    message.warning('请先选择云盘账号')
    return
  }
  if (sourceForm.sourceType === 'family' && !sourceForm.familyId) {
    message.warning('家庭目录模式下请先选择家庭组')
    return
  }
  sourceLoading.value = true
  try {
    if (sourceForm.sourceType === 'person') {
      const res = await getPersonFiles({
        pageNum: 1,
        pageSize: 100,
        cloudToken: sourceForm.cloudToken,
        parentId: sourceForm.parentId || '-11',
      })
      sourceEntries.value = res.data?.data || []
    } else {
      const res = await getFamilyFiles({
        pageNum: 1,
        pageSize: 100,
        cloudToken: sourceForm.cloudToken,
        familyId: sourceForm.familyId!,
        parentId: sourceForm.parentId || '',
      })
      sourceEntries.value = res.data?.data || []
    }
  } catch (err: any) {
    message.error(err?.message || '加载 CAS 来源目录失败')
  } finally {
    sourceLoading.value = false
  }
}

const enterFolder = async (row: FileNode) => {
  sourceFolderStack.value.push({ id: row.id, name: row.name })
  sourceForm.parentId = row.id
  sourceForm.parentName = row.name
  await loadSourceRoot()
}

const goParentFolder = async () => {
  sourceFolderStack.value.pop()
  const current = sourceFolderStack.value[sourceFolderStack.value.length - 1]
  sourceForm.parentId = current?.id || defaultParentId.value
  sourceForm.parentName = current?.name || '根目录'
  await loadSourceRoot()
}

const useCasFile = (row: FileNode) => {
  const fullPath = `${sourcePathLabel.value}/${row.name}`
  restoreForm.inputMode = 'path'
  restoreForm.casPath = fullPath
  restoreForm.casFileId = row.id
  restoreForm.casFileName = row.name
  message.success('已把该 .cas 带入恢复表单')
}

const saveSourceConfig = async () => {
  if (!sourceForm.cloudToken) {
    message.warning('请先选择云盘账号')
    return
  }
  if (!sourceForm.parentId && sourceForm.sourceType === 'family') {
    message.warning('请先选择家庭目录')
    return
  }

  const resolvedCasPath = sourcePathLabel.value
  sourceForm.casAccessPath = resolvedCasPath
  sourceForm.enabled = true
  sourceForm.autoCollectEnabled = true

  const savedFolderId = sourceForm.parentId || defaultParentId.value
  await modifySettingAddition({
    casTargetEnabled: true,
    casTargetTokenId: sourceForm.cloudToken,
    casTargetType: sourceForm.sourceType,
    casTargetFamilyId: sourceForm.familyId,
    casTargetFolderId: savedFolderId,
    casAccessPath: resolvedCasPath,
    casAutoCollectEnabled: true,
    casAutoCollectPreservePath: sourceForm.preservePath,
  })

  savedSourceFolderId.value = savedFolderId
  restoreForm.inputMode = 'path'
  restoreForm.casPath = resolvedCasPath
  message.success(`CAS归集路径已保存：${resolvedCasPath}`)
}

const loadSourceConfigFromServer = async () => {
  const res = await getSettingAddition()
  const addition: Models.SettingAddition = res.data || ({} as Models.SettingAddition)
  sourceForm.enabled = addition.casTargetEnabled !== false
  sourceForm.autoCollectEnabled = addition.casAutoCollectEnabled !== false
  sourceForm.preservePath = addition.casAutoCollectPreservePath !== false
  sourceForm.cloudToken = addition.casTargetTokenId || undefined
  sourceForm.sourceType = (addition.casTargetType as SourceType) || 'person'
  sourceForm.familyId = addition.casTargetFamilyId || undefined
  sourceForm.parentId = addition.casTargetFolderId || (sourceForm.sourceType === 'person' ? '-11' : '')
  savedSourceFolderId.value = addition.casTargetFolderId || ''
  sourceForm.parentName = addition.casAccessPath || '根目录'
  sourceForm.casAccessPath = addition.casAccessPath || ''
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

onMounted(async () => {
  loadDefaults()
  await loadCloudTokens()
  await loadSourceConfigFromServer()
  applyDefaultsToRestore()
  if (sourceForm.cloudToken && sourceForm.sourceType === 'family') {
    await loadFamilyList()
  }
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
