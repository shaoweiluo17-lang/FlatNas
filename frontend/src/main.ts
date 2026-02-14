import "./assets/main.css";
import "./assets/grid-layout.css";
import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import { useMainStore } from "./stores/main";
import { attachErrorCapture, ensureOverlayHandled } from "./utils/overlay";

if (typeof document !== "undefined" && typeof navigator !== "undefined") {
  const ua = navigator.userAgent || "";
  const isHarmony = /(harmonyos|hongmeng|hm os)/i.test(ua);
  const isHuawei = /(huaweibrowser|huawei)/i.test(ua);
  const isAlook = /alook/i.test(ua);
  if (isHarmony || isHuawei) {
    document.documentElement.classList.add("harmony-os");
  }
  if (isAlook) {
    document.documentElement.classList.add("alook-browser");
  }
}

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);

// Initialize store globally to ensure configuration is loaded
const store = useMainStore();
store.init();

app.mount("#app");

if (import.meta.env.DEV) {
  attachErrorCapture();
  ensureOverlayHandled();
}

// Service Worker 注册
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js')
        .then((registration) => {
          console.log('[SW] Service Worker registered:', registration.scope);

          // 监听更新
          registration.addEventListener('updatefound', () => {
            const newWorker = registration.installing;
            if (newWorker) {
              newWorker.addEventListener('statechange', () => {
                if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                  // 有新版本可用
                  console.log('[SW] New version available');
                  // 可以在这里显示更新提�?
                }
              });
            }
          });
        })
        .catch((error) => {
          console.error('[SW] Service Worker registration failed:', error);
        });
  });
}

// 监听 Service Worker 控制变化
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    console.log('[SW] Controller changed, reloading page');
    window.location.reload();
  });
}
