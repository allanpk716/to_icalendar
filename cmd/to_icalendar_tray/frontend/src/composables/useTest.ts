import { ref, readonly, onMounted, onUnmounted } from 'vue'
import { WailsAPI } from '@/api/wails'
import { useWails } from './useWails'
import type { TestResult, TestProgress, TestLogEvent } from '@/types/api'

// 测试状态管理
export function useTest() {
  const { callAPI } = useWails()

  // 测试状态
  const testResult = ref<TestResult | null>(null)
  const progress = ref<number>(0) // 0-100 进度值
  const isRunning = ref<boolean>(false)
  const currentTest = ref<string>('')
  const progressMessage = ref<string>('')
  const testLogs = ref<TestLogEvent[]>([]) // 测试日志

  // 计算进度百分比
  const calculateProgress = (current: number, total: number): number => {
    return Math.round((current / total) * 100)
  }

  // 开始测试
  const startTest = async (): Promise<{ success: boolean; result?: TestResult; error?: string }> => {
    if (isRunning.value) {
      return {
        success: false,
        error: '测试正在运行中，请稍候'
      }
    }

    try {
      // 重置状态
      isRunning.value = true
      progress.value = 0
      testResult.value = null
      currentTest.value = ''
      progressMessage.value = '准备开始测试...'

      // 调用测试API
      const result = await callAPI(() => WailsAPI.TestConfiguration(), {
        loadingMessage: '正在运行配置测试...',
        errorMessage: '配置测试失败'
      })

      if (result.success && result.data) {
        testResult.value = result.data
        progress.value = 100
        currentTest.value = '测试完成'
        progressMessage.value = result.data.overallSuccess ? '所有测试通过' : '部分测试失败'

        return {
          success: true,
          result: result.data
        }
      } else {
        return {
          success: false,
          error: result.error || '测试失败'
        }
      }
    } catch (error) {
      return {
        success: false,
        error: `测试执行失败: ${error}`
      }
    } finally {
      isRunning.value = false
    }
  }

  // 处理测试进度事件
  const handleTestProgress = (progressData: TestProgress) => {
    currentTest.value = progressData.testName
    progressMessage.value = progressData.message
    progress.value = calculateProgress(progressData.current, progressData.total)
  }

  // 处理测试日志事件
  const handleTestLog = (logData: TestLogEvent) => {
    testLogs.value.push(logData)
    // 保持最新的100条日志
    if (testLogs.value.length > 100) {
      testLogs.value = testLogs.value.slice(-100)
    }
  }

  // 重置测试状态
  const resetTest = () => {
    testResult.value = null
    progress.value = 0
    isRunning.value = false
    currentTest.value = ''
    progressMessage.value = ''
    testLogs.value = []
  }

  // 格式化测试持续时间
  const formatDuration = (duration: number): string => {
    if (duration < 1000) {
      return `${duration}ms`
    } else if (duration < 60000) {
      return `${(duration / 1000).toFixed(2)}s`
    } else {
      const minutes = Math.floor(duration / 60000)
      const seconds = ((duration % 60000) / 1000).toFixed(2)
      return `${minutes}m ${seconds}s`
    }
  }

  // 获取测试状态文本
  const getTestStatusText = (success: boolean): { text: string; type: 'success' | 'error' | 'warning' } => {
    if (success) {
      return {
        text: '测试通过',
        type: 'success'
      }
    } else {
      return {
        text: '测试失败',
        type: 'error'
      }
    }
  }

  // 事件监听器设置
  let progressEventListener: ((event: any) => void) | null = null
  let logEventListener: ((event: any) => void) | null = null

  onMounted(() => {
    // 监听测试进度事件
    if (typeof window !== 'undefined') {
      progressEventListener = (event: any) => {
        if (event.type === 'testProgress') {
          handleTestProgress(event.data as TestProgress)
        }
      }

      // 监听测试日志事件
      logEventListener = (event: any) => {
        if (event.type === 'testLog') {
          handleTestLog(event.data as TestLogEvent)
        }
      }

      // 注册事件监听器（这里假设Wails提供了事件监听接口）
      // 具体实现可能需要根据Wails的事件系统调整
      try {
        ;(window as any).runtime?.EventsOn('testProgress', progressEventListener)
        ;(window as any).runtime?.EventsOn('testLog', logEventListener)
      } catch (error) {
        console.warn('无法注册测试事件监听器:', error)
      }
    }
  })

  onUnmounted(() => {
    // 清理事件监听器
    if (progressEventListener && typeof window !== 'undefined') {
      try {
        ;(window as any).runtime?.EventsOff('testProgress', progressEventListener)
        ;(window as any).runtime?.EventsOff('testLog', logEventListener)
      } catch (error) {
        console.warn('无法移除测试事件监听器:', error)
      }
    }
  })

  return {
    // 状态（只读）
    testResult: readonly(testResult),
    progress: readonly(progress),
    isRunning: readonly(isRunning),
    currentTest: readonly(currentTest),
    progressMessage: readonly(progressMessage),
    testLogs: readonly(testLogs),

    // 方法
    startTest,
    resetTest,
    formatDuration,
    getTestStatusText,

    // 计算属性
    isTestCompleted: () => testResult.value !== null,
    hasTestPassed: () => testResult.value?.overallSuccess || false
  }
}