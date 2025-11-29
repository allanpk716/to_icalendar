import { appState, STATUS } from '../state.js';

export class CleanModule {
    constructor(container, logPanel, statusBar) {
        this.container = container;
        this.logPanel = logPanel;
        this.statusBar = statusBar;
        this.scanResults = null;
        this.cleanResults = null;
        this.init();
    }

    init() {
        this.render();
        this.bindEvents();
        this.setupStateListener();
    }

    render() {
        this.container.innerHTML = `
            <div class="module-content clean-module">
                <h2>🧹 缓存清理</h2>
                <p class="module-description">清理应用缓存文件和临时数据</p>

                <!-- 清理类型选择 -->
                <div class="content-section">
                    <h3>🗂️ 清理类型</h3>
                    <div class="clean-options">
                        <label class="checkbox-option">
                            <input type="checkbox" id="cleanTasks" checked>
                            <span class="checkmark"></span>
                            <span class="option-label">
                                <strong>任务缓存</strong>
                                <small>任务去重缓存数据</small>
                            </span>
                        </label>

                        <label class="checkbox-option">
                            <input type="checkbox" id="cleanImages" checked>
                            <span class="checkmark"></span>
                            <span class="option-label">
                                <strong>图片缓存</strong>
                                <small>剪贴板图片缓存</small>
                            </span>
                        </label>

                        <label class="checkbox-option">
                            <input type="checkbox" id="cleanImageHashes">
                            <span class="checkmark"></span>
                            <span class="option-label">
                                <strong>图片哈希</strong>
                                <small>图片去重哈希数据</small>
                            </span>
                        </label>

                        <label class="checkbox-option">
                            <input type="checkbox" id="cleanTemp">
                            <span class="checkmark"></span>
                            <span class="option-label">
                                <strong>临时文件</strong>
                                <small>应用临时文件</small>
                            </span>
                        </label>

                        <label class="checkbox-option">
                            <input type="checkbox" id="cleanGenerated">
                            <span class="checkmark"></span>
                            <span class="option-label">
                                <strong>生成文件</strong>
                                <small>AI生成的JSON文件</small>
                            </span>
                        </label>
                    </div>
                </div>

                <!-- 时间过滤选项 -->
                <div class="content-section">
                    <h3>⏰ 时间过滤</h3>
                    <div class="time-filter">
                        <label class="radio-option">
                            <input type="radio" name="timeFilter" value="all" checked>
                            <span class="radio-mark"></span>
                            <span class="option-label">所有时间</span>
                        </label>

                        <label class="radio-option">
                            <input type="radio" name="timeFilter" value="7d">
                            <span class="radio-mark"></span>
                            <span class="option-label">7天前</span>
                        </label>

                        <label class="radio-option">
                            <input type="radio" name="timeFilter" value="30d">
                            <span class="radio-mark"></span>
                            <span class="option-label">30天前</span>
                        </label>

                        <label class="radio-option">
                            <input type="radio" name="timeFilter" value="90d">
                            <span class="radio-mark"></span>
                            <span class="option-label">90天前</span>
                        </label>

                        <label class="radio-option">
                            <input type="radio" name="timeFilter" value="custom">
                            <span class="radio-mark"></span>
                            <span class="option-label">
                                自定义:
                                <input type="number" id="customDays" min="1" max="365" value="30" disabled>
                                天
                            </span>
                        </label>
                    </div>
                </div>

                <!-- 操作模式 -->
                <div class="content-section">
                    <h3>⚙️ 操作模式</h3>
                    <div class="operation-mode">
                        <label class="radio-option">
                            <input type="radio" name="operationMode" value="preview" checked>
                            <span class="radio-mark"></span>
                            <span class="option-label">
                                <strong>预览模式</strong>
                                <small>仅查看将要清理的文件，不实际删除</small>
                            </span>
                        </label>

                        <label class="radio-option">
                            <input type="radio" name="operationMode" value="clean">
                            <span class="radio-mark"></span>
                            <span class="option-label">
                                <strong>执行模式</strong>
                                <small>实际删除选定的文件（不可撤销）</small>
                            </span>
                        </label>
                    </div>
                </div>

                <!-- 操作按钮 -->
                <div class="content-section">
                    <h3>🎯 执行操作</h3>
                    <div class="action-buttons">
                        <button class="btn btn-primary" id="scanBtn">
                            <span class="btn-icon">🔍</span>
                            扫描文件
                        </button>
                        <button class="btn btn-success" id="cleanBtn" disabled>
                            <span class="btn-icon">🧹</span>
                            开始清理
                        </button>
                    </div>
                    <div class="clean-status" id="cleanStatus">点击扫描按钮查看将要清理的文件</div>
                </div>

                <!-- 扫描结果 -->
                <div class="content-section" id="scanResultsSection" style="display: none;">
                    <h3>📊 扫描结果</h3>
                    <div class="scan-results" id="scanResults"></div>
                </div>

                <!-- 清理进度 -->
                <div class="content-section" id="progressSection" style="display: none;">
                    <h3>🚀 清理进度</h3>
                    <div class="clean-progress">
                        <div class="progress-bar">
                            <div class="progress-fill" id="cleanProgressFill"></div>
                        </div>
                        <div class="progress-info">
                            <span class="progress-text" id="cleanProgressText">准备清理...</span>
                            <span class="progress-count" id="cleanProgressCount">0/0</span>
                        </div>
                    </div>
                </div>

                <!-- 清理结果 -->
                <div class="content-section" id="cleanResultsSection" style="display: none;">
                    <h3>✅ 清理结果</h3>
                    <div class="clean-results" id="cleanResults"></div>
                </div>
            </div>
        `;

        this.scanBtn = this.container.querySelector('#scanBtn');
        this.cleanBtn = this.container.querySelector('#cleanBtn');
        this.cleanStatus = this.container.querySelector('#cleanStatus');
        this.scanResultsSection = this.container.querySelector('#scanResultsSection');
        this.scanResults = this.container.querySelector('#scanResults');
        this.progressSection = this.container.querySelector('#progressSection');
        this.cleanProgressFill = this.container.querySelector('#cleanProgressFill');
        this.cleanProgressText = this.container.querySelector('#cleanProgressText');
        this.cleanProgressCount = this.container.querySelector('#cleanProgressCount');
        this.cleanResultsSection = this.container.querySelector('#cleanResultsSection');
        this.cleanResults = this.container.querySelector('#cleanResults');

        // 时间过滤相关
        this.customDaysInput = this.container.querySelector('#customDays');
        this.setupTimeFilterEvents();
    }

