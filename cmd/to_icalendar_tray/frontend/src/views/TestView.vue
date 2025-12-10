<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Tools,
  SuccessFilled,
  CircleCloseFilled,
  WarningFilled,
  Refresh,
  Connection,
  DocumentChecked,
  Setting,
  View
} from '@element-plus/icons-vue'
import { useTest } from '@/composables/useTest'
import { useResponsiveDialog } from '@/composables/useResponsiveDialog'
import { WailsAPI } from '@/api/wails'
import type { TestItemResult } from '@/types/api'
import TestItemDetail from '@/components/TestItemDetail.vue'
import OAuthWebViewDialog from '@/components/OAuthWebViewDialog.vue'

// 使用测试状态管理
const {
  testResult,
  progress,
  isRunning,
  currentTest,
  progressMessage,
  testLogs,
  startTest,
  resetTest,
  formatDuration,
  getTestStatusText,
  isTestCompleted,
  hasTestPassed
} = useTest()

// 使用响应式对话框管理
const { calculateDialogWidth } = useResponsiveDialog()

// 本地状态
const error = ref<string>('')

// 新增对话框状态
const showResultDialog = ref(false)
const showErrorDetailDialog = ref(false)
const selectedError = ref<any>(null)
const activeCollapse = ref<string[]>(['config', 'todo', 'dify'])

// OAuth WebView授权对话框
const showWebViewAuth = ref(false)
const isAuthenticating = ref(false)

// 计算属性
const canStartTest = computed(() => !isRunning.value)
const showResults = computed(() => testResult.value !== null)

// 计算对话框宽度
const testResultDialogWidth = computed(() => calculateDialogWidth(800, 1000))
const errorDialogWidth = computed(() => calculateDialogWidth(600, 800))

// 获取测试项状态图标和颜色
const getTestItemIcon = (item: TestItemResult) => {
  if (item.success) {
    return { icon: SuccessFilled, color: '#67C23A' }
  } else {
    return { icon: CircleCloseFilled, color: '#F56C6C' }
  }
}

// 获取测试项状态文本
const getTestItemStatus = (item: TestItemResult) => {
  return item.success ? '通过' : '失败'
}

// 开始测试
const handleStartTest = async () => {
  try {
    error.value = ''

    const result = await startTest()

    if (result.success) {
      showResultDialog.value = true
      ElMessage.success('测试完成！')
    } else {
      // 检查是否需要重新授权
      if (result.error && result.error.includes('需要重新认证')) {
        await ElMessageBox.confirm(
          'Microsoft Todo 授权已过期，需要重新授权。是否立即进行授权？',
          '需要重新授权',
          {
            confirmButtonText: '立即授权',
            cancelButtonText: '稍后再说',
            type: 'warning'
          }
        )
        await handleAuthentication()
        return
      }
      throw new Error(result.error || '测试失败')
    }
  } catch (err: any) {
    const error = err as Error
    if (error.message !== 'cancel') {
      ElMessage.error(`测试失败: ${error.message}`)
    }
  }
}

// 重新测试
const handleRetest = () => {
  showResultDialog.value = false
  handleStartTest()
}

// 显示错误详情
const showErrorDetail = (errorItem: any) => {
  selectedError.value = errorItem
  showErrorDetailDialog.value = true
}

// 重置测试
const handleResetTest = () => {
  resetTest()
  error.value = ''
  ElMessage.info('测试状态已重置')
}

// 查看详细错误
const showDetailedError = async (item: TestItemResult) => {
  if (!item.error) return

  const fullErrorMessage = item.error + (item.details ? '\n\n详细信息：\n' + item.details : '')

  await ElMessageBox.alert(
    fullErrorMessage,
    `${item.name} - 详细错误信息`,
    {
      confirmButtonText: '确定',
      type: 'error'
    }
  )
}

// 处理认证
const handleAuthentication = async () => {
  showWebViewAuth.value = true
  isAuthenticating.value = true
}

// 授权成功回调
const onAuthSuccess = () => {
  ElMessage.success('授权成功，重新测试连接')
  isAuthenticating.value = false
  // 重新执行测试
  handleStartTest()
}

// 授权失败回调
const onAuthError = (error: string) => {
  ElMessage.error(`授权失败: ${error}`)
  isAuthenticating.value = false
}
</script>

