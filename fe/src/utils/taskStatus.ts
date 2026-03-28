import {
  PlayOutline,
  CheckmarkCircleOutline,
  CloseCircleOutline,
  InformationCircleOutline,
} from '@vicons/ionicons5'
import type { Component } from 'vue'

// 执行状态常量
export const TaskStatus = {
  PENDING: 'pending',
  RUNNING: 'running',
  COMPLETED: 'completed',
  FAILED: 'failed',
} as const

export type TaskStatusType = (typeof TaskStatus)[keyof typeof TaskStatus]

// 执行状态信息接口
export interface TaskStatusInfo {
  text: string
  type: 'default' | 'info' | 'success' | 'error'
  icon: Component
  color: string
}

// 执行状态映射
export const getTaskStatusInfo = (status: string): TaskStatusInfo => {
  const statusMap: Record<string, TaskStatusInfo> = {
    [TaskStatus.PENDING]: {
      text: '等待中',
      type: 'default',
      icon: InformationCircleOutline,
      color: '#909399',
    },
    [TaskStatus.RUNNING]: {
      text: '进行中',
      type: 'info',
      icon: PlayOutline,
      color: '#409eff',
    },
    [TaskStatus.COMPLETED]: {
      text: '已完成',
      type: 'success',
      icon: CheckmarkCircleOutline,
      color: '#67c23a',
    },
    [TaskStatus.FAILED]: {
      text: '失败',
      type: 'error',
      icon: CloseCircleOutline,
      color: '#f56c6c',
    },
  }

  return statusMap[status] || statusMap[TaskStatus.PENDING]
}
