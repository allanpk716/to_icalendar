import { ref, reactive, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { useAppState } from './useAppState'
import { useWails } from './useWails'

// 在函数外部声明局部变量，避免全局状态污染
let refreshDebounceTimer: number | null = null
let updateFrameId: number | null = null

// 全局剪贴板访问锁机制
let clipboardAccessLock = false
let lastAccessTime = 0
const CLIPBOARD_ACCESS_COOLDOWN = 3000 // 3秒冷却时间

// 检查剪贴板访问权限
const checkClipboardAccess = (): boolean => {
  const now = Date.now()
  if (clipboardAccessLock && (now - lastAccessTime) < CLIPBOARD_ACCESS_COOLDOWN) {
    return false // 3秒内不允许重复访问
  }
  return true
}

// 获取剪贴板访问锁
const acquireClipboardLock = (): boolean => {
  if (!checkClipboardAccess()) {
    return false
  }
  clipboardAccessLock = true
  lastAccessTime = Date.now()
  return true
}

// 释放剪贴板访问锁
const releaseClipboardLock = () => {
  setTimeout(() => {
    clipboardAccessLock = false
  }, 1000) // 1秒后释放锁，确保操作完成
}

export interface ProcessResult {
  success: boolean
  title: string
  description: string
  message: string
  list?: string
  priority?: string
}

export interface LogMessage {
  type: 'info' | 'success' | 'error' | 'warning'
  message: string
  time: string
}

export function useClipboardUpload() {
  const { globalStatus } = useAppState()

  // 状态
  const clipboardBase64 = ref<string>('')
  const hasImage = ref(false)
  const isProcessing = ref(false)
  const previewUrl = ref('')
  const processResult = ref<ProcessResult | null>(null)
  const logs = ref<LogMessage[]>([])

  // 进度跟踪
  const progress = reactive({
    step: 0,
    message: ''
  })

  // 添加日志
  const addLog = (type: LogMessage['type'], message: string) => {
    const log: LogMessage = {
      type,
      message,
      time: new Date().toLocaleTimeString()
    }
    logs.value.push(log)
  }

  // 优化防抖刷新
  const debouncedRefresh = async () => {
    if (refreshDebounceTimer) {
      clearTimeout(refreshDebounceTimer)
    }

    refreshDebounceTimer = setTimeout(async () => {
      await getClipboardImage(false)
      refreshDebounceTimer = null
    }, 300)
  }

  // 优化图片URL更新
  const updatePreviewUrl = (newUrl: string) => {
    if (updateFrameId) {
      cancelAnimationFrame(updateFrameId)
    }

    updateFrameId = requestAnimationFrame(() => {
      previewUrl.value = newUrl
      updateFrameId = null
    })
  }

  // 监听来自后端的日志事件
  EventsOn('clipboardLog', (data: LogMessage) => {
    logs.value.push(data)
  })

  // 获取剪贴板图片
  const getClipboardImage = async (showMessage = true) => {
    try {
      // 检查剪贴板访问锁
      if (!acquireClipboardLock()) {
        if (showMessage) {
          addLog('warning', '剪贴板访问冷却中，请稍后重试')
        }
        return // 静默返回，不显示错误
      }

      isProcessing.value = true

      // 动态导入 Wails API
      const { GetClipboardFromWindows } = await import('../../wailsjs/go/main/App')
      const base64Data = await GetClipboardFromWindows()

      // 严格的内容变化检测
      if (base64Data && base64Data !== clipboardBase64.value) {
        // 立即清理旧URL，避免累积
        if (previewUrl.value) {
          URL.revokeObjectURL(previewUrl.value)
          previewUrl.value = ''
        }

        // 批量更新状态
        clipboardBase64.value = base64Data
        hasImage.value = true

        const newPreviewUrl = `data:image/png;base64,${base64Data}`
        updatePreviewUrl(newPreviewUrl)

        if (showMessage) {
          ElMessage.success('成功获取剪贴板图片')
        }
      } else if (!base64Data) {
        hasImage.value = false
        if (previewUrl.value) {
          URL.revokeObjectURL(previewUrl.value)
          previewUrl.value = ''
        }
        if (showMessage) {
          ElMessage.warning('剪贴板中没有图片内容')
        }
      }
      // 如果内容相同，静默处理
    } catch (error) {
      hasImage.value = false
      if (previewUrl.value) {
        URL.revokeObjectURL(previewUrl.value)
        previewUrl.value = ''
      }
      addLog('error', `获取剪贴板失败: ${error}`)
      if (showMessage) {
        ElMessage.error(`获取剪贴板失败: ${error}`)
      }
    } finally {
      isProcessing.value = false
      // 释放剪贴板访问锁
      releaseClipboardLock()
    }
  }

  // 处理剪贴板内容并创建Todo任务
  const processImageToTodo = async (): Promise<ProcessResult | null> => {
    if (!hasImage.value || !clipboardBase64.value) {
      addLog('warning', '请先获取剪贴板图片')
      return null
    }

    try {
      isProcessing.value = true
      if (globalStatus) {
        (globalStatus as any).value = 'processing'
      }

      // 重置进度和日志
      progress.step = 0
      progress.message = '开始处理...'
      logs.value = []
      addLog('info', '开始处理图片并创建任务...')

      // 步骤1：准备上传
      progress.step = 1
      progress.message = '正在上传图片到AI服务...'
      addLog('info', '正在上传图片到AI服务...')

      // 步骤2：AI分析
      progress.step = 2
      progress.message = 'AI正在分析图片内容...'
      addLog('info', 'AI正在分析图片内容...')

      // 调用后端处理方法
      const { ProcessImageToTodo } = await import('../../wailsjs/go/main/App')
      const resultJson = await ProcessImageToTodo(clipboardBase64.value)

      progress.step = 3
      progress.message = '正在创建Microsoft Todo任务...'
      addLog('info', '正在创建Microsoft Todo任务...')

      if (resultJson) {
        const processResultData: ProcessResult = JSON.parse(resultJson)

        if (processResultData.success) {
          progress.step = 4
          progress.message = '处理完成！'
          addLog('success', '任务创建成功！')
          addLog('info', `任务标题: ${processResultData.title}`)

          processResult.value = processResultData
          console.log('剪贴板内容处理成功！')
        } else {
          addLog('error', `处理失败: ${processResultData.message}`)
          console.error('处理失败:', processResultData.message)
        }

        return processResultData
      }

      return null
    } catch (error) {
      addLog('error', `处理失败: ${error}`)
      console.error('处理失败:', error)
      return null
    } finally {
      isProcessing.value = false
      if (globalStatus) {
        (globalStatus as any).value = 'idle'
      }
    }
  }

  // 清除结果
  const clearResult = () => {
    processResult.value = null
    logs.value = []
    progress.step = 0
    progress.message = ''
  }

  // 清理资源
  const cleanup = () => {
    if (previewUrl.value) {
      URL.revokeObjectURL(previewUrl.value)
      previewUrl.value = ''
    }
  }

  return {
    clipboardBase64,
    hasImage,
    isProcessing,
    progress,
    processResult,
    logs,
    previewUrl,
    getClipboardImage,
    processImageToTodo,
    clearResult,
    cleanup
  }
}