import { appState, STATUS } from '../state.js';

export class InitModule {
    constructor(container, logPanel, statusBar) {
        this.container = container;
        this.logPanel = logPanel;
        this.statusBar = statusBar;
        this.isInitializing = false;
        this.init();
    }

    init() {
        this.render();
        this.bindEvents();
        this.setupStateListener();
        this.setupEventListeners();
    }

    render() {
        this.container.innerHTML = `
            <div class="module-content init-module">
                <h2>⚙️ 配置初始化</h2>
                <p class="module-description">首次使用需要初始化配置文件以连接 Microsoft Todo 服务</p>

                <!-- 初始化状态 -->
                <div class="content-section">
                    <div class="init-status" id="initStatusCard">
                        <div class="status-icon">⏸️</div>
                        <div class="status-content">
                            <h3>配置状态检查</h3>
                            <p id="configStatusText">正在检查配置文件...</p>
                            <div class="status-details" id="statusDetails"></div>
                        </div>
                    </div>
                </div>

                <!-- 初始化按钮区域 -->
                <div class="content-section" id="initSection">
                    <button class="btn btn-primary btn-large" id="initBtn">
                        <span class="btn-icon">🚀</span>
                        初始化配置文件
                    </button>
                    <div class="init-description">
                        <p>这将创建必要的配置文件和目录结构</p>
                    </div>
                </div>

                <!-- 日志显示区域 -->
                <div class="content-section" id="logSection" style="display: none;">
                    <h3>📝 初始化日志</h3>
                    <div class="log-container" id="initLogContainer"></div>
                </div>

                <!-- 结果展示区域 -->
                <div class="content-section" id="resultSection" style="display: none;">
                    <h3>✅ 初始化结果</h3>
                    <div class="result-content" id="resultContent"></div>

                    <div class="next-actions" id="nextActions">
                        <h4>🎯 下一步操作</h4>
                        <div class="action-steps">
                            <div class="step">
                                <span class="step-number">1</span>
                                <div class="step-content">
                                    <strong>编辑配置文件</strong>
                                    <p>修改 server.yaml 文件中的 Microsoft Todo 配置信息</p>
                                    <button class="btn btn-secondary" id="openConfigBtn">
                                        <span class="btn-icon">📁</span>
                                        打开配置目录
                                    </button>
                                </div>
                            </div>

                            <div class="step">
                                <span class="step-number">2</span>
                                <div class="step-content">
                                    <strong>获取 Azure AD 信息</strong>
                                    <p>在 Azure Portal 中获取租户ID、客户端ID和密钥</p>
                                </div>
                            </div>

                            <div class="step">
                                <span class="step-number">3</span>
                                <div class="step-content">
                                    <strong>测试连接</strong>
                                    <p>配置完成后，使用测试功能验证连接</p>
                                    <button class="btn btn-secondary" id="goToTestBtn">
                                        <span class="btn-icon">🔍</span>
                                        前往测试
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 配置文件路径显示 -->
                <div class="content-section" id="configPathSection" style="display: none;">
                    <h3>📁 配置文件路径</h3>
                    <div class="config-path-info">
                        <div class="path-item">
                            <strong>配置目录:</strong>
                            <code id="configDirPath">~/.to_icalendar</code>
                        </div>
                        <div class="path-item">
                            <strong>主配置文件:</strong>
                            <code id="configFilePath">server.yaml</code>
                        </div>
                    </div>
                </div>
            </div>
        `;

        this.initBtn = this.container.querySelector('#initBtn');
        this.initStatusCard = this.container.querySelector('#initStatusCard');
        this.configStatusText = this.container.querySelector('#configStatusText');
        this.statusDetails = this.container.querySelector('#statusDetails');
        this.initSection = this.container.querySelector('#initSection');
        this.logSection = this.container.querySelector('#logSection');
        this.initLogContainer = this.container.querySelector('#initLogContainer');
        this.resultSection = this.container.querySelector('#resultSection');
        this.resultContent = this.container.querySelector('#resultContent');
        this.nextActions = this.container.querySelector('#nextActions');
        this.configPathSection = this.container.querySelector('#configPathSection');
        this.configDirPath = this.container.querySelector('#configDirPath');
        this.configFilePath = this.container.querySelector('#configFilePath');
        this.openConfigBtn = this.container.querySelector('#openConfigBtn');
        this.goToTestBtn = this.container.querySelector('#goToTestBtn');
    }

    bindEvents() {
        this.initBtn.addEventListener('click', () => {
            this.startInitialization();
        });

        this.openConfigBtn.addEventListener('click', () => {
            this.openConfigDirectory();
        });

        this.goToTestBtn.addEventListener('click', () => {
            // 切换到测试标签
            this.container.dispatchEvent(new CustomEvent('switchTab', {
                detail: { tabId: 'test' }
            }));
        });
    }

    setupStateListener() {
        appState.subscribe('module:init', (state) => {
            this.updateUI(state);
        });
    }

    setupEventListeners() {
        // 监听后端日志事件
        if (window.runtime && window.runtime.EventsOn) {
            window.runtime.EventsOn("initLog", (logMessage) => {
                this.appendLog(logMessage.type, logMessage.message);
            });

            window.runtime.EventsOn("initResult", (result) => {
                this.handleInitResult(result);
            });
        }
    }

