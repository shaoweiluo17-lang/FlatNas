<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from "vue";
import { useMainStore } from "../stores/main";
import { VueDraggable } from "vue-draggable-plus";
import type { RssFeed, WidgetConfig } from "@/types";

defineProps<{ widget: WidgetConfig }>();

const store = useMainStore();

interface RssItem {
  title: string;
  link: string;
  pubDate?: string;
  content?: string;
  contentSnippet?: string;
}

// Backend handles caching (6 hours). Frontend refreshes every 15 minutes.
const REFRESH_INTERVAL = 15 * 60 * 1000;

const activeFeedId = ref<string>("");
const list = ref<RssItem[]>([]);
const loading = ref(false);
const errorMsg = ref("");
let activeCleanup: (() => void) | undefined;
let refreshTimer: ReturnType<typeof setInterval> | undefined;

// Get enabled feeds
const enabledFeeds = computed(() => store.rssFeeds.filter((f) => f.enable));

// Draggable local state
const localFeeds = ref<RssFeed[]>([]);

watch(
  enabledFeeds,
  (newVal) => {
    // Only update localFeeds if length differs or IDs don't match (avoid resetting during drag if possible, 
    // though usually enabledFeeds won't change during drag unless store updates)
    // Simple deep sync is safer to ensure we have latest data
    const currentIds = localFeeds.value.map((f) => f.id).join(",");
    const newIds = newVal.map((f) => f.id).join(",");
    if (currentIds !== newIds) {
      localFeeds.value = [...newVal];
    }
  },
  { immediate: true, deep: true },
);

const onDragEnd = () => {
  // Reconstruct store.rssFeeds: new order of enabled + existing disabled
  const disabled = store.rssFeeds.filter((f) => !f.enable);
  store.rssFeeds = [...localFeeds.value, ...disabled];
  store.saveData();
};

// Watch for feed changes to reset/update
watch(
  enabledFeeds,
  (newFeeds) => {
    if (newFeeds.length > 0) {
      // If current active feed is gone, switch to first
      if (!newFeeds.find((f) => f.id === activeFeedId.value)) {
        const first = newFeeds[0];
        if (first) {
          activeFeedId.value = first.id;
          fetchFeed(first);
        }
      }
    } else {
      activeFeedId.value = "";
      list.value = [];
    }
  },
  { deep: true },
);

const fetchFeed = async (feed: RssFeed) => {
  if (!feed) return;
  
  // Cleanup previous listeners
  if (activeCleanup) {
    activeCleanup();
    activeCleanup = undefined;
  }

  // Clear previous timer if we are switching feeds manually
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = undefined;
  }

  activeFeedId.value = feed.id;
  errorMsg.value = "";

  // Always set loading true initially, backend is fast if cached
  loading.value = true;
  list.value = [];

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const onData = (payload: any) => {
    if (payload.url === feed.url) {
      const items = payload.data.items || [];
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      list.value = items.map((item: any) => ({
        title: item.title,
        link: item.link,
        pubDate: item.pubDate,
        contentSnippet: item.contentSnippet,
      }));
      loading.value = false;
      
      if (activeCleanup) {
        activeCleanup();
        activeCleanup = undefined;
      }
    }
  };

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const onError = (payload: any) => {
    if (payload.url === feed.url) {
      console.error(`Failed to load RSS: ${feed.title}`, payload.error);
      errorMsg.value = "加载失败";
      list.value = [];
      loading.value = false;
      
      if (activeCleanup) {
        activeCleanup();
        activeCleanup = undefined;
      }
    }
  };

  activeCleanup = () => {
    store.socket.off("rss:data", onData);
    store.socket.off("rss:error", onError);
  };

  store.socket.on("rss:data", onData);
  store.socket.on("rss:error", onError);
  
  // Use Socket.IO
  const doFetch = () => {
    store.socket.emit("rss:fetch", { url: feed.url });
  };

  doFetch();
  
  // Setup 15 min refresh interval
  refreshTimer = setInterval(doFetch, REFRESH_INTERVAL);
};

onMounted(() => {
  const first = enabledFeeds.value[0];
  if (first) {
    activeFeedId.value = first.id;
    fetchFeed(first);
  }
});

onUnmounted(() => {
  if (activeCleanup) {
    activeCleanup();
    activeCleanup = undefined;
  }
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = undefined;
  }
});