    setupTimeFilterEvents() {
        const customRadio = this.container.querySelector('input[value="custom"]');
        this.customDaysInput.addEventListener('change', () => {
            if (customRadio.checked) {
                this.validateCustomDays();
            }
        });

        // 监听所有时间过滤选项
        this.container.querySelectorAll('input[name="timeFilter"]').forEach(radio => {
            radio.addEventListener('change', () => {
                const isCustom = radio.value === 'custom';
                this.customDaysInput.disabled = !isCustom;
            });
        });
    }

    validateCustomDays() {
        const value = parseInt(this.customDaysInput.value);
        if (isNaN(value) || value < 1 || value > 365) {
            this.customDaysInput.value = 30;
        }
    }

    bindEvents() {
        this.scanBtn.addEventListener('click', () => {
            this.scanFiles();
        });

        this.cleanBtn.addEventListener('click', () => {
            this.startCleaning();
        });

        // 监听清理选项变化，启用/禁用清理按钮
        this.container.querySelectorAll('.clean-options input').forEach(checkbox => {
            checkbox.addEventListener('change', () => {
                this.validateCleanButton();
            });
        });

        // 监听操作模式变化
        this.container.querySelectorAll('input[name="operationMode"]').forEach(radio => {
            radio.addEventListener('change', () => {
                this.updateButtonText();
            });
        });
    }

    setupStateListener() {
        appState.subscribe('module:clean', (state) => {
            this.updateUI(state);
        });
    }

    validateCleanButton() {
        const hasSelectedTypes = this.getSelectedCleanTypes().length > 0;
        const hasScanResults = this.scanResults !== null;

        this.cleanBtn.disabled = !hasSelectedTypes || !hasScanResults;
    }

