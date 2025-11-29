export class LogPanel {
    constructor(container) {
        this.container = container;
        this.logs = [];
        this.maxLogs = 100; // 最大日志条数
        this.logLevels = ['debug', 'info', 'warn', 'error'];
        this.currentFilter = 'all'; // 当前过滤级别
        this.init();
    }

    init() {
        this.render();
        this.bindEvents();
    }

    render() {
        this.container.innerHTML = `
            <div class="log-panel">
                <div class="log-header">
                    <h3>日志面板</h3>
                    <div class="log-controls">
                        <select class="log-filter" id="logFilter">
                            <option value="all">全部</option>
                            <option value="info">信息</option>
                            <option value="warn">警告</option>
                            <option value="error">错误</option>
                        </select>
                        <button class="log-clear" id="logClear">清空</button>
                        <button class="log-export" id="logExport">导出</button>
                    </div>
                </div>
                <div class="log-container" id="logContainer"></div>
            </div>
        `;

        this.logContainer = this.container.querySelector('#logContainer');
        this.logFilter = this.container.querySelector('#logFilter');
        this.logClear = this.container.querySelector('#logClear');
        this.logExport = this.container.querySelector('#logExport');
    }

    bindEvents() {
        this.logFilter.addEventListener('change', (e) => {
            this.currentFilter = e.target.value;
            this.renderLogs();
        });

        this.logClear.addEventListener('click', () => {
            this.clearLogs();
        });

        this.logExport.addEventListener('click', () => {
            this.exportLogs();
        });
    }

    addLog(type, message, source = '') {
        const log = {
            id: Date.now() + Math.random(),
            timestamp: new Date(),
            type: type.toLowerCase(),
            message,
            source
        };

        // 检查日志级别
        if (!this.isValidLogLevel(log.type)) {
            console.warn(`Invalid log type: ${log.type}`);
            return;
        }

        this.logs.push(log);

        // 限制日志数量
        if (this.logs.length > this.maxLogs) {
            this.logs = this.logs.slice(-this.maxLogs);
        }

        this.renderLogs();
        this.scrollToBottom();
    }

    isValidLogLevel(type) {
        return this.logLevels.includes(type) || type === 'debug';
    }

    shouldShowLog(log) {
        if (this.currentFilter === 'all') {
            return true;
        }
        return log.type === this.currentFilter;
    }

    renderLogs() {
        const filteredLogs = this.logs.filter(log => this.shouldShowLog(log));

        this.logContainer.innerHTML = filteredLogs.map(log => this.renderLogEntry(log)).join('');

        if (filteredLogs.length === 0) {
            this.logContainer.innerHTML = '<div class="log-empty">暂无日志</div>';
        }
    }

    renderLogEntry(log) {
        const timestamp = log.timestamp.toLocaleTimeString();
        const sourceStr = log.source ? `[${log.source}] ` : '';

        return `
            <div class="log-entry log-${log.type}" data-id="${log.id}">
                <span class="log-time">[${timestamp}]</span>
                <span class="log-source">${sourceStr}</span>
                <span class="log-message">${this.escapeHtml(log.message)}</span>
            </div>
        `;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    scrollToBottom() {
        this.logContainer.scrollTop = this.logContainer.scrollHeight;
    }

    clearLogs() {
        this.logs = [];
        this.renderLogs();
    }

    exportLogs() {
        if (this.logs.length === 0) {
            alert('没有日志可以导出');
            return;
        }

        const logText = this.logs.map(log => {
            const timestamp = log.timestamp.toISOString();
            return `[${timestamp}] [${log.type.toUpperCase()}] ${log.source ? `[${log.source}] ` : ''}${log.message}`;
        }).join('\n');

        const blob = new Blob([logText], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `to_icalendar_logs_${new Date().toISOString().split('T')[0]}.txt`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }

    // 便捷方法
    info(message, source = '') {
        this.addLog('info', message, source);
    }

    warn(message, source = '') {
        this.addLog('warn', message, source);
    }

    error(message, source = '') {
        this.addLog('error', message, source);
    }

    debug(message, source = '') {
        this.addLog('debug', message, source);
    }

    success(message, source = '') {
        this.addLog('info', `✅ ${message}`, source);
    }

    // 获取日志统计
    getStats() {
        const stats = {
            total: this.logs.length,
            info: 0,
            warn: 0,
            error: 0,
            debug: 0
        };

        this.logs.forEach(log => {
            if (stats.hasOwnProperty(log.type)) {
                stats[log.type]++;
            }
        });

        return stats;
    }
}