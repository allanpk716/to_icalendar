<script setup lang="ts">
import { useClipboardUpload } from '@/composables/useClipboardUpload'
import { useResponsiveDialog } from '@/composables/useResponsiveDialog'
import type { ProcessResult } from '@/types'
import {
    CircleCloseFilled,
    DocumentCopy,
    Loading,
    Picture,
    QuestionFilled,
    Refresh,
    SuccessFilled,
    Tools,
    WarningFilled
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

// 使用剪贴板上传功能
const {
  clipboardBase64,
  hasImage,
  isProcessing,
  progress,
  processResult,
  logs,
  previewUrl,
  configStatus,
  checkConfigStatus,
  getClipboardImage,
  processImageToTodo,
  clearResult,
  resetProcessingState
} = useClipboardUpload()

// 使用响应式对话框管理
const { calculateDialogWidth } = useResponsiveDialog()

// 本地状态
const showImageDialog = ref(false)
const showResultDialog = ref(false)
const showErrorDetailDialog = ref(false)
const selectedError = ref<ProcessResult | null>(null)
const autoRefresh = ref(false)
const refreshInterval = ref<number>()
const logContainer = ref<HTMLElement>()
const imageError = ref(false)
const isUserOperation = ref(false) // 用户操作标识

// 图片加载处理
const handleImageLoad = () => {
  imageError.value = false
}

const handleImageError = () => {
  imageError.value = true
}

// 计算属性
const canProcess = computed(() => hasImage.value && !isProcessing.value && configStatus.value.ready)
const processButtonText = computed(() => isProcessing.value ? '处理中...' : '分析并创建任务')

// 计算对话框宽度
const resultDialogWidth = computed(() => calculateDialogWidth(600, 800))
const errorDialogWidth = computed(() => calculateDialogWidth(500, 700))

// 进度相关计算属性
const progressPercentage = computed(() => {
  // 确保进度基于实际的处理状态
  if (!isProcessing.value && progress.step === 0) return 0

  // 基于步骤计算百分比，确保与任务状态同步
  const percentage = Math.min(100, Math.max(0, progress.step * (100 / 6)))

  // 如果处理完成且有结果，确保显示100%
  if (!isProcessing.value && processResult.value && processResult.value.success) {
    return 100
  }

  return Math.round(percentage)
})

const progressStatus = computed(() => {
  // 如果有处理结果，根据结果显示状态
  if (processResult.value) {
    if (processResult.value.success) {
      return 'success'
    } else {
      return 'exception'
    }
  }

  // 处理成功完成时显示成功状态（基于进度）
  if (progress.step >= 6) {
    return 'success'
  }

  // 正在处理中，不显示特定状态
  if (isProcessing.value) {
    return undefined
  }

  // 未开始处理
  return undefined
})

// 获取错误类型图标和颜色
const getErrorTypeInfo = (errorType?: string) => {
  switch (errorType) {
    case 'config':
      return { icon: WarningFilled, color: '#E6A23C', text: '配置错误' }
    case 'network':
      return { icon: WarningFilled, color: '#E6A23C', text: '网络错误' }
    case 'parsing':
      return { icon: CircleCloseFilled, color: '#F56C6C', text: '解析错误' }
    case 'api':
      return { icon: WarningFilled, color: '#E6A23C', text: 'API错误' }
    default:
      return { icon: CircleCloseFilled, color: '#F56C6C', text: '未知错误' }
  }
}

// 获取剪贴板图片
const handleGetClipboard = async () => {
  await getClipboardImage()
  if (hasImage.value) {
    ElMessage.success('成功获取剪贴板图片')
  }
}

// 手动刷新剪贴板
const handleManualRefresh = async () => {
  await getClipboardImage(true)
}

// 处理并创建任务
const handleProcessUpload = async () => {
  if (!hasImage.value) {
    ElMessage.warning('请先获取剪贴板图片')
    return
  }

  if (!configStatus.value.ready) {
    ElMessage.error('配置未就绪，请检查配置后重试')
    return
  }

  // 先清除之前的结果和状态
  resetProcessingState()

  // 设置用户操作标识，禁用剪贴板操作
  isUserOperation.value = true
  autoRefresh.value = false

  try {
    const taskID = await processImageToTodo()

    // 等待任务完成，通过监听 processResult 的变化来处理结果
    // 结果将通过轮询机制自动更新 processResult.value
    // 然后通过监听器显示结果弹窗

    // 如果需要显示结果弹窗，可以通过监听 processResult 的变化来实现
    // 这里我们让轮询机制处理结果显示
  } catch (error) {
    console.error('处理失败:', error)
    ElMessage.error('处理失败，请查看详细信息')
  }
  // 处理完成后不立即恢复，让用户手动控制
  // 注释掉自动恢复
  // setTimeout(() => {
  //   isUserOperation.value = false
  //   autoRefresh.value = true
  // }, 3000)
}

// 清除结果
const handleClearResult = () => {
  clearResult()
  showResultDialog.value = false
}

// 自动刷新剪贴板
const startAutoRefresh = () => {
  if (autoRefresh.value) {
    refreshInterval.value = setInterval(async () => {
      // 只有在窗口可见、未处理且非用户操作时才自动刷新
      if (!document.hidden && !isProcessing.value && !isUserOperation.value) {
        // 使用防抖机制
        if (!window.clipboardRefreshPending) {
          window.clipboardRefreshPending = true
          await getClipboardImage(false) // 静默获取，不显示消息

          // 增加防抖时间到2秒，减少剪贴板锁定概率
          setTimeout(() => {
            window.clipboardRefreshPending = false
          }, 2000)
        }
      }
    }, 5000) // 保持5秒刷新间隔
  }
}

const stopAutoRefresh = () => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
}