    updateButtonText() {
        const operationMode = this.container.querySelector('input[name="operationMode"]:checked').value;
        if (operationMode === 'preview') {
            this.cleanBtn.innerHTML = '<span class="btn-icon">👀</span> 预览清理';
        } else {
            this.cleanBtn.innerHTML = '<span class="btn-icon">🧹</span> 开始清理';
        }
    }

    getSelectedCleanTypes() {
        const types = [];
        if (this.container.querySelector('#cleanTasks').checked) types.push('tasks');
        if (this.container.querySelector('#cleanImages').checked) types.push('images');
        if (this.container.querySelector('#cleanImageHashes').checked) types.push('imageHashes');
        if (this.container.querySelector('#cleanTemp').checked) types.push('temp');
        if (this.container.querySelector('#cleanGenerated').checked) types.push('generated');
        return types;
    }

    getTimeFilter() {
        const selectedRadio = this.container.querySelector('input[name="timeFilter"]:checked');
        if (selectedRadio.value === 'custom') {
            this.validateCustomDays();
            return `${this.customDaysInput.value}d`;
        }
        return selectedRadio.value;
    }

    async scanFiles() {
        try {
            this.setScanRunning(true);
            this.scanResults = null;
            this.cleanResults = null;
            this.cleanBtn.disabled = true;

            const cleanTypes = this.getSelectedCleanTypes();
            const timeFilter = this.getTimeFilter();

            this.cleanStatus.textContent = '正在扫描文件...';
            this.logPanel.info('开始扫描缓存文件', 'clean');

            // 调用后端API扫描文件
            const options = {
                types: cleanTypes,
                timeFilter: timeFilter,
                dryRun: true
            };

            const response = await window.backend.ScanCleanFiles(options);

            if (response.success) {
                this.scanResults = response.results;
                this.showScanResults(response.results);
                this.validateCleanButton();
                this.cleanStatus.textContent = `扫描完成，发现 ${response.results.fileCount} 个文件，占用 ${this.formatFileSize(response.results.totalSize)}`;
                this.logPanel.info(`扫描完成：${response.results.fileCount} 个文件，${this.formatFileSize(response.results.totalSize)}`, 'clean');
            } else {
                throw new Error(response.error || '扫描失败');
            }

        } catch (error) {
            this.logPanel.error(`扫描失败: ${error.message}`, 'clean');
            this.cleanStatus.textContent = `❌ 扫描失败: ${error.message}`;
        } finally {
            this.setScanRunning(false);
        }
    }

    async startCleaning() {
        if (!this.scanResults) {
            this.logPanel.error('请先扫描文件', 'clean');
            return;
        }

        const operationMode = this.container.querySelector('input[name="operationMode"]:checked').value;

        if (operationMode === 'clean') {
            // 执行模式需要确认
            const confirmed = confirm(`确定要删除 ${this.scanResults.fileCount} 个文件吗？\n此操作不可撤销。`);
            if (!confirmed) {
                return;
            }
        }

        try {
            this.setCleanRunning(true);

            const cleanTypes = this.getSelectedCleanTypes();
            const timeFilter = this.getTimeFilter();

            const options = {
                types: cleanTypes,
                timeFilter: timeFilter,
                dryRun: operationMode === 'preview'
            };

            this.cleanStatus.textContent = operationMode === 'preview' ? '预览清理操作...' : '正在清理文件...';
            this.logPanel.info(operationMode === 'preview' ? '开始预览清理操作' : '开始清理文件', 'clean');

            const response = await window.backend.ExecuteClean(options);

            if (response.success) {
                this.cleanResults = response.results;
                this.showCleanResults(response.results, operationMode);

                if (operationMode === 'preview') {
                    this.cleanStatus.textContent = `预览完成：将删除 ${response.results.fileCount} 个文件，释放 ${this.formatFileSize(response.results.totalSize)} 空间`;
                    this.logPanel.info(`预览完成：${response.results.fileCount} 个文件`, 'clean');
                } else {
                    this.cleanStatus.textContent = `清理完成：已删除 ${response.results.fileCount} 个文件，释放 ${this.formatFileSize(response.results.totalSize)} 空间`;
                    this.logPanel.success(`清理完成：${response.results.fileCount} 个文件，释放${this.formatFileSize(response.results.totalSize)}`, 'clean');
                    this.statusBar.showModuleStatus('clean', STATUS.SUCCESS, '清理完成');
                }
            } else {
                throw new Error(response.error || '清理失败');
            }

        } catch (error) {
            this.logPanel.error(`清理失败: ${error.message}`, 'clean');
            this.cleanStatus.textContent = `❌ 清理失败: ${error.message}`;
            this.statusBar.showModuleStatus('clean', STATUS.ERROR, '清理失败');
        } finally {
            this.setCleanRunning(false);
        }
    }

