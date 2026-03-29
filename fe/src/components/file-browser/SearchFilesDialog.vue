<template>
  <n-modal
    :show="show"
    preset="card"
    title="文件搜索"
    style="width: 1000px; max-width: 92vw"
    :bordered="false"
    @update:show="onUpdateShow"
  >
    <div class="search-toolbar">
      <n-checkbox v-model:checked="globalSearch">全局搜索</n-checkbox>
      <n-input
        v-model:value="keyword"
        placeholder="请输入搜索关键词"
        clearable
        @keyup.enter="doSearch"
        class="keyword-input"
      />
      <n-button type="primary" secondary :loading="searching" @click="doSearch">
        <template #icon>
          <n-icon :component="SearchOutline" />
        </template>
        搜索
      </n-button>
    </div>

    <div v-if="searching" class="loading-container">
      <n-spin />
    </div>

    <template v-else>
      <div class="dialog-section">
        <n-empty v-if="!list.length" description="暂无搜索结果" />
        <n-data-table
          v-else
          :columns="columns"
          :data="list"
          :loading="searching"
          :bordered="false"
          :single-line="false"
          :row-props="rowProps"
        />
      </div>
      <div class="pagination">
        <div class="summary">共 {{ total }} 条结果，第 {{ currentPage }} / {{ totalPages }} 页</div>
        <div class="pager">
          <n-button size="small" tertiary :disabled="currentPage <= 1" @click="prevPage"
            >上一页</n-button
          >
          <n-button
            size="small"
            type="primary"
            ghost
            :disabled="currentPage >= totalPages"
            @click="nextPage"
            >下一页</n-button
          >
        </div>
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch, computed, h } from 'vue'
import {
  NModal,
  NCheckbox,
  NInput,
  NButton,
  NIcon,
  NDataTable,
  NSpin,
  NEmpty,
  NEllipsis,
  NTooltip,
  useMessage,
} from 'naive-ui'
import { SearchOutline } from '@vicons/ionicons5'
import { searchFiles, type FileSearchItem, type FileSearchResponse } from '@/api/file'
import { formatFileSize } from '@/utils/format'

interface Props {
  show: boolean
  // 当前目录的文件 id（用于局部搜索时作为 pid）
  currentDirId?: number | null
  // 默认页大小
  pageSize?: number
}

const props = defineProps<Props>()
const emits = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'select', row: FileSearchItem): void
}>()

const message = useMessage()

const show = ref<boolean>(props.show)
const keyword = ref<string>('')
const globalSearch = ref<boolean>(true)
const searching = ref<boolean>(false)

const list = ref<FileSearchItem[]>([])
const total = ref<number>(0)
const currentPage = ref<number>(1)
const pageSize = ref<number>(props.pageSize ?? 15)

const totalPages = computed(() => {
  if (total.value === 0) return 1
  return Math.max(1, Math.ceil(total.value / pageSize.value))
})

watch(
  () => props.show,
  (val) => {
    show.value = val
    if (val) {
      // 重置分页、列表
      currentPage.value = 1
      list.value = []
      total.value = 0
      globalSearch.value = true
    }
  }
)

const onUpdateShow = (val: boolean) => {
  show.value = val
  emits('update:show', val)
}

const columns = [
  {
    title: '名称',
    key: 'name',
    render(row: FileSearchItem) {
      return row.name
    },
  },
  {
    title: '类型',
    key: 'type',
    render(row: FileSearchItem) {
      return row.isDir ? '目录' : '文件'
    },
    width: 100,
  },
  {
    title: '大小',
    key: 'size',
    render(row: FileSearchItem) {
      return row.isDir ? '-' : formatFileSize(row.size || 0)
    },
    width: 140,
  },
  {
    title: '路径',
    key: 'fullPath',
    render(row: FileSearchItem) {
      const text = row.fullPath || '-'
      return h(
        NTooltip,
        { placement: 'top' },
        {
          default: () => text,
          trigger: () =>
            h(NEllipsis, { style: 'max-width: 100%;', lineClamp: 1 }, { default: () => text }),
        }
      )
    },
  },
]

const rowProps = (row: FileSearchItem) => {
  return {
    style: 'cursor: pointer;',
    onClick: () => {
      emits('select', row)
      emits('update:show', false)
    },
  }
}

const buildQuery = () => {
  const q: {
    keyword?: string
    pid?: number
    global?: boolean
    pageSize: number
    currentPage: number
  } = {
    pageSize: pageSize.value,
    currentPage: currentPage.value,
  }
  if (keyword.value?.trim()) {
    q.keyword = keyword.value.trim()
  }
  if (globalSearch.value) {
    q.global = true
  } else if (props.currentDirId != null) {
    q.pid = Number(props.currentDirId)
  }
  return q
}

const doSearch = () => {
  searching.value = true
  const query = buildQuery()
  searchFiles(query)
    .then((res) => {
      if (res.code === 200 && res.data) {
        const data = res.data as FileSearchResponse
        list.value = data.data || []
        total.value = data.total || 0
        currentPage.value = data.currentPage || query.currentPage
        pageSize.value = data.pageSize || query.pageSize
      } else {
        message.error(res.msg || '搜索失败')
      }
    })
    .catch((err) => {
      console.error('搜索失败: ', err)
      message.error('搜索失败')
    })
    .finally(() => {
      searching.value = false
    })
}

const prevPage = () => {
  if (currentPage.value > 1) {
    currentPage.value -= 1
    doSearch()
  }
}

const nextPage = () => {
  if (currentPage.value < totalPages.value) {
    currentPage.value += 1
    doSearch()
  }
}
</script>

<style scoped>
.search-toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 16px;
  padding: 10px;
  background: var(--n-color-embedded);
  border: 1px solid var(--n-border-color);
  border-radius: 8px;
}

.keyword-input {
  flex: 1;
}

:deep(.keyword-input .n-input) {
  --n-color: var(--n-card-color);
  --n-color-focus: var(--n-card-color);
  --n-text-color: var(--n-text-color);
  --n-placeholder-color: var(--n-text-color-disabled);
  --n-caret-color: var(--n-primary-color);
  --n-border: 1px solid var(--n-border-color);
  --n-border-hover: 1px solid var(--n-primary-color-hover);
  --n-border-focus: 1px solid var(--n-primary-color);
  --n-box-shadow-focus: 0 0 0 2px rgb(24 160 88 / 15%);
}

:deep(.keyword-input .n-input__input-el) {
  color: var(--n-text-color) !important;
  -webkit-text-fill-color: var(--n-text-color);
}

.dialog-section {
  max-height: 60vh;
  overflow: auto;
  border: 1px solid #eef0f3;
  border-radius: 8px;
}

:deep(.n-data-table) {
  --td-padding: 10px 12px;
}

:deep(.n-data-table .n-data-table-th) {
  background: #fafafa;
}

:deep(.n-data-table .n-data-table-tr:hover) {
  background: #f7f9fc;
}

.loading-container {
  display: flex;
  justify-content: center;
  padding: 24px 0;
}

.pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
  padding-top: 8px;
}

.summary {
  font-size: 12px;
  color: #666;
}

.pager {
  display: flex;
  gap: 8px;
}
</style>