<template>
  <div class="test-container">
    <div class="test-content">
      <!-- 页面标题 -->
      <div class="page-header">
        <h2 class="page-title">
          <el-icon><Tools /></el-icon>
          配置测试
        </h2>
        <p class="page-description">
          测试系统配置的完整性和服务连接状态，确保所有组件正常工作
        </p>
      </div>

      <!-- 主要内容区 -->
      <div class="main-section">
        <!-- 测试控制卡片 -->
        <el-card class="test-control-card" shadow="hover">
          <div class="card-content">
            <div class="test-icon">
              <el-icon size="60" color="#409EFF">
                <Tools />
              </el-icon>
            </div>

            <div class="test-text">
              <h3>系统配置测试</h3>
              <p>验证配置文件格式、Microsoft Todo连接和Dify服务状态</p>
            </div>

            <!-- 错误提示 -->
            <el-alert
              v-if="error"
              :title="error"
              type="error"
              show-icon
              :closable="false"
              class="error-alert"
            />

            <!-- 进度条 -->
            <div v-if="isRunning || progress > 0" class="progress-section">
              <div class="progress-info">
                <span class="current-test">{{ currentTest }}</span>
                <span class="progress-percent">{{ progress }}%</span>
              </div>
              <el-progress
                :percentage="progress"
                :status="isRunning ? undefined : (hasTestPassed() ? 'success' : 'exception')"
                :stroke-width="6"
              />
              <div class="progress-message">{{ progressMessage }}</div>
            </div>

            <!-- 操作按钮 -->
            <div class="action-buttons">
              <el-button
                type="primary"
                size="large"
                :loading="isRunning"
                :disabled="!canStartTest"
                @click="handleStartTest"
              >
                <el-icon><Connection /></el-icon>
                {{ isRunning ? '测试中...' : '开始测试' }}
              </el-button>

              <el-button
                size="large"
                :disabled="isRunning"
                @click="handleResetTest"
              >
                <el-icon><Refresh /></el-icon>
                重置
              </el-button>

              <el-button
                type="warning"
                size="large"
                :disabled="isRunning || isAuthenticating"
                :loading="isAuthenticating"
                @click="handleAuthentication"
              >
                <el-icon><Setting /></el-icon>
                {{ isAuthenticating ? '授权中...' : 'Microsoft授权' }}
              </el-button>
            </div>
          </div>
        </el-card>
      </div>

      <!-- 测试日志区域 -->
      <div v-if="testLogs.length > 0 || isRunning" class="log-section">
        <el-card class="log-card" shadow="hover">
          <template #header>
            <div class="log-header">
              <span class="log-title">测试日志</span>
              <el-button
                v-if="testLogs.length > 0"
                size="small"
                text
                @click="testLogs = []"
              >
                清空日志
              </el-button>
            </div>
          </template>
          <div class="log-content">
            <div
              v-for="(log, index) in testLogs"
              :key="index"
              class="log-item"
              :class="`log-${log.type}`"
            >
              <span class="log-time">{{ log.timestamp }}</span>
              <span class="log-message">{{ log.message }}</span>
            </div>
            <div v-if="isRunning" class="log-item log-info">
              <span class="log-time">{{ new Date().toLocaleTimeString() }}</span>
              <span class="log-message">{{ progressMessage }}</span>
            </div>
          </div>
        </el-card>
      </div>
    </div>

    <!-- 测试结果对话框 -->
    <el-dialog
      v-model="showResultDialog"
      title="测试结果"
      :width="testResultDialogWidth"
      :close-on-click-modal="false"
    >
      <div class="test-result-content">
        <!-- 总体状态 -->
        <div class="overall-status">
          <el-result
            :icon="hasTestPassed() ? 'success' : 'error'"
            :title="hasTestPassed() ? '测试通过' : '测试失败'"
            :sub-title="`总耗时: ${formatDuration(testResult?.duration || 0)}`"
          >
            <template #extra>
              <el-tag :type="hasTestPassed() ? 'success' : 'danger'" size="large">
                {{ hasTestPassed() ? '所有测试项通过' : '存在失败的测试项' }}
              </el-tag>
            </template>
          </el-result>
        </div>

        <!-- 详细测试结果 -->
        <div class="detailed-results">
          <el-collapse v-model="activeCollapse">
            <!-- 配置文件验证 -->
            <el-collapse-item
              v-if="testResult?.configTest"
              title="配置文件验证"
              name="config"
            >
              <test-item-detail
                :test-item="testResult.configTest"
                @show-error="showErrorDetail"
              />
            </el-collapse-item>

            <!-- Microsoft Todo 服务测试 -->
            <el-collapse-item
              v-if="testResult?.todoTest"
              title="Microsoft Todo 服务测试"
              name="todo"
            >
              <test-item-detail
                :test-item="testResult.todoTest"
                @show-error="showErrorDetail"
              />
            </el-collapse-item>

            <!-- Dify 服务测试 -->
            <el-collapse-item
              v-if="testResult?.difyTest"
              title="Dify 服务测试"
              name="dify"
            >
              <test-item-detail
                :test-item="testResult.difyTest"
                @show-error="showErrorDetail"
              />
            </el-collapse-item>
          </el-collapse>
        </div>
      </div>

      <template #footer>
        <el-button @click="showResultDialog = false">关闭</el-button>
        <el-button v-if="!hasTestPassed()" type="primary" @click="handleRetest">重新测试</el-button>
      </template>
    </el-dialog>

    <!-- 错误详情对话框 -->
    <el-dialog
      v-model="showErrorDetailDialog"
      title="错误详情"
      :width="errorDialogWidth"
      :close-on-click-modal="false"
    >
      <div class="error-detail-content">
        <el-alert
          :title="selectedError?.name || '错误'"
          type="error"
          :description="selectedError?.message || '未知错误'"
          show-icon
          :closable="false"
        />

        <div v-if="selectedError?.details" class="error-details">
          <h4>详细信息</h4>
          <pre>{{ selectedError.details }}</pre>
        </div>
      </div>

      <template #footer>
        <el-button @click="showErrorDetailDialog = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- OAuth WebView授权对话框 -->
    <OAuthWebViewDialog
      v-model="showWebViewAuth"
      @success="onAuthSuccess"
      @error="onAuthError"
    />
  </div>
