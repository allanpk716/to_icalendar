import { appState, STATUS } from '../state.js';

export class TestModule {
    constructor(container, logPanel, statusBar) {
        this.container = container;
        this.logPanel = logPanel;
        this.statusBar = statusBar;
        this.testResults = [];
        this.init();
    }

    init() {
        this.render();
        this.bindEvents();
        this.setupStateListener();
    }

    render() {
        this.container.innerHTML = `
            <div class="module-content test-module">
                <h2>🔍 系统测试</h2>
                <p class="module-description">测试系统连接和配置状态</p>

                <!-- 测试控制区域 -->
                <div class="content-section">
                    <h3>🧪 执行测试</h3>
                    <button class="btn btn-primary" id="runAllTestsBtn">
                        <span class="btn-icon">🚀</span>
                        运行所有测试
                    </button>
                    <div class="test-status" id="testStatus">点击按钮开始系统测试</div>
                </div>

                <!-- 测试项目列表 -->
                <div class="content-section">
                    <h3>📋 测试项目</h3>
                    <div class="test-items">
                        <div class="test-item" data-test="config">
                            <div class="test-item-header">
                                <span class="test-icon" id="configIcon">⏸️</span>
                                <span class="test-name">配置文件验证</span>
                                <span class="test-status" id="configStatus">待测试</span>
                            </div>
                            <div class="test-details" id="configDetails"></div>
                        </div>

                        <div class="test-item" data-test="microsoft">
                            <div class="test-item-header">
                                <span class="test-icon" id="microsoftIcon">⏸️</span>
                                <span class="test-name">Microsoft Todo 服务测试</span>
                                <span class="test-status" id="microsoftStatus">待测试</span>
                            </div>
                            <div class="test-details" id="microsoftDetails"></div>
                        </div>

                        <div class="test-item" data-test="dify">
                            <div class="test-item-header">
                                <span class="test-icon" id="difyIcon">⏸️</span>
                                <span class="test-name">Dify AI 服务测试</span>
                                <span class="test-status" id="difyStatus">待测试</span>
                            </div>
                            <div class="test-details" id="difyDetails"></div>
                        </div>

                        <div class="test-item" data-test="permissions">
                            <div class="test-item-header">
                                <span class="test-icon" id="permissionsIcon">⏸️</span>
                                <span class="test-name">API 权限验证</span>
                                <span class="test-status" id="permissionsStatus">待测试</span>
                            </div>
                            <div class="test-details" id="permissionsDetails"></div>
                        </div>
                    </div>
                </div>

                <!-- 测试进度 -->
                <div class="content-section" id="progressSection" style="display: none;">
                    <h3>📊 测试进度</h3>
                    <div class="test-progress">
                        <div class="progress-bar">
                            <div class="progress-fill" id="testProgressFill"></div>
                        </div>
                        <div class="progress-info">
                            <span class="progress-text" id="testProgressText">准备测试...</span>
                            <span class="progress-count" id="testProgressCount">0/4</span>
                        </div>
                    </div>
                </div>

                <!-- 测试报告 -->
                <div class="content-section" id="reportSection" style="display: none;">
                    <h3>📄 测试报告</h3>
                    <div class="test-report" id="testReport"></div>
                </div>
            </div>
        `;

        this.runAllTestsBtn = this.container.querySelector('#runAllTestsBtn');
        this.testStatus = this.container.querySelector('#testStatus');
        this.progressSection = this.container.querySelector('#progressSection');
        this.testProgressFill = this.container.querySelector('#testProgressFill');
        this.testProgressText = this.container.querySelector('#testProgressText');
        this.testProgressCount = this.container.querySelector('#testProgressCount');
        this.reportSection = this.container.querySelector('#reportSection');
        this.testReport = this.container.querySelector('#testReport');

        this.testItems = {
            config: {
                icon: this.container.querySelector('#configIcon'),
                status: this.container.querySelector('#configStatus'),
                details: this.container.querySelector('#configDetails')
            },
            microsoft: {
                icon: this.container.querySelector('#microsoftIcon'),
                status: this.container.querySelector('#microsoftStatus'),
                details: this.container.querySelector('#microsoftDetails')
            },
            dify: {
                icon: this.container.querySelector('#difyIcon'),
                status: this.container.querySelector('#difyStatus'),
                details: this.container.querySelector('#difyDetails')
            },
            permissions: {
                icon: this.container.querySelector('#permissionsIcon'),
                status: this.container.querySelector('#permissionsStatus'),
                details: this.container.querySelector('#permissionsDetails')
            }
        };
    }

    bindEvents() {
        this.runAllTestsBtn.addEventListener('click', () => {
            this.runAllTests();
        });

        // 单个测试项点击事件（可选：允许单独运行测试）
        Object.keys(this.testItems).forEach(testKey => {
            const testItem = this.container.querySelector(`[data-test="${testKey}"]`);
            testItem.addEventListener('click', () => {
                this.runSingleTest(testKey);
            });
        });
    }

