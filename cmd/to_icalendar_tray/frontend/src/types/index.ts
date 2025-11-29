// 通用类型定义
export interface BaseResponse<T = any> {
  success: boolean
  data: T
  message?: string
  code?: number
}

// 分页相关
export interface PaginationParams {
  page: number
  pageSize: number
}

export interface PaginationResponse<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

// 通用状态
export type StatusType = 'idle' | 'loading' | 'success' | 'error'