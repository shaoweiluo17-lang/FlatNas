// FlatNas 加速器 - 后台脚本
const DEFAULT_SETTINGS = {
  serverUrl: 'https://bswy.cc',
  localUrl: 'http://192.168.31.90:23000',
  enableCache: true,
  enableOffline: true,
  enablePreload: true,
  cacheSize: 100,
  autoUpdateInterval: 3600,
  enableOptimization: true,
  maxCacheAge: 86400
};

const CACHE_PREFIX = 'flatnas_cache_';

chrome.runtime.onInstalled.addListener(async (details) => {
  console.log('[FlatNas Extension] Installed:', details.reason);
  
  if (details.reason === 'install') {
    await chrome.storage.local.set({ settings: DEFAULT_SETTINGS });
    await preloadDefaultConfig();
  } else if (details.reason === 'update') {
    await migrateSettings();
  }
  
  setupAlarms();
});

chrome.alarms.create('checkUpdate', { periodInMinutes: 10 });
chrome.alarms.create('cleanCache', { periodInMinutes: 60 });

chrome.alarms.onAlarm.addListener(async (alarm) => {
  const settings = await getSettings();
  
  if (alarm.name === 'checkUpdate' && settings.enableCache) {
    await checkForUpdates();
  } else if (alarm.name === 'cleanCache') {
    await cleanExpiredCache();
  }
});

async function checkForUpdates() {
  const settings = await getSettings();
  try {
    await updateConfigCache(settings.serverUrl);
    await updateConfigCache(settings.localUrl);
    console.log('[FlatNas Extension] Cache updated');
  } catch (error) {
    console.error('[FlatNas Extension] Update failed:', error);
  }
}

async function updateConfigCache(baseUrl) {
  const endpoints = ['/api/config', '/api/system-config'];
  
  for (const endpoint of endpoints) {
    try {
      const url = baseUrl + endpoint;
      const response = await fetch(url);
      
      if (response.ok) {
        const data = await response.json();
        const cacheKey = CACHE_PREFIX + btoa(url);
        
        await chrome.storage.local.set({
          [cacheKey]: { data, timestamp: Date.now(), url }
        });
      }
    } catch (error) {
      console.error(`[FlatNas Extension] Failed to cache ${endpoint}:`, error);
    }
  }
}

async function cleanExpiredCache() {
  const settings = await getSettings();
  const items = await chrome.storage.local.get();
  const now = Date.now();
  const keysToRemove = [];
  
  for (const [key, value] of Object.entries(items)) {
    if (key.startsWith(CACHE_PREFIX)) {
      const age = (now - value.timestamp) / 1000;
      if (age > settings.maxCacheAge) {
        keysToRemove.push(key);
      }
    }
  }
  
  if (keysToRemove.length > 0) {
    await chrome.storage.local.remove(keysToRemove);
    console.log(`[FlatNas Extension] Cleaned ${keysToRemove.length} expired items`);
  }
}

async function preloadDefaultConfig() {
  try {
    const response = await fetch(chrome.runtime.getURL('assets/default-config.json'));
    const defaultConfig = await response.json();
    await chrome.storage.local.set({ 'default_config': defaultConfig });
    console.log('[FlatNas Extension] Default config preloaded');
  } catch (error) {
    console.error('[FlatNas Extension] Failed to preload default config:', error);
  }
}

async function migrateSettings() {
  const result = await chrome.storage.local.get('settings');
  const oldSettings = result.settings || {};
  const newSettings = { ...DEFAULT_SETTINGS, ...oldSettings };
  await chrome.storage.local.set({ settings: newSettings });
}

async function getSettings() {
  const result = await chrome.storage.local.get('settings');
  return result.settings || DEFAULT_SETTINGS;
}

function setupAlarms() {
  chrome.alarms.create('checkUpdate', { periodInMinutes: 10 });
  chrome.alarms.create('cleanCache', { periodInMinutes: 60 });
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'getSettings') {
    getSettings().then(settings => sendResponse(settings));
    return true;
  }
  
  if (message.type === 'updateCache') {
    checkForUpdates().then(() => sendResponse({ success: true }));
    return true;
  }
  
  if (message.type === 'clearCache') {
    clearAllCache().then(() => sendResponse({ success: true }));
    return true;
  }
  
  if (message.type === 'getCacheInfo') {
    getCacheInfo().then(info => sendResponse(info));
    return true;
  }
});

async function clearAllCache() {
  const items = await chrome.storage.local.get();
  const keysToRemove = Object.keys(items).filter(key => 
    key.startsWith(CACHE_PREFIX) || key === 'default_config'
  );
  await chrome.storage.local.remove(keysToRemove);
  console.log(`[FlatNas Extension] Cleared ${keysToRemove.length} items`);
}

async function getCacheInfo() {
  const items = await chrome.storage.local.get();
  const cacheItems = Object.entries(items).filter(([key]) => key.startsWith(CACHE_PREFIX));
  
  let totalSize = 0;
  for (const [key, value] of cacheItems) {
    totalSize += JSON.stringify(value).length;
  }
  
  return {
    count: cacheItems.length,
    size: totalSize,
    sizeMB: (totalSize / 1024 / 1024).toFixed(2)
  };
}
