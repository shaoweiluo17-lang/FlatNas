import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import vueDevTools from 'vite-plugin-vue-devtools';

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  base: '/',
  publicDir: '../server/public',
  build: {
    sourcemap: false,
    // 确保静态资源被正确复制到输出目录
    rollupOptions: {
      output: {
        assetFileNames: 'assets/[name]-[hash][extname]',
        chunkFileNames: 'assets/[name]-[hash].js',
        entryFileNames: 'assets/[name]-[hash].js',
      },
    },
    // 启用代码分割
    chunkSizeWarningLimit: 1000,
  },
  plugins: [vue(), mode === 'development' && vueDevTools()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  // ✨✨✨ 关键修改：增加了 /music 的代理 ✨✨✨
  server: {
    host: '0.0.0.0',
    watch: {
      ignored: ['**/data/**', '**/server/**'],
    },
    proxy: {
      // 告诉 Vite：遇到 /api 开头的请求，转给 3000 端口
      '/api': {
        target: process.env.VITE_BACKEND || 'http://127.0.0.1:3000',
        changeOrigin: true,
      },
      // ✨ 新增：告诉 Vite：遇到 /music 开头的请求，也转给 3000 端口！
      '/music': {
        target: process.env.VITE_BACKEND || 'http://127.0.0.1:3000',
        changeOrigin: true,
      },
      // ✨ Backgrounds 代理
      '/backgrounds': {
        target: process.env.VITE_BACKEND || 'http://127.0.0.1:3000',
        changeOrigin: true,
      },
      '/mobile_backgrounds': {
        target: process.env.VITE_BACKEND || 'http://127.0.0.1:3000',
        changeOrigin: true,
      },
      '/icon-cache': {
        target: process.env.VITE_BACKEND || 'http://127.0.0.1:3000',
        changeOrigin: true,
      },
      // ✨ CGI 代理
      '^.*\\.cgi.*': {
        target: process.env.VITE_BACKEND || 'http://127.0.0.1:3000',
        changeOrigin: true,
      },
      // ✨ Socket.IO 代理
      '/socket.io': {
        target: process.env.VITE_BACKEND || 'http://127.0.0.1:3000',
        ws: true,
        changeOrigin: true,
      },
    },
  },
}));
