import type { StatusType } from './index'

// 基础应用状态
export interface AppState {
  globalStatus: StatusType
  title?: string
}