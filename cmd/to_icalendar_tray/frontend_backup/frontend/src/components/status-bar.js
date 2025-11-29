import { appState, STATUS } from '../state.js';

export class StatusBar {
    constructor(container) {
        this.container = container;
        this.statusText = '';
        this.lastOperationTime = null;
        this.init();
    }

    init() {
        this.render();
        this.setupStateListener();
    }

    render() {
        this.container.innerHTML = `
            <div class="status-bar">
                <div class="status-main">
                    <span class="status-icon" id="statusIcon">✅</span>
                    <span class="status-text" id="statusText">应用程序已启动，就绪</span>
                </div>
                <div class="status-info">
                    <span class="last-sync" id="lastSyncTime">尚未执行操作</span>
                </div>
            </div>
        `;
        this.statusIcon = this.container.querySelector('#statusIcon');
        this.statusText = this.container.querySelector('#statusText');
        this.lastSyncTime = this.container.querySelector('#lastSyncTime');
    }

    setupStateListener() {
        appState.subscribe('globalStatus', (status) => {
            this.updateStatus(status);
        });
    }

    updateStatus(status, message = null) {
        this.statusIcon.className = 'status-icon';

        switch (status) {
            case STATUS.READY:
                this.statusIcon.textContent = '✅';
                this.statusText.textContent = message || '就绪';
                break;
            case STATUS.PROCESSING:
                this.statusIcon.textContent = '⏳';
                this.statusText.textContent = message || '处理中...';
                this.statusIcon.className = 'status-icon processing';
                break;
            case STATUS.SUCCESS:
                this.statusIcon.textContent = '✅';
                this.statusText.textContent = message || '操作成功';
                break;
            case STATUS.ERROR:
                this.statusIcon.textContent = '❌';
                this.statusText.textContent = message || '操作失败';
                this.statusIcon.className = 'status-icon error';
                break;
        }

        if (status !== STATUS.READY) {
            this.lastOperationTime = new Date();
            this.updateLastSyncTime();
        }
    }

    updateLastSyncTime() {
        if (this.lastOperationTime) {
            const timeStr = this.lastOperationTime.toLocaleTimeString();
            this.lastSyncTime.textContent = `上次操作: ${timeStr}`;
        }
    }

    showMessage(message, type = STATUS.READY) {
        this.updateStatus(type, message);
    }

    // 模块特定的状态更新
    showModuleStatus(module, status, message) {
        let prefix = '';
        switch (module) {
            case 'init':
                prefix = '初始化: ';
                break;
            case 'clipboard':
                prefix = '剪贴板: ';
                break;
            case 'test':
                prefix = '测试: ';
                break;
            case 'clean':
                prefix = '清理: ';
                break;
        }
        this.showMessage(prefix + message, status);
    }

    // 清除状态
    clear() {
        this.updateStatus(STATUS.READY, '就绪');
    }
}