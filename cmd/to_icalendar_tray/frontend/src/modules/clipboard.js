import { appState, STATUS } from '../state.js';

export class ClipboardModule {
    constructor(container, logPanel, statusBar) {
        this.container = container;
        this.logPanel = logPanel;
        this.statusBar = statusBar;
        this.clipboardContent = null;
        this.taskLists = [];
        this.init();
    }

    init() {
        this.render();
        this.bindEvents();
        this.setupStateListener();
    }

    render() {
        this.container.innerHTML = `
            <div class="module-content clipboard-module">
                <h2>📋 剪贴板内容处理</h2>
                <p class="module-description">获取剪贴板内容并上传到 Microsoft Todo</p>

                <!-- 获取剪贴板内容区域 -->
                <div class="content-section">
                    <h3>📥 获取剪贴板内容</h3>
                    <button class="btn btn-primary" id="getClipboardBtn">
                        <span class="btn-icon">📋</span>
                        获取剪贴板内容
                    </button>
                    <div class="clipboard-status" id="clipboardStatus">点击按钮获取最新剪贴板内容</div>
                </div>

                <!-- 内容预览区域 -->
                <div class="content-section" id="previewSection" style="display: none;">
                    <h3>👀 内容预览</h3>
                    <div class="content-preview" id="contentPreview">
                        <div class="preview-placeholder">暂无内容</div>
                    </div>
                </div>

                <!-- 配置选项区域 -->
                <div class="content-section" id="optionsSection" style="display: none;">
                    <h3>⚙️ 处理选项</h3>
                    <div class="option-group">
                        <label class="checkbox-option">
                            <input type="checkbox" id="enableAI" checked>
                            <span class="checkmark"></span>
                            启用AI智能解析
                        </label>
                        <small class="option-help">使用Dify AI服务智能分析和解析剪贴板内容</small>
                    </div>

                    <div class="option-group">
                        <label for="targetList">🎯 目标任务列表:</label>
                        <select id="targetList" class="select-input">
                            <option value="">使用默认列表</option>
                            <option value="Tasks">Tasks</option>
                            <option value="Work">工作</option>
                            <option value="Personal">个人</option>
                        </select>
                        <small class="option-help">选择任务要添加到的列表</small>
                    </div>
                </div>

                <!-- 处理和上传区域 -->
                <div class="content-section" id="uploadSection" style="display: none;">
                    <h3>🚀 处理并上传</h3>
                    <button class="btn btn-success" id="processBtn" disabled>
                        <span class="btn-icon">⬆️</span>
                        处理并上传到 Microsoft Todo
                    </button>
                    <div class="processing-status" id="processingStatus" style="display: none;">
                        <div class="progress-bar">
                            <div class="progress-fill" id="progressFill"></div>
                        </div>
                        <div class="progress-text" id="progressText">处理中...</div>
                    </div>
                </div>

                <!-- 结果展示区域 -->
                <div class="content-section" id="resultSection" style="display: none;">
                    <h3>📄 处理结果</h3>
                    <div class="result-content" id="resultContent"></div>
                </div>
            </div>
        `;

        this.getClipboardBtn = this.container.querySelector('#getClipboardBtn');
        this.clipboardStatus = this.container.querySelector('#clipboardStatus');
        this.previewSection = this.container.querySelector('#previewSection');
        this.contentPreview = this.container.querySelector('#contentPreview');
        this.optionsSection = this.container.querySelector('#optionsSection');
        this.enableAI = this.container.querySelector('#enableAI');
        this.targetList = this.container.querySelector('#targetList');
        this.uploadSection = this.container.querySelector('#uploadSection');
        this.processBtn = this.container.querySelector('#processBtn');
        this.processingStatus = this.container.querySelector('#processingStatus');
        this.progressFill = this.container.querySelector('#progressFill');
        this.progressText = this.container.querySelector('#progressText');
        this.resultSection = this.container.querySelector('#resultSection');
        this.resultContent = this.container.querySelector('#resultContent');
    }

    bindEvents() {
        this.getClipboardBtn.addEventListener('click', () => {
            this.getClipboardContent();
        });

        this.processBtn.addEventListener('click', () => {
            this.processAndUpload();
        });

        this.enableAI.addEventListener('change', () => {
            this.validateUploadButton();
        });

        // 监听内容变化，动态更新按钮状态
        this.container.addEventListener('contentChanged', () => {
            this.validateUploadButton();
        });
    }

    setupStateListener() {
        appState.subscribe('module:clipboard', (state) => {
            this.updateUI(state);
        });
    }

