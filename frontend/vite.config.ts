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
    // 确保 sw.js 正确设置 Content-Type
    headers: {
      'Service-Worker-Allowed': '/',
    },
    // 添加中间件来正确处理 sw.js 的 Content-Type
    configureServer(server: { middlewares: { use: (arg0: (req: any, res: any, next: any) => void) => void; }; }) {
      return () => {
        server.middlewares.use((req, res, next) => {
          if (req.url === '/sw.js' || req.url?.endsWith('/sw.js')) {
            res.setHeader('Content-Type', 'application/javascript; charset=utf-8');
            res.setHeader('Service-Worker-Allowed', '/');
            res.setHeader('Cache-Control', 'no-cache, no-store, must-revalidate');
          }
          if (req.url === '/manifest.json') {
            res.setHeader('Content-Type', 'application/manifest+json; charset=utf-8');
          }
          next();
        });
      };
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
