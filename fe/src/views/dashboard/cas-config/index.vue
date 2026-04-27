<template>
  <div class="cas-config-page">
    <n-space vertical size="large">
      <n-grid :cols="24" :x-gap="16" :y-gap="16">
        <n-grid-item :span="24">
          <CasSourceConfigCard
            :source-form="sourceForm"
            :cloud-token-options="cloudTokenOptions"
            :source-type-options="sourceTypeOptions"
            :family-options="familyOptions"
            :retention-options="retentionOptions"
            :cloud-token-loading="cloudTokenLoading"
            :family-loading="familyLoading"
            :source-loading="sourceLoading"
            :source-path-label="sourcePathLabel"
            :current-folder-id-label="currentFolderIdLabel"
            :family-group-folder-id-label="familyGroupFolderIdLabel"
            :saved-folder-id-label="savedFolderIdLabel"
            :saved-cas-path-label="savedCasPathLabel"
            :source-folder-stack="sourceFolderStack"
            :source-columns="sourceColumns"
            :source-entries="sourceEntries"
            @load-root="loadSourceRoot"
            @go-parent="goParentFolder"
            @save-source="saveSourceConfig"
          />
        </n-grid-item>
        <n-grid-item :span="24">
          <CasLocalStrmCard
            :source-form="sourceForm"
            :manual-scanning="manualLocalStrmScanning"
            :manual-fallback-recycle-running="manualFallbackRecycleRunning"
            @save-settings="saveLocalStrmSettings"
            @manual-scan="handleManualLocalStrmScan"
            @manual-fallback-recycle="handleManualFallbackRecycle"
          />
        </n-grid-item>
      </n-grid>
    </n-space>
  </div>
</template>

<script setup lang="ts">
/**
 * CAS 配置页开发规范（必须遵守）
 *
 * 这个页面已经按“功能区拆分 + 页面编排层收口”的方式重构完成。
 * 后续无论是人还是 AI 修改这里，都不要再把它改回单文件大泥坑。
 *
 * 一眼看懂的规则：
 * 1. index.vue 只做页面编排、状态组装、跨卡片联动，不承载臃肿 UI 细节。
 * 2. 每个功能区都应该是独立组件：来源目录、默认配置、手动恢复、请求预览、链路说明、恢复结果。
 * 3. 带确认弹窗/异步 loading/消息提示的动作逻辑，优先放 composables 或独立组件，不要继续塞回 index.vue。
 * 4. 不要为了“快”把按钮、弹窗、接口调用、表单状态再堆回一个超长 SFC。
 * 5. 如果新增 CAS 功能区：优先新建 components/*Card.vue；如果新增复用动作：优先新建 composables/useXxx.ts。
 * 6. 只有跨多个卡片共享的业务状态，才放在 index.vue；卡片私有交互应留在子组件内部。
 * 7. 修改后必须保证：前端 build 通过，且不要留下重复 style/script 片段、脏尾巴、未使用 import/变量。
 *
 * 禁止事项：
 * - 禁止把清空缓存、重建缓存、恢复动作确认逻辑重新塞回单文件页面。
 * - 禁止把多个功能区的模板/状态/接口调用混写成一个几百上千行的块。
 * - 禁止在未拆分职责的前提下继续“顺手加几行”式补丁开发。
 */
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { NAlert, NButton, NGrid, NGridItem, NSpace, NText, useMessage, type DataTableColumns } from 'naive-ui'
import type { RestoreCasRequest, CasDestinationType, CasUploadRoute } from '@/api/media'
import { rebuildLocalCASSTRM, restoreCas, runFallbackRecycleOnce } from '@/api/media'
import { getCloudTokenList } from '@/api/cloudtoken'
import { getSettingAddition, modifySettingAddition } from '@/api/setting'
import { getFamilyFiles, getFamilyList, getPersonFiles, type FileNode } from '@/api/storage/advance'
import CasDefaultsCard from './components/CasDefaultsCard.vue'
import CasLinkInfoCard from './components/CasLinkInfoCard.vue'
import CasManualRestoreCard from './components/CasManualRestoreCard.vue'
import CasRequestPreviewCard from './components/CasRequestPreviewCard.vue'
import CasResultCard from './components/CasResultCard.vue'
import CasLocalStrmCard from './components/CasLocalStrmCard.vue'
import CasSourceConfigCard from './components/CasSourceConfigCard.vue'

