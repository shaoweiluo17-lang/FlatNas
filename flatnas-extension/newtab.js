// FlatNas 新标签页脚本
console.log('[FlatNas Extension] Newtab script loaded');

// 获取设置并决定是否跳转
chrome.storage.local.get(['flatnasUrl', 'autoOpen'], (result) => {
  const url = result.flatnasUrl || 'http://localhost:3000';
  const autoOpen = result.autoOpen !== false; // 默认启用

  console.log('[FlatNas Extension] Settings:', { url, autoOpen, result });

  if (autoOpen) {
    // 如果启用了自动打开，跳转到 FlatNas
    console.log('[FlatNas Extension] Auto-open enabled, redirecting to:', url);
    setTimeout(() => {
      console.log('[FlatNas Extension] Performing redirect...');
      window.location.href = url;
    }, 500);
  } else {
    // 如果禁用了自动打开，显示手动打开按钮
    console.log('[FlatNas Extension] Auto-open disabled, showing manual open button');
    const container = document.querySelector('.container');
    container.innerHTML = `
      <img src="icons/icon128.png" alt="FlatNas" class="logo">
      <h1>FlatNas</h1>
      <p>个人导航页与仪表盘系统</p>
      <button class="button" id="openFlatNas">打开 FlatNas</button>
      <div class="info">
        您已禁用自动打开功能。点击上方按钮可手动打开 FlatNas。<br>
        如需启用自动打开，请前往扩展设置页面。
      </div>
    `;

    // 添加按钮点击事件
    document.getElementById('openFlatNas').addEventListener('click', () => {
      console.log('[FlatNas Extension] Manual open clicked, redirecting to:', url);
      window.location.href = url;
    });
  }
});