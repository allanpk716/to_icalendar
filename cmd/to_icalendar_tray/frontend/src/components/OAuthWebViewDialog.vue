<template>
  <el-dialog
    v-model="dialogVisible"
    title="Microsoft 账户授权"
    width="700px"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="!processing"
    destroy-on-close
  >
    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <!-- 自动授权标签页 -->
      <el-tab-pane label="自动授权" name="auto">
        <div class="auto-auth-content">
          <!-- 授权说明 -->
          <div class="auth-intro">
            <el-alert
              title="自动授权说明"
              type="info"
              :closable="false"
              show-icon
            >
              <template #default>
                <p>将在系统浏览器中打开Microsoft账户授权页面。</p>
                <p>请在浏览器中完成登录和授权。</p>
              </template>
            </el-alert>
          </div>

          <!-- 认证状态 -->
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

          <!-- 授权窗口信息 -->
          <div class="auth-window-info" v-if="processing && !authCompleted">
            <el-alert
              title="浏览器已打开"
              type="success"
              :closable="false"
              show-icon
            >
              <template #default>
                <p>已在系统浏览器中打开Microsoft授权页面，请在该窗口中完成登录和授权。</p>
                <p>如果浏览器窗口被遮挡，请检查任务栏或切换窗口。</p>
              </template>
            </el-alert>
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

          <!-- 回调URL输入区域 -->
          <div class="callback-url-section" v-if="processing && !authCompleted">
            <el-alert
              title="如果在浏览器中看到回调页面"
              type="warning"
              :closable="false"
              show-icon
            >
              <template #default>
                <p>如果在浏览器授权完成后看到类似下面的地址：</p>
                <p><code>http://localhost:8080/callback?code=xxx&state=xxx</code></p>
                <p>请复制这个完整地址并粘贴到下方输入框中</p>
              </template>
            </el-alert>

            <el-input
              v-model="callbackURL"
              type="textarea"
              :rows="3"
              placeholder="请在这里粘贴浏览器中的回调URL..."
              class="callback-url-input"
              @input="validateURL"
            />

            <div class="callback-url-actions">
              <el-button @click="pasteFromClipboard">
                <el-icon><DocumentCopy /></el-icon>
                从剪贴板粘贴
              </el-button>
              <el-button type="primary" @click="submitCallbackURL" :loading="submitting" :disabled="!isURLValid">
                提交回调URL
              </el-button>
            </div>

            <!-- URL验证提示 -->
            <div class="url-validation" v-if="callbackURL">
              <el-icon v-if="isURLValid" color="#67C23A"><Check /></el-icon>
              <el-icon v-else color="#F56C6C"><Close /></el-icon>
              <span :style="{ color: isURLValid ? '#67C23A' : '#F56C6C' }">
                {{ validationMessage }}
              </span>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- 手动输入URL标签页 -->
      <el-tab-pane label="手动输入URL" name="manual">
        <div class="manual-auth-content">
          <!-- 步骤指导 -->
          <el-steps :active="currentStep" direction="vertical" class="auth-steps">
            <el-step title="打开浏览器" description="点击下方按钮在系统浏览器中打开Microsoft授权页面" />
            <el-step title="完成授权" description="登录您的Microsoft账户并同意授权" />
            <el-step title="复制URL" description="授权后，复制浏览器地址栏中的完整URL" />
            <el-step title="粘贴并完成" description="将URL粘贴到下方输入框并点击完成授权" />
          </el-steps>

          <!-- 操作按钮 -->
          <div class="action-buttons" v-if="currentStep === 0">
            <el-button type="primary" size="large" @click="openBrowserForAuth">
              <el-icon><Link /></el-icon>
              打开浏览器授权
            </el-button>
          </div>

          <!-- URL输入区域 -->
          <div class="url-input-section" v-if="currentStep >= 2">
            <el-alert
              title="请粘贴完整的回调URL"
              type="info"
              :closable="false"
              show-icon
            >
              <template #default>
                <p>URL应该类似：<code>http://localhost:8080/callback?code=xxx&state=xxx</code></p>
                <p>请确保复制地址栏中的完整URL，包括所有参数</p>
              </template>
            </el-alert>

            <el-input
              v-model="callbackURL"
              type="textarea"
              :rows="3"
              placeholder="请粘贴回调URL..."
              class="url-input"
              @input="validateURL"
            />

            <div class="input-actions">
              <el-button @click="pasteFromClipboard">
                <el-icon><DocumentCopy /></el-icon>
                从剪贴板粘贴
              </el-button>
              <el-button type="primary" @click="submitCallbackURL" :loading="submitting" :disabled="!isURLValid">
                完成授权
              </el-button>
            </div>

            <!-- URL验证提示 -->
            <div class="url-validation" v-if="callbackURL">
              <el-icon v-if="isURLValid" color="#67C23A"><Check /></el-icon>
              <el-icon v-else color="#F56C6C"><Close /></el-icon>
              <span :style="{ color: isURLValid ? '#67C23A' : '#F56C6C' }">
                {{ validationMessage }}
              </span>
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 错误提示 -->
    <div class="error-container" v-if="error">
      <el-alert
        :title="error"
        type="error"
        show-icon
        :closable="false"
      />
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
import { Loading, Check, Warning, Link, DocumentCopy, Close } from '@element-plus/icons-vue'
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
const authCompleted = ref(false)
const showAuthStatus = ref(false)
const statusTitle = ref('')
const statusMessage = ref('')
const progress = ref(0)
const progressStatus = ref<'success' | 'exception' | 'warning' | ''>('')
const progressText = ref('')

