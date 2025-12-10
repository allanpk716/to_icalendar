<template>
  <el-dialog
    v-model="dialogVisible"
    title="Microsoft 账户授权"
    width="600px"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="!processing"
    destroy-on-close
  >
    <!-- 授权说明 -->
    <div class="auth-intro">
      <el-alert
        title="授权说明"
        type="info"
        :closable="false"
        show-icon
      >
        <template #default>
          <p>我们将为您打开系统浏览器进行Microsoft账户授权。</p>
          <p>授权完成后，此窗口将自动关闭。</p>
        </template>
      </el-alert>
    </div>

    <!-- 浏览器认证状态 -->
    <div class="auth-status" v-if="showAuthStatus">
      <div class="status-icon">
        <el-icon class="is-loading" v-if="loading" :size="48">
          <Loading />
        </el-icon>
        <el-icon v-else-if="authCompleted" :size="48" color="#67C23A">
          <Check />
        </el-icon>
        <el-icon v-else :size="48" color="#E6A23C">
          <Warning />
        </el-icon>
      </div>
      <div class="status-text">
        <h3>{{ statusTitle }}</h3>
        <p>{{ statusMessage }}</p>
      </div>
    </div>

    <!-- 手动打开浏览器链接 -->
    <div class="manual-open" v-if="authUrl && !browserOpened">
      <el-alert
        title="浏览器未自动打开？"
        type="warning"
        :closable="false"
        show-icon
      >
        <template #default>
          <p>请点击下方链接手动打开授权页面：</p>
          <el-button
            type="primary"
            link
            @click="openAuthUrlManually"
            class="auth-link"
          >
            {{ authUrlText }}
          </el-button>
        </template>
      </el-alert>
    </div>

    <!-- 错误提示 -->
    <div class="error-container" v-if="error">
      <el-alert
        :title="error"
        type="error"
        show-icon
        :closable="false"
      />
    </div>

    <!-- 进度指示器 -->
    <div class="progress-container" v-if="processing && !authCompleted">
      <el-progress
        :percentage="progress"
        :status="progressStatus"
        :stroke-width="8"
      />
      <p class="progress-text">{{ progressText }}</p>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleCancel" :disabled="processing">
          {{ processing ? '处理中...' : '取消' }}
        </el-button>
        <el-button @click="handleRetry" :disabled="processing || !error" type="primary">
          重试
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading, Check, Warning } from '@element-plus/icons-vue'
import { WailsAPI } from '@/api/wails'

interface Props {
  modelValue: boolean
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'success', data: any): void
  (e: 'error', error: string): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 状态
const dialogVisible = ref(false)
const loading = ref(false)
const error = ref('')
const processing = ref(false)
const authUrl = ref('')
const authUrlText = ref('')
const browserOpened = ref(false)
const authCompleted = ref(false)
const showAuthStatus = ref(false)
const statusTitle = ref('')
const statusMessage = ref('')
const progress = ref(0)
const progressStatus = ref<'success' | 'exception' | 'warning' | ''>('')
const progressText = ref('')

// 监听对话框显示状态
watch(() => props.modelValue, (newVal) => {
  dialogVisible.value = newVal
  if (newVal) {
    startBrowserAuth()
  } else {
    resetState()
  }
})

// 开始浏览器认证流程
const startBrowserAuth = async () => {
  try {
    loading.value = true
    processing.value = true
    error.value = ''
    authCompleted.value = false
    showAuthStatus.value = true
    statusTitle.value = '正在启动浏览器授权'
    statusMessage.value = '请稍候，正在准备授权流程...'
    progress.value = 10
    progressText.value = '初始化认证流程'

    // 调用后端API启动浏览器OAuth
    const result = await WailsAPI.StartBrowserOAuth()

    if (result.success && result.data) {
      statusTitle.value = '请在浏览器中完成授权'
      statusMessage.value = result.data.message || '正在打开系统浏览器...'
      progress.value = 30
      progressText.value = '等待用户完成授权'

      // 设置一个超时，模拟浏览器打开
      setTimeout(() => {
        if (!authCompleted.value) {
          browserOpened.value = true
          statusMessage.value = '请在浏览器中登录您的Microsoft账户并授权访问'
          progress.value = 60
          progressText.value = '等待用户授权'
        }
      }, 2000)
    } else {
      throw new Error(result.error || '启动浏览器认证失败')
    }
  } catch (err: any) {
    error.value = err.message || '启动浏览器认证失败'
    statusTitle.value = '认证启动失败'
    statusMessage.value = error.value
    progressStatus.value = 'exception'
    ElMessage.error(error.value)
  } finally {
    loading.value = false
  }
}

