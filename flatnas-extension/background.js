// FlatNas 扩展后台脚本
console.log('[FlatNas Extension] Background script loaded');

// 安装事件
chrome.runtime.onInstalled.addListener((details) => {
  console.log('[FlatNas Extension] Extension installed/updated:', details.reason);

  if (details.reason === 'install') {
    // 首次安装时设置默认配置
    chrome.storage.local.set({
      flatnasUrl: 'http://localhost:3000',
      autoOpen: true,
      notifications: true
    }, () => {
      console.log('[FlatNas Extension] Default settings saved');
    });
  }
});

// 监听扩展图标点击
chrome.action.onClicked.addListener((tab) => {
  chrome.storage.local.get(['flatnasUrl'], (result) => {
    const url = result.flatnasUrl || 'http://localhost:3000';
    chrome.tabs.create({ url: url });
  });
});

// 监听新标签页更新事件
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  console.log('[FlatNas Extension] Tab updated:', tabId, 'status:', changeInfo.status, 'url:', tab.url, 'pendingUrl:', tab.pendingUrl);

  // 只在页面加载完成或加载中时处理
  if (changeInfo.status === 'complete' || changeInfo.status === 'loading') {
    chrome.storage.local.get(['flatnasUrl', 'autoOpen'], (result) => {
      const autoOpen = result.autoOpen !== false;
      const url = result.flatnasUrl || 'http://localhost:3000';

      console.log('[FlatNas Extension] Auto-open setting:', autoOpen, 'Current URL:', tab.url);

      // 检查是否是新标签页（支持 chrome://newtab/ 和 edge://newtab/）
      const isNewTab = tab.url === 'chrome://newtab/' ||
                       tab.url === 'edge://newtab' ||
                       tab.url === 'edge://newtab/' ||
                       tab.url === '' ||
                       tab.url === 'about:blank';

      console.log('[FlatNas Extension] Is new tab:', isNewTab);

      // 只有在 autoOpen 为 true 且是新标签页时才跳转
      if (autoOpen && isNewTab) {
        console.log('[FlatNas Extension] Redirecting new tab to FlatNas:', url);
        chrome.tabs.update(tabId, { url: url });
      }
      // 如果 autoOpen 为 false，不做任何操作，浏览器会显示默认新标签页
    });
  }
});



// 监听存储变化
chrome.storage.onChanged.addListener((changes, area) => {
  console.log('[FlatNas Extension] Storage changed:', area, changes);
});

// 处理来自 content script 的消息
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  console.log('[FlatNas Extension] Message received:', request);

  if (request.action === 'getFlatnasUrl') {
    chrome.storage.local.get(['flatnasUrl'], (result) => {
      sendResponse({ url: result.flatnasUrl || 'http://localhost:3000' });
    });
    return true;
  }

  if (request.action === 'checkConnection') {
    sendResponse({ status: 'ok' });
    return true;
  }
});

// 定期检查 FlatNas 服务状态
chrome.alarms.create('checkFlatnasStatus', { periodInMinutes: 5 });

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'checkFlatnasStatus') {
    console.log('[FlatNas Extension] Checking FlatNas service status');
    // 这里可以添加健康检查逻辑
  }
});

console.log('[FlatNas Extension] Background script initialized');