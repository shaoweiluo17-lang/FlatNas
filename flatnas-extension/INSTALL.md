# FlatNas 浏览器扩展安装指南

## 📦 扩展包位置

扩展文件位于：`D:\1_lsw\project\FlatNas\flatnas-extension`

## 🚀 快速安装（Chrome/Edge）

### 方法 1：直接加载（推荐）

1. **打开扩展管理页面**
   - Chrome: 地址栏输入 `chrome://extensions/`
   - Edge: 地址栏输入 `edge://extensions/`

2. **启用开发者模式**
   - 点击右上角的"开发者模式"开关

3. **加载扩展**
   - 点击"加载已解压的扩展程序"按钮
   - 浏览到 `D:\1_lsw\project\FlatNas\flatnas-extension` 文件夹
   - 点击"选择文件夹"

4. **完成安装**
   - 扩展将立即安装并显示在浏览器工具栏中

### 方法 2：打包安装

1. **打包扩展**
   - 在 `chrome://extensions/` 页面
   - 点击"打包扩展程序"
   - 选择 `flatnas-extension` 文件夹
   - 生成 `.crx` 文件

2. **安装扩展**
   - 将 `.crx` 文件拖拽到浏览器中
   - 确认安装

## 🦊 Firefox 安装

1. **打开调试页面**
   - 地址栏输入 `about:debugging#/runtime/this-firefox`

2. **加载临时扩展**
   - 点击"临时载入附加组件"
   - 选择 `flatnas-extension/manifest.json` 文件

3. **完成安装**
   - 扩展将安装并在会话期间有效

## ⚙️ 首次配置

1. **点击扩展图标**
   - 在浏览器工具栏中找到 FlatNas 图标并点击

2. **打开设置**
   - 在弹窗中点击"设置"按钮

3. **配置服务地址**
   - 在"FlatNas 服务地址"输入框中输入：
     - 开发环境：`http://localhost:3000`
     - 局域网：`http://192.168.x.x:23000`
     - 生产环境：`https://your-domain.com`

4. **保存设置**
   - 点击"保存设置"按钮

## 📱 使用功能

### 快速访问
- 点击工具栏图标 → "打开 FlatNas"

### 新标签页
- 设置 → 勾选"自动在新标签页中打开 FlatNas"
- 或直接使用扩展提供的新标签页页面

### 状态监控
- 点击工具栏图标查看服务状态

## 🔍 验证安装

1. **检查扩展状态**
   - 访问 `chrome://extensions/`
   - 确认 FlatNas 扩展已启用

2. **测试功能**
   - 点击扩展图标
   - 点击"打开 FlatNas"
   - 应该能成功打开您的 FlatNas 服务

## ❗ 常见问题

### Q: 扩展加载失败？
A: 确保 `manifest.json` 文件在正确的位置，并且没有语法错误。

### Q: 无法连接到 FlatNas？
A: 检查服务地址是否正确，确认 FlatNas 服务正在运行。

### Q: 新标签页不生效？
A: 检查浏览器设置，确认没有其他扩展覆盖了新标签页。

### Q: Firefox 中扩展不持久？
A: 临时加载的扩展在浏览器重启后会消失。建议使用开发者账户发布扩展。

## 📞 技术支持

如遇问题，请访问：
- GitHub Issues: https://github.com/shaoweiluo17-lang/FlatNas/issues
- 查看详细文档: `flatnas-extension/README.md`

---

**享受 FlatNas 带来的便捷体验！** 🎉