// 监听自动刷新开关
watch(autoRefresh, (newVal) => {
  if (newVal) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
})

// 监听日志变化，自动滚动到底部
watch(logs, () => {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}, { deep: true })

// 监听页面可见性变化
const handleVisibilityChange = () => {
  if (document.hidden) {
    stopAutoRefresh()
  } else if (autoRefresh.value) {
    startAutoRefresh()
  }
}

// 组件挂载时
onMounted(() => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
  // 不自动启动刷新，让用户手动控制

  // 监听处理结果变化
  watch(processResult, (newResult) => {
    if (newResult && newResult.success) {
      showResultDialog.value = true
      // 任务完成后延迟恢复用户操作状态
      setTimeout(() => {
        isUserOperation.value = false
      }, 2000) // 2秒后恢复，避免竞态条件
    } else if (newResult && !newResult.success) {
      showResultDialog.value = true
      // 任务失败后也延迟恢复用户操作状态
      setTimeout(() => {
        isUserOperation.value = false
      }, 2000)
    }
  }, { deep: true })
})

// 组件卸载时清理
onUnmounted(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  stopAutoRefresh()
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
  }
})

// 获取优先级类型
const getPriorityType = (priority: string) => {
  switch (priority?.toLowerCase()) {
    case 'high':
      return 'danger'
    case 'medium':
      return 'warning'
    case 'low':
      return 'success'
    default:
      return 'info'
  }
}

// 显示错误详情
const showErrorDetail = (error: ProcessResult) => {
  selectedError.value = error
  showErrorDetailDialog.value = true
}

// 重新尝试处理
const handleRetry = async () => {
  showResultDialog.value = false
  showErrorDetailDialog.value = false
  await handleProcessUpload()
}

// 格式化处理时间
const formatDuration = (duration?: number) => {
  if (!duration) return ''
  if (duration < 1000) {
    return `${duration}ms`
  }
  return `${(duration / 1000).toFixed(1)}s`
}

// 获取步骤状态
const getStepStatus = (stepIndex: number) => {
  if (processResult.value) {
    if (processResult.value.success) return 'finish'
    if (stepIndex < progress.step - 1) return 'finish'
    if (stepIndex === progress.step - 1) return 'error'
    return 'wait'
  }

  if (isProcessing.value) {
    if (stepIndex < progress.step - 1) return 'finish'
    if (stepIndex === progress.step - 1) return 'process'
    return 'wait'
  }

  if (progress.step >= 6) return 'finish'
  if (progress.step > 0) {
    if (stepIndex < progress.step) return 'finish'
    return 'wait'
  }
  return 'wait'
}

// 获取步骤图标
const getStepIcon = (stepIndex: number) => {
  const status = getStepStatus(stepIndex)
  switch (status) {
    case 'finish':
      return SuccessFilled
    case 'process':
      return Loading
    case 'error':
      return CircleCloseFilled
    default:
      return undefined
  }
}
</script>