void [NAlert, NText, CasDefaultsCard, CasLinkInfoCard, CasManualRestoreCard, CasRequestPreviewCard, CasResultCard]

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
  cloudToken?: number
  sourceType: SourceType
  familyId?: string
  fixedFamilyId?: string
  parentId: string
  parentName: string
  casAccessPath: string
  retentionHours?: number
  localCasAutoScanEnabled: boolean
  localCasAutoScanIntervalMin: number
}

const DEFAULTS_KEY = 'cas.config.defaults'
const message = useMessage()
const restoring = ref(false)
const resultText = ref('')
const cloudTokenLoading = ref(false)
const familyLoading = ref(false)
const sourceLoading = ref(false)
const manualLocalStrmScanning = ref(false)
const manualFallbackRecycleRunning = ref(false)
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

const retentionOptions = [
  { label: '永久保留', value: 0 },
  { label: '1 小时', value: 1 },
  { label: '6 小时', value: 6 },
  { label: '12 小时', value: 12 },
  { label: '1 天', value: 24 },
  { label: '3 天', value: 72 },
  { label: '7 天', value: 168 },
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
  cloudToken: undefined,
  sourceType: 'person',
  familyId: undefined,
  fixedFamilyId: undefined,
  parentId: '-11',
  parentName: '根目录',
  casAccessPath: '',
  retentionHours: undefined,
  localCasAutoScanEnabled: false,
  localCasAutoScanIntervalMin: 10,
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

const defaultParentId = computed(() => (sourceForm.sourceType === 'family' ? '-16' : '-11'))
const sourcePathLabel = computed(() => {
  const root = sourceForm.sourceType === 'person' ? '/个人云盘' : '/家庭云盘'
  if (sourceFolderStack.value.length === 0) {
    return root
  }
  return `${root}/${sourceFolderStack.value.map((item) => item.name).join('/')}`
})

const savedCasPathLabel = computed(() => sourceForm.casAccessPath || '未保存')
const currentFolderIdLabel = computed(() => {
  return sourceForm.parentId || defaultParentId.value || ''
})
const familyGroupFolderIdLabel = computed(() => sourceForm.fixedFamilyId || sourceForm.familyId || '未选择')
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
  { title: '名称', key: 'name' },
  { title: '类型', key: 'isFolder', render: (row) => (row.isFolder === 1 ? '目录' : '文件') },
  {
    title: '操作',
    key: 'actions',
    render: (row) => {
      if (row.isFolder === 1) {
        return h(NButton, { size: 'small', onClick: () => enterFolder(row) }, { default: () => '进入目录' })
      }
      if ((row.name || '').toLowerCase().endsWith('.cas')) {
        return h(NButton, { size: 'small', type: 'primary', onClick: () => useCasFile(row) }, { default: () => '使用这个 .cas' })
      }
      return h('span', { style: 'color: var(--n-text-color-3); font-size: 12px;' }, '仅目录或 .cas 可操作')
    },
  },
]

watch(
  () => defaultForm.uploadRoute,
  (val) => {
    if (val === 'person') defaultForm.destinationType = 'person'
  }
)

watch(
  () => restoreForm.uploadRoute,
  (val) => {
    if (val === 'person') restoreForm.destinationType = 'person'
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
    if (sourceForm.sourceType === 'family') await loadFamilyList()
  }
)

watch(
  () => sourceForm.sourceType,
  async (val) => {
    sourceEntries.value = []
    sourceFolderStack.value = []
    sourceForm.familyId = val === 'family' ? sourceForm.fixedFamilyId : undefined
    sourceForm.parentId = val === 'family' ? '-16' : '-11'
    sourceForm.parentName = '根目录'
    if (val !== 'family') {
      sourceForm.fixedFamilyId = undefined
    }
    familyOptions.value = []
    if (val === 'family' && sourceForm.cloudToken) await loadFamilyList()
  }
)

watch(
  () => sourceForm.fixedFamilyId,
  (val) => {
    if (sourceForm.sourceType !== 'family') return
    sourceForm.familyId = val || undefined
    sourceEntries.value = []
    sourceFolderStack.value = []
    sourceForm.parentId = '-16'
    sourceForm.parentName = '根目录'
  }
)

watch(
  () => sourceForm.familyId,
  () => {
    sourceEntries.value = []
    sourceFolderStack.value = []
    sourceForm.parentId = '-16'
    sourceForm.parentName = '根目录'
  }
)

