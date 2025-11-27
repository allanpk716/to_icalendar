import './style.css';
import './app.css';

import { Show, Hide, IsWindowVisible, Quit } from '../wailsjs/go/main/App';

// 导入组件和模块
import { appState, TABS } from './state.js';
import { TabNavigation } from './components/tab-navigation.js';
import { StatusBar } from './components/status-bar.js';
import { LogPanel } from './components/log-panel.js';
import { InitModule } from './modules/init.js';
import { TestModule } from './modules/test.js';
import { ClipboardModule } from './modules/clipboard.js';
import { CleanModule } from './modules/clean.js';

// 应用主类
class ToICalendarApp {
    constructor() {
        this.tabNavigation = null;
        this.statusBar = null;
        this.logPanel = null;
        this.modules = {};
        this.isInitialized = false;
    }

    async init() {
        try {
            this.render();
            this.setupComponents();
            this.bindEvents();
            this.setupStateListeners();

            // 初始化窗口状态
            await this.initializeWindow();

            // 检查配置状态
            await this.checkConfiguration();

            this.isInitialized = true;
            console.log('to_icalendar 应用初始化完成');

        } catch (error) {
            console.error('应用初始化失败:', error);
            this.showError('应用初始化失败', error.message);
        }
    }

    render() {
        document.querySelector('#app').innerHTML = `
            <div class="app-container">
                <!-- 标签导航 -->
                <div class="tab-navigation-container" id="tabNavigation"></div>

                <!-- 状态栏 -->
                <div class="status-bar-container" id="statusBar"></div>

                <!-- 模块内容区域 -->
                <div class="module-content-area" id="moduleContent">
                    <!-- 模块内容将在这里动态加载 -->
                </div>

                <!-- 日志面板 -->
                <div class="log-panel-container" id="logPanel"></div>
            </div>
        `;
    }

    setupComponents() {
        // 初始化组件
        const tabNavContainer = document.getElementById('tabNavigation');
        const statusBarContainer = document.getElementById('statusBar');
        const logPanelContainer = document.getElementById('logPanel');
        const moduleContentArea = document.getElementById('moduleContent');

        this.tabNavigation = new TabNavigation(tabNavContainer);
        this.statusBar = new StatusBar(statusBarContainer);
        this.logPanel = new LogPanel(logPanelContainer);

        // 初始化模块
        this.modules = {
            [TABS.INIT]: new InitModule(moduleContentArea, this.logPanel, this.statusBar),
            [TABS.TEST]: new TestModule(moduleContentArea, this.logPanel, this.statusBar),
            [TABS.CLIPBOARD]: new ClipboardModule(moduleContentArea, this.logPanel, this.statusBar),
            [TABS.CLEAN]: new CleanModule(moduleContentArea, this.logPanel, this.statusBar)
        };

        // 默认显示剪贴板模块
        this.showModule(TABS.CLIPBOARD);
    }

    bindEvents() {
        // 标签切换事件
        this.tabNavigation.container.addEventListener('tabchange', (e) => {
            this.showModule(e.detail.tabId);
        });

        // 模块切换到其他标签的事件
        Object.values(this.modules).forEach(module => {
            if (module.container) {
                module.container.addEventListener('switchTab', (e) => {
                    this.tabNavigation.switchTab(e.detail.tabId);
                });
            }
        });
    }

    setupStateListeners() {
        // 监听应用状态变化
        appState.subscribe('currentTab', (tabId) => {
            this.showModule(tabId);
        });
    }

    showModule(tabId) {
        // 隐藏所有模块
        Object.values(this.modules).forEach(module => {
            if (module.container) {
                module.container.style.display = 'none';
            }
        });

        // 显示选中的模块
        const module = this.modules[tabId];
        if (module && module.container) {
            module.container.style.display = 'block';
        }

        // 更新状态栏
        this.statusBar.showMessage(`当前模块: ${this.getModuleDisplayName(tabId)}`);
    }

    getModuleDisplayName(tabId) {
        const names = {
            [TABS.INIT]: '配置初始化',
            [TABS.TEST]: '系统测试',
            [TABS.CLIPBOARD]: '剪贴板处理',
            [TABS.CLEAN]: '缓存清理'
        };
        return names[tabId] || tabId;
    }

    async initializeWindow() {
        try {
            // 窗口控制功能已移除，不需要检查窗口状态
            console.log('应用窗口初始化完成');
        } catch (error) {
            console.error('初始化窗口状态失败:', error);
        }
    }

    async checkConfiguration() {
        try {
            // 检查配置状态，可能需要调整默认标签
            const configStatus = await this.checkConfigFileStatus();

            if (!configStatus.exists) {
                // 如果配置不存在，默认显示初始化标签
                this.tabNavigation.switchTab(TABS.INIT);
            }
        } catch (error) {
            console.log('检查配置状态失败，使用默认设置:', error);
        }
    }

    async checkConfigFileStatus() {
        // 这里可以调用后端API检查配置文件状态
        // 暂时返回存在状态，让默认显示剪贴板标签
        return { exists: true, valid: true };
    }

    showError(title, message) {
        console.error(title, message);
        document.body.innerHTML = `
            <div class="error-screen">
                <h1>❌ ${title}</h1>
                <p>${message}</p>
                <button onclick="location.reload()">重新加载应用</button>
            </div>
        `;
    }
}

// 创建应用实例
const app = new ToICalendarApp();

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', () => {
    app.init();
});

// 将必要的函数暴露到全局作用域（向后兼容）
// 窗口控制功能已移除

// 导出应用实例
export default app;
