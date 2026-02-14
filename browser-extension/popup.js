// FlatNas 加速器 - 弹窗脚本
document.addEventListener('DOMContentLoaded', async () => {
  await loadSettings();
  await updateStatus();
  bindEvents();
});

async function loadSettings() {
  try {
    const response = await chrome.runtime.sendMessage({ type: 'getSettings' });
    const settings = await response;
    
    document.getElementById('enableCache').checked = settings.enableCache;
    document.getElementById('enableOffline').checked = settings.enableOffline;
    document.getElementById('enableAutoUpdate').checked = settings.autoUpdateInterval > 0;
    
    document.getElementById('serverUrl').value = settings.serverUrl;
    document.getElementById('localUrl').value = settings.localUrl;
  } catch (error) {
    console.error('Failed to load settings:', error);
  }
}

async function updateStatus() {
  try {
    const response = await chrome.runtime.sendMessage({ type: 'getCacheInfo' });
    const info = await response;
    
    document.getElementById('cacheStatus').textContent = info.count > 0 ? '已启用' : '未启用';
    document.getElementById('cacheSize').textContent = `${info.sizeMB} MB`;
    document.getElementById('cacheCount').textContent = info.count;
  } catch (error) {
    console.error('Failed to update status:', error);
  }
}

function bindEvents() {
  document.getElementById('enableCache').addEventListener('change', async (e) => {
    await updateSetting('enableCache', e.target.checked);
  });
  
  document.getElementById('enableOffline').addEventListener('change', async (e) => {
    await updateSetting('enableOffline', e.target.checked);
  });
  
  document.getElementById('enableAutoUpdate').addEventListener('change', async (e) => {
    await updateSetting('autoUpdateInterval', e.target.checked ? 3600 : 0);
  });
  
  document.getElementById('forceUpdate').addEventListener('click', async () => {
    const btn = document.getElementById('forceUpdate');
    const originalText = btn.textContent;
    
    btn.textContent = '⏳ 更新中...';
    btn.disabled = true;
    
    try {
      await chrome.runtime.sendMessage({ type: 'updateCache' });
      await updateStatus();
      alert('更新成功！');
    } catch (error) {
      alert('更新失败：' + error.message);
    } finally {
      btn.textContent = originalText;
      btn.disabled = false;
    }
  });
  
  document.getElementById('clearCache').addEventListener('click', async () => {
    if (confirm('确定要清除所有缓存吗？')) {
      try {
        await chrome.runtime.sendMessage({ type: 'clearCache' });
        await updateStatus();
        alert('缓存已清除');
      } catch (error) {
        alert('清除失败：' + error.message);
      }
    }
  });
  
  document.getElementById('saveConfig').addEventListener('click', async () => {
    const serverUrl = document.getElementById('serverUrl').value.trim();
    const localUrl = document.getElementById('localUrl').value.trim();
    
    if (!serverUrl || !localUrl) {
      alert('请填写完整的服务器地址');
      return;
    }
    
    try {
      await updateSetting('serverUrl', serverUrl);
      await updateSetting('localUrl', localUrl);
      alert('配置已保存');
    } catch (error) {
      alert('保存失败：' + error.message);
    }
  });
}

async function updateSetting(key, value) {
  try {
    const response = await chrome.runtime.sendMessage({ type: 'getSettings' });
    const settings = await response;
    settings[key] = value;
    await chrome.storage.local.set({ settings });
  } catch (error) {
    console.error('Failed to update setting:', error);
    throw error;
  }
}