const loadDefaults = () => {
  try {
    const raw = localStorage.getItem(DEFAULTS_KEY)
    if (!raw) return
    Object.assign(defaultForm, emptyDefaults(), JSON.parse(raw) as Partial<CasDefaults>)
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
  restoreForm.uploadRoute = 'family'
  restoreForm.destinationType = sourceForm.sourceType
  restoreForm.targetFolderId = savedSourceFolderId.value || defaultForm.targetFolderId
  restoreForm.inputMode = defaultForm.inputMode
  message.success('已按当前生效链路应用恢复配置')
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
    if (sourceForm.sourceType === 'family') {
      if (!sourceForm.fixedFamilyId && familyOptions.value.length > 0) {
        sourceForm.fixedFamilyId = familyOptions.value[0].value
      }
      if (!sourceForm.familyId) {
        sourceForm.familyId = sourceForm.fixedFamilyId
      }
    }
  } finally {
    familyLoading.value = false
  }
}

const loadSourceEntries = async (parentId: string) => {
  if (!sourceForm.cloudToken) {
    message.warning('请先选择云盘账号')
    return
  }
  if (sourceForm.sourceType === 'family' && !(sourceForm.familyId || sourceForm.fixedFamilyId)) {
    message.warning('家庭目录模式下请先选择 CAS 指定恢复位置')
    return
  }
  sourceLoading.value = true
  try {
    if (sourceForm.sourceType === 'person') {
      const res = await getPersonFiles({ pageNum: 1, pageSize: 100, cloudToken: sourceForm.cloudToken, parentId })
      sourceEntries.value = res.data?.data || []
    } else {
      const res = await getFamilyFiles({
        pageNum: 1,
        pageSize: 100,
        cloudToken: sourceForm.cloudToken,
        familyId: (sourceForm.familyId || sourceForm.fixedFamilyId)!,
        parentId,
      })
      sourceEntries.value = res.data?.data || []
    }
  } catch (err: any) {
    message.error(err?.message || '加载 CAS 来源目录失败')
  } finally {
    sourceLoading.value = false
  }
}

const loadSourceRoot = async () => {
  sourceFolderStack.value = []
  sourceForm.parentId = defaultParentId.value
  sourceForm.parentName = '根目录'
  await loadSourceEntries(defaultParentId.value)
}

const enterFolder = async (row: FileNode) => {
  sourceFolderStack.value.push({ id: row.id, name: row.name })
  sourceForm.parentId = row.id
  sourceForm.parentName = row.name
  await loadSourceEntries(row.id)
}

const goParentFolder = async () => {
  sourceFolderStack.value.pop()
  const current = sourceFolderStack.value[sourceFolderStack.value.length - 1]
  sourceForm.parentId = current?.id || defaultParentId.value
  sourceForm.parentName = current?.name || '根目录'
  await loadSourceEntries(sourceForm.parentId)
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

  const resolvedTargetPath = sourcePathLabel.value
  sourceForm.casAccessPath = resolvedTargetPath

  const savedFolderId = sourceForm.sourceType === 'family' && sourceFolderStack.value.length === 0
    ? (sourceForm.fixedFamilyId || sourceForm.familyId || '')
    : (sourceForm.parentId || defaultParentId.value)
  await modifySettingAddition({
    casTargetEnabled: true,
    casTargetType: sourceForm.sourceType,
    casPersonTargetTokenId: sourceForm.sourceType === 'person' ? sourceForm.cloudToken : 0,
    casPersonTargetFolderId: sourceForm.sourceType === 'person' ? savedFolderId : '',
    casPersonAccessPath: sourceForm.sourceType === 'person' ? resolvedTargetPath : '',
    casFamilyTargetTokenId: sourceForm.sourceType === 'family' ? sourceForm.cloudToken : 0,
    casFamilyTargetFamilyId: sourceForm.sourceType === 'family' ? (sourceForm.fixedFamilyId || sourceForm.familyId || '') : '',
    casFamilyTargetFolderId: sourceForm.sourceType === 'family' ? savedFolderId : '',
    casFamilyAccessPath: sourceForm.sourceType === 'family' ? resolvedTargetPath : '',
    casRestoreRetentionHours: sourceForm.retentionHours,
    localCasAutoScanEnabled: sourceForm.localCasAutoScanEnabled,
    localCasAutoScanIntervalMin: sourceForm.localCasAutoScanIntervalMin,
  })

  savedSourceFolderId.value = savedFolderId
  defaultForm.uploadRoute = 'family'
  defaultForm.destinationType = sourceForm.sourceType
  defaultForm.targetFolderId = savedFolderId
  restoreForm.uploadRoute = 'family'
  restoreForm.destinationType = sourceForm.sourceType
  restoreForm.targetFolderId = savedFolderId
  restoreForm.inputMode = 'path'
  restoreForm.casPath = resolvedTargetPath
  message.success(`CAS最终目录已保存：${resolvedTargetPath}（当前仅生效 ${sourceForm.sourceType === 'person' ? '家庭→个人' : '家庭→家庭'} 链路）`)
}

