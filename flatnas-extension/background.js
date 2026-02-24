// FlatNas 扩展后台脚本
console.log('[FlatNas Extension] Background script loaded');

// 安装事件
chrome.runtime.onInstalled.addListener((details) => {
  console.log('[FlatNas Extension] Extension installed/updated:', details.reason);

  if (details.reason === 'install') {
    // 首次安装时设置默认配置
    chrome.storage.local.set({
      flatnasUrl: 'https://bswy.cc',
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

// 跟踪已跳转的标签页,防止重复跳转
const redirectedTabs = new Set();

// 监听新标签页更新事件
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  console.log('[FlatNas Extension] Tab updated:', tabId, 'status:', changeInfo.status, 'url:', tab.url, 'pendingUrl:', tab.pendingUrl);

  // 只在页面加载完成时处理新标签页
  if (changeInfo.status === 'complete') {
    // 如果已经跳转过这个标签页,不再处理
    if (redirectedTabs.has(tabId)) {
      console.log('[FlatNas Extension] Tab already redirected, skipping:', tabId);
      return;
    }

    chrome.storage.local.get(['flatnasUrl', 'autoOpen'], (result) => {
      const autoOpen = result.autoOpen !== false;
      const url = result.flatnasUrl || 'http://localhost:3000';

      console.log('[FlatNas Extension] Auto-open setting:', autoOpen, 'Current URL:', tab.url, 'FlatNas URL:', url);

      // 检查是否是新标签页（支持多种浏览器的新标签页 URL）
      const isNewTab = !tab.url ||
                       tab.url === '' ||
                       tab.url === 'about:blank' ||
                       tab.url === 'chrome://newtab/' ||
                       tab.url === 'edge://newtab' ||
                       tab.url === 'edge://newtab/' ||
                       tab.url.startsWith('chrome://newtab') ||
                       tab.url.startsWith('edge://newtab');

      console.log('[FlatNas Extension] Is new tab:', isNewTab);

      // 只有在 autoOpen 为 true 且是新标签页时才跳转
      if (autoOpen && isNewTab) {
        console.log('[FlatNas Extension] Redirecting new tab to FlatNas:', url);
        // 标记为已跳转
        redirectedTabs.add(tabId);

        chrome.tabs.update(tabId, { url: url }, () => {
          if (chrome.runtime.lastError) {
            console.error('[FlatNas Extension] Redirect error:', chrome.runtime.lastError);
            // 如果跳转失败,从集合中移除,允许重试
            redirectedTabs.delete(tabId);
          } else {
            console.log('[FlatNas Extension] Redirect successful');
            // 3秒后从集合中移除,允许后续更新
            setTimeout(() => {
              redirectedTabs.delete(tabId);
              console.log('[FlatNas Extension] Tab redirection tracking cleared:', tabId);
            }, 3000);
          }
        });
      }
      // 如果 autoOpen 为 false，不做任何操作，浏览器会显示默认新标签页
    });
  }
});

// 监听标签页关闭事件,清理跟踪记录
chrome.tabs.onRemoved.addListener((tabId) => {
  if (redirectedTabs.has(tabId)) {
    redirectedTabs.delete(tabId);
    console.log('[FlatNas Extension] Tab closed, tracking cleared:', tabId);
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