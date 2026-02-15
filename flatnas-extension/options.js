// FlatNas 扩展设置脚本
console.log('[FlatNas Extension] Options script loaded');

// 获取 DOM 元素
const flatnasUrlInput = document.getElementById('flatnasUrl');
const autoOpenCheckbox = document.getElementById('autoOpen');
const notificationsCheckbox = document.getElementById('notifications');
const saveBtn = document.getElementById('saveBtn');
const resetBtn = document.getElementById('resetBtn');
const messageEl = document.getElementById('message');

// 显示消息
function showMessage(text, type) {
  messageEl.textContent = text;
  messageEl.className = `message ${type}`;
  messageEl.style.display = 'block';

  setTimeout(() => {
    messageEl.style.display = 'none';
  }, 3000);
}

// 加载设置
async function loadSettings() {
  const result = await chrome.storage.local.get([
    'flatnasUrl',
    'autoOpen',
    'notifications'
  ]);

  flatnasUrlInput.value = result.flatnasUrl || 'http://localhost:3000';
  autoOpenCheckbox.checked = result.autoOpen !== false;
  notificationsCheckbox.checked = result.notifications !== false;
}

// 保存设置
async function saveSettings() {
  const flatnasUrl = flatnasUrlInput.value.trim();

  if (!flatnasUrl) {
    showMessage('请输入 FlatNas 服务地址', 'error');
    return;
  }

  try {
    await chrome.storage.local.set({
      flatnasUrl: flatnasUrl,
      autoOpen: autoOpenCheckbox.checked,
      notifications: notificationsCheckbox.checked
    });

    showMessage('设置已保存', 'success');
  } catch (error) {
    console.error('[FlatNas Extension] Save settings error:', error);
    showMessage('保存设置失败', 'error');
  }
}

// 重置设置
async function resetSettings() {
  if (!confirm('确定要重置为默认设置吗？')) {
    return;
  }

  try {
    await chrome.storage.local.set({
      flatnasUrl: 'http://localhost:3000',
      autoOpen: true,
      notifications: true
    });

    await loadSettings();
    showMessage('已重置为默认设置', 'success');
  } catch (error) {
    console.error('[FlatNas Extension] Reset settings error:', error);
    showMessage('重置设置失败', 'error');
  }
}

// 事件监听
saveBtn.addEventListener('click', saveSettings);
resetBtn.addEventListener('click', resetSettings);

// 初始化
document.addEventListener('DOMContentLoaded', () => {
  loadSettings();
});

console.log('[FlatNas Extension] Options script initialized');