<script setup lang="ts">
import { useClipboardUpload } from '@/composables/useClipboardUpload'
import {
  DocumentCopy,
  Loading,
  Picture,
  Refresh,
  Tools
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
  getClipboardImage,
  processImageToTodo,
  clearResult
} = useClipboardUpload()

// 本地状态
const showImageDialog = ref(false)
const showResultDialog = ref(false)
const autoRefresh = ref(true)
const refreshInterval = ref<number>()
const logContainer = ref<HTMLElement>()
const imageError = ref(false)

// 图片加载处理
const handleImageLoad = () => {
  imageError.value = false
}

const handleImageError = () => {
  imageError.value = true
}

// 计算属性
const canProcess = computed(() => hasImage.value && !isProcessing.value)
const processButtonText = computed(() => isProcessing.value ? '处理中...' : '分析并创建任务')

// 获取剪贴板图片
const handleGetClipboard = async () => {
  await getClipboardImage()
  if (hasImage.value) {
    ElMessage.success('成功获取剪贴板图片')
  }
}

// 处理并创建任务
const handleProcessUpload = async () => {
  if (!hasImage.value) {
    ElMessage.warning('请先获取剪贴板图片')
    return
  }

  const result = await processImageToTodo()
  if (result?.success) {
    showResultDialog.value = true
  }
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
      // 只有在窗口可见且未处理时才自动刷新
      if (!document.hidden && !isProcessing.value) {
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

// 组件挂载时开始自动刷新
onMounted(() => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
  startAutoRefresh()
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
</script>

<template>
  <div class="clipboard-view">
    <!-- 顶部操作栏 -->
    <div class="action-bar">
      <div class="left-actions">
        <el-button type="primary" :icon="DocumentCopy" @click="handleGetClipboard" :loading="isProcessing">
          获取剪贴板图片
        </el-button>

        <el-button type="success" :icon="Tools" @click="handleProcessUpload" :disabled="!canProcess"
          :loading="isProcessing">
          {{ processButtonText }}
        </el-button>

        <el-button :icon="Refresh" @click="getClipboardImage" :loading="isProcessing" circle />
      </div>

      <div class="right-actions">
        <el-switch v-model="autoRefresh" active-text="自动刷新" inactive-text="手动刷新" />
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
          <el-steps :active="progress.step" direction="vertical" finish-status="success">
            <el-step title="检查剪贴板" description="验证剪贴板内容" />
            <el-step title="AI 分析" description="使用 Dify AI 分析内容" />
            <el-step title="创建任务" description="在 Microsoft Todo 创建任务" />
            <el-step title="完成" description="处理完成" />
          </el-steps>

          <div v-if="progress.message" class="progress-message">
            <el-icon class="is-loading" v-if="isProcessing">
              <Loading />
            </el-icon>
            {{ progress.message }}
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
    <el-dialog v-model="showResultDialog" title="处理结果" width="60%" :center="true">
      <div v-if="processResult" class="result-content">
        <el-result :icon="processResult.success ? 'success' : 'error'" :title="processResult.success ? '处理成功' : '处理失败'"
          :sub-title="processResult.message">
          <template #extra v-if="processResult.success">
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
      </div>

      <template #footer>
        <el-button @click="handleClearResult">关闭</el-button>
        <el-button type="primary" @click="showResultDialog = false">
          确定
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

  .progress-message {
    margin-top: 16px;
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--el-text-color-primary);

    .el-icon {
      font-size: 16px;
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
}
</style>
