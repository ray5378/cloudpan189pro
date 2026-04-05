<template>
  <div class="login-logs-page">
    <!-- 头部筛选 -->
    <div class="header">
      <div class="header-left">
        <n-input
          v-model:value="state.username"
          placeholder="用户名"
          clearable
          style="width: 140px"
        />
        <n-input v-model:value="state.addr" placeholder="地址/IP" clearable style="width: 140px" />
        <n-select
          v-model:value="state.event"
          :options="eventOptions"
          clearable
          placeholder="事件"
          style="width: 120px"
        />
        <n-select
          v-model:value="state.status"
          :options="statusOptions"
          clearable
          placeholder="状态"
          style="width: 120px"
        />
        <n-date-picker
          v-model:value="state.dateRange"
          type="datetimerange"
          clearable
          style="width: 280px"
          format="yyyy-MM-dd HH:mm:ss"
          value-format="yyyy-MM-ddTHH:mm:ssXXX"
          placeholder="选择时间范围"
        />
        <n-button type="primary" @click="handleSearch">搜索</n-button>
        <n-button @click="handleReset">重置</n-button>
      </div>
      <div class="header-right">
        <n-space>
          <n-button @click="handleRefresh">
            <template #icon>
              <n-icon>
                <RefreshOutline />
              </n-icon>
            </template>
            刷新
          </n-button>
          <n-popconfirm positive-text="清除" negative-text="取消" @positive-click="handleCleanup">
            <template #trigger>
              <n-button type="error" quaternary>清除登录日志</n-button>
            </template>
            确认按保留策略清理历史登录日志？
          </n-popconfirm>
        </n-space>
      </div>
    </div>

    <!-- 表格 -->
    <n-data-table
      :columns="columns"
      :data="state.tableData"
      :loading="state.loading"
      :pagination="paginationReactive"
      :row-key="(row: Models.LoginLog) => row.id"
      class="login-logs-table"
      :scroll-x="1100"
      remote
    />
  </div>
</template>

<script setup lang="ts">
import { reactive, h, onMounted } from 'vue'
import {
  NDataTable,
  NInput,
  NButton,
  NIcon,
  NSelect,
  NTag,
  NText,
  NSpace,
  NPopconfirm,
  useMessage,
  type DataTableColumns,
  type PaginationProps,
  NDatePicker,
} from 'naive-ui'
import {
  RefreshOutline,
  GlobeOutline,
  KeyOutline,
  PersonCircleOutline,
  ShieldOutline,
  WarningOutline,
  CheckmarkCircleOutline,
  CloseCircleOutline,
  RefreshCircleOutline,
} from '@vicons/ionicons5'
import { getLoginLogList, cleanupLoginLogs, type LoginLogListQuery } from '@/api/loginlog'
import { formatDate } from '@/utils/format'
import {
  LOGIN_EVENT_OPTIONS,
  LOGIN_EVENT_TEXT_MAP,
  LOGIN_STATUS_OPTIONS,
  LOGIN_STATUS_TEXT_MAP,
  LOGIN_STATUS_TAG_MAP,
} from '@/constants/loginLog'

const message = useMessage()

const state = reactive({
  loading: false,
  tableData: [] as Models.LoginLog[],
  username: '',
  addr: '',
  event: null as Enums.LoginEvent | null,
  status: null as Enums.LoginStatus | null,
  dateRange: null as [number, number] | null,
})

// 选项
const eventOptions = LOGIN_EVENT_OPTIONS
const statusOptions = LOGIN_STATUS_OPTIONS

// 分页
const paginationReactive = reactive<PaginationProps>({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  prefix: ({ itemCount }) => `共 ${itemCount} 条`,
  onChange: (page) => {
    paginationReactive.page = page
    fetchList()
  },
  onUpdatePageSize: (pageSize) => {
    paginationReactive.pageSize = pageSize
    paginationReactive.page = 1
    fetchList()
  },
})

const getStatusTagType = (s: string) => {
  return LOGIN_STATUS_TAG_MAP[s as keyof typeof LOGIN_STATUS_TAG_MAP] || 'default'
}
const getStatusIcon = (s: string) => {
  switch (s) {
    case 'success':
      return CheckmarkCircleOutline
    case 'failed':
      return CloseCircleOutline
    case 'blocked':
      return WarningOutline
    default:
      return ShieldOutline
  }
}
const getMethodIcon = (m: string) => {
  switch (m) {
    case 'web':
      return GlobeOutline
    case 'api':
      return KeyOutline
    case 'app':
      return PersonCircleOutline
    case 'cli':
      return RefreshCircleOutline
    default:
      return GlobeOutline
  }
}

