<script setup lang="ts">
import { useRouter } from 'vue-router'
import { Calendar, Setting, Connection, Upload } from '@element-plus/icons-vue'

const router = useRouter()

// 快速访问初始化页面
const goToInit = () => {
  router.push({ name: 'init' })
}
</script>

<template>
  <div class="home-container">
    <div class="home-content">
      <h1 class="home-title">
        <el-icon size="32">
          <Calendar />
        </el-icon>
        to_icalendar
      </h1>
      <p class="home-description">
        Microsoft Todo 提醒事项管理工具
      </p>

      <!-- 快速操作卡片 -->
      <div class="action-cards">
        <el-card class="action-card" shadow="hover" @click="goToInit">
          <div class="card-content">
            <el-icon size="40" color="#409EFF">
              <Setting />
            </el-icon>
            <h3>初始化配置</h3>
            <p>创建或重置配置文件</p>
          </div>
        </el-card>

        <el-card class="action-card disabled" shadow="hover">
          <div class="card-content">
            <el-icon size="40" color="#909399">
              <Connection />
            </el-icon>
            <h3>连接测试</h3>
            <p>测试Microsoft Todo连接</p>
          </div>
        </el-card>

        <el-card class="action-card disabled" shadow="hover">
          <div class="card-content">
            <el-icon size="40" color="#909399">
              <Upload />
            </el-icon>
            <h3>发送提醒</h3>
            <p>创建并发送提醒事项</p>
          </div>
        </el-card>
      </div>

      <!-- 状态信息 -->
      <div class="status-section">
        <el-alert
          title="欢迎使用 to_icalendar"
          description="请先进行配置初始化，然后设置您的Microsoft Todo API凭证"
          type="info"
          show-icon
          :closable="false"
        />
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.home-container {
  height: 100%;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.home-content {
  max-width: 1000px;
  width: 100%;
  text-align: center;
}

.home-title {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  font-size: 36px;
  color: var(--text-color-primary);
  margin-bottom: 16px;
}

.home-description {
  font-size: 18px;
  color: var(--text-color-secondary);
  margin-bottom: 40px;
}

.action-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 24px;
  margin-bottom: 40px;
}

.action-card {
  cursor: pointer;
  transition: all 0.3s ease;

  &:hover {
    transform: translateY(-4px);
  }

  &.disabled {
    cursor: not-allowed;
    opacity: 0.6;

    &:hover {
      transform: none;
    }
  }

  .card-content {
    padding: 24px;
    text-align: center;

    h3 {
      font-size: 18px;
      color: var(--text-color-primary);
      margin: 16px 0 8px;
    }

    p {
      color: var(--text-color-secondary);
      margin: 0;
      line-height: 1.5;
    }
  }
}

.status-section {
  max-width: 600px;
  margin: 0 auto;
}

// 响应式设计
@media (max-width: 768px) {
  .home-container {
    padding: 16px;
  }

  .home-title {
    font-size: 28px !important;
  }

  .home-description {
    font-size: 16px;
  }

  .action-cards {
    grid-template-columns: 1fr;
    gap: 16px;
  }
}
</style>
