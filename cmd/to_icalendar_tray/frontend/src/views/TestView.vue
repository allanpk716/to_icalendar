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
import type { TestItemResult } from '@/types/api'

// 使用测试状态管理
const {
  testResult,
  progress,
  isRunning,
  currentTest,
  progressMessage,
  startTest,
  resetTest,
  formatDuration,
  getTestStatusText,
  isTestCompleted,
  hasTestPassed
} = useTest()

// 本地状态
const error = ref<string>('')

// 计算属性
const canStartTest = computed(() => !isRunning.value)
const showResults = computed(() => testResult.value !== null)

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

    // 调试输出：打印完整的测试结果
    console.log('测试结果:', result)
    if (result.result) {
      console.log('配置测试:', result.result.configTest)
      console.log('配置测试错误:', result.result.configTest?.error)
      console.log('Todo测试:', result.result.todoTest)
      console.log('Todo测试错误:', result.result.todoTest?.error)
      console.log('Dify测试:', result.result.difyTest)
      console.log('Dify测试错误:', result.result.difyTest?.error)
    }

    if (result.success) {
      ElMessage.success('测试完成！')
    } else {
      throw new Error(result.error || '测试失败')
    }
  } catch (err: any) {
    error.value = err.message || '测试执行失败'
    ElMessage.error(`测试失败: ${error.value}`)
  }
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

      <!-- 主要内容区 - 使用Grid布局 -->
      <div class="main-section">
        <!-- 左侧：测试控制 -->
        <div class="left-panel">
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
              </div>
            </div>
          </el-card>
        </div>

        <!-- 右侧：测试结果 - 直接显示 -->
        <div class="right-panel">
          <el-card v-if="showResults" class="result-card compact" shadow="hover">
            <template #header>
              <div class="card-header">
                <el-icon :color="hasTestPassed() ? '#67C23A' : '#F56C6C'">
                  <component :is="hasTestPassed() ? SuccessFilled : CircleCloseFilled" />
                </el-icon>
                <span>测试结果</span>
                <el-tag :type="hasTestPassed() ? 'success' : 'danger'" size="small">
                  {{ hasTestPassed() ? '通过' : '失败' }}
                </el-tag>
              </div>
            </template>

            <div class="result-content">
              <!-- 紧凑的总体信息 -->
              <div class="overall-info-compact">
                <div class="info-row">
                  <span class="label">总体状态:</span>
                  <el-tag :type="hasTestPassed() ? 'success' : 'danger'" size="small">
                    {{ hasTestPassed() ? '测试通过' : '测试失败' }}
                  </el-tag>
                </div>
                <div class="info-row">
                  <span class="label">总耗时:</span>
                  <span class="value">{{ formatDuration(testResult?.duration || 0) }}</span>
                </div>
              </div>

              <!-- 紧凑的测试结果项 -->
              <div class="test-items-compact">
                <!-- 配置文件验证 -->
                <div v-if="testResult?.configTest" class="test-item-compact">
                  <div class="item-header">
                    <el-icon :color="getTestItemIcon(testResult.configTest).color" size="16">
                      <DocumentChecked />
                    </el-icon>
                    <span class="item-name">{{ testResult.configTest.name }}</span>
                    <el-tag
                      :type="testResult.configTest.success ? 'success' : 'danger'"
                      size="small"
                    >
                      {{ getTestItemStatus(testResult.configTest) }}
                    </el-tag>
                  </div>
                  <!-- 失败时直接显示错误信息 -->
                  <div v-if="!testResult.configTest.success && testResult.configTest.error" class="error-info-compact">
                    <div class="error-content">
                      <el-icon color="#F56C6C" size="14"><WarningFilled /></el-icon>
                      <span class="error-text">{{ testResult.configTest.error }}</span>
                    </div>
                    <el-button
                      type="text"
                      size="small"
                      @click="showDetailedError(testResult.configTest)"
                      class="detail-btn"
                      title="查看详细错误信息"
                    >
                      <el-icon size="12"><View /></el-icon>
                    </el-button>
                  </div>
                </div>

                <!-- Microsoft Todo 服务测试 -->
                <div v-if="testResult?.todoTest" class="test-item-compact">
                  <div class="item-header">
                    <el-icon :color="getTestItemIcon(testResult.todoTest).color" size="16">
                      <Connection />
                    </el-icon>
                    <span class="item-name">{{ testResult.todoTest.name }}</span>
                    <el-tag
                      :type="testResult.todoTest.success ? 'success' : 'danger'"
                      size="small"
                    >
                      {{ getTestItemStatus(testResult.todoTest) }}
                    </el-tag>
                  </div>
                  <!-- 失败时直接显示错误信息 -->
                  <div v-if="!testResult.todoTest.success && testResult.todoTest.error" class="error-info-compact">
                    <div class="error-content">
                      <el-icon color="#F56C6C" size="14"><WarningFilled /></el-icon>
                      <span class="error-text">{{ testResult.todoTest.error }}</span>
                    </div>
                    <el-button
                      type="text"
                      size="small"
                      @click="showDetailedError(testResult.todoTest)"
                      class="detail-btn"
                      title="查看详细错误信息"
                    >
                      <el-icon size="12"><View /></el-icon>
                    </el-button>
                  </div>
                </div>

                <!-- Dify 服务测试 -->
                <div v-if="testResult?.difyTest" class="test-item-compact">
                  <div class="item-header">
                    <el-icon :color="getTestItemIcon(testResult.difyTest).color" size="16">
                      <Setting />
                    </el-icon>
                    <span class="item-name">{{ testResult.difyTest.name }}</span>
                    <el-tag
                      :type="testResult.difyTest.success ? 'success' : 'warning'"
                      size="small"
                    >
                      {{ getTestItemStatus(testResult.difyTest) }}
                    </el-tag>
                  </div>
                  <!-- 失败时直接显示错误信息 -->
                  <div v-if="!testResult.difyTest.success && testResult.difyTest.error" class="error-info-compact">
                    <div class="error-content">
                      <el-icon color="#F56C6C" size="14"><WarningFilled /></el-icon>
                      <span class="error-text">{{ testResult.difyTest.error }}</span>
                    </div>
                    <el-button
                      type="text"
                      size="small"
                      @click="showDetailedError(testResult.difyTest)"
                      class="detail-btn"
                      title="查看详细错误信息"
                    >
                      <el-icon size="12"><View /></el-icon>
                    </el-button>
                  </div>
                </div>
              </div>

              <!-- 建议操作 -->
              <div class="suggestions-compact">
                <el-alert
                  v-if="hasTestPassed()"
                  title="所有测试通过"
                  type="success"
                  show-icon
                  :closable="false"
                />
                <el-alert
                  v-else
                  title="部分测试失败"
                  type="warning"
                  show-icon
                  :closable="false"
                />
              </div>
            </div>
          </el-card>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.test-container {
  height: 100%;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.test-content {
  max-width: 1200px;
  width: 100%;
  height: 100vh;
  display: flex;
  flex-direction: column;
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
  flex: 1;
  align-items: stretch;
  min-height: 0;
}

.left-panel, .right-panel {
  min-height: 0;
}

.test-control-card, .result-card {
  height: fit-content;
  min-height: 300px;
  max-height: none;
  overflow-y: auto;

  &.compact {
    .card-content {
      padding: 16px;
    }
  }

  .card-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    text-align: center;
  }

  .card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    color: var(--text-color-primary);
    font-size: 14px;
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

.result-content {
  width: 100%;

  .overall-info-compact {
    display: flex;
    gap: 16px;
    margin-bottom: 16px;
    padding: 8px;
    background-color: var(--background-color-light);
    border-radius: 4px;

    .info-row {
      display: flex;
      align-items: center;
      gap: 8px;

      .label {
        font-size: 13px;
        color: var(--text-color-secondary);
      }

      .value {
        font-size: 13px;
        color: var(--text-color-primary);
        font-weight: 500;
      }
    }
  }

  .test-items-compact {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-bottom: 16px;

    .test-item-compact {
      padding: 12px;
      border: 1px solid var(--border-color);
      border-radius: 6px;
      background-color: var(--background-color-light);

      .item-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 8px;

        .item-name {
          font-weight: 500;
          color: var(--text-color-primary);
          font-size: 13px;
          flex: 1;
        }
      }

      .error-info-compact {
        display: flex;
        align-items: flex-start;
        gap: 8px;
        padding: 8px;
        background-color: #FEF0F0;
        border: 1px solid #F56C6C;
        border-radius: 4px;

        .error-content {
          flex: 1;
          display: flex;
          align-items: center;
          gap: 6px;

          .error-text {
            font-size: 12px;
            color: #F56C6C;
            line-height: 1.3;
            word-break: break-all;
          }
        }

        .detail-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 4px;
          color: #F56C6C;
          background-color: transparent;
          border: 1px solid #F56C6C;
          border-radius: 3px;
          font-size: 12px;
          transition: all 0.2s ease;
          min-width: 24px;
          height: 24px;

          &:hover {
            background-color: #F56C6C;
            color: white;
          }
        }
      }
    }
  }

  .suggestions-compact {
    .el-alert {
      :deep(.el-alert__description) {
        display: none;
      }
    }
  }
}

// 响应式设计
@media (max-width: 1024px) {
  .main-section {
    gap: 16px;
  }
}

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

  .overall-info-compact {
    flex-direction: column;
    gap: 8px;
  }
}

// 动画效果
.test-item-compact {
  transition: all 0.3s ease;

  &:hover {
    border-color: var(--color-primary);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  }
}

.el-progress {
  transition: all 0.3s ease;
}
</style>