// 列
const columns: DataTableColumns<Models.LoginLog> = [
  {
    title: '序号',
    key: 'index',
    width: 80,
    align: 'center',
    render(_, index) {
      const start = ((paginationReactive.page || 1) - 1) * (paginationReactive.pageSize || 10)
      return start + index + 1
    },
  },
  { title: '用户名', key: 'username', minWidth: 100, align: 'center', ellipsis: { tooltip: true } },
  {
    title: '来源',
    key: 'method',
    width: 80,
    align: 'center',
    render(row) {
      return h(
        NTag,
        { type: 'info', size: 'small' },
        {
          icon: () => h(NIcon, { size: 12 }, { default: () => h(getMethodIcon(row.method)) }),
          default: () => row.method,
        }
      )
    },
  },
  {
    title: '事件',
    key: 'event',
    width: 100,
    align: 'center',
    render(row) {
      const text = LOGIN_EVENT_TEXT_MAP[row.event as keyof typeof LOGIN_EVENT_TEXT_MAP] || row.event
      return h(NText, null, { default: () => text })
    },
  },
  {
    title: '状态',
    key: 'status',
    width: 80,
    align: 'center',
    render(row) {
      return h(
        NTag,
        { type: getStatusTagType(row.status), size: 'small' },
        {
          icon: () => h(NIcon, { size: 12 }, { default: () => h(getStatusIcon(row.status)) }),
          default: () =>
            LOGIN_STATUS_TEXT_MAP[row.status as keyof typeof LOGIN_STATUS_TEXT_MAP] || row.status,
        }
      )
    },
  },
  { title: '地址/IP', key: 'addr', minWidth: 120, align: 'center', ellipsis: { tooltip: true } },
  { title: 'UA', key: 'userAgent', minWidth: 160, align: 'center', ellipsis: { tooltip: true } },
  {
    title: '原因',
    key: 'reason',
    minWidth: 200,
    align: 'center',
    ellipsis: { tooltip: true },
  },
  {
    title: '时间',
    key: 'createdAt',
    width: 180,
    align: 'center',
    render(row) {
      return formatDate(row.createdAt)
    },
  },
]

// 拉取列表
const fetchList = () => {
  state.loading = true
  const params: LoginLogListQuery = {
    currentPage: paginationReactive.page ?? 1,
    pageSize: paginationReactive.pageSize ?? 10,
  }
  if (state.username) params.username = state.username
  if (state.addr) params.addr = state.addr
  if (state.event) params.event = state.event
  if (state.status) params.status = state.status
  if (state.dateRange && state.dateRange.length === 2) {
    params.beginAt = new Date(state.dateRange[0]).toISOString()
    params.endAt = new Date(state.dateRange[1]).toISOString()
  }

  getLoginLogList(params)
    .then((res) => {
      if (res.data) {
        state.tableData = res.data.data || []
        paginationReactive.itemCount = res.data.total || 0
      }
    })
    .catch((err) => {
      console.error(err)
      message.error(err?.message || '获取登录日志失败')
    })
    .finally(() => {
      state.loading = false
    })
}

const handleSearch = () => {
  paginationReactive.page = 1
  fetchList()
}
const handleReset = () => {
  state.username = ''
  state.addr = ''
  state.event = null
  state.status = null
  state.dateRange = null
  paginationReactive.page = 1
  fetchList()
}
const handleRefresh = () => fetchList()

// 清理登录日志
const handleCleanup = async () => {
  try {
    state.loading = true
    const res = await cleanupLoginLogs()
    const deleted = (res.data as any)?.deleted ?? 0
    const days = (res.data as any)?.retentionDays ?? 15
    message.success(`已清理 ${deleted} 条（保留 ${days} 天）`)
    fetchList()
  } catch (e: any) {
    console.error(e)
    message.error(e?.message || '清理登录日志失败')
  } finally {
    state.loading = false
  }
}

// 仅在组件挂载时发起请求，配合 Tabs 的 v-if 保证按需加载
onMounted(() => {
  fetchList()
})
</script>

<style scoped>
.login-logs-page {
  padding: 0;
}

.header {
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.header-left {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.login-logs-table {
  background: var(--n-card-color);
  border-radius: 6px;
}

.login-logs-table :deep(.n-data-table-th) {
  text-align: center;
  font-weight: 600;
}

.login-logs-table :deep(.n-data-table-td) {
  text-align: center;
}
</style>
