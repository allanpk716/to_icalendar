<script setup lang="ts">
// 简化的标签导航组件
</script>

<template>
  <div class="tab-navigation">
    <el-tabs type="border-card">
      <el-tab-pane label="主页" name="home">
        <template #label>
          <div class="tab-label">
            <el-icon>
              <House />
            </el-icon>
            <span>主页</span>
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