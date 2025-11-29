// 配置相关类型
export interface ServerConfig {
  microsoft_todo: {
    tenant_id: string
    client_id: string
    client_secret: string
    timezone: string
  }
}

export interface ReminderConfig {
  title: string
  description: string
  date: string
  time: string
  remind_before: string
  priority: 'low' | 'medium' | 'high'
  list: string
}

// Wails API 返回类型
export interface WailsResponse<T = any> {
  success: boolean
  data?: T
  error?: string
}

// 测试相关类型
export interface TestItem {
  name: string
  description: string
  status: 'pending' | 'running' | 'success' | 'error'
  result?: string
  duration?: number
}

export interface TestProgress {
  current: number
  total: number
  currentTest: string
}

// 剪贴板相关类型
export interface ClipboardContent {
  type: 'text' | 'image' | 'file'
  content: string
  size?: number
  timestamp: Date
}

export interface ParseResult {
  title: string
  description: string
  date?: string
  time?: string
  priority?: 'low' | 'medium' | 'high'
  list?: string
  confidence: number
}

// 清理相关类型
export interface CacheInfo {
  path: string
  size: number
  lastModified: Date
  type: string
}

export interface CleanProgress {
  scanned: number
  found: number
  cleaned: number
  currentPath?: string
}