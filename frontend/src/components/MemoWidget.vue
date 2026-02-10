<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import type { WidgetConfig } from "@/types";
import { useMainStore } from "../stores/main";
import MemoEditor from "./Memo/MemoEditor.vue";
import MemoToolbar from "./Memo/MemoToolbar.vue";
import { useMemoPersistence } from "./Memo/useMemoPersistence";

const props = defineProps<{ widget: WidgetConfig }>();
const store = useMainStore();

// State
const mode = ref<"simple" | "rich">("simple");
const localData = ref(""); // Stores HTML for rich mode or text for simple mode
const editorRef = ref<InstanceType<typeof MemoEditor> | null>(null);
const isEditing = ref(false);
const localUpdatedAt = ref(0);
const lastInputAt = ref(0);
const isBroadcasting = ref(false);
const isPageVisible = ref(document.visibilityState === "visible");

// Persistence
const { saveToIndexedDB, loadFromIndexedDB, status, progress } = useMemoPersistence(
  props.widget.id,
  localData,
  mode
);

// Toast State
const showToast = ref(false);
const toastMessage = ref("");

// Computed Styles
const containerStyle = computed(() => ({
  backgroundColor: `rgba(254, 249, 195, ${props.widget.opacity ?? 0.9})`,
  color: props.widget.textColor || "#374151",
}));

// Methods
const handleCommand = (cmd: string, val?: string) => {
  editorRef.value?.execCommand(cmd, val);
};

const triggerSave = async () => {
  await saveToIndexedDB();
  if (status.value === "success") {
    // Triple Feedback 2: Toast
    toastMessage.value = "已保存，刷新不丢失";
    showToast.value = true;
    setTimeout(() => (showToast.value = false), 3000);
  }
  saveToServer(true);
};

const toggleMode = () => {
  mode.value = mode.value === "simple" ? "rich" : "simple";
  const payload = buildPayload();
  localUpdatedAt.value = payload.updatedAt;
  saveToServer(true);
  if (store.isLogged) {
    store.socket?.emit("memo:update", {
      token: store.token || localStorage.getItem("flat-nas-token"),
      widgetId: props.widget.id,
      content: payload,
    });
  }
};

const parsePayload = (payload: WidgetConfig["data"]) => {
  let content = "";
  let payloadMode: "simple" | "rich" = mode.value;
  let updatedAt = 0;

  if (typeof payload === "string") {
    content = payload;
  } else if (payload && typeof payload === "object") {
    const data = payload as Record<string, unknown>;
    if (typeof data.content === "string") {
      content = data.content;
    } else if (typeof data.rich === "string") {
      content = data.rich;
    } else if (typeof data.simple === "string") {
      content = data.simple;
    }
    if (data.mode === "simple" || data.mode === "rich") {
      payloadMode = data.mode;
    }
    if (typeof data.updatedAt === "number") {
      updatedAt = data.updatedAt;
    }
  }

  return { content, mode: payloadMode, updatedAt };
};

const buildPayload = () => ({
  content: localData.value,
  mode: mode.value,
  updatedAt: Date.now(),
});

let serverSaveTimer: ReturnType<typeof setTimeout> | null = null;
let broadcastTimer: ReturnType<typeof setTimeout> | null = null;
const ACTIVE_INPUT_WINDOW = 800;
const POLL_ACTIVE_INTERVAL = 800;
const POLL_IDLE_INTERVAL = 5000;
const saveToServer = (immediate = false) => {
  if (!store.isLogged) return;
  const doSave = () => {
    const payload = buildPayload();
    localUpdatedAt.value = payload.updatedAt;
    store.saveWidget(props.widget.id, payload);
  };

  if (immediate) {
    doSave();
    return;
  }

  if (serverSaveTimer) clearTimeout(serverSaveTimer);
  serverSaveTimer = setTimeout(() => {
    serverSaveTimer = null;
    doSave();
  }, 800);
};

const applyRemotePayload = (payload: WidgetConfig["data"]) => {
  const parsed = parsePayload(payload);
  if (!parsed.content) return;
  if (isEditing.value) return;
  if (parsed.updatedAt && parsed.updatedAt <= localUpdatedAt.value) return;
  if (parsed.content !== localData.value) {
    localData.value = parsed.content;
    mode.value = parsed.mode;
    localUpdatedAt.value = parsed.updatedAt || Date.now();
  }
};

const scheduleBroadcast = () => {
  if (!isBroadcasting.value || !store.isLogged) return;
  if (broadcastTimer) clearTimeout(broadcastTimer);
  broadcastTimer = setTimeout(() => {
    if (!isBroadcasting.value || !store.isLogged) return;
    const payload = buildPayload();
    store.socket?.emit("memo:update", {
      token: store.token || localStorage.getItem("flat-nas-token"),
      widgetId: props.widget.id,
      content: payload,
    });
  }, 300);
};

