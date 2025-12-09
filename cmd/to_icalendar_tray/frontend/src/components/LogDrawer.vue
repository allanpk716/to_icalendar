<script setup lang="ts">
import type { LogMessage } from '@/types'
import { computed, nextTick, ref, watch } from 'vue'

const props = defineProps<{
  modelValue: boolean
  logs: LogMessage[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'clear'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v)
})

const logContainer = ref<HTMLElement>()

watch(() => props.logs, () => {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}, { deep: true })
</script>

<template>
  <el-drawer v-model="visible" title="处理日志" direction="rtl" size="80%">
    <div class="action-bar" style="margin-bottom: 8px; justify-content: flex-end; display: flex;">
      <el-button type="warning" @click="emit('clear')">清空日志</el-button>
    </div>
    <div class="log-container" ref="logContainer">
      <div v-for="(log, i) in props.logs" :key="i" :class="['log-item', `log-${log.type}`]">
        <span class="log-time">{{ log.time }}</span>
        <span class="log-message">{{ log.message }}</span>
      </div>
      <div v-if="props.logs.length === 0" class="log-empty">暂无日志</div>
    </div>
  </el-drawer>
</template>

<style scoped>
.log-container { height: 100%; overflow-y: auto; padding: 8px; background-color: var(--el-bg-color); border-radius: 4px; font-family: 'Consolas','Monaco', monospace; font-size: 13px; }
.log-item { display: flex; gap: 12px; margin-bottom: 4px; line-height: 1.5; color: var(--el-text-color-primary); }
.log-time { flex-shrink: 0; color: var(--el-text-color-secondary); font-weight: 500; }
.log-message { flex: 1; word-break: break-word; }
.log-item.log-error { color: var(--el-color-danger); }
.log-item.log-success { color: var(--el-color-success); }
.log-item.log-warning { color: var(--el-color-warning); }
.log-empty { display: flex; justify-content: center; align-items: center; height: 100%; color: var(--el-text-color-secondary); font-style: italic; }
</style>

