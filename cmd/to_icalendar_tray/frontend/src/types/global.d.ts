// Element Plus 全局组件类型声明
declare global {
  const ElMessage: {
    success: (message: string) => void
    error: (message: string) => void
    warning: (message: string) => void
    info: (message: string) => void
  }

  const ElMessageBox: {
    confirm: (message: string, title?: string, options?: any) => Promise<void>
    alert: (message: string, title?: string, options?: any) => Promise<void>
    prompt: (message: string, title?: string, options?: any) => Promise<any>
  }
}

export {}