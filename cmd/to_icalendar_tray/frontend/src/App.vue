<script setup lang="ts">
import { onMounted } from 'vue'
import { useAppState } from '@/composables/useAppState'
import { useWails } from '@/composables/useWails'
import TabNavigation from '@/components/TabNavigation.vue'
import Home from '@/views/Home.vue'

// 状态管理
const { globalStatus } = useAppState()

// Wails通信
const { init: initWails } = useWails()

// 应用初始化
onMounted(async () => {
  try {
    // 初始化Wails连接
    await initWails()
    console.log('应用初始化完成')
  } catch (error) {
    console.error('应用初始化失败:', error)
  }
})
</script>

<template>
  <div id="app" class="app-container">
    <!-- 标签导航 -->
    <div class="nav-section">
      <TabNavigation />
    </div>

    <!-- 主要内容区域 -->
    <main class="main-content">
      <div class="content-container">
        <!-- 主页面内容 -->
        <Home />
      </div>
    </main>

    <!-- 全局状态指示器 -->
    <div v-if="globalStatus === 'loading'" class="global-loading">
      <el-icon class="is-loading">
        <Loading />
      </el-icon>
      <span>处理中...</span>
    </div>
  </div>
</template>

<style scoped lang="scss">
.app-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: var(--background-color-base);
  color: var(--text-color-primary);
}


.nav-section {
  background-color: lighten($background-color-base, 5%);
  border-bottom: 1px solid $border-color-base;
  flex-shrink: 0;
}

.main-content {
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

.content-container {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.global-loading {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  background-color: rgba(0, 0, 0, 0.8);
  color: white;
  padding: 16px 24px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 9999;

  .el-icon {
    font-size: 18px;
  }
}

// 响应式设计
@media (max-width: 768px) {
  .content-container {
    padding: 12px;
  }
}
</style>