<script setup lang="ts">
import type { ProcessResult } from '@/types';
import { CircleCloseFilled, QuestionFilled, SuccessFilled } from '@element-plus/icons-vue';
import { computed, ref } from 'vue';

const props = defineProps<{
  modelValue: boolean
  processResult: ProcessResult | null
  width?: string | number
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'retry'): void
  (e: 'close'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const showErrorDetailDialog = ref(false)
const selectedError = ref<ProcessResult | null>(null)

const getPriorityType = (p?: string) => {
  const v = p?.toLowerCase()
  if (v === 'high') return 'danger'
  if (v === 'medium') return 'warning'
  if (v === 'low') return 'success'
  return 'info'
}

const formatDuration = (d?: number) => {
  if (!d) return ''
  if (d < 1000) return `${d}ms`
  return `${(d / 1000).toFixed(1)}s`
}

const getErrorTypeInfo = (t?: string) => {
  switch (t) {
    case 'config': return { icon: CircleCloseFilled, color: '#E6A23C', text: '配置错误' }
    case 'network': return { icon: CircleCloseFilled, color: '#E6A23C', text: '网络错误' }
    case 'parsing': return { icon: CircleCloseFilled, color: '#F56C6C', text: '解析错误' }
    case 'api': return { icon: CircleCloseFilled, color: '#E6A23C', text: 'API错误' }
    default: return { icon: CircleCloseFilled, color: '#F56C6C', text: '未知错误' }
  }
}

const showErrorDetail = (err: ProcessResult) => {
  selectedError.value = err
  showErrorDetailDialog.value = true
}
</script>

<template>
  <el-dialog v-model="visible" title="处理结果" :width="props.width" :center="true" :close-on-click-modal="false">
    <div v-if="props.processResult" class="result-content">
      <el-result v-if="props.processResult.success" icon="success" title="处理成功"
        :sub-title="props.processResult.message + (props.processResult.duration ? ` (耗时: ${formatDuration(props.processResult.duration)})` : '')">
        <template #extra>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="任务标题">{{ props.processResult.title }}</el-descriptions-item>
            <el-descriptions-item label="任务描述" v-if="props.processResult.description">{{ props.processResult.description
              }}</el-descriptions-item>
            <el-descriptions-item label="任务列表" v-if="props.processResult.list">{{ props.processResult.list
              }}</el-descriptions-item>
            <el-descriptions-item label="优先级" v-if="props.processResult.priority">
              <el-tag :type="getPriorityType(props.processResult.priority)">{{ props.processResult.priority }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="AI解析内容" v-if="props.processResult.parsedAnswer">
              <pre class="code-block">{{ props.processResult.parsedAnswer }}</pre>
            </el-descriptions-item>
          </el-descriptions>
        </template>
      </el-result>

      <el-result v-else icon="error" title="处理失败"
        :sub-title="props.processResult.message + (props.processResult.duration ? ` (耗时: ${formatDuration(props.processResult.duration)})` : '')">
        <template #extra>
          <div class="error-type-info" v-if="props.processResult.errorType">
            <el-tag :type="getErrorTypeInfo(props.processResult.errorType).color === '#F56C6C' ? 'danger' : 'warning'"
              class="error-type-tag">
              <el-icon class="tag-icon">
                <component :is="getErrorTypeInfo(props.processResult.errorType).icon" />
              </el-icon>
              {{ getErrorTypeInfo(props.processResult.errorType).text }}
            </el-tag>
          </div>
          <div class="suggestions" v-if="props.processResult.suggestions && props.processResult.suggestions.length > 0">
            <h4>解决建议：</h4>
            <ul>
              <li v-for="(s, i) in props.processResult.suggestions" :key="i">{{ s }}</li>
            </ul>
          </div>
          <div class="error-detail">
            <el-button type="info" plain size="small" @click="showErrorDetail(props.processResult)"
              class="detail-button">
              <el-icon>
                <QuestionFilled />
              </el-icon>
              查看详细错误
            </el-button>
          </div>
        </template>
      </el-result>
    </div>
    <template #footer>
      <el-button @click="emit('close'); visible = false">关闭</el-button>
      <el-button v-if="!props.processResult?.success && props.processResult?.canRetry" type="warning"
        @click="emit('retry')">重新尝试</el-button>
      <el-button type="primary" @click="visible = false">确定</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="showErrorDetailDialog" title="错误详情" width="520px" :center="true" :close-on-click-modal="false">
    <div class="error-detail-content" v-if="selectedError">
      <div class="error-type-section">
        <el-alert :title="getErrorTypeInfo(selectedError?.errorType).text" type="error"
          :description="selectedError?.error" show-icon :closable="false" />
      </div>
      <div class="suggestions-section" v-if="selectedError?.suggestions && selectedError?.suggestions.length > 0">
        <h4>解决建议</h4>
        <div class="suggestions-list">
          <div v-for="(s, i) in selectedError?.suggestions" :key="i" class="suggestion-item">
            <el-icon>
              <SuccessFilled />
            </el-icon>
            <span>{{ s }}</span>
          </div>
        </div>
      </div>
      <div class="retry-info" v-if="selectedError?.canRetry !== undefined">
        <el-tag :type="selectedError?.canRetry ? 'success' : 'danger'">{{ selectedError?.canRetry ? '可以重试' : '不建议重试'
          }}</el-tag>
      </div>
    </div>
    <template #footer>
      <el-button @click="showErrorDetailDialog = false">关闭</el-button>
      <el-button v-if="selectedError?.canRetry" type="warning"
        @click="emit('retry'); showErrorDetailDialog = false">重新尝试</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.code-block {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
}
</style>
