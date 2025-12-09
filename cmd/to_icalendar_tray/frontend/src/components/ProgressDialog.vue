<script setup lang="ts">
import type { LogMessage, ProcessResult } from '@/types';
import { CircleCloseFilled, Loading, SuccessFilled } from '@element-plus/icons-vue';
import { computed, nextTick, ref, watch } from 'vue';

const props = defineProps<{
    modelValue: boolean
    isProcessing: boolean
    progress: { step: number; message: string }
    logs: LogMessage[]
    processResult: ProcessResult | null
    width?: string | number
}>()

const emit = defineEmits<{
    (e: 'update:modelValue', v: boolean): void
}>()

const visible = computed({
    get: () => props.modelValue,
    set: (v: boolean) => emit('update:modelValue', v)
})

const activeTab = ref('progress')
const logContainer = ref<HTMLElement>()

const progressPercentage = computed(() => {
    if (!props.isProcessing && props.progress.step === 0) return 0
    const p = Math.min(100, Math.max(0, props.progress.step * (100 / 6)))
    if (!props.isProcessing && props.processResult && props.processResult.success) return 100
    return Math.round(p)
})

const progressStatus = computed(() => {
    if (props.processResult) {
        return props.processResult.success ? 'success' : 'exception'
    }
    if (props.progress.step >= 6) return 'success'
    return undefined
})

const getStepStatus = (i: number) => {
    if (props.processResult) {
        if (props.processResult.success) return 'finish'
        if (i < props.progress.step - 1) return 'finish'
        if (i === props.progress.step - 1) return 'error'
        return 'wait'
    }
    if (props.isProcessing) {
        if (i < props.progress.step - 1) return 'finish'
        if (i === props.progress.step - 1) return 'process'
        return 'wait'
    }
    if (props.progress.step >= 6) return 'finish'
    if (props.progress.step > 0) {
        if (i < props.progress.step) return 'finish'
        return 'wait'
    }
    return 'wait'
}

const getStepIcon = (i: number) => {
    const s = getStepStatus(i)
    if (s === 'finish') return SuccessFilled
    if (s === 'process') return Loading
    if (s === 'error') return CircleCloseFilled
    return undefined
}

watch(() => props.logs, () => {
    nextTick(() => {
        if (logContainer.value) {
            logContainer.value.scrollTop = logContainer.value.scrollHeight
        }
    })
}, { deep: true })
</script>

<template>
    <el-dialog v-model="visible" title="处理进度" :width="props.width" :center="true" :close-on-click-modal="false">
        <el-tabs v-model="activeTab">
            <el-tab-pane label="进度" name="progress">
                <div class="progress-card">
                    <el-steps :active="props.progress.step" direction="vertical" finish-status="success"
                        :process-status="props.isProcessing ? 'process' : 'wait'" align-center>
                        <el-step title="解码图片" description="解码剪贴板图片数据" :status="getStepStatus(0)"
                            :icon="getStepIcon(0)" />
                        <el-step title="上传AI服务" description="上传图片到Dify AI服务" :status="getStepStatus(1)"
                            :icon="getStepIcon(1)" />
                        <el-step title="AI分析" description="AI正在分析图片内容" :status="getStepStatus(2)"
                            :icon="getStepIcon(2)" />
                        <el-step title="解析结果" description="解析AI响应结果" :status="getStepStatus(3)"
                            :icon="getStepIcon(3)" />
                        <el-step title="创建Todo任务" description="在Microsoft Todo创建任务" :status="getStepStatus(4)"
                            :icon="getStepIcon(4)" />
                        <el-step title="完成" description="处理完成" :status="getStepStatus(5)" :icon="getStepIcon(5)" />
                    </el-steps>

                    <div v-if="props.progress.message" class="progress-message">
                        <el-icon class="is-loading" v-if="props.isProcessing">
                            <Loading />
                        </el-icon>
                        {{ props.progress.message }}
                    </div>

                    <div v-if="props.isProcessing || props.processResult || props.progress.step > 0"
                        class="realtime-progress">
                        <el-progress :percentage="progressPercentage" :status="progressStatus" :stroke-width="8"
                            :show-text="true" :indeterminate="props.isProcessing && progressPercentage === 0">
                            <template #default="{ percentage }">
                                <span class="progress-text">{{ Math.round(percentage as number) }}%</span>
                            </template>
                        </el-progress>

                        <div class="progress-info">
                            <div class="progress-message">
                                <el-icon class="is-loading" v-if="props.isProcessing">
                                    <Loading />
                                </el-icon>
                                {{ props.progress.message }}
                            </div>
                            <div class="progress-tips">
                                <el-text size="small" type="info">正在处理，请稍候...整个过程可能需要10-30秒</el-text>
                            </div>
                        </div>
                    </div>

                    <div v-if="props.processResult?.parsedAnswer" class="parsed-answer">
                        <el-card header="AI解析内容" class="parsed-card">
                            <pre class="code-block">{{ props.processResult.parsedAnswer }}</pre>
                        </el-card>
                    </div>
                </div>
            </el-tab-pane>
            <el-tab-pane label="日志" name="logs">
                <div class="log-container" ref="logContainer">
                    <div v-for="(log, i) in props.logs" :key="i" :class="['log-item', `log-${log.type}`]">
                        <span class="log-time">{{ log.time }}</span>
                        <span class="log-message">{{ log.message }}</span>
                    </div>
                    <div v-if="props.logs.length === 0" class="log-empty">暂无日志</div>
                </div>
            </el-tab-pane>
        </el-tabs>
        <template #footer>
            <el-button @click="visible = false">关闭</el-button>
        </template>
    </el-dialog>
</template>

<style scoped lang="scss">
.progress-card {
    flex: 0 0 auto;
}

.progress-message {
    margin-top: 16px;
    display: flex;
    align-items: center;
    gap: 8px;
}

.realtime-progress {
    margin-top: 20px;
    padding: 16px;
    background-color: var(--el-fill-color-lighter);
    border-radius: 8px;
    border: 1px solid var(--el-border-color-light);
}

.progress-text {
    font-weight: 600;
    color: var(--el-color-primary);
}

.parsed-answer {
    margin-top: 12px;
}

.parsed-card {
    padding: 8px;
}

.code-block {
    white-space: pre-wrap;
    word-break: break-word;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 13px;
}

.log-container {
    height: 100%;
    overflow-y: auto;
    padding: 8px;
    background-color: var(--el-bg-color);
    border-radius: 4px;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 13px;
}

.log-item {
    display: flex;
    gap: 12px;
    margin-bottom: 4px;
    line-height: 1.5;
    color: var(--el-text-color-primary);
}

.log-time {
    flex-shrink: 0;
    color: var(--el-text-color-secondary);
    font-weight: 500;
}

.log-message {
    flex: 1;
    word-break: break-word;
}

.log-item.log-error {
    color: var(--el-color-danger);
}

.log-item.log-success {
    color: var(--el-color-success);
}

.log-item.log-warning {
    color: var(--el-color-warning);
}

.log-empty {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100%;
    color: var(--el-text-color-secondary);
    font-style: italic;
}
</style>
