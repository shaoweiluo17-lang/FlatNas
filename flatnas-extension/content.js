// FlatNas 扩展内容脚本
console.log('[FlatNas Extension] Content script loaded');

// 检测当前页面是否是 FlatNas
function isFlatNasPage() {
  const hostname = window.location.hostname;
  return hostname === 'localhost' ||
         hostname.startsWith('192.168.') ||
         hostname.startsWith('10.') ||
         hostname.startsWith('172.');
}

// 如果是 FlatNas 页面，注入一些增强功能
if (isFlatNasPage()) {
  console.log('[FlatNas Extension] FlatNas page detected');

  // 监听来自后台的消息
  chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.action === 'ping') {
      sendResponse({ status: 'alive', url: window.location.href });
    }
  });

  // 通知后台脚本页面已加载
  chrome.runtime.sendMessage({
    action: 'pageLoaded',
    url: window.location.href
  });

  // 可以在这里添加页面增强功能
  // 例如：添加快捷键、性能监控等
}

// 创建快捷键支持
document.addEventListener('keydown', (e) => {
  // Ctrl+Shift+F 快速打开设置
  if (e.ctrlKey && e.shiftKey && e.key === 'F') {
    e.preventDefault();
    chrome.runtime.openOptionsPage();
  }
});

console.log('[FlatNas Extension] Content script initialized');