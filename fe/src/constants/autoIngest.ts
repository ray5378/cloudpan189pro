/**
 * 自动入库（Auto Ingest）相关常量与类型
 * 与后端 swagger 定义保持一致
 */

// 入库来源 SourceType（目前仅 subscribe）
export const AUTO_INGEST_SOURCE_TYPES = ['subscribe'] as const
export type AutoIngestSourceType = (typeof AUTO_INGEST_SOURCE_TYPES)[number]

// 入库来源下拉选项
export const AUTO_INGEST_SOURCE_TYPE_OPTIONS: { label: string; value: AutoIngestSourceType }[] = [
  { label: '订阅号', value: 'subscribe' },
]

// 冲突处理策略 OnConflict（rename | abandon）
export const AUTO_INGEST_ON_CONFLICTS = ['rename', 'abandon'] as const
export type AutoIngestOnConflict = (typeof AUTO_INGEST_ON_CONFLICTS)[number]

// 冲突处理策略选项
export const AUTO_INGEST_ON_CONFLICT_OPTIONS: { label: string; value: AutoIngestOnConflict }[] = [
  { label: '重命名', value: 'rename' },
  { label: '忽略', value: 'abandon' },
]

// 数值范围常量（与 swagger 约束一致）
export const AUTO_INGEST_INTERVAL_MIN = 5 // 自动入库间隔(分钟) 最小值
export const REFRESH_INTERVAL_MIN = 30 // 刷新间隔(分钟) 最小值
export const REFRESH_INTERVAL_MAX = 1440 // 刷新间隔(分钟) 最大值
export const AUTO_REFRESH_DAYS_MIN = 1 // 自动刷新持续天数 最小值
export const AUTO_REFRESH_DAYS_MAX = 365 // 自动刷新持续天数 最大值