<template>
  <div class="clipboard-view">
    <!-- 配置状态提示 -->
    <div v-if="!configStatus.ready" class="config-status-banner">
      <el-alert
        :title="configStatus.error || '配置未就绪'"
        :type="configStatus.configExists ? 'warning' : 'error'"
        :closable="false"
        show-icon
      >
        <template #default>
          <div v-if="configStatus.suggestions && configStatus.suggestions.length > 0">
            <p>建议：</p>
            <ul>
              <li v-for="(suggestion, index) in configStatus.suggestions" :key="index">{{ suggestion }}</li>
            </ul>
          </div>
          <div v-if="!configStatus.configExists" class="config-actions">
            <el-button type="primary" size="small" @click="$router.push('/init')">
              前往初始化
            </el-button>
          </div>
        </template>
      </el-alert>
    </div>

    <!-- 顶部操作栏 -->
    <div class="action-bar">
      <div class="left-actions">
        <el-button type="primary" :icon="DocumentCopy" @click="handleGetClipboard" :loading="isProcessing">
          获取剪贴板图片
        </el-button>

        <el-button type="success" :icon="Tools" @click="handleProcessUpload" :disabled="!canProcess || !configStatus.ready"
          :loading="isProcessing">
          {{ processButtonText }}
        </el-button>

        <el-button :icon="Refresh" @click="handleManualRefresh" :loading="isProcessing" circle />
      </div>

      </div>

    <!-- 主内容区域 -->
    <div class="main-content">
      <!-- 左侧：图片预览 -->
      <div class="preview-section">
        <el-card header="图片预览" class="preview-card">
          <div v-if="hasImage" class="image-container" @click="showImageDialog = true">
            <el-image v-if="previewUrl" :src="previewUrl" fit="contain" class="preview-image-el" @load="handleImageLoad"
              @error="handleImageError" />
            <div v-if="isProcessing" class="image-placeholder">
              <el-icon class="is-loading">
                <Loading />
              </el-icon>
              <span>加载中...</span>
            </div>
            <div v-if="imageError" class="image-error">
              <el-icon>
                <Picture />
              </el-icon>
              <span>图片加载失败</span>
            </div>
          </div>
          <el-empty v-else description="暂无剪贴板图片" :image-size="120">
            <el-button type="primary" @click="handleGetClipboard">
              获取剪贴板内容
            </el-button>
          </el-empty>
        </el-card>
      </div>

      <!-- 右侧：处理信息 -->
      <div class="process-section">
        <!-- 进度显示 -->
        <el-card header="处理进度" class="progress-card">
          <el-steps
            :active="progress.step"
            direction="vertical"
            finish-status="success"
            :process-status="isProcessing ? 'process' : 'wait'"
            align-center
          >
            <el-step
              title="解码图片"
              description="解码剪贴板图片数据"
              :status="getStepStatus(0)"
              :icon="getStepIcon(0)"
            />
            <el-step
              title="上传AI服务"
              description="上传图片到Dify AI服务"
              :status="getStepStatus(1)"
              :icon="getStepIcon(1)"
            />
            <el-step
              title="AI分析"
              description="AI正在分析图片内容"
              :status="getStepStatus(2)"
              :icon="getStepIcon(2)"
            />
            <el-step
              title="解析结果"
              description="解析AI响应结果"
              :status="getStepStatus(3)"
              :icon="getStepIcon(3)"
            />
            <el-step
              title="创建Todo任务"
              description="在Microsoft Todo创建任务"
              :status="getStepStatus(4)"
              :icon="getStepIcon(4)"
            />
            <el-step
              title="完成"
              description="处理完成"
              :status="getStepStatus(5)"
              :icon="getStepIcon(5)"
            />
          </el-steps>

          <div v-if="progress.message" class="progress-message">
            <el-icon class="is-loading" v-if="isProcessing">
              <Loading />
            </el-icon>
            <el-icon v-else-if="processResult && processResult.success" class="success-icon">
              <SuccessFilled />
            </el-icon>
            {{ progress.message }}
          </div>

          <!-- 实时进度条 -->
          <div v-if="isProcessing || processResult || progress.step > 0" class="realtime-progress">
            <el-progress
              :percentage="progressPercentage"
              :status="progressStatus"
              :stroke-width="8"
              :show-text="true"
              :indeterminate="isProcessing && progressPercentage === 0"
            >
              <template #default="{ percentage }">
                <span class="progress-text">{{ Math.round(percentage) }}%</span>
              </template>
            </el-progress>

            <div class="progress-info">
              <div class="progress-message">
                <el-icon class="is-loading" v-if="isProcessing">
                  <Loading />
                </el-icon>
                {{ progress.message }}
              </div>
              <div class="progress-tips">
                <el-text size="small" type="info">
                  正在处理，请稍候...整个过程可能需要10-30秒
                </el-text>
              </div>
            </div>
          </div>
        </el-card>

        <!-- 日志输出 -->
        <el-card header="处理日志" class="log-card">
          <div class="log-container" ref="logContainer">
            <div v-for="(log, index) in logs" :key="index" :class="['log-item', `log-${log.type}`]">
              <span class="log-time">{{ log.time }}</span>
              <span class="log-message">{{ log.message }}</span>
            </div>
            <div v-if="logs.length === 0" class="log-empty">
              暂无日志
            </div>
          </div>
        </el-card>
      </div>
    </div>

    <!-- 图片预览对话框 -->
    <el-dialog v-model="showImageDialog" title="图片预览" width="80%" :center="true">
      <el-image v-if="previewUrl" :src="previewUrl" fit="contain" style="width: 100%; max-height: 70vh;" />
    </el-dialog>

    <!-- 处理结果对话框 -->
    <el-dialog
      v-model="showResultDialog"
      title="处理结果"
      :width="resultDialogWidth"
      :center="true"
      :close-on-click-modal="false"
    >
      <div v-if="processResult" class="result-content">
        <!-- 成功结果 -->
        <el-result
          v-if="processResult.success"
          icon="success"
          title="处理成功"
          :sub-title="processResult.message + (processResult.duration ? ` (耗时: ${formatDuration(processResult.duration)})` : '')"
        >
          <template #extra>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="任务标题">
                {{ processResult.title }}
              </el-descriptions-item>
              <el-descriptions-item label="任务描述" v-if="processResult.description">
                {{ processResult.description }}
              </el-descriptions-item>
              <el-descriptions-item label="任务列表" v-if="processResult.list">
                {{ processResult.list }}
              </el-descriptions-item>
              <el-descriptions-item label="优先级" v-if="processResult.priority">
                <el-tag :type="getPriorityType(processResult.priority)">
                  {{ processResult.priority }}
                </el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </template>
        </el-result>

        <!-- 失败结果 -->
        <el-result
          v-else
          icon="error"
          title="处理失败"
          :sub-title="processResult.message + (processResult.duration ? ` (耗时: ${formatDuration(processResult.duration)})` : '')"
        >
          <template #extra>
            <!-- 错误类型显示 -->
            <div class="error-type-info" v-if="processResult.errorType">
              <el-tag
                :type="getErrorTypeInfo(processResult.errorType).color === '#F56C6C' ? 'danger' : 'warning'"
                class="error-type-tag"
              >
                <el-icon class="tag-icon">
                  <component :is="getErrorTypeInfo(processResult.errorType).icon" />
                </el-icon>
                {{ getErrorTypeInfo(processResult.errorType).text }}
              </el-tag>
            </div>

            <!-- 解决建议 -->
            <div class="suggestions" v-if="processResult.suggestions && processResult.suggestions.length > 0">
              <h4>解决建议：</h4>
              <ul>
                <li v-for="(suggestion, index) in processResult.suggestions" :key="index">
                  {{ suggestion }}
                </li>
              </ul>
            </div>

            <!-- 详细错误信息 -->
            <div class="error-detail">
              <el-button
                type="info"
                plain
                size="small"
                @click="showErrorDetail(processResult)"
                class="detail-button"
              >
                <el-icon><QuestionFilled /></el-icon>
                查看详细错误
              </el-button>
            </div>
          </template>
        </el-result>
      </div>

      <template #footer>
        <el-button @click="handleClearResult">关闭</el-button>
        <el-button
          v-if="!processResult?.success && processResult?.canRetry"
          type="warning"
          @click="handleRetry"
        >
          重新尝试
        </el-button>
        <el-button type="primary" @click="showResultDialog = false">
          确定
        </el-button>
      </template>
    </el-dialog>

    <!-- 错误详情对话框 -->
    <el-dialog
      v-model="showErrorDetailDialog"
      title="错误详情"
      :width="errorDialogWidth"
      :center="true"
      :close-on-click-modal="false"
    >
      <div class="error-detail-content" v-if="selectedError">
        <!-- 错误类型 -->
        <div class="error-type-section">
          <el-alert
            :title="getErrorTypeInfo(selectedError.errorType).text"
            type="error"
            :description="selectedError.error"
            show-icon
            :closable="false"
          />
        </div>

        <!-- 解决建议 -->
        <div class="suggestions-section" v-if="selectedError.suggestions && selectedError.suggestions.length > 0">
          <h4>解决建议</h4>
          <div class="suggestions-list">
            <div
              v-for="(suggestion, index) in selectedError.suggestions"
              :key="index"
              class="suggestion-item"
            >
              <el-icon><SuccessFilled /></el-icon>
              <span>{{ suggestion }}</span>
            </div>
          </div>
        </div>

        <!-- 重试信息 -->
        <div class="retry-info" v-if="selectedError.canRetry !== undefined">
          <el-tag :type="selectedError.canRetry ? 'success' : 'danger'">
            {{ selectedError.canRetry ? '可以重试' : '不建议重试' }}
          </el-tag>
        </div>
      </div>

      <template #footer>
        <el-button @click="showErrorDetailDialog = false">关闭</el-button>
        <el-button
          v-if="selectedError?.canRetry"
          type="warning"
          @click="handleRetry"
        >
          重新尝试
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.clipboard-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
  background-color: var(--el-bg-color-page);
}

