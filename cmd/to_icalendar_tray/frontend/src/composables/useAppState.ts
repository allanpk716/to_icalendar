import { ref, readonly } from 'vue'
import type { StatusType } from '@/types/index'
import { STATUS } from '@/utils/constants'

// 基础状态管理
export function useAppState() {
  // 全局状态
  const globalStatus = ref<StatusType>(STATUS.IDLE)

  // 应用标题
  const title = ref<string>('to_icalendar')

  // 设置全局状态
  const setGlobalStatus = (status: StatusType) => {
    globalStatus.value = status
  }

  return {
    // 状态
    globalStatus: readonly(globalStatus),
    title: readonly(title),

    // 方法
    setGlobalStatus
  }
}