    showScanResults(results) {
        this.scanResultsSection.style.display = 'block';

        let html = `
            <div class="scan-summary">
                <div class="summary-stats">
                    <div class="stat-item">
                        <span class="stat-value">${results.fileCount}</span>
                        <span class="stat-label">文件数量</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-value">${this.formatFileSize(results.totalSize)}</span>
                        <span class="stat-label">占用空间</span>
                    </div>
                </div>
            </div>
        `;

        if (results.byType && Object.keys(results.byType).length > 0) {
            html += '<div class="type-breakdown"><h4>按类型分类</h4>';
            Object.entries(results.byType).forEach(([type, info]) => {
                html += `
                    <div class="type-item">
                        <span class="type-name">${this.getTypeDisplayName(type)}</span>
                        <span class="type-count">${info.count} 个文件</span>
                        <span class="type-size">${this.formatFileSize(info.size)}</span>
                    </div>
                `;
            });
            html += '</div>';
        }

        if (results.files && results.files.length > 0) {
            html += '<div class="file-list"><h4>文件列表</h4>';
            html += '<div class="file-items">';

            const displayFiles = results.files.slice(0, 50); // 最多显示50个文件
            displayFiles.forEach(file => {
                html += `
                    <div class="file-item">
                        <span class="file-name">${file.name}</span>
                        <span class="file-size">${this.formatFileSize(file.size)}</span>
                        <span class="file-date">${new Date(file.modified).toLocaleDateString()}</span>
                    </div>
                `;
            });

            if (results.files.length > 50) {
                html += `<div class="file-more">... 还有 ${results.files.length - 50} 个文件未显示</div>`;
            }

            html += '</div></div>';
        }

        this.scanResults.innerHTML = html;
    }

    showCleanResults(results, operationMode) {
        this.cleanResultsSection.style.display = 'block';

        const isPreview = operationMode === 'preview';

        let html = `
            <div class="clean-summary ${isPreview ? 'preview' : 'success'}">
                <h4>${isPreview ? '预览结果' : '清理结果'}</h4>
                <div class="summary-stats">
                    <div class="stat-item">
                        <span class="stat-value">${results.fileCount}</span>
                        <span class="stat-label">${isPreview ? '将处理' : '已处理'}文件</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-value">${this.formatFileSize(results.totalSize)}</span>
                        <span class="stat-label">${isPreview ? '将释放' : '已释放'}空间</span>
                    </div>
                </div>
            </div>
        `;

        if (results.errors && results.errors.length > 0) {
            html += '<div class="clean-errors"><h4>错误信息</h4>';
            results.errors.forEach(error => {
                html += `<div class="error-item">❌ ${error}</div>`;
            });
            html += '</div>';
        }

        this.cleanResults.innerHTML = html;
    }

    setScanRunning(running) {
        this.scanBtn.disabled = running;
        if (running) {
            this.scanBtn.innerHTML = '<span class="btn-icon">⏳</span> 扫描中...';
        } else {
            this.scanBtn.innerHTML = '<span class="btn-icon">🔍</span> 扫描文件';
        }
    }

    setCleanRunning(running) {
        this.cleanBtn.disabled = running;
        if (running) {
            this.progressSection.style.display = 'block';
        } else {
            this.progressSection.style.display = 'none';
        }
    }

    getTypeDisplayName(type) {
        const names = {
            tasks: '任务缓存',
            images: '图片缓存',
            imageHashes: '图片哈希',
            temp: '临时文件',
            generated: '生成文件'
        };
        return names[type] || type;
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    }

    updateUI(state) {
        // 根据状态更新UI
        if (state.running) {
            this.setCleanRunning(true);
        } else {
            this.setCleanRunning(false);
        }
    }
}