.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background-color: var(--el-bg-color);
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);

  .left-actions {
    display: flex;
    gap: 12px;
  }
}

.main-content {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  min-height: 0;
}

.preview-section {
  align-self: start;
}

.preview-card,
.progress-card,
.log-card {
  height: 100%;
  display: flex;
  flex-direction: column;

  :deep(.el-card__header) {
    flex: 0 0 auto;
  }

  :deep(.el-card__body) {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    position: relative;
    contain: layout paint;
    padding: 0 !important;
  }
}

.preview-card {
  height: 60vh;
  max-height: 70vh;
}

.image-container {
  height: 100%;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  cursor: pointer;
  overflow: hidden;
  contain: layout paint;
}

.preview-image-el {
  position: absolute;
  inset: 0;
}

:deep(.preview-image-el .el-image__inner) {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: top left;
  display: block;
}

// 更新占位符样式
.image-placeholder,
.image-error {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-secondary);

  .el-icon {
    font-size: 24px;
  }
}

.process-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.progress-card {
  flex: 0 0 auto;

  :deep(.el-steps) {
    .el-step__head {
      .el-step__icon {
        transition: all 0.3s ease;
      }
    }

    .el-step__title {
      font-size: 14px;
      font-weight: 500;
      transition: all 0.3s ease;
    }

    .el-step__description {
      font-size: 12px;
      color: var(--el-text-color-secondary);
      margin-top: 4px;
      transition: all 0.3s ease;
    }

    .el-step.is-process {
      .el-step__title {
        color: var(--el-color-primary);
        font-weight: 600;
      }

      .el-step__description {
        color: var(--el-color-primary);
      }

      .el-step__icon {
        color: var(--el-color-primary);
        border-color: var(--el-color-primary);
      }
    }

    .el-step.is-finish {
      .el-step__title {
        color: var(--el-color-success);
        font-weight: 500;
      }

      .el-step__description {
        color: var(--el-text-color-regular);
      }

      .el-step__icon {
        color: var(--el-color-success);
        border-color: var(--el-color-success);
      }
    }

    .el-step.is-error {
      .el-step__title {
        color: var(--el-color-danger);
        font-weight: 600;
      }

      .el-step__description {
        color: var(--el-color-danger);
      }

      .el-step__icon {
        color: var(--el-color-danger);
        border-color: var(--el-color-danger);
      }
    }
  }

  .progress-message {
    margin-top: 16px;
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--el-text-color-primary);
    font-size: 14px;
    font-weight: 500;

    .el-icon {
      font-size: 16px;
    }
  }

  .realtime-progress {
    margin-top: 20px;
    padding: 16px;
    background-color: var(--el-fill-color-lighter);
    border-radius: 8px;
    border: 1px solid var(--el-border-color-light);

    .progress-text {
      font-weight: 600;
      color: var(--el-color-primary);
    }

    .progress-info {
      margin-top: 12px;

      .progress-message {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 8px;
        font-size: 14px;
        color: var(--el-text-color-primary);
        font-weight: 500;

        .el-icon {
          font-size: 16px;
        }
      }

      .progress-tips {
        text-align: center;
      }
    }
  }
}

