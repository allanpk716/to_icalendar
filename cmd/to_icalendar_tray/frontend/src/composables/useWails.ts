import { ref } from 'vue'
import { WailsAPI } from '@/api/wails'
import { useAppState } from './useAppState'
import { STATUS } from '@/utils/constants'
import type { WailsResponse } from '@/types/api'

// Wails通信管理
export function useWails() {
  const { setGlobalStatus } = useAppState()

  // 是否已连接到Wails
  const isConnected = ref<boolean>(false)

  // 通用API调用包装器
  const callAPI = async <T>(
    apiCall: () => Promise<WailsResponse<T>>,
    options: {
      loadingMessage?: string
      successMessage?: string
      errorMessage?: string
      showLoading?: boolean
    } = {}
  ): Promise<{ success: boolean; data?: T; error?: string }> => {
    const {
      loadingMessage = '正在处理...',
      successMessage,
      errorMessage,
      showLoading = true
    } = options

    try {
      if (showLoading) {
        setGlobalStatus(STATUS.LOADING)
      }

      const response = await apiCall()

      if (response.success) {
        return {
          success: true,
          data: response.data
        }
      } else {
        const errorMsg = errorMessage || response.error || '操作失败'
        return {
          success: false,
          error: errorMsg
        }
      }
    } catch (error) {
      const errorMsg = errorMessage || `API调用失败: ${error}`
      return {
        success: false,
        error: errorMsg
      }
    } finally {
      if (showLoading) {
        setGlobalStatus(STATUS.IDLE)
      }
    }
  }

  // 测试Wails连接
  const testConnection = async (): Promise<boolean> => {
    try {
      const result = await callAPI(() => WailsAPI.GetAppVersion(), {
        loadingMessage: '正在测试Wails连接...',
        successMessage: 'Wails连接成功',
        errorMessage: 'Wails连接失败'
      })

      isConnected.value = result.success
      return result.success
    } catch (error) {
      isConnected.value = false
      return false
    }
  }

  // 显示通知
  const showNotification = async (title: string, message: string) => {
    return await callAPI(() => WailsAPI.ShowNotification(title, message), {
      showLoading: false
    })
  }

  // 获取应用版本
  const getAppVersion = async (): Promise<string> => {
    const result = await callAPI(() => WailsAPI.GetAppVersion(), {
      showLoading: false
    })
    return result.data || 'unknown'
  }

  // 初始化Wails连接
  const init = async () => {
    console.log('正在初始化Wails连接...')

    // 测试连接
    const connected = await testConnection()

    if (connected) {
      console.log('Wails初始化完成')
    } else {
      console.error('Wails初始化失败')
    }

    return connected
  }

  return {
    // 状态
    isConnected,

    // 通用方法
    callAPI,

    // 具体API方法
    init,
    testConnection,
    showNotification,
    getAppVersion
  }
}