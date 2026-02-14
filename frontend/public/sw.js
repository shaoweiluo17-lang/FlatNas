const CACHE_VERSION = 'v1.0.0';
const CACHE_PREFIX = 'flatnas-';
const STATIC_CACHE = `${CACHE_PREFIX}static-${CACHE_VERSION}`;
const CONFIG_CACHE = `${CACHE_PREFIX}config-${CACHE_VERSION}`;
const API_CACHE = `${CACHE_PREFIX}api-${CACHE_VERSION}`;

// 需要预缓存的静态资源
const PRECACHE_URLS = [
    '/',
    '/index.html',
    '/favicon.ico',
    '/assets/icons/*.svg',
    '/assets/logo.svg'
];

// 需要缓存的静态资源模式
const STATIC_PATTERNS = [
    /\.js$/,
    /\.css$/,
    /\.svg$/,
    /\.png$/,
    /\.jpg$/,
    /\.jpeg$/,
    /\.gif$/,
    /\.webp$/,
    /\.ico$/,
    /\.woff$/,
    /\.woff2$/,
    /\.ttf$/,
    /\.eot$/
];

// 需要缓存的 API 端点
const CACHEABLE_APIS = [
    '/api/config',
    '/api/system-config',
    '/api/data',
    '/api/widgets/memo',
    '/api/widgets/todo'
];

// 安装 Service Worker
self.addEventListener('install', (event) => {
    console.log('[SW] Install event');

    event.waitUntil(
        caches.open(STATIC_CACHE).then((cache) => {
            return cache.addAll(
                PRECACHE_URLS.filter(url => url.includes('*') ? false : url)
            );
        }).then(() => {
            return self.skipWaiting();
        })
    );
});

// 激活 Service Worker
self.addEventListener('activate', (event) => {
    console.log('[SW] Activate event');

    event.waitUntil(
        caches.keys().then((cacheNames) => {
            return Promise.all(
                cacheNames
                    .filter((cacheName) => {
                        return cacheName.startsWith(CACHE_PREFIX) &&
                            !cacheName.includes(CACHE_VERSION);
                    })
                    .map((cacheName) => {
                        console.log('[SW] Deleting old cache:', cacheName);
                        return caches.delete(cacheName);
                    })
            );
        }).then(() => {
            return self.clients.claim();
        })
    );
});

// 拦截网络请求
self.addEventListener('fetch', (event) => {
    const url = new URL(event.request.url);

    // 跳过非 GET 请求
    if (event.request.method !== 'GET') {
        return;
    }

    // 跳过 chrome-extension 等
    if (!url.protocol.startsWith('http')) {
        return;
    }

    // 处理静态资源
    if (isStaticResource(url.pathname)) {
        event.respondWith(handleStaticResource(event.request));
        return;
    }

    // 处理 API 请求
    if (isCacheableAPI(url.pathname)) {
        event.respondWith(handleAPIRequest(event.request));
        return;
    }

    // 处理导航请求
    if (event.request.mode === 'navigate') {
        event.respondWith(handleNavigation(event.request));
        return;
    }
});

// 处理静态资源
async function handleStaticResource(request) {
    const cache = await caches.open(STATIC_CACHE);
    const cachedResponse = await cache.match(request);

    if (cachedResponse) {
        // 后台更新
        fetch(request).then(response => {
            if (response.ok) {
                cache.put(request, response);
            }
        });
        return cachedResponse;
    }

    const response = await fetch(request);
    if (response.ok) {
        const responseClone = response.clone();
        cache.put(request, responseClone);
    }
    return response;
}

// 处理 API 请求（Network First）
async function handleAPIRequest(request) {
    try {
        const response = await fetch(request);

        if (response.ok) {
            const cache = await caches.open(API_CACHE);
            const responseClone = response.clone();
            cache.put(request, responseClone);
        }

        return response;
    } catch (error) {
        // 网络失败，使用缓存
        const cache = await caches.open(API_CACHE);
        const cachedResponse = await cache.match(request);

        if (cachedResponse) {
            return cachedResponse;
        }

        // 返回离线响应
        return new Response(
            JSON.stringify({
                error: 'offline',
                message: '网络已断开，使用缓存数据',
                cached: true
            }),
            {
                status: 503,
                headers: {
                    'Content-Type': 'application/json',
                    'X-From-Cache': 'true'
                }
            }
        );
    }
}

// 处理导航请求（Network First，离线时返回首页）
async function handleNavigation(request) {
    try {
        const response = await fetch(request);

        if (response.ok) {
            return response;
        }
    } catch (error) {
        console.log('[SW] Navigation failed, returning cached index');
    }

    // 返回缓存的首页
    const cache = await caches.open(STATIC_CACHE);
    const cachedIndex = await cache.match('/');

    if (cachedIndex) {
        return cachedIndex;
    }

    return new Response('离线模式', {
        status: 503,
        headers: { 'Content-Type': 'text/plain' }
    });
}

// 判断是否为静态资源
function isStaticResource(pathname) {
    return STATIC_PATTERNS.some(pattern => pattern.test(pathname));
}

// 判断是否为可缓存的 API
function isCacheableAPI(pathname) {
    return CACHEABLE_APIS.some(api => pathname.startsWith(api));
}

// 监听消息
self.addEventListener('message', (event) => {
    if (event.data && event.data.type === 'SKIP_WAITING') {
        self.skipWaiting();
    }

    if (event.data && event.data.type === 'CLEAR_CACHE') {
        clearAllCaches().then(() => {
            event.ports[0].postMessage({ success: true });
        });
    }
});

// 清除所有缓存
async function clearAllCaches() {
    const cacheNames = await caches.keys();
    return Promise.all(
        cacheNames
            .filter(name => name.startsWith(CACHE_PREFIX))
            .map(name => caches.delete(name))
    );
}

// 获取缓存信息
async function getCacheInfo() {
    const cacheNames = await caches.keys();
    const flatnasCaches = cacheNames.filter(name => name.startsWith(CACHE_PREFIX));

    let totalSize = 0;
    const cacheDetails = {};

    for (const cacheName of flatnasCaches) {
        const cache = await caches.open(cacheName);
        const keys = await cache.keys();
        let cacheSize = 0;

        for (const request of keys) {
            const response = await cache.match(request);
            if (response) {
                const blob = await response.blob();
                cacheSize += blob.size;
            }
        }

        totalSize += cacheSize;
        cacheDetails[cacheName] = {
            count: keys.length,
            size: cacheSize
        };
    }

    return {
        totalSize,
        count: flatnasCaches.length,
        details: cacheDetails
    };
}