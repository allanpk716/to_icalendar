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

// 日志消息结构
export interface LogMessage {
  type: string    // info, debug, error, success, warn
  message: string
  time: string
}

// 初始化结果结构
export interface InitResult {
  success: boolean
  message: string
  configDir: string
  serverConfig: string
}

// 剪贴板上传结果
export interface ClipUploadResult {
  success: boolean
  title: string
  description: string
  message: string
  list?: string
  priority?: string
  error?: string
  errorType?: string    // 错误类型：config, network, parsing, api
  canRetry?: boolean     // 是否可重试
  suggestions?: string[]  // 解决建议
  duration?: number      // 处理耗时（毫秒）
}

// 处理结果（用于前端显示）
export interface ProcessResult {
  success: boolean
  title: string
  description: string
  message: string
  list?: string
  priority?: string
  error?: string            // 错误信息
  errorType?: string        // 错误类型：config, network, parsing, api, unknown
  canRetry?: boolean        // 是否可重试
  suggestions?: string[]    // 解决建议
  duration?: number         // 处理耗时（毫秒）
  timestamp?: string        // 时间戳
  parsedAnswer?: string     // AI解析原始内容
}

// 配置状态
export interface ConfigStatus {
  configDir: string
  configExists: boolean
  configValid?: boolean
  serviceInitialized: boolean
  ready?: boolean
  error?: string
  suggestions?: string[]
}
