import './style.css';
import './app.css';

import logo from './assets/images/logo-universal.png';
import { Show, Hide, IsWindowVisible, Quit } from '../wailsjs/go/main/App';

document.querySelector('#app').innerHTML = `
    <div class="container">
        <img id="logo" class="logo" alt="to_icalendar logo">
        <h1>to_icalendar 系统托盘</h1>
        <div class="status" id="status">应用程序正在运行...</div>
        <div class="controls">
            <button class="btn" id="showBtn" onclick="showWindow()">显示窗口</button>
            <button class="btn" id="hideBtn" onclick="hideWindow()">隐藏到托盘</button>
            <button class="btn btn-danger" id="quitBtn" onclick="quitApp()">退出应用</button>
        </div>
        <div class="info">
            <p>to_icalendar 系统托盘应用程序</p>
            <p>应用程序最小化到系统托盘后继续在后台运行</p>
        </div>
    </div>
`;

document.getElementById('logo').src = logo;

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
