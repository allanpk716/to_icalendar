<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { Setting, Tools, InfoFilled } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()

// 切换标签页
const handleTabClick = (tabName: string) => {
  router.push({ name: tabName })
}
</script>

<template>
  <div class="tab-navigation">
    <el-tabs
      type="border-card"
      :model-value="route.name"
      @tab-click="({ paneName }) => handleTabClick(paneName)"
    >
      <el-tab-pane label="初始化" name="init">
        <template #label>
          <div class="tab-label">
            <el-icon>
              <Setting />
            </el-icon>
            <span>初始化</span>
          </div>
        </template>
      </el-tab-pane>

      <el-tab-pane label="配置测试" name="test">
        <template #label>
          <div class="tab-label">
            <el-icon>
              <Tools />
            </el-icon>
            <span>配置测试</span>
          </div>
        </template>
      </el-tab-pane>

      <el-tab-pane label="关于" name="about">
        <template #label>
          <div class="tab-label">
            <el-icon>
              <InfoFilled />
            </el-icon>
            <span>关于</span>
          </div>
        </template>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped lang="scss">
.tab-navigation {
  .el-tabs {
    background-color: transparent;
    border: none;

    :deep(.el-tabs__header) {
      background-color: lighten($background-color-base, 5%);
      margin: 0;
      border-bottom: 1px solid $border-color-base;
    }

    :deep(.el-tabs__nav-wrap) {
      padding: 0;
    }

    :deep(.el-tabs__nav) {
      border: none;
    }

    :deep(.el-tabs__item) {
      color: $text-color-secondary;
      border: none;
      background-color: transparent;
      padding: 0 20px;
      height: 50px;
      line-height: 50px;
      font-weight: 500;

      &:hover {
        color: $text-color-primary;
        background-color: lighten($background-color-base, 8%);
      }

      &.is-active {
        color: $primary-color;
        background-color: $background-color-base;
        border-bottom: 2px solid $primary-color;
      }
    }

    :deep(.el-tabs__content) {
      padding: 0;
      display: none; // 内容由路由处理
    }
  }

  .tab-label {
    display: flex;
    align-items: center;
    gap: 8px;

    .el-icon {
      font-size: 16px;
    }

    span {
      font-size: 14px;
    }
  }
}
</style>