    setupStateListener() {
        appState.subscribe('module:test', (state) => {
            this.updateUI(state);
        });
    }

    async runAllTests() {
        try {
            this.setTestRunning(true);
            this.resetAllTests();
            this.progressSection.style.display = 'block';
            this.reportSection.style.display = 'none';

            this.testStatus.textContent = '正在运行系统测试...';
            this.logPanel.info('开始运行系统测试', 'test');

            const testOrder = ['config', 'microsoft', 'permissions', 'dify'];
            const results = [];

            for (let i = 0; i < testOrder.length; i++) {
                const testKey = testOrder[i];
                this.updateProgress((i + 1) / testOrder.length * 100, `正在测试: ${this.getTestName(testKey)}`);
                this.updateProgressCount(i + 1, testOrder.length);

                const result = await this.runSingleTest(testKey);
                results.push(result);

                // 如果配置测试失败，停止后续测试
                if (testKey === 'config' && !result.success) {
                    this.logPanel.error('配置测试失败，停止后续测试', 'test');
                    break;
                }

                // 添加小延迟，让用户看到进度
                await this.delay(500);
            }

            this.showTestReport(results);
            this.setTestRunning(false);

            const successCount = results.filter(r => r.success).length;
            const totalCount = results.length;

            if (successCount === totalCount) {
                this.testStatus.textContent = `✅ 所有测试通过 (${successCount}/${totalCount})`;
                this.statusBar.showModuleStatus('test', STATUS.SUCCESS, '所有测试通过');
                this.logPanel.success(`所有测试通过 (${successCount}/${totalCount})`, 'test');
            } else {
                this.testStatus.textContent = `⚠️ 部分测试失败 (${successCount}/${totalCount})`;
                this.statusBar.showModuleStatus('test', STATUS.ERROR, '部分测试失败');
                this.logPanel.warn(`部分测试失败 (${successCount}/${totalCount})`, 'test');
            }

        } catch (error) {
            this.logPanel.error(`测试运行异常: ${error.message}`, 'test');
            this.testStatus.textContent = `❌ 测试异常: ${error.message}`;
            this.setTestRunning(false);
        }
    }

    async runSingleTest(testKey) {
        const testItem = this.testItems[testKey];

        try {
            this.updateTestStatus(testKey, 'running');
            this.logPanel.info(`开始测试: ${this.getTestName(testKey)}`, 'test');

            let result;
            switch (testKey) {
                case 'config':
                    result = await this.testConfig();
                    break;
                case 'microsoft':
                    result = await this.testMicrosoftTodo();
                    break;
                case 'dify':
                    result = await this.testDify();
                    break;
                case 'permissions':
                    result = await this.testPermissions();
                    break;
                default:
                    throw new Error(`未知的测试类型: ${testKey}`);
            }

            this.updateTestResult(testKey, result);
            return result;

        } catch (error) {
            const errorResult = {
                test: testKey,
                success: false,
                error: error.message,
                duration: 0
            };
            this.updateTestResult(testKey, errorResult);
            return errorResult;
        }
    }

    async testConfig() {
        const startTime = Date.now();

        // 调用后端API测试配置文件
        // 这里需要等待后端实现
        try {
            const response = await window.backend.TestConfigFile();
            const duration = Date.now() - startTime;

            return {
                test: 'config',
                success: response.success,
                message: response.message || (response.success ? '配置文件验证通过' : '配置文件验证失败'),
                details: response.details || {},
                duration: duration
            };
        } catch (error) {
            return {
                test: 'config',
                success: false,
                message: `配置文件测试失败: ${error.message}`,
                details: { error: error.message },
                duration: Date.now() - startTime
            };
        }
    }

    async testMicrosoftTodo() {
        const startTime = Date.now();

        try {
            const response = await window.backend.TestMicrosoftTodo();
            const duration = Date.now() - startTime;

            return {
                test: 'microsoft',
                success: response.success,
                message: response.message || (response.success ? 'Microsoft Todo 服务连接正常' : 'Microsoft Todo 服务连接失败'),
                details: response.details || {},
                duration: duration
            };
        } catch (error) {
            return {
                test: 'microsoft',
                success: false,
                message: `Microsoft Todo 测试失败: ${error.message}`,
                details: { error: error.message },
                duration: Date.now() - startTime
            };
        }
    }

    async testDify() {
        const startTime = Date.now();

        try {
            const response = await window.backend.TestDifyService();
            const duration = Date.now() - startTime;

            return {
                test: 'dify',
                success: response.success,
                message: response.message || (response.success ? 'Dify AI 服务连接正常' : 'Dify AI 服务连接失败'),
                details: response.details || {},
                duration: duration
            };
        } catch (error) {
            return {
                test: 'dify',
                success: false,
                message: `Dify AI 测试失败: ${error.message}`,
                details: { error: error.message },
                duration: Date.now() - startTime
            };
        }
    }