</template>

<style scoped lang="scss">
.test-container {
  height: 100%;
  width: 100%;
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow-y: auto;
}

.test-content {
  max-width: 800px;
  width: 100%;
  margin: 0 auto;
}

.page-header {
  text-align: center;
  margin-bottom: 24px;

  .page-title {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    font-size: 24px;
    color: var(--text-color-primary);
    margin-bottom: 8px;
  }

  .page-description {
    font-size: 14px;
    color: var(--text-color-secondary);
    margin: 0;
  }
}

.main-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
}

.test-control-card {
  height: fit-content;
  min-height: 300px;

  .card-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    text-align: center;
  }
}

.test-icon {
  margin-bottom: 8px;
}

.test-text {
  h3 {
    font-size: 18px;
    color: var(--text-color-primary);
    margin-bottom: 8px;
  }

  p {
    color: var(--text-color-secondary);
    margin: 0;
    line-height: 1.4;
    font-size: 13px;
  }
}

.error-alert {
  width: 100%;
}

.progress-section {
  width: 100%;

  .progress-info {
    display: flex;
    justify-content: space-between;
    margin-bottom: 8px;
    font-size: 13px;

    .current-test {
      color: var(--text-color-primary);
      font-weight: 500;
    }

    .progress-percent {
      color: var(--text-color-secondary);
    }
  }

  .progress-message {
    margin-top: 8px;
    font-size: 12px;
    color: var(--text-color-secondary);
  }
}

.action-buttons {
  display: flex;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
}

.test-result-content {
  .overall-status {
    margin-bottom: 24px;
  }

  .detailed-results {
    .el-collapse {
      border: none;

      :deep(.el-collapse-item__header) {
        font-weight: 500;
      }
    }
  }
}

.error-detail-content {
  .error-details {
    margin-top: 20px;

    h4 {
      margin-bottom: 12px;
      color: var(--el-text-color-primary);
    }

    pre {
      background-color: var(--el-fill-color-lighter);
      padding: 12px;
      border-radius: 6px;
      font-size: 12px;
      line-height: 1.5;
      overflow-x: auto;
      margin: 0;
    }
  }
}

// 响应式设计
@media (max-width: 768px) {
  .test-container {
    padding: 16px;
  }

  .page-title {
    font-size: 20px !important;
  }

  .action-buttons {
    flex-direction: column;
    width: 100%;

    .el-button {
      width: 100%;
    }
  }
}

// 对话框样式优化
:deep(.el-dialog) {
  // 使用 Element Plus 的响应式行为，移除固定宽度
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);

  .el-dialog__body {
    padding: 20px;
    max-height: 70vh;
    overflow-y: auto;
  }

  // 保持对话框居中
  margin: 5vh auto;
}

// 测试结果对话框内容样式
// 测试结果对话框内容样式补充
.test-result-content {
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
}

// 测试日志样式
.log-section {
  margin-top: 20px;
}

.log-card {
  .log-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .log-title {
    font-weight: 500;
    color: var(--el-text-color-primary);
  }
}

.log-content {
  max-height: 300px;
  overflow-y: auto;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  background-color: var(--el-fill-color-lighter);
  border-radius: 6px;
  padding: 12px;
}

.log-item {
  display: flex;
  gap: 12px;
  padding: 4px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &:last-child {
    border-bottom: none;
  }

  .log-time {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    min-width: 80px;
    flex-shrink: 0;
  }

  .log-message {
    flex: 1;
    word-break: break-all;
  }

  // 不同日志类型的颜色
  &.log-info {
    .log-message {
      color: var(--el-color-info);
    }
  }

  &.log-success {
    .log-message {
      color: var(--el-color-success);
    }
  }

  &.log-warn {
    .log-message {
      color: var(--el-color-warning);
    }
  }

  &.log-error {
    .log-message {
      color: var(--el-color-error);
    }
  }

  &.log-debug {
    .log-message {
      color: var(--el-text-color-secondary);
      opacity: 0.8;
    }
  }
}
</style>