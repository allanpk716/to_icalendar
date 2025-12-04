<template>
  <div class="test-item-detail">
    <div class="item-header">
      <el-icon
        :color="testItem.success ? '#67C23A' : '#F56C6C'"
        size="20"
      >
        <component :is="testItem.success ? SuccessFilled : CircleCloseFilled" />
      </el-icon>
      <span class="item-name">{{ testItem.name }}</span>
      <el-tag :type="testItem.success ? 'success' : 'danger'" size="small">
        {{ testItem.success ? '通过' : '失败' }}
      </el-tag>
      <span v-if="testItem.duration" class="duration">
        耗时: {{ formatDuration(testItem.duration) }}
      </span>
    </div>

    <div v-if="testItem.message" class="item-message">
      <el-alert
        :type="testItem.success ? 'success' : 'error'"
        :title="testItem.message"
        :closable="false"
        show-icon
      />
    </div>

    <div v-if="testItem.details" class="item-details">
      <div class="details-header">
        <span>详细信息</span>
        <el-button
          v-if="!testItem.success"
          type="primary"
          size="small"
          @click="$emit('show-error', testItem)"
        >
          查看错误
        </el-button>
      </div>
      <pre>{{ testItem.details }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { SuccessFilled, CircleCloseFilled } from '@element-plus/icons-vue'

interface TestItem {
  name: string
  success: boolean
  message?: string
  details?: string
  duration?: number
}

defineProps<{
  testItem: TestItem
}>()

defineEmits<{
  'show-error': [error: TestItem]
}>()

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
}
</script>

<style scoped lang="scss">
.test-item-detail {
  .item-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;

    .item-name {
      font-weight: 500;
      flex: 1;
    }

    .duration {
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
  }

  .item-message {
    margin-bottom: 16px;
  }

  .item-details {
    .details-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;
      font-weight: 500;
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
</style>