.log-card {
  flex: 1;
  min-height: 0;
}

.log-container {
  height: 100%;
  overflow-y: auto;
  padding: 8px;
  background-color: var(--el-bg-color);
  border-radius: 4px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-thumb {
    background-color: var(--el-border-color-darker);
    border-radius: 3px;
  }
}

.log-item {
  display: flex;
  gap: 12px;
  margin-bottom: 4px;
  line-height: 1.5;

  .log-time {
    flex-shrink: 0;
    color: var(--el-text-color-secondary);
    font-weight: 500;
  }

  .log-message {
    flex: 1;
    word-break: break-word;
  }

  &.log-error {
    color: var(--el-color-danger);
  }

  &.log-success {
    color: var(--el-color-success);
  }

  &.log-warning {
    color: var(--el-color-warning);
  }
}

.log-empty {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  color: var(--el-text-color-secondary);
  font-style: italic;
}

// 配置状态提示样式
.config-status-banner {
  margin: 16px;
  margin-bottom: 8px;
}

.config-status-banner ul {
  margin: 8px 0 0 0;
  padding-left: 20px;
}

.config-status-banner li {
  margin: 4px 0;
}

.config-actions {
  margin-top: 8px;
}

// 结果对话框样式
.result-content {
  .error-type-info {
    margin: 16px 0;
    text-align: center;

    .error-type-tag {
      font-size: 14px;
      padding: 8px 16px;

      .tag-icon {
        margin-right: 6px;
      }
    }
  }

  .suggestions {
    margin: 20px 0;
    background-color: var(--el-fill-color-lighter);
    border-radius: 6px;
    padding: 16px;

    h4 {
      margin: 0 0 12px 0;
      color: var(--el-text-color-primary);
      font-size: 14px;
      font-weight: 600;
    }

    ul {
      margin: 0;
      padding-left: 20px;
      color: var(--el-text-color-regular);

      li {
        margin: 8px 0;
        line-height: 1.5;
        font-size: 13px;
      }
    }
  }

  .error-detail {
    margin-top: 16px;
    text-align: center;

    .detail-button {
      .el-icon {
        margin-right: 4px;
      }
    }
  }
}