    async getClipboardContent() {
        try {
            this.setStatus('正在获取剪贴板内容...', STATUS.PROCESSING);
            this.getClipboardBtn.disabled = true;
            this.getClipboardBtn.innerHTML = '<span class="btn-icon">⏳</span> 获取中...';

            // 调用后端API获取剪贴板内容
            // 这里需要等待后端实现 GetClipboardContent 方法
            const response = await window.backend.GetClipboardContent();

            if (response.success) {
                this.clipboardContent = response.content;
                this.showContentPreview(this.clipboardContent);
                this.showOptionsAndUpload();
                this.setStatus(`已获取${response.content.type === 'text' ? '文本' : '图片'}内容`, STATUS.SUCCESS);
                this.logPanel.info(`成功获取剪贴板${response.content.type === 'text' ? '文本' : '图片'}内容`, 'clipboard');
            } else {
                throw new Error(response.error || '获取剪贴板内容失败');
            }
        } catch (error) {
            this.logPanel.error(`获取剪贴板内容失败: ${error.message}`, 'clipboard');
            this.setStatus(`获取失败: ${error.message}`, STATUS.ERROR);
        } finally {
            this.getClipboardBtn.disabled = false;
            this.getClipboardBtn.innerHTML = '<span class="btn-icon">📋</span> 获取剪贴板内容';
        }
    }

    showContentPreview(content) {
        this.previewSection.style.display = 'block';

        if (content.type === 'text') {
            this.contentPreview.innerHTML = `
                <div class="text-preview">
                    <div class="preview-header">
                        <span class="content-type">📝 文本内容</span>
                        <span class="content-length">${content.text.length} 字符</span>
                    </div>
                    <div class="text-content">${this.escapeHtml(content.text)}</div>
                </div>
            `;
        } else if (content.type === 'image') {
            this.contentPreview.innerHTML = `
                <div class="image-preview">
                    <div class="preview-header">
                        <span class="content-type">🖼️ 图片内容</span>
                        <span class="content-info">${content.width}×${content.height} | ${this.formatFileSize(content.size)}</span>
                    </div>
                    <div class="image-container">
                        <img src="${content.dataUrl}" alt="剪贴板图片" class="preview-image">
                    </div>
                </div>
            `;
        }

        // 通知其他组件内容已变化
        this.container.dispatchEvent(new CustomEvent('contentChanged', {
            detail: { content: content }
        }));
    }

    showOptionsAndUpload() {
        this.optionsSection.style.display = 'block';
        this.uploadSection.style.display = 'block';
        this.validateUploadButton();
    }

    validateUploadButton() {
        const hasContent = this.clipboardContent !== null;
        this.processBtn.disabled = !hasContent;
    }

    async processAndUpload() {
        if (!this.clipboardContent) {
            this.logPanel.error('没有可处理的内容', 'clipboard');
            return;
        }

        try {
            this.setProcessingStatus(true);
            this.processBtn.disabled = true;

            const options = {
                enableAI: this.enableAI.checked,
                targetList: this.targetList.value || null
            };

            this.updateProgress(10, '准备处理...');

            // 调用后端API处理剪贴板内容
            // 这里需要等待后端实现 ProcessClipboard 方法
            const response = await window.backend.ProcessClipboard(this.clipboardContent, options.enableAI, options.targetList);

            if (response.success) {
                this.updateProgress(100, '上传成功!');
                this.showResult(response.result, true);
                this.logPanel.success(`成功上传到Microsoft Todo: ${response.result.title}`, 'clipboard');
                this.statusBar.showModuleStatus('clipboard', STATUS.SUCCESS, '上传成功');
            } else {
                throw new Error(response.error || '处理失败');
            }
        } catch (error) {
            this.logPanel.error(`处理失败: ${error.message}`, 'clipboard');
            this.showResult({ error: error.message }, false);
            this.statusBar.showModuleStatus('clipboard', STATUS.ERROR, '处理失败');
        } finally {
            this.setProcessingStatus(false);
            this.processBtn.disabled = false;
        }
    }

    setProcessingStatus(isProcessing) {
        if (isProcessing) {
            this.processingStatus.style.display = 'block';
            this.updateProgress(0, '处理中...');
        } else {
            this.processingStatus.style.display = 'none';
        }
    }

    updateProgress(percent, text) {
        this.progressFill.style.width = `${percent}%`;
        this.progressText.textContent = text;
    }

    showResult(result, success) {
        this.resultSection.style.display = 'block';

        if (success) {
            this.resultContent.innerHTML = `
                <div class="result-success">
                    <h4>✅ 上传成功</h4>
                    <div class="task-details">
                        <p><strong>任务标题:</strong> ${result.title}</p>
                        <p><strong>任务描述:</strong> ${result.description || '无'}</p>
                        <p><strong>目标任务列表:</strong> ${result.targetList || '默认'}</p>
                        <p><strong>AI解析:</strong> ${result.aiProcessed ? '已启用' : '未启用'}</p>
                        <p><strong>创建时间:</strong> ${new Date(result.createdAt).toLocaleString()}</p>
                    </div>
                </div>
            `;
        } else {
            this.resultContent.innerHTML = `
                <div class="result-error">
                    <h4>❌ 上传失败</h4>
                    <p>${result.error}</p>
                </div>
            `;
        }
    }

    setStatus(message, status) {
        this.clipboardStatus.textContent = message;
        this.clipboardStatus.className = `clipboard-status status-${status}`;
    }

    updateUI(state) {
        // 根据状态更新UI
        if (state.processing) {
            this.setProcessingStatus(true);
        } else {
            this.setProcessingStatus(false);
        }
    }

    // 工具方法
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }
}