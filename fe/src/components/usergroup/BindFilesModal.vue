<template>
  <n-modal v-model:show="visible" preset="dialog" title="绑定存储" style="width: 800px">
    <div class="bind-files-modal">
      <!-- 搜索区域 -->
      <div class="search-section">
        <n-space>
          <n-input
            v-model:value="searchKeyword"
            placeholder="搜索存储挂载点..."
            clearable
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <n-icon :component="SearchOutline" />
            </template>
          </n-input>
          <n-button type="primary" @click="handleSearch">搜索</n-button>
        </n-space>
      </div>

      <!-- 存储列表 -->
      <div class="file-list-section">
        <n-data-table
          :columns="columns"
          :data="storageList"
          :loading="loading"
          :row-key="(row) => row.id"
          :checked-row-keys="selectedStorageIds"
          @update:checked-row-keys="handleSelectionChange"
        />
      </div>

      <!-- 已选择的存储 -->
      <div class="selected-section" v-if="selectedStorageIds.length > 0">
        <n-divider style="margin: 12px 0 8px" />
        <div class="selected-header">
          <span>已选择 {{ selectedStorageIds.length }} 个存储挂载点</span>
          <n-button text type="error" @click="clearSelection">清空选择</n-button>
        </div>
        <div class="selected-files">
          <!-- 显示前10个标签 -->
          <n-tag
            v-for="storageId in displayedStorageIds"
            :key="storageId"
            size="small"
            closable
            :title="getFullStorageName(storageId)"
            @close="removeSelection(storageId)"
          >
            {{ getStorageName(storageId) }}
          </n-tag>
          <!-- 如果超过10个，显示省略提示 -->
          <n-tag v-if="selectedStorageIds.length > 10" size="small" type="info">
            +{{ selectedStorageIds.length - 10 }} 更多...
          </n-tag>
        </div>
      </div>
    </div>

    <template #action>
      <n-space>
        <n-button @click="handleCancel">取消</n-button>
        <n-button
          type="primary"
          :loading="submitting"
          :disabled="selectedStorageIds.length === 0"
          @click="handleConfirm"
        >
          确定绑定
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import {
  NModal,
  NInput,
  NButton,
  NSpace,
  NIcon,
  NDataTable,
  NDivider,
  NTag,
  useMessage,
  type DataTableColumns,
} from 'naive-ui'
import { SearchOutline } from '@vicons/ionicons5'
import { getStorageSelectList, type StorageSelectItem } from '@/api/storage'
import { batchBindFiles, getBindFiles } from '@/api/usergroup'

interface Props {
  show: boolean
  userGroupInfo?: Models.UserGroup | null
}

interface Emits {
  (e: 'update:show', value: boolean): void
  (e: 'success'): void
}

const props = withDefaults(defineProps<Props>(), {
  show: false,
  userGroupInfo: null,
})

const emit = defineEmits<Emits>()
const message = useMessage()

// 响应式数据
const visible = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value),
})

const searchKeyword = ref('')
const loading = ref(false)
const submitting = ref(false)
const storageList = ref<StorageSelectItem[]>([])
const selectedStorageIds = ref<number[]>([])

// 显示的存储ID列表（最多显示10个）
const displayedStorageIds = computed(() => {
  return selectedStorageIds.value.slice(0, 10)
})

// 表格列配置
const columns: DataTableColumns<StorageSelectItem> = [
  {
    type: 'selection',
  },
  {
    title: '存储名称',
    key: 'name',
    ellipsis: {
      tooltip: true,
    },
  },
  {
    title: '存储路径',
    key: 'path',
    ellipsis: {
      tooltip: true,
    },
  },
]

// 监听 props 变化
watch(
  () => props.show,
  (newShow) => {
    if (newShow) {
      // 重置状态
      searchKeyword.value = ''
      selectedStorageIds.value = []
      // 初始加载存储列表和已绑定文件
      nextTick(() => {
        Promise.all([handleSearch(), loadBindFiles()])
      })
    }
  }
)

