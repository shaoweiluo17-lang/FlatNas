// FlatNas 扩展弹窗脚本
console.log('[FlatNas Extension] Popup script loaded');

// 获取 DOM 元素
const openFlatNasBtn = document.getElementById('openFlatNas');
const openSettingsBtn = document.getElementById('openSettings');
const statusEl = document.getElementById('status');

// 打开 FlatNas
openFlatNasBtn.addEventListener('click', async () => {
  const result = await chrome.storage.local.get(['flatnasUrl']);
  const url = result.flatnasUrl || 'http://localhost:3000';
  chrome.tabs.create({ url: url });
  window.close();
});

// 打开设置
openSettingsBtn.addEventListener('click', () => {
  chrome.runtime.openOptionsPage();
  window.close();
});

// 检查服务状态
async function checkStatus() {
  try {
    const result = await chrome.storage.local.get(['flatnasUrl']);
    const url = result.flatnasUrl || 'http://localhost:3000';

    const response = await fetch(url, {
      method: 'HEAD',
      mode: 'no-cors'
    });

    statusEl.textContent = '在线';
    statusEl.className = 'status-value status-online';
  } catch (error) {
    statusEl.textContent = '离线';
    statusEl.className = 'status-value status-offline';
  }
}

// 初始化
document.addEventListener('DOMContentLoaded', () => {
  checkStatus();
});

console.log('[FlatNas Extension] Popup script initialized');