const updateSyncMode = () => {
  const active =
    isEditing.value && Date.now() - lastInputAt.value <= ACTIVE_INPUT_WINDOW;
  const targetInterval = isPageVisible.value ? POLL_ACTIVE_INTERVAL : POLL_IDLE_INTERVAL;
  if (active) {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
    isBroadcasting.value = true;
  } else {
    isBroadcasting.value = false;
    if (!pollTimer) {
      pollTimer = setInterval(pollRemote, targetInterval);
    } else if (pollTimer && targetInterval !== currentPollInterval) {
      clearInterval(pollTimer);
      pollTimer = setInterval(pollRemote, targetInterval);
    }
  }
  currentPollInterval = targetInterval;
};

const handleInputActivity = () => {
  lastInputAt.value = Date.now();
  updateSyncMode();
  scheduleBroadcast();
};

// Initial Load
loadFromIndexedDB().then(() => {
  if (!localData.value && props.widget.data) {
     // Fallback to widget prop data if IDB is empty
     if (typeof props.widget.data === 'string') {
        localData.value = props.widget.data;
     } else {
        const d = props.widget.data as { rich?: string; simple?: string; mode?: "simple" | "rich" };
        localData.value = d.rich || d.simple || "";
        mode.value = d.mode || "simple";
     }
     localUpdatedAt.value = Date.now();
  }
});

// Auto-save wrapper (optional, but requested "Persistent Button" behavior implies manual action is the focus, 
// but user data usually needs autosave. The prompt emphasizes the "Persistent Button" feedback.)
// I will keep manual save for the "Persistent Button" requirement demo, and maybe autosave silently.
let autoSaveTimer: ReturnType<typeof setTimeout> | undefined;
watch([localData, mode], () => {
  clearTimeout(autoSaveTimer);
  autoSaveTimer = setTimeout(() => {
    saveToIndexedDB();
    saveToServer();
  }, 800);
});

watch(
  () => props.widget.data,
  (data) => {
    if (!data) return;
    applyRemotePayload(data);
  }
);

let pollTimer: ReturnType<typeof setInterval> | null = null;
let idleCheckTimer: ReturnType<typeof setInterval> | null = null;
let currentPollInterval = POLL_ACTIVE_INTERVAL;
const handleVisibilityChange = () => {
  isPageVisible.value = document.visibilityState === "visible";
  updateSyncMode();
};
const pollRemote = async () => {
  if (!store.isLogged || !store.isConnected || isEditing.value) return;
  const id = props.widget.id;
  if (!id) return;
  if (import.meta.env.MODE === "test") return;
  try {
    const res = await fetch(`/api/widgets/${id}`, { headers: store.getHeaders() });
    if (!res.ok) return;
    const data = await res.json();
    if (data?.data) {
      applyRemotePayload(data.data);
    }
  } catch {
    return;
  }
};

onMounted(() => {
  updateSyncMode();
  idleCheckTimer = setInterval(updateSyncMode, 1000);
  document.addEventListener("visibilitychange", handleVisibilityChange);
});

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer);
  if (idleCheckTimer) clearInterval(idleCheckTimer);
  if (serverSaveTimer) clearTimeout(serverSaveTimer);
  if (autoSaveTimer) clearTimeout(autoSaveTimer);
  if (broadcastTimer) clearTimeout(broadcastTimer);
  document.removeEventListener("visibilitychange", handleVisibilityChange);
});

const handleFocus = () => {
  isEditing.value = true;
  updateSyncMode();
};

const handleBlur = () => {
  isEditing.value = false;
  updateSyncMode();
};

</script>

