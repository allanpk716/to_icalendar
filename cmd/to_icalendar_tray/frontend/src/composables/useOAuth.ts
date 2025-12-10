import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'

export function useOAuth() {
  // 状态
  const isAuthenticating = ref(false)
  const authError = ref('')
  const lastAuthTime = ref<Date | null>(null)

  // 计算属性
  const canRetry = computed(() => authError.value && !isAuthenticating.value)
  const timeSinceLastAuth = computed(() => {
    if (!lastAuthTime.value) return null
    const seconds = Math.floor((Date.now() - lastAuthTime.value.getTime()) / 1000)
    if (seconds < 60) return `${seconds}秒前`
    if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟前`
    return `${Math.floor(seconds / 3600)}小时前`
  })

  // 方法
  const authenticate = async () => {
    try {
      isAuthenticating.value = true
      authError.value = ''

      // 这里应该触发授权对话框
      // 具体实现取决于父组件如何处理
      return true
    } catch (error: any) {
      authError.value = error.message || '认证失败'
      ElMessage.error(authError.value)
      return false
    } finally {
      isAuthenticating.value = false
    }
  }

  const clearError = () => {
    authError.value = ''
  }

  const reset = () => {
    isAuthenticating.value = false
    authError.value = ''
    lastAuthTime.value = null
  }

  return {
    // 状态
    isAuthenticating,
    authError,
    lastAuthTime,
    // 计算属性
    canRetry,
    timeSinceLastAuth,
    // 方法
    authenticate,
    clearError,
    reset
  }
}