const handleScrollIsolation = (e: WheelEvent) => {
  const el = e.currentTarget as HTMLDivElement;
  const { scrollTop, scrollHeight, clientHeight } = el;
  const delta = e.deltaY;

  const isAtTop = scrollTop <= 0;
  const isAtBottom = scrollTop + clientHeight >= scrollHeight - 1;

  if ((isAtTop && delta < 0) || (isAtBottom && delta > 0)) {
    e.preventDefault();
    e.stopPropagation();
  }
};

const tabsRef = ref<HTMLDivElement | null>(null);

const handleHorizontalScroll = (e: WheelEvent) => {
  if (!tabsRef.value) return;
  if (e.deltaY !== 0) {
    tabsRef.value.scrollLeft += e.deltaY;
  }
};
</script>

<template>
  <div
    class="w-full h-full rounded-2xl backdrop-blur border border-white/10 overflow-hidden flex flex-col text-white relative transition-shadow"
    :style="{
      backgroundColor: `rgba(0,0,0,${Math.min(0.85, Math.max(0.15, widget?.opacity ?? 0.35))})`,
      color: '#fff',
    }"
  >
    <!-- Header / Tabs -->
    <VueDraggable
      ref="tabsRef"
      v-model="localFeeds"
      @wheel.prevent="handleHorizontalScroll"
      :animation="150"
      @end="onDragEnd"
      class="flex border-b border-white/10 bg-white/10 select-none overflow-x-auto custom-scrollbar"
    >
      <div
        v-if="enabledFeeds.length === 0"
        class="w-full py-2.5 text-xs text-white/60 text-center"
      >
        暂无订阅源
      </div>
      <button
        v-for="feed in localFeeds"
        :key="feed.id"
        @click="fetchFeed(feed)"
        class="flex-shrink-0 px-4 py-2.5 text-xs font-bold transition-all flex items-center justify-center gap-1.5 relative whitespace-nowrap cursor-move"
        :class="
          activeFeedId === feed.id
            ? 'text-white bg-white/15'
            : 'text-white/60 hover:bg-white/10 hover:text-white'
        "
      >
        <span>{{ feed.title }}</span>
        <div
          v-if="activeFeedId === feed.id"
          class="absolute bottom-0 left-0 right-0 h-0.5 bg-white/60"
        ></div>
      </button>
    </VueDraggable>

    <!-- Content -->
    <div class="flex-1 overflow-hidden relative">
      <div class="h-full overflow-y-auto custom-scrollbar p-0" @wheel="handleScrollIsolation">
        <div
          v-if="enabledFeeds.length === 0"
          class="h-full flex flex-col items-center justify-center text-white/60 p-4 text-center"
        >
          <span class="text-2xl mb-2">📡</span>
          <span class="text-xs">请在设置中添加并启用 RSS 订阅源</span>
        </div>

        <div
          v-else-if="loading && list.length === 0"
          class="p-8 text-center text-white/60 text-xs animate-pulse"
        >
          加载中...
        </div>

        <div v-else-if="errorMsg" class="p-8 text-center text-white/70 text-xs">
          {{ errorMsg }}
          <button
            @click="fetchFeed(enabledFeeds.find((f) => f.id === activeFeedId)!)"
            class="block mx-auto mt-2 text-white/80 hover:text-white hover:underline"
          >
            重试
          </button>
        </div>

        <div v-else class="flex flex-col py-1">
          <a
            v-for="(item, index) in list"
            :key="index"
            :href="item.link"
            target="_blank"
            class="block px-3 py-2 hover:bg-white/10 transition-colors group/item border-b border-white/10 last:border-0"
          >
            <div
              class="text-sm text-white/80 group-hover/item:text-white transition-colors font-medium line-clamp-2 mb-1"
            >
              {{ item.title }}
            </div>
            <div class="flex justify-between items-center">
              <div
                v-if="item.contentSnippet"
                class="text-[10px] text-white/50 line-clamp-1 flex-1 mr-2"
              >
                {{ item.contentSnippet }}
              </div>
              <div v-if="item.pubDate" class="text-[10px] text-white/40 whitespace-nowrap">
                {{ new Date(item.pubDate).toLocaleDateString() }}
              </div>
            </div>
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
  height: 4px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: rgba(0, 0, 0, 0.05);
  border-radius: 4px;
}
.custom-scrollbar:hover::-webkit-scrollbar-thumb {
  background-color: rgba(0, 0, 0, 0.1);
}
.no-scrollbar::-webkit-scrollbar {
  display: none;
}
.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
