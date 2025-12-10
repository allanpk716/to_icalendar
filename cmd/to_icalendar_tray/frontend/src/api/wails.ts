import type {
  WailsResponse,
  ServerConfig,
  ReminderConfig,
  TestItem,
  TestProgress,
  TestResult,
  ClipboardContent,
  ParseResult,
  CacheInfo,
  CleanProgress
} from '@/types/api'

// Wails API 封装类
export class WailsAPI {
  // 配置初始化
  static async InitConfig(): Promise<WailsResponse<ServerConfig>> {
    try {
      const result = await (window as any).go.main.App.InitConfig()
      return {
        success: true,
        data: result
      }
    } catch (error) {
      return {
        success: false,
        error: `配置初始化失败: ${error}`
      }
    }
  }

  // 检查配置是否存在
  static async CheckConfigExists(): Promise<WailsResponse<boolean>> {
    try {
      const result = await (window as any).go.main.App.CheckConfigExists()
      return {
        success: true,
        data: result
      }
    } catch (error) {
      return {
        success: false,
        error: `检查配置失败: ${error}`
      }
    }
  }

  // 配置测试
  static async TestConfiguration(): Promise<WailsResponse<TestResult>> {
    try {
      const result = await (window as any).go.main.App.TestConfiguration()
      const testResult: TestResult = JSON.parse(result)
      return {
        success: true,
        data: testResult
      }
    } catch (error) {
      return {
        success: false,
        error: `配置测试失败: ${error}`
      }
    }
  }

  // 获取剪贴板内容
  static async GetClipboardContent(): Promise<WailsResponse<ClipboardContent>> {
    try {
      const result = await (window as any).go.main.App.GetClipboardContent()
      return {
        success: true,
        data: result
      }
    } catch (error) {
      return {
        success: false,
        error: `获取剪贴板内容失败: ${error}`
      }
    }
  }

  // 解析剪贴板内容
  static async ParseClipboardContent(content: string): Promise<WailsResponse<ParseResult>> {
    try {
      const result = await (window as any).go.main.App.ParseClipboardContent(content)
      return {
        success: true,
        data: result
      }
    } catch (error) {
      return {
        success: false,
        error: `解析剪贴板内容失败: ${error}`
      }
    }
  }

  // 发送到Microsoft Todo
  static async SendToTodo(config: ReminderConfig): Promise<WailsResponse<boolean>> {
    try {
      const result = await (window as any).go.main.App.SendToTodo(config)
      return {
        success: true,
        data: result
      }
    } catch (error) {
      return {
        success: false,
        error: `发送到Microsoft Todo失败: ${error}`
      }
    }
  }

  // 扫描缓存文件
  static async ScanCacheFiles(): Promise<WailsResponse<CacheInfo[]>> {
    try {
      const result = await (window as any).go.main.App.ScanCacheFiles()
      return {
        success: true,
        data: result
      }
    } catch (error) {
      return {
        success: false,
        error: `扫描缓存文件失败: ${error}`
      }
    }
  }

  // 清理缓存文件
  static async CleanCacheFiles(paths: string[]): Promise<WailsResponse<number>> {
    try {
      const result = await (window as any).go.main.App.CleanCacheFiles(paths)
      return {
        success: true,
        data: result
      }
    } catch (error) {
      return {
        success: false,
        error: `清理缓存文件失败: ${error}`
      }
    }
  }

  // 获取应用版本
  static async GetAppVersion(): Promise<WailsResponse<string>> {
    try {
      const result = await (window as any).go.main.App.GetAppVersion()
      return {
        success: true,
        data: result
      }
    } catch (error) {
      return {
        success: false,
        error: `获取应用版本失败: ${error}`
      }
    }
  }

  // 显示通知
  static async ShowNotification(title: string, message: string): Promise<WailsResponse<void>> {
    try {
      await (window as any).go.main.App.ShowNotification(title, message)
      return {
        success: true
      }
    } catch (error) {
      return {
        success: false,
        error: `显示通知失败: ${error}`
      }
    }
  }

  // 打开配置目录
  static async OpenConfigDirectory(): Promise<WailsResponse<void>> {
    try {
      await (window as any).go.main.App.OpenConfigDirectory()
      return {
        success: true
      }
    } catch (error) {
      return {
        success: false,
        error: `打开配置目录失败: ${error}`
      }
    }
  }

  // 启动浏览器OAuth认证
  static async StartBrowserOAuth(): Promise<WailsResponse<{
    message: string
    type: string
  }>> {
    try {
      const result = await (window as any).go.main.App.StartBrowserOAuth()
      return {
        success: true,
        data: result
      }
    } catch (error) {
      return {
        success: false,
        error: `启动浏览器认证失败: ${error}`
      }
    }
  }
}