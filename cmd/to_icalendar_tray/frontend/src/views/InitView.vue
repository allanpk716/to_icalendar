<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Setting, Tools, QuestionFilled, SuccessFilled, Refresh, CircleCheckFilled } from '@element-plus/icons-vue'
import { WailsAPI } from '@/api/wails'
import type { WailsResponse, ServerConfig } from '@/types/api'
import { useAppState } from '@/composables/useAppState'

// 状态管理
const { setGlobalStatus } = useAppState()
const isInitializing = ref(false)
const initResult = ref<ServerConfig | null>(null)
const error = ref<string>('')
const guideDialogVisible = ref(false)

// 新增对话框状态
const showSuccessDialog = ref(false)
const showErrorDialog = ref(false)
const errorMessage = ref('')

// 计算属性
const canInit = computed(() => !isInitializing.value)

// 配置指南步骤数据
const configSteps = [
  {
    title: '访问 Azure Portal',
    description: '前往 Azure 管理门户',
    content: '打开 https://portal.azure.com'
  },
  {
    title: '注册应用程序',
    description: '创建新的Azure AD应用或选择现有应用',
    content: '在 Azure Active Directory -> 应用注册 中操作'
  },
  {
    title: '配置 API 权限',
    description: '添加 Microsoft Graph API 权限',
    content: '添加 Tasks.ReadWrite.All 权限'
  },
  {
    title: '创建客户端密钥',
    description: '生成应用程序密钥',
    content: '在证书和密钥部分创建新的客户端密钥'
  },
  {
    title: '配置应用信息',
    description: '将获取的信息填入配置文件',
    content: '编辑 ~/.to_icalendar/server.yaml 文件'
  }
]

// 初始化配置
const handleInit = async () => {
  try {
    isInitializing.value = true
    setGlobalStatus('loading')
    error.value = ''

    // 确认对话框
    const confirmResult = await ElMessageBox.confirm(
      '此操作将创建默认配置文件，如果配置文件已存在将被忽略。是否继续？',
      '确认初始化',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info',
      }
    )

    if (confirmResult !== 'confirm') {
      return
    }

    // 调用后端API
    const response: WailsResponse<ServerConfig> = await WailsAPI.InitConfig()

    if (response.success && response.data) {
      initResult.value = response.data
      showSuccessDialog.value = true
      ElMessage.success('配置初始化成功！')
    } else {
      throw new Error(response.error || '初始化失败')
    }
  } catch (err: any) {
    if (err !== 'cancel') {
      const error = err as Error
      errorMessage.value = error.message
      showErrorDialog.value = true
      ElMessage.error('初始化失败，请查看详细信息')
    }
  } finally {
    isInitializing.value = false
    setGlobalStatus('idle')
  }
}

// 查看配置指南
const showGuide = () => {
  guideDialogVisible.value = true
}

// 打开配置目录
const openConfigLocation = async () => {
  try {
    // 调用后端API打开配置目录
    await WailsAPI.OpenConfigDirectory()
    showSuccessDialog.value = false
  } catch (err) {
    ElMessage.error('无法打开配置目录')
  }
}
</script>

