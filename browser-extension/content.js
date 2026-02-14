// FlatNas 加速器 - 内容脚本
(function() {
  'use strict';
  console.log('[FlatNas Extension] Content script loaded');
  
  // 监听网络状态
  function updateNetworkStatus() {
    const isOnline = navigator.onLine;
    document.body.classList.toggle('offline', !isOnline);
    if (!isOnline) {
      showOfflineNotification();
    } else {
      hideOfflineNotification();
    }
  }
  
  window.addEventListener('online', updateNetworkStatus);
  window.addEventListener('offline', updateNetworkStatus);
  updateNetworkStatus();
  
  // 优化页面加载
  document.addEventListener('DOMContentLoaded', () => {
    optimizeImages();
    preloadCriticalResources();
  });
  
  function optimizeImages() {
    const images = document.querySelectorAll('img');
    images.forEach(img => {
      if (!img.loading) {
        img.loading = 'lazy';
      }
      img.addEventListener('error', function() {
        this.style.display = 'none';
      });
    });
  }
  
  function preloadCriticalResources() {
    const criticalResources = ['/api/config', '/api/system-config'];
    criticalResources.forEach(url => {
      fetch(url, { cache: 'force-cache' })
        .then(response => {
          if (response.ok) return response.json();
        })
        .catch(error => {
          console.log('[FlatNas Extension] Preload failed:', url);
        });
    });
  }
  
  function showOfflineNotification() {
    if (document.querySelector('[data-offline-notification]')) return;
    const notification = document.createElement('div');
    notification.dataset.offlineNotification = 'true';
    notification.style.cssText = 'position:fixed;top:20px;right:20px;background:#ff4444;color:white;padding:12px 24px;border-radius:8px;z-index:10000;animation:slideIn 0.3s ease;';
    notification.textContent = '🌐 网络已断开，使用缓存数据';
    document.body.appendChild(notification);
  }
  
  function hideOfflineNotification() {
    const notification = document.querySelector('[data-offline-notification]');
    if (notification) notification.remove();
  }
  
  // 拦截 fetch 请求
  const originalFetch = window.fetch;
  window.fetch = async function(...args) {
    const url = args[0];
    
    if (typeof url === 'string' && url.includes('/api/')) {
      try {
        const cached = await getFromCache(url);
        if (cached) {
          console.log('[FlatNas Extension] Using cached data:', url);
          return new Response(JSON.stringify(cached), {
            headers: { 'Content-Type': 'application/json' }
          });
        }
      } catch (error) {
        console.error('[FlatNas Extension] Cache error:', error);
      }
    }
    
    const response = await originalFetch.apply(this, args);
    
    if (response.ok && typeof url === 'string' && url.includes('/api/')) {
      try {
        const data = await response.clone().json();
        await saveToCache(url, data);
      } catch (error) {}
    }
    
    return response;
  };
  
  async function getFromCache(url) {
    const cacheKey = 'flatnas_cache_' + btoa(url);
    const result = await chrome.storage.local.get(cacheKey);
    return result[cacheKey]?.data;
  }
  
  async function saveToCache(url, data) {
    const cacheKey = 'flatnas_cache_' + btoa(url);
    await chrome.storage.local.set({
      [cacheKey]: { data, timestamp: Date.now() }
    });
  }
})();