    async checkConfigStatus() {
        try {
            // 检查配置文件状态
            const response = await window.backend.CheckConfigStatus();

            if (response.success) {
                this.updateConfigStatus(response);

                if (response.exists && response.valid) {
                    this.configStatusText.textContent = '✅ 配置文件已存在且有效';
                    this.initStatusCard.querySelector('.status-icon').textContent = '✅';
                    this.showConfigPaths(response);
                    this.hideInitButton();
                } else if (response.exists && !response.valid) {
                    this.configStatusText.textContent = '⚠️ 配置文件存在但格式有误';
                    this.initStatusCard.querySelector('.status-icon').textContent = '⚠️';
                    this.statusDetails.innerHTML = `<p class="error-detail">${response.error}</p>`;
                    this.showConfigPaths(response);
                    this.showRecreateOption();
                } else {
                    this.configStatusText.textContent = '❌ 配置文件不存在';
                    this.initStatusCard.querySelector('.status-icon').textContent = '❌';
                    this.showInitButton();
                }
            }
        } catch (error) {
            this.configStatusText.textContent = '❌ 检查配置状态失败';
            this.statusDetails.innerHTML = `<p class="error-detail">${error.message}</p>`;
            this.logPanel.error(`检查配置状态失败: ${error.message}`, 'init');
        }
    }

    updateConfigStatus(statusInfo) {
        if (statusInfo.configDir) {
            this.configDirPath.textContent = statusInfo.configDir;
        }
        if (statusInfo.configFile) {
            this.configFilePath.textContent = statusInfo.configFile;
        }
    }

    showConfigPaths(statusInfo) {
        this.configPathSection.style.display = 'block';
    }

    hideInitButton() {
        this.initSection.style.display = 'none';
    }

    showInitButton() {
        this.initSection.style.display = 'block';
    }

    showRecreateOption() {
        this.initBtn.innerHTML = '<span class="btn-icon">🔄</span> 重新创建配置文件';
        this.initSection.style.display = 'block';
    }

    async startInitialization() {
        if (this.isInitializing) return;

        try {
            this.isInitializing = true;
            appState.setModuleState('init', { running: true });

            this.initBtn.disabled = true;
            this.initBtn.innerHTML = '<span class="btn-icon">⏳</span> 正在初始化...';

            this.logSection.style.display = 'block';
            this.resultSection.style.display = 'none';
            this.clearLogs();

            this.logPanel.info('开始初始化配置文件', 'init');
            this.statusBar.showModuleStatus('init', STATUS.PROCESSING, '正在初始化配置...');

            // 滚动到日志区域
            this.logSection.scrollIntoView({ behavior: 'smooth' });

            // 调用后端初始化方法
            await window.backend.InitConfigWithStreaming();

        } catch (error) {
            this.appendLog('error', `初始化异常: ${error.message}`);
            this.logPanel.error(`初始化异常: ${error.message}`, 'init');
            this.statusBar.showModuleStatus('init', STATUS.ERROR, '初始化失败');
            this.resetInitButton();
            appState.setModuleState('init', { running: false });
        }
    }

    handleInitResult(result) {
        this.isInitializing = false;
        this.resetInitButton();
        appState.setModuleState('init', { running: false, completed: true });

        if (result.success) {
            this.showSuccessResult(result);
            this.logPanel.success('配置文件初始化成功', 'init');
            this.statusBar.showModuleStatus('init', STATUS.SUCCESS, '初始化成功');
        } else {
            this.showErrorResult(result);
            this.logPanel.error(`初始化失败: ${result.message}`, 'init');
            this.statusBar.showModuleStatus('init', STATUS.ERROR, '初始化失败');
        }
    }

    showSuccessResult(result) {
        this.resultSection.style.display = 'block';
        this.configPathSection.style.display = 'block';

        this.resultContent.innerHTML = `
            <div class="result-success">
                <h4>🎉 初始化成功</h4>
                <p><strong>${result.message}</strong></p>
                <div class="success-details">
                    <p><strong>配置目录:</strong> <code>${result.configDir}</code></p>
                    <p><strong>配置文件:</strong> <code>${result.serverConfig}</code></p>
                </div>
            </div>
        `;

        if (result.configDir) {
            this.configDirPath.textContent = result.configDir;
        }
        if (result.serverConfig) {
            this.configFilePath.textContent = result.serverConfig;
        }

        this.hideInitButton();
    }

    showErrorResult(result) {
        this.resultSection.style.display = 'block';

        this.resultContent.innerHTML = `
            <div class="result-error">
                <h4>❌ 初始化失败</h4>
                <p>${result.message}</p>
                ${result.error ? `<p class="error-detail">${result.error}</p>` : ''}
            </div>
        `;
    }

    resetInitButton() {
        this.initBtn.disabled = false;
        this.initBtn.innerHTML = '<span class="btn-icon">🚀</span> 初始化配置文件';
    }

    appendLog(type, message) {
        const logEntry = document.createElement('div');
        logEntry.className = `log-entry log-${type}`;

        const timestamp = new Date().toLocaleTimeString();
        logEntry.innerHTML = `<span class="log-time">[${timestamp}]</span> ${message}`;

        this.initLogContainer.appendChild(logEntry);
        this.initLogContainer.scrollTop = this.initLogContainer.scrollHeight;
    }

    clearLogs() {
        this.initLogContainer.innerHTML = '';
    }

    async openConfigDirectory() {
        try {
            const response = await window.backend.OpenConfigDirectory();
            if (!response.success) {
                throw new Error(response.error || '打开配置目录失败');
            }
        } catch (error) {
            this.logPanel.error(`打开配置目录失败: ${error.message}`, 'init');
            alert(`打开配置目录失败: ${error.message}`);
        }
    }

    updateUI(state) {
        // 根据状态更新UI
        if (state.running) {
            this.setInitRunning(true);
        } else {
            this.setInitRunning(false);
        }
    }

    setInitRunning(running) {
        this.initBtn.disabled = running;
        if (running) {
            this.initBtn.innerHTML = '<span class="btn-icon">⏳</span> 正在初始化...';
        } else {
            this.initBtn.innerHTML = '<span class="btn-icon">🚀</span> 初始化配置文件';
        }
    }
}