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

// 监听新标签页打开
chrome.tabs.onCreated.addListener((tab) => {
  console.log('[FlatNas Extension] Tab created:', tab.id, tab.url, tab.pendingUrl);

  chrome.storage.local.get(['flatnasUrl', 'autoOpen'], (result) => {
    const autoOpen = result.autoOpen !== false;
    const url = result.flatnasUrl || 'http://localhost:3000';

    console.log('[FlatNas Extension] Auto-open setting:', autoOpen);

    if (autoOpen) {
      // 延迟执行，确保标签页完全创建
      setTimeout(() => {
        chrome.tabs.get(tab.id, (updatedTab) => {
          console.log('[FlatNas Extension] Updated tab info:', updatedTab.url, updatedTab.pendingUrl);

          // 检查是否是新标签页
          if (updatedTab.url === 'chrome://newtab/' ||
              updatedTab.pendingUrl === 'chrome://newtab/' ||
              updatedTab.url === '' ||
              updatedTab.pendingUrl === '') {
            console.log('[FlatNas Extension] Redirecting new tab to FlatNas:', url);
            chrome.tabs.update(tab.id, { url: url });
          }
        });
      }, 100);
    }
  });
});

// 监听导航完成事件（作为备用方案）
chrome.webNavigation.onCompleted.addListener((details) => {
  if (details.frameId === 0) { // 只处理主框架
    chrome.storage.local.get(['flatnasUrl', 'autoOpen'], (result) => {
      const autoOpen = result.autoOpen !== false;
      const url = result.flatnasUrl || 'http://localhost:3000';

      if (autoOpen && details.url === 'chrome://newtab/') {
        console.log('[FlatNas Extension] Navigation completed on newtab, redirecting to:', url);
        chrome.tabs.update(details.tabId, { url: url });
      }
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