// 标签页状态
const activeTab = ref('auto')
const currentStep = ref(0)
const callbackURL = ref('')
const isURLValid = ref(false)
const validationMessage = ref('')
const submitting = ref(false)

// 监听对话框显示状态
watch(() => props.modelValue, (newVal) => {
  dialogVisible.value = newVal
  if (newVal) {
    if (activeTab.value === 'auto') {
      startBrowserAuth()
    }
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
    statusTitle.value = '正在启动系统浏览器授权'
    statusMessage.value = '请稍候，正在准备授权流程...'
    progress.value = 10
    progressText.value = '初始化认证流程'

    // 调用后端API启动系统浏览器OAuth
    const result = await WailsAPI.StartBrowserOAuth()

    if (result.success && result.data) {
      statusTitle.value = '请在授权窗口中完成登录'
      statusMessage.value = result.data.message || '已在系统浏览器中打开授权窗口...'
      progress.value = 40
      progressText.value = '等待用户在授权窗口中完成操作'

      // 设置一个超时，确保用户看到授权窗口已打开
      setTimeout(() => {
        if (!authCompleted.value) {
          statusMessage.value = '请在弹出的授权窗口中登录您的Microsoft账户并授权访问'
          progress.value = 60
          progressText.value = '等待用户授权'
        }
      }, 1500)
    } else {
      throw new Error(result.error || '启动系统浏览器授权失败')
    }
  } catch (err: any) {
    error.value = err.message || '启动系统浏览器授权失败'
    statusTitle.value = '认证启动失败'
    statusMessage.value = error.value
    progressStatus.value = 'exception'
    ElMessage.error(error.value)
  } finally {
    loading.value = false
  }
}

// 打开浏览器进行手动授权
const openBrowserForAuth = async () => {
  try {
    loading.value = true
    const result = await WailsAPI.StartBrowserOAuth()
    if (result.success) {
      currentStep.value = 1
      ElMessage.success('已在浏览器中打开授权页面，请完成授权')
      // 2秒后进入下一步
      setTimeout(() => {
        currentStep.value = 2
      }, 2000)
    }
  } catch (error) {
    ElMessage.error('打开浏览器失败：' + error)
  } finally {
    loading.value = false
  }
}

// 验证URL
const validateURL = () => {
  if (!callbackURL.value) {
    isURLValid.value = false
    validationMessage.value = ''
    return
  }

  try {
    const url = new URL(callbackURL.value)
    if (url.hostname !== 'localhost' || url.port !== '8080' || !url.pathname.includes('callback')) {
      isURLValid.value = false
      validationMessage.value = 'URL格式不正确，应该是localhost:8080的回调地址'
      return
    }

    const params = new URLSearchParams(url.search)
    const hasCode = params.has('code')
    const hasState = params.has('state')

    if (!hasCode && !params.has('error')) {
      isURLValid.value = false
      validationMessage.value = 'URL中未找到授权码或错误信息'
      return
    }

    isURLValid.value = true
    validationMessage.value = hasCode ? 'URL格式正确，包含授权码' : 'URL包含错误信息'
    currentStep.value = 3
  } catch (e) {
    isURLValid.value = false
    validationMessage.value = '无效的URL格式'
  }
}

// 从剪贴板粘贴
const pasteFromClipboard = async () => {
  try {
    const text = await navigator.clipboard.readText()
    callbackURL.value = text
    validateURL()
    ElMessage.success('已从剪贴板粘贴')
  } catch (error) {
    ElMessage.error('无法访问剪贴板，请手动粘贴 (Ctrl+V)')
  }
}

// 提交回调URL
const submitCallbackURL = async () => {
  if (!isURLValid.value) {
    ElMessage.error('请输入有效的回调URL')
    return
  }

  submitting.value = true
  try {
    const result = await WailsAPI.ProcessOAuthCallback(callbackURL.value)
    if (result.success) {
      ElMessage.success('授权成功！')

      // 更新状态显示
      statusTitle.value = '授权成功！'
      statusMessage.value = 'Microsoft Todo授权已完成，正在保存令牌...'
      progress.value = 100
      progressStatus.value = 'success'
      progressText.value = '授权成功，正在保存令牌'

      // 延迟触发配置验证，确保token有足够时间保存到缓存文件
      setTimeout(() => {
        statusMessage.value = '令牌已保存，正在触发配置验证...'
        emit('success', result)

        // 再延迟1.5秒后自动关闭对话框
        setTimeout(() => {
          closeDialog()
        }, 1500)
      }, 2000) // 增加到2秒延迟
    } else {
      error.value = result.error || '授权失败'
    }
  } catch (error) {
    error.value = error.message || '处理回调URL失败'
  } finally {
    submitting.value = false
  }
}

// 设置事件监听器
const setupEventListeners = () => {
  // 监听OAuth结果
  const runtime = (window as any).runtime
  if (runtime) {
    // 监听OAuth开始事件
    runtime.EventsOn('oauthStarted', (data: any) => {
      console.log('OAuth流程已启动:', data)
    })

    // 监听OAuth结果
    runtime.EventsOn('oauthResult', (result: any) => {
      handleOAuthResult(result)
    })

    // 监听OAuth错误
    runtime.EventsOn('oauthError', (errorMsg: string) => {
      handleOAuthError(errorMsg)
    })
  }
}

// 处理OAuth结果
const handleOAuthResult = (result: any) => {
  processing.value = false
  authCompleted.value = true

  if (result.success) {
    statusTitle.value = '授权成功！'
    statusMessage.value = 'Microsoft Todo授权已完成，正在保存令牌...'
    progress.value = 100
    progressStatus.value = 'success'
    progressText.value = '授权成功，正在保存令牌'

    ElMessage.success('Microsoft账户授权成功！')

    // 延迟触发配置验证，确保token有足够时间保存到缓存文件
    setTimeout(() => {
      statusMessage.value = '令牌已保存，正在触发配置验证...'
      emit('success', result)

      // 再延迟1.5秒后自动关闭对话框
      setTimeout(() => {
        closeDialog()
      }, 1500)
    }, 2000) // 增加到2秒延迟
  } else {
    error.value = result.error || result.error_description || '授权失败'
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
  if (activeTab.value === 'auto') {
    startBrowserAuth()
  } else {
    resetManualAuthState()
  }
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
    runtime.EventsOff('oauthStarted')
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
  authCompleted.value = false
  showAuthStatus.value = false
  statusTitle.value = ''
  statusMessage.value = ''
  progress.value = 0
  progressStatus.value = ''
  progressText.value = ''
  resetManualAuthState()
  resetURLInputState()
}

// 重置手动认证状态
const resetManualAuthState = () => {
  currentStep.value = 0
  callbackURL.value = ''
  isURLValid.value = false
  validationMessage.value = ''
  submitting.value = false
}

// 重置URL输入状态（用于自动授权流程）
const resetURLInputState = () => {
  callbackURL.value = ''
  isURLValid.value = false
  validationMessage.value = ''
  submitting.value = false
}

// 切换标签页时重置状态
const handleTabChange = (tabName: string) => {
  if (tabName === 'manual') {
    resetManualAuthState()
  } else if (tabName === 'auto') {
    resetURLInputState()
  }
}

// 组件挂载时
onMounted(() => {
  setupEventListeners()
  if (dialogVisible.value) {
    if (activeTab.value === 'auto') {
      startBrowserAuth()
    }
  }
})

// 组件卸载时清理
onUnmounted(() => {
  const runtime = (window as any).runtime
  if (runtime) {
    runtime.EventsOff('oauthStarted')
    runtime.EventsOff('oauthResult')
    runtime.EventsOff('oauthError')
  }
})
</script>

<style scoped lang="scss">
.auto-auth-content, .manual-auth-content {
  padding: 20px;
}

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

.auth-window-info {
  margin: 20px 0;

  .el-alert {
    p {
      margin: 4px 0;
    }
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

.callback-url-section {
  margin: 24px 0;
  padding: 20px;
  background: #fffbe6;
  border: 1px solid #f7ba2a;
  border-radius: 8px;

  .el-alert {
    margin-bottom: 20px;

    p {
      margin: 4px 0;
    }
  }

  .callback-url-input {
    margin-bottom: 15px;

    :deep(.el-textarea__inner) {
      font-family: 'Consolas', 'Monaco', monospace;
      font-size: 14px;
    }
  }

  .callback-url-actions {
    display: flex;
    gap: 10px;
    margin-bottom: 15px;

    .el-button {
      flex: 1;
    }
  }

  .url-validation {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
  }
}

.manual-auth-content {
  .auth-steps {
    margin-bottom: 30px;
  }

  .action-buttons {
    text-align: center;
    margin: 30px 0;

    .el-button {
      padding: 12px 30px;
      font-size: 16px;
    }
  }

  .url-input-section {
    .url-input {
      margin: 20px 0;

      :deep(.el-textarea__inner) {
        font-family: 'Consolas', 'Monaco', monospace;
        font-size: 14px;
      }
    }

    .input-actions {
      display: flex;
      gap: 10px;
      margin-top: 15px;

      .el-button {
        flex: 1;
      }
    }

    .url-validation {
      margin-top: 10px;
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 14px;
    }
  }
}

.dialog-footer {
  text-align: right;
}

.code {
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Consolas', 'Monaco', monospace;
}
</style>