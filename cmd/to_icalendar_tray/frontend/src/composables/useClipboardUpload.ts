import type { LogMessage, ProcessResult } from '@/types'
import { ElMessage } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { GetConfigStatus } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { useAppState } from './useAppState'

// 任务状态管理
interface TaskInfo {
  id: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  progress: number
  step: string
  message: string
  result?: string
  error?: string
  start_time: string
  end_time?: string
}

// 在函数外部声明局部变量，避免全局状态污染
let refreshDebounceTimer: number | null = null
let updateFrameId: number | null = null

// 全局剪贴板访问锁机制
let clipboardAccessLock = false
let lastAccessTime = 0
const CLIPBOARD_ACCESS_COOLDOWN = 3000 // 3秒冷却时间

// 任务轮询管理
const taskPollingMap = ref<Map<string, number>>(new Map())

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

export function useClipboardUpload() {
  const { globalStatus } = useAppState()

  // 状态
  const clipboardBase64 = ref<string>('')
  const hasImage = ref(false)
  const isProcessing = ref(false)
  const previewUrl = ref('')
  const processResult = ref<ProcessResult | null>(null)
  const logs = ref<LogMessage[]>([])
  const configStatus = ref({
    configDir: '',
    configExists: false,
    configValid: false,
    serviceInitialized: false,
    ready: false,
    error: '',
    suggestions: [] as string[]
  })

  // 进度跟踪
  const progress = reactive({
    step: 0,
    message: ''
  })

  // 检查配置状态
  const checkConfigStatus = async () => {
    try {
      const status = await GetConfigStatus()
      configStatus.value = {
        configDir: status.configDir || '',
        configExists: status.configExists || false,
        configValid: status.configValid || false,
        serviceInitialized: status.serviceInitialized || false,
        ready: status.ready || false,
        error: status.error || '',
        suggestions: status.suggestions || []
      }
    } catch (error) {
      console.error('获取配置状态失败:', error)
      configStatus.value = {
        configDir: '',
        configExists: false,
        configValid: false,
        serviceInitialized: false,
        ready: false,
        error: '获取配置状态失败',
        suggestions: []
      }
    }
  }

  // 添加日志
  const addLog = (type: LogMessage['type'], message: string) => {
    const log: LogMessage = {
      type,
      message,
      time: new Date().toLocaleTimeString()
    }
    logs.value.push(log)
    if (logs.value.length > 500) {
      logs.value.shift()
    }
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

  // 监听来自后端的日志事件，并智能更新进度状态
  EventsOn('clipboardLog', (data: LogMessage) => {
    logs.value.push(data)
    if (logs.value.length > 500) {
      logs.value.shift()
    }

    // 根据日志内容智能更新进度状态
    if (isProcessing.value) {
      const message = data.message.toLowerCase()

      if (message.includes('解码图片') || message.includes('解码成功')) {
        progress.step = 1
        progress.message = '图片解码完成'
      } else if (message.includes('上传图片到ai服务') || message.includes('正在上传')) {
        progress.step = 2
        progress.message = '正在上传图片到AI服务...'
      } else if (message.includes('ai服务调用成功')) {
        progress.step = 3
        progress.message = 'AI服务调用成功，正在分析...'
      } else if (message.includes('ai正在分析图片内容') || message.includes('ai分析完成')) {
        progress.step = 4
        progress.message = 'AI分析完成，正在解析结果...'
      } else if (message.includes('解析ai响应') || message.includes('解析结果')) {
        progress.step = 4
        progress.message = '正在解析AI响应结果...'
      } else if (message.includes('创建microsoft todo任务') || message.includes('正在创建')) {
        progress.step = 5
        progress.message = '正在创建Microsoft Todo任务...'
      } else if (message.includes('任务创建成功')) {
        progress.step = 6
        progress.message = '任务创建成功！'
      } else if (data.type === 'error') {
        progress.message = '处理出现错误'
      }
    }
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
      const { GetClipboardBase64 } = await import('../../wailsjs/go/main/App')
      const base64Data = await GetClipboardBase64()

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

  // 异步处理方法
  const processImageToTodo = async (): Promise<string> => {
    if (!hasImage.value || !clipboardBase64.value) {
      addLog('warning', '请先获取剪贴板图片')
      return ''
    }

    try {
      isProcessing.value = true
      if (globalStatus) {
        (globalStatus as any).value = 'processing'
      }

      // 重置进度和日志
      progress.step = 0
      progress.message = '正在启动处理任务...'
      logs.value = []

      // 启动异步任务
      const { StartProcessImageToTodo } = await import('../../wailsjs/go/main/App')
      const taskID = await StartProcessImageToTodo(clipboardBase64.value)

      // 开始轮询任务状态
      await startTaskPolling(taskID)

      return taskID
    } catch (error) {
      addLog('error', `启动任务失败: ${error}`)
      console.error('启动任务失败:', error)
      return ''
    }
  }

  // 任务状态轮询
  const startTaskPolling = async (taskID: string): Promise<void> => {
    const { GetTaskStatus } = await import('../../wailsjs/go/main/App')

    const pollTask = async () => {
      try {
        const taskInfo = await GetTaskStatus(taskID)

        // 首先检查任务状态，如果已完成或失败，不再更新进度
        if (taskInfo.status === 'completed') {
          if (taskInfo.result) {
            const resultData: ProcessResult = JSON.parse(taskInfo.result)
            processResult.value = resultData
            addLog('success', '任务创建成功！')
          }
          // 确保进度显示为完成
          progress.step = 6
          progress.message = '任务完成'
          stopTaskPolling(taskID)
          // 设置处理完成，但让结果保留
          isProcessing.value = false
          return // 立即返回，不再继续执行
        } else if (taskInfo.status === 'failed') {
          addLog('error', `处理失败: ${taskInfo.error}`)

          // 🔧 关键修复：创建完整的错误结果对象
          const errorResult: ProcessResult = {
            success: false,
            title: '',
            description: '',
            message: taskInfo.step || '处理失败',
            error: taskInfo.error || '未知错误',
            errorType: determineErrorType(taskInfo.error), // 智能错误分类
            canRetry: determineRetryability(taskInfo.error), // 智能重试判断
            suggestions: generateSuggestions(taskInfo.error), // 生成解决建议
            duration: taskInfo.end_time ?
              new Date(taskInfo.end_time).getTime() - new Date(taskInfo.start_time).getTime() : 0
          }
          processResult.value = errorResult // 触发错误弹窗

          stopTaskPolling(taskID)
          // 设置处理完成，但让结果保留
          isProcessing.value = false
          return // 立即返回，不再继续执行
        }

        // 只有任务仍在运行时才更新进度
        progress.step = Math.floor(taskInfo.progress / 100 * 6) // 转换为6步进度
        progress.message = taskInfo.step

        // 添加日志
        if (taskInfo.message && logs.value[logs.value.length - 1]?.message !== taskInfo.message) {
          addLog('info', taskInfo.message)
        }
      } catch (error) {
        addLog('error', `获取任务状态失败: ${error}`)
        stopTaskPolling(taskID)
        isProcessing.value = false
      }
    }

    // 立即执行一次
    await pollTask()

    // 设置定时轮询（每500ms检查一次）
    const timer = setInterval(pollTask, 500)
    taskPollingMap.value.set(taskID, timer)
  }

  // 停止任务轮询
  const stopTaskPolling = (taskID: string) => {
    const timer = taskPollingMap.value.get(taskID)
    if (timer) {
      clearInterval(timer)
      taskPollingMap.value.delete(taskID)
    }
  }

  const clearResult = () => {
    processResult.value = null
  }

  const clearLogs = () => {
    logs.value = []
  }

  // 重置处理状态（用于开始新任务前）
  const resetProcessingState = () => {
    progress.step = 0
    progress.message = ''
    logs.value = []
    processResult.value = null
    isProcessing.value = false
  }

  const resetAllStates = () => {
    logs.value = []
    processResult.value = null
    isProcessing.value = false
  }

  // 清理资源
  const cleanup = () => {
    if (previewUrl.value) {
      URL.revokeObjectURL(previewUrl.value)
      previewUrl.value = ''
    }

    // 清理所有任务轮询
    for (const [taskID, timer] of taskPollingMap.value) {
      clearInterval(timer)
    }
    taskPollingMap.value.clear()
  }

  // 智能错误分类函数
  const determineErrorType = (errorMsg?: string): string => {
    if (!errorMsg) return 'unknown'
    const msg = errorMsg.toLowerCase()

    if (msg.includes('配置') || msg.includes('config')) return 'config'
    if (msg.includes('网络') || msg.includes('network') || msg.includes('connection')) return 'network'
    if (msg.includes('解析') || msg.includes('parse') || msg.includes('格式')) return 'parsing'
    if (msg.includes('api') || msg.includes('服务') || msg.includes('service')) return 'api'
    if (msg.includes('解码') || msg.includes('decode')) return 'processing'

    return 'unknown'
  }

  // 判断可重试性
  const determineRetryability = (errorMsg?: string): boolean => {
    if (!errorMsg) return true
    const nonRetryableErrors = ['解析失败', '格式错误', '图片格式不支持']
    return !nonRetryableErrors.some(pattern => errorMsg.includes(pattern))
  }

  // 生成解决建议
  const generateSuggestions = (errorMsg?: string): string[] => {
    if (!errorMsg) return ['请稍后重试']

    const errorType = determineErrorType(errorMsg)
    const suggestions: string[] = []

    switch (errorType) {
      case 'config':
        suggestions.push('检查配置文件是否完整')
        suggestions.push('确认 API 密钥是否正确')
        break
      case 'network':
        suggestions.push('检查网络连接')
        suggestions.push('确认服务是否可访问')
        suggestions.push('稍后重试')
        break
      case 'parsing':
        suggestions.push('检查图片内容是否清晰')
        suggestions.push('尝试重新截图')
        break
      case 'api':
        suggestions.push('检查服务配置')
        suggestions.push('确认API配额是否充足')
        suggestions.push('稍后重试')
        break
      default:
        suggestions.push('请检查图片内容')
        suggestions.push('稍后重试')
    }

    return suggestions
  }

  // 组件挂载时检查配置状态
  onMounted(() => {
    checkConfigStatus()
  })

  return {
    clipboardBase64,
    hasImage,
    isProcessing,
    progress,
    processResult,
    logs,
    previewUrl,
    configStatus,
    checkConfigStatus,
    getClipboardImage,
    processImageToTodo,
    clearResult,
    clearLogs,
    resetProcessingState,
    resetAllStates,
    cleanup
  }
}
