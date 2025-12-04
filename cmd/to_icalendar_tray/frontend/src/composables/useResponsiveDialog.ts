import { ref, readonly, onMounted, onUnmounted, computed } from 'vue'

// 响应式对话框管理
export function useResponsiveDialog() {
  // 窗口宽度响应式数据
  const windowWidth = ref(window.innerWidth)

  // 监听窗口大小变化
  const updateWindowWidth = () => {
    windowWidth.value = window.innerWidth
  }

  // 计算对话框宽度
  const calculateDialogWidth = (baseWidth: number, maxWidth: number = 1200) => {
    // 根据窗口宽度动态计算
    if (windowWidth.value < 768) {
      return '95%'
    } else if (windowWidth.value < 1200) {
      return '90%'
    } else {
      return Math.min(baseWidth, maxWidth) + 'px'
    }
  }

  // 创建响应式宽度计算器
  const createDialogWidth = (baseWidth: number, maxWidth?: number) => {
    return computed(() => calculateDialogWidth(baseWidth, maxWidth))
  }

  // 获取当前窗口宽度
  const getWindowWidth = () => windowWidth.value

  // 判断是否为小屏幕
  const isSmallScreen = computed(() => windowWidth.value < 768)

  // 判断是否为中等屏幕
  const isMediumScreen = computed(() => windowWidth.value >= 768 && windowWidth.value < 1200)

  // 判断是否为大屏幕
  const isLargeScreen = computed(() => windowWidth.value >= 1200)

  // 生命周期钩子
  onMounted(() => {
    window.addEventListener('resize', updateWindowWidth)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', updateWindowWidth)
  })

  return {
    // 状态
    windowWidth: readonly(windowWidth),
    isSmallScreen,
    isMediumScreen,
    isLargeScreen,

    // 方法
    calculateDialogWidth,
    createDialogWidth,
    getWindowWidth
  }
}