// 错误详情对话框样式
.error-detail-content {
  .error-type-section {
    margin-bottom: 20px;
  }

  .suggestions-section {
    margin: 20px 0;

    h4 {
      margin: 0 0 12px 0;
      color: var(--el-text-color-primary);
      font-size: 14px;
      font-weight: 600;
    }

    .suggestions-list {
      .suggestion-item {
        display: flex;
        align-items: center;
        gap: 8px;
        margin: 12px 0;
        padding: 12px;
        background-color: var(--el-fill-color-lighter);
        border-radius: 6px;
        color: var(--el-text-color-regular);
        font-size: 13px;

        .el-icon {
          color: var(--el-color-success);
          flex-shrink: 0;
        }

        span {
          line-height: 1.4;
        }
      }
    }
  }

  .retry-info {
    margin-top: 20px;
    text-align: center;

    .el-tag {
      font-size: 13px;
      padding: 6px 12px;
    }
  }
}

// 响应式设计
@media (max-width: 1024px) {
  .main-content {
    grid-template-columns: 1fr;
    grid-template-rows: 1fr 1fr;
  }
}

@media (max-width: 768px) {
  .clipboard-view {
    padding: 8px;
  }

  .action-bar {
    flex-direction: column;
    gap: 12px;

    .left-actions {
      width: 100%;
      justify-content: center;
    }
  }

  .result-content {
    .suggestions {
      padding: 12px;

      h4 {
        font-size: 13px;
      }

      ul li {
        font-size: 12px;
      }
    }
  }

  .error-detail-content {
    .suggestions-section {
      h4 {
        font-size: 13px;
      }

      .suggestions-list .suggestion-item {
        padding: 8px;
        font-size: 12px;
      }
    }
  }
}
</style>
