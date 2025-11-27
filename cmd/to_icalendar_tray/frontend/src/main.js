import './style.css';
import './app.css';

import logo from './assets/images/logo-universal.png';
import { Show, Hide, IsWindowVisible, Quit, InitConfigWithStreaming } from '../wailsjs/go/main/App';

document.querySelector('#app').innerHTML = `
    <div class="container">
        <h1 class="app-title">to_icalendar 系统托盘</h1>

        <!-- 状态显示 -->
        <div class="status" id="status">应用程序正在运行...</div>

        <!-- 初始化区域 -->
        <div class="init-section" id="initSection">
            <h3>配置初始化</h3>
            <p>首次使用需要初始化配置文件以连接 Microsoft Todo 服务</p>
            <button class="btn btn-primary" id="initBtn" onclick="initConfig()">
                初始化配置文件
            </button>
        </div>

        <!-- 日志显示区域 -->
        <div class="log-section" id="logSection" style="display: none;">
            <h3>初始化日志</h3>
            <div class="log-container" id="logContainer"></div>
            <div class="result-section" id="resultSection" style="display: none;">
                <h4>初始化结果</h4>
                <div id="resultContent"></div>
            </div>
        </div>

        <!-- 控制按钮 -->
        <div class="controls">
            <button class="btn" id="showBtn" onclick="showWindow()">显示窗口</button>
            <button class="btn" id="hideBtn" onclick="hideWindow()">隐藏到托盘</button>
            <button class="btn btn-danger" id="quitBtn" onclick="quitApp()">退出应用</button>
        </div>
    </div>
`;

// Logo已移除，界面更加简洁

let statusElement = document.getElementById("status");
let showBtn = document.getElementById("showBtn");
let hideBtn = document.getElementById("hideBtn");
let quitBtn = document.getElementById("quitBtn");

// 初始化界面
async function initializeUI() {
    try {
        const isVisible = await IsWindowVisible();
        updateUIState(isVisible);
        statusElement.textContent = "应用程序已启动，托盘功能正常运行";
    } catch (err) {
        console.error("初始化失败:", err);
        statusElement.textContent = "初始化失败: " + err;
    }
}

// 更新UI状态
function updateUIState(isVisible) {
    if (isVisible) {
        showBtn.disabled = true;
        hideBtn.disabled = false;
        statusElement.textContent = "窗口可见";
    } else {
        showBtn.disabled = false;
        hideBtn.disabled = true;
        statusElement.textContent = "窗口已隐藏到托盘";
    }
}

// 显示窗口
window.showWindow = async function () {
    try {
        await Show();
        updateUIState(true);
        statusElement.textContent = "窗口已显示";
    } catch (err) {
        console.error("显示窗口失败:", err);
        statusElement.textContent = "显示窗口失败: " + err;
    }
};

// 隐藏窗口
window.hideWindow = async function () {
    try {
        await Hide();
        updateUIState(false);
        statusElement.textContent = "窗口已隐藏到托盘";
    } catch (err) {
        console.error("隐藏窗口失败:", err);
        statusElement.textContent = "隐藏窗口失败: " + err;
    }
};

// 退出应用
window.quitApp = async function () {
    if (confirm("确定要退出应用程序吗？")) {
        try {
            statusElement.textContent = "正在退出应用程序...";
            await Quit();
        } catch (err) {
            console.error("退出应用失败:", err);
            statusElement.textContent = "退出应用失败: " + err;
        }
    }
};

// 添加初始化相关变量
let isInitializing = false;

// 初始化配置
window.initConfig = async function () {
    if (isInitializing) return;

    try {
        isInitializing = true;
        const initBtn = document.getElementById('initBtn');
        initBtn.disabled = true;
        initBtn.textContent = '正在初始化...';

        // 显示日志区域
        document.getElementById('logSection').style.display = 'block';

        // 清空现有内容
        document.getElementById('logContainer').innerHTML = '';
        document.getElementById('resultSection').style.display = 'none';

        // 滚动到日志区域
        document.getElementById('logSection').scrollIntoView({ behavior: 'smooth' });

        // 调用后端初始化方法
        await InitConfigWithStreaming();

    } catch (err) {
        appendLog('error', `初始化异常: ${err}`);
        // 重置按钮状态
        resetInitButton();
    }
};

// 重置初始化按钮状态
function resetInitButton() {
    const initBtn = document.getElementById('initBtn');
    initBtn.disabled = false;
    initBtn.textContent = '初始化配置文件';
    isInitializing = false;
}

// 添加日志到显示区域
function appendLog(type, message) {
    const logContainer = document.getElementById('logContainer');
    const logEntry = document.createElement('div');
    logEntry.className = `log-entry log-${type}`;

    const timestamp = new Date().toLocaleTimeString();
    logEntry.innerHTML = `<span class="log-time">[${timestamp}]</span> ${message}`;

    logContainer.appendChild(logEntry);
    logContainer.scrollTop = logContainer.scrollHeight;
}

// 显示初始化结果
function showResult(result) {
    const resultSection = document.getElementById('resultSection');
    const resultContent = document.getElementById('resultContent');

    if (result.success) {
        resultContent.innerHTML = `
            <div class="result-success">
                <p>✅ ${result.message}</p>
                <p><strong>配置目录:</strong> ${result.configDir}</p>
                <p><strong>配置文件:</strong> ${result.serverConfig}</p>
                <div class="next-steps">
                    <h4>下一步操作:</h4>
                    <ol>
                        <li>编辑 server.yaml 文件，配置 Microsoft Todo 信息</li>
                        <li>获取 Azure AD 租户 ID、客户端 ID 和密钥</li>
                        <li>配置完成后运行测试连接验证</li>
                    </ol>
                </div>
            </div>
        `;
    } else {
        resultContent.innerHTML = `
            <div class="result-error">
                <p>❌ ${result.message}</p>
            </div>
        `;
    }

    resultSection.style.display = 'block';
    resultSection.scrollIntoView({ behavior: 'smooth' });
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', initializeUI);

// 定期检查窗口状态
setInterval(async () => {
    try {
        const isVisible = await IsWindowVisible();
        updateUIState(isVisible);
    } catch (err) {
        // 忽略定期检查的错误，避免控制台噪音
    }
}, 2000); // 每2秒检查一次

// 事件监听器 - 监听后端日志
runtime.EventsOn("initLog", (logMessage) => {
    appendLog(logMessage.type, logMessage.message);
});

// 事件监听器 - 监听初始化结果
runtime.EventsOn("initResult", (result) => {
    resetInitButton();
    showResult(result);
});