<template>
  <div class="init-container">
    <div class="init-content">
      <!-- 页面标题 -->
      <div class="page-header">
        <h2 class="page-title">
          <el-icon><Setting /></el-icon>
          配置初始化
        </h2>
        <p class="page-description">
          初始化Microsoft Todo配置文件，设置API凭证和默认参数
        </p>
      </div>

      <!-- 主要内容区 -->
      <div class="main-section">
        <!-- 初始化卡片 -->
        <el-card class="init-card" shadow="hover">
          <div class="card-content">
            <div class="init-icon">
              <el-icon size="80" color="#409EFF">
                <Tools />
              </el-icon>
            </div>

            <div class="init-text">
              <h3>初始化配置文件</h3>
              <p>创建默认的server.yaml配置文件，包含Microsoft Todo API设置</p>
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

            <!-- 操作按钮 -->
            <div class="action-buttons">
              <el-button
                type="primary"
                size="large"
                :loading="isInitializing"
                :disabled="!canInit"
                @click="handleInit"
              >
                <el-icon><Refresh /></el-icon>
                {{ isInitializing ? '初始化中...' : 'Init 初始化' }}
              </el-button>

              <el-button
                type="primary"
                size="large"
                @click="showGuide"
              >
                <el-icon><QuestionFilled /></el-icon>
                配置指南
              </el-button>
            </div>
          </div>
        </el-card>
      </div>
    </div>

    <!-- 初始化成功对话框 -->
    <el-dialog
      v-model="showSuccessDialog"
      title="初始化成功"
      width="600px"
      :close-on-click-modal="false"
    >
      <div class="success-content">
        <el-result
          icon="success"
          title="配置初始化成功"
          sub-title="配置文件已成功创建"
        >
          <template #extra>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="配置目录">
                ~/.to_icalendar/
              </el-descriptions-item>
              <el-descriptions-item label="服务器配置">
                server.yaml
              </el-descriptions-item>
              <el-descriptions-item label="状态">
                <el-tag type="success">已创建</el-tag>
              </el-descriptions-item>
            </el-descriptions>

            <div class="next-steps">
              <h4>后续步骤</h4>
              <ol>
                <li>编辑配置文件填入Azure AD信息</li>
                <li>配置Microsoft Todo API权限</li>
                <li>运行连接测试验证配置</li>
              </ol>
            </div>
          </template>
        </el-result>
      </div>

      <template #footer>
        <el-button @click="showSuccessDialog = false">关闭</el-button>
        <el-button type="primary" @click="openConfigLocation">打开配置目录</el-button>
      </template>
    </el-dialog>

    <!-- 初始化失败对话框 -->
    <el-dialog
      v-model="showErrorDialog"
      title="初始化失败"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-result
        icon="error"
        title="初始化失败"
        :sub-title="errorMessage"
      >
        <template #extra>
          <el-button type="primary" @click="showErrorDialog = false">确定</el-button>
        </template>
      </el-result>
    </el-dialog>

    <!-- 配置指南对话框 -->
    <el-dialog
      v-model="guideDialogVisible"
      title="配置指南"
      width="600px"
      :center="true"
    >
      <div class="guide-content">
        <h4 class="guide-subtitle">Microsoft Todo API 配置步骤</h4>
        <p class="guide-description">
          按照以下步骤配置 Azure AD 应用程序，以获取访问 Microsoft Todo 所需的凭证信息。
        </p>

        <el-timeline class="guide-timeline">
          <el-timeline-item
            v-for="(step, index) in configSteps"
            :key="index"
            :icon="CircleCheckFilled"
            :type="index === configSteps.length - 1 ? 'primary' : 'success'"
            size="large"
          >
            <div class="timeline-content">
              <h5 class="step-title">{{ step.title }}</h5>
              <p class="step-description">{{ step.description }}</p>
              <p class="step-content">{{ step.content }}</p>
            </div>
          </el-timeline-item>
        </el-timeline>

        <div class="guide-note">
          <el-alert
            title="注意事项"
            description="配置完成后，请确保保存好客户端密钥信息，它只显示一次。"
            type="warning"
            show-icon
            :closable="false"
          />
        </div>
      </div>

      <template #footer>
        <span class="dialog-footer">
          <el-button type="primary" @click="guideDialogVisible = false">
            确定
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.init-container {
  height: 100%;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.init-content {
  max-width: 800px;
  width: 100%;
}

.page-header {
  text-align: center;
  margin-bottom: 32px;

  .page-title {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    font-size: 28px;
    color: var(--text-color-primary);
    margin-bottom: 8px;
  }

  .page-description {
    font-size: 16px;
    color: var(--text-color-secondary);
    margin: 0;
  }
}

.main-section {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.init-card {
  .card-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 24px;
    text-align: center;
  }
}

.success-content {
  .next-steps {
    margin-top: 20px;
    padding: 16px;
    background-color: var(--el-fill-color-lighter);
    border-radius: 8px;

    h4 {
      margin: 0 0 12px 0;
      color: var(--el-text-color-primary);
    }

    ol {
      margin: 0;
      padding-left: 20px;

      li {
        margin-bottom: 8px;
        color: var(--el-text-color-regular);

        &:last-child {
          margin-bottom: 0;
        }
      }
    }
  }
}

.init-icon {
  margin-bottom: 16px;
}

.init-text {
  h3 {
    font-size: 20px;
    color: var(--text-color-primary);
    margin-bottom: 8px;
  }

  p {
    color: var(--text-color-secondary);
    margin: 0;
    line-height: 1.6;
  }
}

.error-alert {
  width: 100%;
}

.action-buttons {
  display: flex;
  gap: 16px;
  justify-content: center;
  flex-wrap: wrap;
}


// 响应式设计
@media (max-width: 768px) {
  .init-container {
    padding: 16px;
  }

  .page-title {
    font-size: 24px !important;
  }

  .action-buttons {
    flex-direction: column;
    width: 100%;

    .el-button {
      width: 100%;
    }
  }
}

// 配置指南对话框样式
.guide-content {
  .guide-subtitle {
    color: var(--text-color-primary);
    margin-bottom: 16px;
    font-size: 18px;
    font-weight: 600;
    text-align: center;
  }

  .guide-description {
    color: var(--text-color-secondary);
    margin-bottom: 24px;
    line-height: 1.6;
    text-align: center;
  }

  .guide-timeline {
    margin: 24px 0;

    .timeline-content {
      text-align: left;
      padding-left: 8px;

      .step-title {
        color: var(--text-color-primary);
        font-size: 16px;
        font-weight: 600;
        margin-bottom: 8px;
        margin-top: 0;
      }

      .step-description {
        color: var(--text-color-secondary);
        font-size: 14px;
        margin-bottom: 4px;
        line-height: 1.5;
      }

      .step-content {
        color: var(--text-color-regular);
        font-size: 13px;
        margin: 0;
        line-height: 1.4;
        font-style: italic;
      }
    }
  }

  .guide-note {
    margin-top: 24px;
  }
}

// 响应式对话框
@media (max-width: 768px) {
  :deep(.el-dialog) {
    width: 90% !important;
    margin: 5vh auto;
  }

  .guide-content {
    .guide-timeline {
      .timeline-content {
        .step-title {
          font-size: 15px;
        }

        .step-description {
          font-size: 13px;
        }

        .step-content {
          font-size: 12px;
        }
      }
    }
  }
}
</style>