    async testPermissions() {
        const startTime = Date.now();

        try {
            const response = await window.backend.TestAPIPermissions();
            const duration = Date.now() - startTime;

            return {
                test: 'permissions',
                success: response.success,
                message: response.message || (response.success ? 'API 权限验证通过' : 'API 权限验证失败'),
                details: response.details || {},
                duration: duration
            };
        } catch (error) {
            return {
                test: 'permissions',
                success: false,
                message: `权限测试失败: ${error.message}`,
                details: { error: error.message },
                duration: Date.now() - startTime
            };
        }
    }

    updateTestStatus(testKey, status) {
        const testItem = this.testItems[testKey];

        switch (status) {
            case 'running':
                testItem.icon.textContent = '⏳';
                testItem.status.textContent = '测试中...';
                testItem.status.className = 'test-status status-running';
                break;
            case 'success':
                testItem.icon.textContent = '✅';
                testItem.status.textContent = '成功';
                testItem.status.className = 'test-status status-success';
                break;
            case 'error':
                testItem.icon.textContent = '❌';
                testItem.status.textContent = '失败';
                testItem.status.className = 'test-status status-error';
                break;
            case 'warning':
                testItem.icon.textContent = '⚠️';
                testItem.status.textContent = '警告';
                testItem.status.className = 'test-status status-warning';
                break;
        }
    }

    updateTestResult(testKey, result) {
        this.updateTestStatus(testKey, result.success ? 'success' : 'error');

        const testItem = this.testItems[testKey];
        if (result.details) {
            testItem.details.innerHTML = this.formatTestDetails(result);
        }

        this.testResults[testKey] = result;
    }

    formatTestDetails(result) {
        let html = `<div class="test-result-details">`;
        html += `<p><strong>状态:</strong> ${result.success ? '✅ 成功' : '❌ 失败'}</p>`;
        html += `<p><strong>耗时:</strong> ${result.duration}ms</p>`;
        html += `<p><strong>消息:</strong> ${result.message}</p>`;

        if (result.details && Object.keys(result.details).length > 0) {
            html += `<div class="test-details-content">`;
            Object.entries(result.details).forEach(([key, value]) => {
                html += `<p><strong>${key}:</strong> ${value}</p>`;
            });
            html += `</div>`;
        }

        html += `</div>`;
        return html;
    }

    showTestReport(results) {
        this.reportSection.style.display = 'block';

        const successCount = results.filter(r => r.success).length;
        const totalCount = results.length;
        const totalDuration = results.reduce((sum, r) => sum + (r.duration || 0), 0);

        let reportHTML = `
            <div class="test-summary">
                <div class="summary-stats">
                    <div class="stat-item ${successCount === totalCount ? 'success' : 'warning'}">
                        <span class="stat-value">${successCount}/${totalCount}</span>
                        <span class="stat-label">测试通过</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-value">${totalDuration}ms</span>
                        <span class="stat-label">总耗时</span>
                    </div>
                </div>
            </div>

            <div class="test-list">
                <h4>详细结果</h4>
        `;

        results.forEach(result => {
            reportHTML += `
                <div class="test-result-item ${result.success ? 'success' : 'error'}">
                    <h5>${this.getTestName(result.test)}</h5>
                    <p class="result-message">${result.message}</p>
                    <p class="result-duration">耗时: ${result.duration}ms</p>
                </div>
            `;
        });

        reportHTML += `</div>`;
        this.testReport.innerHTML = reportHTML;
    }

    setTestRunning(running) {
        this.runAllTestsBtn.disabled = running;
        if (running) {
            this.runAllTestsBtn.innerHTML = '<span class="btn-icon">⏳</span> 测试中...';
        } else {
            this.runAllTestsBtn.innerHTML = '<span class="btn-icon">🚀</span> 运行所有测试';
        }
    }

    resetAllTests() {
        Object.keys(this.testItems).forEach(testKey => {
            this.updateTestStatus(testKey, 'idle');
            this.testItems[testKey].details.innerHTML = '';
        });
        this.testResults = {};
    }

    updateProgress(percent, text) {
        this.testProgressFill.style.width = `${percent}%`;
        this.testProgressText.textContent = text;
    }

    updateProgressCount(current, total) {
        this.testProgressCount.textContent = `${current}/${total}`;
    }

    getTestName(testKey) {
        const names = {
            config: '配置文件验证',
            microsoft: 'Microsoft Todo 服务测试',
            dify: 'Dify AI 服务测试',
            permissions: 'API 权限验证'
        };
        return names[testKey] || testKey;
    }

    updateUI(state) {
        // 根据状态更新UI
        if (state.running) {
            this.setTestRunning(true);
        } else {
            this.setTestRunning(false);
        }
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}