const loadSourceConfigFromServer = async () => {
  const res = await getSettingAddition()
  const addition: Models.SettingAddition = res.data || ({} as Models.SettingAddition)
  sourceForm.sourceType = (addition.casTargetType as SourceType) || 'person'
  if (sourceForm.sourceType === 'family') {
    sourceForm.cloudToken = addition.casFamilyTargetTokenId || undefined
    sourceForm.fixedFamilyId = addition.casFamilyTargetFamilyId || undefined
    sourceForm.familyId = addition.casFamilyTargetFamilyId || undefined
    sourceForm.parentId = '-16'
    savedSourceFolderId.value = addition.casFamilyTargetFolderId || ''
    sourceForm.parentName = addition.casFamilyAccessPath || '根目录'
    sourceForm.casAccessPath = addition.casFamilyAccessPath || ''
  } else {
    sourceForm.cloudToken = addition.casPersonTargetTokenId || undefined
    sourceForm.fixedFamilyId = undefined
    sourceForm.familyId = undefined
    sourceForm.parentId = addition.casPersonTargetFolderId || '-11'
    savedSourceFolderId.value = addition.casPersonTargetFolderId || ''
    sourceForm.parentName = addition.casPersonAccessPath || '根目录'
    sourceForm.casAccessPath = addition.casPersonAccessPath || ''
  }
  sourceForm.retentionHours = addition.casRestoreRetentionHours || undefined
  sourceForm.localCasAutoScanEnabled = !!addition.localCasAutoScanEnabled
  sourceForm.localCasAutoScanIntervalMin = addition.localCasAutoScanIntervalMin || 10
  const resolvedDefaultTargetFolderId = sourceForm.sourceType === 'family'
    ? (addition.casFamilyTargetFolderId || '')
    : (addition.casPersonTargetFolderId || '')
  if (!defaultForm.targetFolderId) {
    defaultForm.targetFolderId = resolvedDefaultTargetFolderId
  }
  if (!restoreForm.targetFolderId) {
    restoreForm.targetFolderId = resolvedDefaultTargetFolderId
  }
}

const saveLocalStrmSettings = async () => {
  await modifySettingAddition({
    localCasAutoScanEnabled: sourceForm.localCasAutoScanEnabled,
    localCasAutoScanIntervalMin: sourceForm.localCasAutoScanIntervalMin,
  })
  message.success('CAS本地STRM设置已保存')
}

const handleManualLocalStrmScan = async () => {
  manualLocalStrmScanning.value = true
  try {
    const res = await rebuildLocalCASSTRM()
    const data = res.data
    message.success(`本地CAS扫描完成：扫描 ${data?.scanned ?? 0}，新建 ${data?.created ?? 0}，跳过 ${data?.skipped ?? 0}，失败 ${data?.failed ?? 0}`)
  } catch (err: any) {
    message.error(err?.message || '本地CAS扫描失败')
  } finally {
    manualLocalStrmScanning.value = false
  }
}

const handleManualFallbackRecycle = async () => {
  manualFallbackRecycleRunning.value = true
  try {
    await runFallbackRecycleOnce()
    message.success('CAS 兜底清理已执行完成')
  } catch (err: any) {
    message.error(err?.message || 'CAS 兜底清理执行失败')
  } finally {
    manualFallbackRecycleRunning.value = false
  }
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
void [uploadRouteOptions, inputModeOptions, defaultDestinationOptions, destinationTypeOptions, saveDefaults, resetDefaults, resetRestore, requestPreview]

const validateBeforeSubmit = (): boolean => {
  const effectiveTargetFolderId = (savedSourceFolderId.value || restoreForm.targetFolderId || '').trim()
  if (!effectiveTargetFolderId) {
    message.warning('请先保存一个生效中的最终目录')
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
void [handleRestore]

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
</style>
