// 应用常量定义

// 状态常量
export const STATUS = {
  IDLE: 'idle',
  LOADING: 'loading',
  SUCCESS: 'success',
  ERROR: 'error'
} as const

// 优先级常量
export const PRIORITY = {
  LOW: 'low',
  MEDIUM: 'medium',
  HIGH: 'high'
} as const

// 应用信息
export const APP_INFO = {
  NAME: 'to_icalendar',
  VERSION: '1.0.0',
  DESCRIPTION: 'Microsoft Todo 剪贴板助手'
} as const

// 键盘快捷键
export const SHORTCUTS = {
  COPY: 'Ctrl+C',
  PASTE: 'Ctrl+V'
} as const

// 文件大小单位
export const FILE_SIZE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const

// 时间格式
export const DATE_FORMATS = {
  DATE: 'YYYY-MM-DD',
  TIME: 'HH:mm:ss',
  DATETIME: 'YYYY-MM-DD HH:mm:ss'
} as const

// 消息提示持续时间（毫秒）
export const MESSAGE_DURATION = {
  SUCCESS: 3000,
  WARNING: 5000,
  ERROR: 8000,
  INFO: 4000
} as const

// API请求超时时间（毫秒）
export const API_TIMEOUT = 30000

// 缓存相关常量
export const CACHE = {
  MAX_SIZE: 100 * 1024 * 1024, // 100MB
  MAX_FILES: 1000,
  CLEANUP_THRESHOLD: 0.8 // 80%使用率时清理
} as const