// 加载已绑定的文件
const loadBindFiles = () => {
  if (!props.userGroupInfo?.id) return Promise.resolve()

  return getBindFiles(props.userGroupInfo.id)
    .then((response) => {
      if (response.code === 200 && response.data) {
        // 预选已绑定的文件ID
        selectedStorageIds.value = response.data.fileIds || []
      } else {
        console.warn('获取已绑定文件失败:', response.msg)
      }
    })
    .catch((error) => {
      console.error('获取已绑定文件失败:', error)
    })
}

// 搜索存储
const handleSearch = () => {
  if (!visible.value) return

  loading.value = true

  const params = {
    name: searchKeyword.value || undefined,
    path: searchKeyword.value || undefined,
  }

  return getStorageSelectList(params)
    .then((response) => {
      if (response.code === 200 && response.data) {
        storageList.value = response.data
      } else {
        message.error(response.msg || '获取存储列表失败')
      }
    })
    .catch((error) => {
      console.error('获取存储列表失败:', error)
      message.error('获取存储列表失败')
    })
    .finally(() => {
      loading.value = false
    })
}

// 处理选择变化
const handleSelectionChange = (keys: Array<string | number>) => {
  selectedStorageIds.value = keys.map((key) => Number(key))
}

// 清空选择
const clearSelection = () => {
  selectedStorageIds.value = []
}

// 移除单个选择
const removeSelection = (storageId: number) => {
  const index = selectedStorageIds.value.indexOf(storageId)
  if (index > -1) {
    selectedStorageIds.value.splice(index, 1)
  }
}

// 获取存储名称（限制长度）
const getStorageName = (storageId: number) => {
  const storage = storageList.value.find((s) => s.id === storageId)
  const name = storage ? storage.name : `存储ID: ${storageId}`
  // 限制标签显示长度，超过20个字符显示省略号
  return name.length > 20 ? name.substring(0, 20) + '...' : name
}

// 获取完整存储名称（用于tooltip）
const getFullStorageName = (storageId: number) => {
  const storage = storageList.value.find((s) => s.id === storageId)
  return storage ? storage.name : `存储ID: ${storageId}`
}

// 取消操作
const handleCancel = () => {
  visible.value = false
}

// 确认绑定
const handleConfirm = () => {
  if (!props.userGroupInfo?.id || selectedStorageIds.value.length === 0) {
    message.warning('请选择要绑定的存储')
    return
  }

  submitting.value = true

  return batchBindFiles({
    groupId: props.userGroupInfo.id,
    fileIds: selectedStorageIds.value,
  })
    .then((response) => {
      if (response.code === 200) {
        message.success('存储绑定成功')
        visible.value = false
        emit('success')
      } else {
        message.error(response.msg || '存储绑定失败')
      }
    })
    .catch((error) => {
      console.error('存储绑定失败:', error)
      message.error('存储绑定失败')
    })
    .finally(() => {
      submitting.value = false
    })
}
</script>

<style scoped>
.bind-files-modal {
  max-height: 600px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.search-section {
  margin-bottom: 16px;
  flex-shrink: 0;
}

.file-list-section {
  flex: 1;
  min-height: 300px;
  max-height: 400px;
  overflow: auto;
}

.selected-section {
  margin-top: 16px;
  flex-shrink: 0;
  max-height: 150px;
  overflow: hidden;
}

.selected-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-weight: 500;
}

.selected-files {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  max-height: 120px;
  overflow-y: auto;
  padding: 4px;
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
  background-color: var(--n-color-target);
}

/* 自定义滚动条样式 */
.file-list-section::-webkit-scrollbar,
.selected-files::-webkit-scrollbar {
  width: 6px;
}

.file-list-section::-webkit-scrollbar-track,
.selected-files::-webkit-scrollbar-track {
  background: var(--n-scrollbar-color);
  border-radius: 3px;
}

.file-list-section::-webkit-scrollbar-thumb,
.selected-files::-webkit-scrollbar-thumb {
  background: var(--n-scrollbar-color-hover);
  border-radius: 3px;
}

.file-list-section::-webkit-scrollbar-thumb:hover,
.selected-files::-webkit-scrollbar-thumb:hover {
  background: var(--n-scrollbar-color-pressed);
}
</style>