<template>
  <div
    class="w-full h-full p-4 rounded-2xl backdrop-blur border border-white/10 relative group flex flex-col transition-colors duration-300 overflow-hidden"
    :style="containerStyle"
  >
    <!-- Triple Feedback 3: Top Progress Bar -->
    <div 
      v-if="progress > 0" 
      class="absolute top-0 left-0 h-1 bg-[#0052D9] transition-all duration-300 rounded-t-2xl z-20"
      :style="{ width: `${progress}%`, opacity: progress === 100 ? 0 : 1 }"
    ></div>

    <!-- Page Curl Toggle -->
    <div 
      class="absolute top-0 left-0 w-3 h-3 cursor-pointer z-50 overflow-hidden group/curl"
      @click="toggleMode"
      title="切换模式 (Switch Mode)"
    >
      <!-- The shadow of the curl -->
      <div class="absolute top-0 left-0 w-0 h-0 border-t-[12px] border-r-[12px] border-t-white/0 border-r-black/20 transform translate-x-0.5 translate-y-0.5 blur-[1px] transition-all duration-300 group-hover/curl:scale-105"></div>
      <!-- The curled part -->
      <div class="absolute top-0 left-0 w-0 h-0 border-t-[12px] border-r-[12px] border-t-white/90 border-r-transparent shadow-sm transition-all duration-300 group-hover/curl:border-t-white group-hover/curl:scale-105"></div>
    </div>

    <!-- Header / Controls -->
    <div class="flex items-center justify-between mb-2 z-10">
      <div class="flex gap-1">
        <!-- Mode indicator removed -->
      </div>

      <!-- Persistent Save Button -->
      <!-- Triple Feedback 1: Button Pulse Animation -->
      <button
        v-if="mode === 'rich'"
        @click="triggerSave"
        class="
          flex items-center gap-1 px-3 py-1 rounded-md text-xs font-medium text-white transition-all duration-300
          focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-[#0052D9]
        "
        :class="[
          status === 'success' ? 'bg-green-500 animate-pulse' : 'bg-[#0052D9] hover:brightness-110',
          status === 'saving' ? 'opacity-70 cursor-wait' : ''
        ]"
        :disabled="status === 'saving'"
        title="保存 (Save)"
      >
        <svg v-if="status === 'success'" class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
        </svg>
        <svg v-else class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
        </svg>
        <span>{{ status === 'success' ? '已保存' : '保存' }}</span>
      </button>
    </div>

    <!-- Content Area -->
    <div class="flex-1 min-h-0 relative">
      <Transition name="page-tear" mode="out-in">
        <div :key="mode" class="w-full h-full">
          <textarea
            v-if="mode === 'simple'"
            v-model="localData"
            class="w-full h-full bg-transparent resize-none outline-none text-sm placeholder-gray-600 font-medium p-2"
            :placeholder="store.isLogged ? '写点什么...' : '请先登录'"
            :readonly="!store.isLogged"
            @focus="handleFocus"
            @blur="handleBlur"
            @input="handleInputActivity"
          ></textarea>
          
          <MemoEditor
            v-else
            ref="editorRef"
            v-model:content="localData"
            :editable="store.isLogged"
            :placeholder="store.isLogged ? '在此输入内容...' : '请先登录'"
            @focus="handleFocus"
            @blur="handleBlur"
            @input="handleInputActivity"
          />
        </div>
      </Transition>
    </div>

    <!-- Toolbar (Rich Mode Only) -->
    <MemoToolbar v-if="mode === 'rich'" @command="handleCommand" />

    <!-- Triple Feedback 2: Toast (Overlay) -->
    <Transition
      enter-active-class="transition ease-out duration-300"
      enter-from-class="transform opacity-0 translate-y-2"
      enter-to-class="transform opacity-100 translate-y-0"
      leave-active-class="transition ease-in duration-200"
      leave-from-class="transform opacity-100 translate-y-0"
      leave-to-class="transform opacity-0 translate-y-2"
    >
      <div 
        v-if="showToast"
        class="absolute top-12 right-4 z-30 bg-gray-800 text-white text-xs px-3 py-1.5 rounded shadow-lg flex items-center gap-2"
      >
        <svg class="w-3 h-3 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        {{ toastMessage }}
      </div>
    </Transition>
  </div>
</template>

<style scoped>
/* Scrollbar styling if needed */
textarea::-webkit-scrollbar,
div::-webkit-scrollbar {
  width: 6px;
}
textarea::-webkit-scrollbar-thumb,
div::-webkit-scrollbar-thumb {
  background-color: rgba(0, 0, 0, 0.1);
  border-radius: 3px;
}

/* Page Tear Animation */
.page-tear-leave-active {
  animation: tear-off 0.6s ease-in forwards;
  transform-origin: top left;
  position: absolute; /* Prevent layout shift */
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 10;
  pointer-events: none; /* Prevent clicks during animation */
}

.page-tear-enter-active {
  animation: fade-in 0.6s ease-out;
}

@keyframes tear-off {
  0% {
    transform: rotate(0deg) translateY(0);
    opacity: 1;
    mask-image: linear-gradient(to bottom, black 100%, transparent 100%);
    -webkit-mask-image: linear-gradient(to bottom, black 100%, transparent 100%);
  }
  100% {
    transform: rotate(-10deg) translateY(120%) translateX(-20px);
    opacity: 0;
    mask-image: linear-gradient(to bottom, black 50%, transparent 100%);
    -webkit-mask-image: linear-gradient(to bottom, black 50%, transparent 100%);
  }
}

@keyframes fade-in {
  0% { opacity: 0; }
  100% { opacity: 1; }
}
</style>
