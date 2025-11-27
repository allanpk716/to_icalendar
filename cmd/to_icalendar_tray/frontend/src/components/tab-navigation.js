import { appState, TABS } from '../state.js';

export class TabNavigation {
    constructor(container) {
        this.container = container;
        this.tabs = [
            { id: TABS.INIT, label: '⚙️ 初始化', icon: '⚙️' },
            { id: TABS.TEST, label: '🔍 测试', icon: '🔍' },
            { id: TABS.CLIPBOARD, label: '📋 剪贴板', icon: '📋' },
            { id: TABS.CLEAN, label: '🧹 清理', icon: '🧹' }
        ];
        this.activeTab = TABS.CLIPBOARD; // 默认显示剪贴板
        this.init();
    }

    init() {
        this.render();
        this.bindEvents();
        this.setupStateListener();
    }

    render() {
        this.container.innerHTML = `
            <div class="tab-navigation">
                ${this.tabs.map(tab => `
                    <button
                        class="tab-button ${tab.id === this.activeTab ? 'active' : ''}"
                        data-tab="${tab.id}"
                        title="${tab.label}"
                    >
                        <span class="tab-icon">${tab.icon}</span>
                        <span class="tab-label">${tab.label}</span>
                    </button>
                `).join('')}
            </div>
        `;
    }

    bindEvents() {
        this.container.addEventListener('click', (e) => {
            const tabButton = e.target.closest('.tab-button');
            if (tabButton && !tabButton.disabled) {
                const tabId = tabButton.dataset.tab;
                this.switchTab(tabId);
            }
        });

        // 添加键盘快捷键支持
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey && e.key >= '1' && e.key <= '4') {
                e.preventDefault();
                const index = parseInt(e.key) - 1;
                if (index < this.tabs.length) {
                    this.switchTab(this.tabs[index].id);
                }
            }
        });
    }

    setupStateListener() {
        appState.subscribe('currentTab', (tab) => {
            this.setActiveTab(tab);
        });
    }

    switchTab(tabId) {
        if (this.tabs.find(tab => tab.id === tabId)) {
            this.activeTab = tabId;
            appState.setCurrentTab(tabId);
            this.updateUI();

            // 触发自定义事件，通知其他组件
            this.container.dispatchEvent(new CustomEvent('tabchange', {
                detail: { tabId: tabId }
            }));
        }
    }

    setActiveTab(tabId) {
        this.activeTab = tabId;
        this.updateUI();
    }

    updateUI() {
        // 更新按钮状态
        const buttons = this.container.querySelectorAll('.tab-button');
        buttons.forEach(button => {
            const tabId = button.dataset.tab;
            if (tabId === this.activeTab) {
                button.classList.add('active');
            } else {
                button.classList.remove('active');
            }
        });
    }

    // 禁用特定标签（比如未初始化时禁用某些功能）
    setTabEnabled(tabId, enabled) {
        const button = this.container.querySelector(`[data-tab="${tabId}"]`);
        if (button) {
            button.disabled = !enabled;
            if (!enabled) {
                button.classList.add('disabled');
                button.title = `${button.label} (暂不可用)`;
            } else {
                button.classList.remove('disabled');
                button.title = button.label;
            }
        }
    }

    // 显示标签提示（比如有新内容时的红点提示）
    showTabIndicator(tabId, show) {
        const button = this.container.querySelector(`[data-tab="${tabId}"]`);
        if (button) {
            if (show) {
                button.classList.add('has-indicator');
            } else {
                button.classList.remove('has-indicator');
            }
        }
    }

    getCurrentTab() {
        return this.activeTab;
    }
}