// 手动打开授权URL
const openAuthUrlManually = () => {
  if (authUrl.value) {
    // 在新窗口中打开授权URL
    window.open(authUrl.value, '_blank')
    browserOpened.value = true
    progress.value = 50
    progressText.value = '已手动打开授权页面'
  }
}

// 设置事件监听器
const setupEventListeners = () => {
  // 监听OAuth结果
  const runtime = (window as any).runtime
  if (runtime) {
    runtime.EventsOn('oauthResult', (result: any) => {
      handleOAuthResult(result)
    })

    runtime.EventsOn('oauthError', (errorMsg: string) => {
      handleOAuthError(errorMsg)
    })
  }
}

// 处理OAuth结果
const handleOAuthResult = (result: any) => {
  processing.value = false
  authCompleted.value = true
  browserOpened.value = true

  if (result.success) {
    statusTitle.value = '授权成功！'
    statusMessage.value = 'Microsoft Todo授权已完成，正在返回...'
    progress.value = 100
    progressStatus.value = 'success'
    progressText.value = '授权成功'

    ElMessage.success('Microsoft账户授权成功！')
    emit('success', result)

    // 1.5秒后自动关闭对话框
    setTimeout(() => {
      closeDialog()
    }, 1500)
  } else {
    error.value = result.error || result.error_desc || '授权失败'
    statusTitle.value = '授权失败'
    statusMessage.value = error.value
    progressStatus.value = 'exception'
    progressText.value = '授权失败'

    ElMessage.error(error.value)
    emit('error', error.value)
  }
}

// 处理OAuth错误
const handleOAuthError = (errorMsg: string) => {
  processing.value = false
  error.value = errorMsg
  statusTitle.value = '认证出错'
  statusMessage.value = errorMsg
  progressStatus.value = 'exception'
  progressText.value = '认证失败'

  ElMessage.error(errorMsg)
  emit('error', errorMsg)
}

// 重试
const handleRetry = () => {
  error.value = ''
  startBrowserAuth()
}

// 取消
const handleCancel = () => {
  if (!processing.value) {
    closeDialog()
  }
}

// 关闭对话框
const closeDialog = () => {
  // 移除事件监听器
  const runtime = (window as any).runtime
  if (runtime) {
    runtime.EventsOff('oauthResult')
    runtime.EventsOff('oauthError')
  }

  emit('update:modelValue', false)
}

// 重置状态
const resetState = () => {
  loading.value = false
  error.value = ''
  processing.value = false
  authUrl.value = ''
  authUrlText.value = ''
  browserOpened.value = false
  authCompleted.value = false
  showAuthStatus.value = false
  statusTitle.value = ''
  statusMessage.value = ''
  progress.value = 0
  progressStatus.value = ''
  progressText.value = ''
}

// 组件挂载时
onMounted(() => {
  setupEventListeners()
  if (dialogVisible.value) {
    startBrowserAuth()
  }
})

// 组件卸载时清理
onUnmounted(() => {
  const runtime = (window as any).runtime
  if (runtime) {
    runtime.EventsOff('oauthResult')
    runtime.EventsOff('oauthError')
  }
})
</script>

<style scoped lang="scss">
.auth-intro {
  margin-bottom: 20px;

  .el-alert {
    p {
      margin: 4px 0;
    }
  }
}

.auth-status {
  display: flex;
  align-items: center;
  gap: 16px;
  margin: 24px 0;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;

  .status-icon {
    flex-shrink: 0;
  }

  .status-text {
    flex: 1;

    h3 {
      margin: 0 0 8px 0;
      font-size: 18px;
      font-weight: 600;
      color: #303133;
    }

    p {
      margin: 0;
      color: #606266;
      line-height: 1.5;
    }
  }
}

.manual-open {
  margin: 20px 0;

  .auth-link {
    margin-top: 8px;
    word-break: break-all;
    font-size: 14px;
  }
}

.error-container {
  margin: 20px 0;
}

.progress-container {
  margin: 24px 0;

  .progress-text {
    margin-top: 12px;
    text-align: center;
    color: #606266;
    font-size: 14px;
  }
}

.dialog-footer {
  text-align: right;
}
</style>