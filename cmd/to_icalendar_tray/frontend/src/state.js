// 应用状态管理
class AppState {
    constructor() {
        this.currentTab = 'clipboard'; // 默认显示剪贴板模块
        this.globalStatus = 'ready'; // ready, processing, success, error
        this.moduleStates = {
            init: { running: false, completed: false },
            clipboard: { hasContent: false, processing: false, content: null },
            test: { running: false, results: null },
            clean: { running: false, results: null }
        };
        this.listeners = new Map();
    }

    // 订阅状态变化
    subscribe(key, callback) {
        if (!this.listeners.has(key)) {
            this.listeners.set(key, []);
        }
        this.listeners.get(key).push(callback);
    }

    // 取消订阅
    unsubscribe(key, callback) {
        if (this.listeners.has(key)) {
            const callbacks = this.listeners.get(key);
            const index = callbacks.indexOf(callback);
            if (index > -1) {
                callbacks.splice(index, 1);
            }
        }
    }

    // 通知状态变化
    notify(key, value) {
        if (this.listeners.has(key)) {
            this.listeners.get(key).forEach(callback => {
                try {
                    callback(value);
                } catch (error) {
                    console.error('State listener error:', error);
                }
            });
        }
    }

    // 设置当前标签
    setCurrentTab(tab) {
        this.currentTab = tab;
        this.notify('currentTab', tab);
    }

    // 获取当前标签
    getCurrentTab() {
        return this.currentTab;
    }

    // 设置全局状态
    setGlobalStatus(status) {
        this.globalStatus = status;
        this.notify('globalStatus', status);
    }

    // 获取全局状态
    getGlobalStatus() {
        return this.globalStatus;
    }

    // 设置模块状态
    setModuleState(module, state) {
        this.moduleStates[module] = { ...this.moduleStates[module], ...state };
        this.notify(`module:${module}`, this.moduleStates[module]);
    }

    // 获取模块状态
    getModuleState(module) {
        return this.moduleStates[module];
    }
}

// 创建全局状态实例
export const appState = new AppState();

// 导出状态常量
export const TABS = {
    INIT: 'init',
    TEST: 'test',
    CLIPBOARD: 'clipboard',
    CLEAN: 'clean'
};

export const STATUS = {
    READY: 'ready',
    PROCESSING: 'processing',
    SUCCESS: 'success',
    ERROR: 'error'
};