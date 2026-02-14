# 图标创建说明

由于浏览器插件需要 PNG 格式的图标文件，请按以下步骤创建：

## 方法 1：使用在线工具
1. 访问 https://www.favicon-generator.org/
2. 上传你的 logo 图片
3. 生成 16x16, 48x48, 128x128 的 PNG 图标
4. 将生成的文件重命名为：
   - icon16.png
   - icon48.png
   - icon128.png
5. 替换 icons 目录中的 SVG 文件

## 方法 2：使用 ImageMagick（如果已安装）
`ash
convert -size 16x16 xc:#667eea -gravity center -pointsize 10 -fill white -annotate +0+0 "FN" icon16.png
convert -size 48x48 xc:#667eea -gravity center -pointsize 24 -fill white -annotate +0+0 "FN" icon48.png
convert -size 128x128 xc:#667eea -gravity center -pointsize 64 -fill white -annotate +0+0 "FN" icon128.png
`

## 方法 3：使用 FlatNas 现有图标
FlatNas 项目中可能已有 logo，可以从以下位置获取：
- frontend/public/favicon.ico
- frontend/public/logo.svg

转换后放置到 browser-extension/icons/ 目录。

## 临时解决方案
在测试阶段，可以使用任何 16x16, 48x48, 128x128 的 PNG 图片作为占位符。
