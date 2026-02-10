<script setup lang="ts">
import { ref, onMounted, watch, computed } from "vue";
import type { WidgetConfig } from "../types";
import { useMainStore } from "@/stores/main";

const props = defineProps<{
  widget: WidgetConfig;
}>();

const store = useMainStore();
const isEditing = ref(false);

// Default data if empty
const htmlContent = ref(
  props.widget.data?.html || '<div class="my-component">Hello Custom Widget</div>',
);
const cssContent = ref(
  props.widget.data?.css || ".my-component { color: blue; font-weight: bold; }",
);

// Safe way to inject styles
const styleId = computed(() => `style-${props.widget.id}`);

const applyStyles = () => {
  let styleEl = document.getElementById(styleId.value) as HTMLStyleElement;
  if (!styleEl) {
    styleEl = document.createElement("style");
    styleEl.id = styleId.value;
    document.head.appendChild(styleEl);
  }

  // Scope CSS to this widget to prevent bleeding
  // This is a naive scoping: preprend #widget-id to rules.
  // A better way is using Shadow DOM, but let's try a simple prefix first or just raw CSS if the user knows what they are doing.
  // User asked for "custom CSS component", they might want full control.
  // But to be safe and nice, let's wrap it in a scope if possible, or just inject it.
  // Given "upload their own custom CSS components", raw CSS injection is probably expected.
  // But to avoid messing up the whole app, we should probably scope it.
  // Let's try to wrap it in a specific class or ID.

  const scopedCss = cssContent.value.replace(
    /([^\r\n,{}]+)(,(?=[^}]*{)|\s*{)/g,
    `#widget-${props.widget.id} $1$2`,
  );

  styleEl.textContent = scopedCss;
};

const save = () => {
  const widget = store.widgets.find((w) => w.id === props.widget.id);
  if (widget) {
    widget.data = {
      html: htmlContent.value,
      css: cssContent.value,
    };
    store.saveData();
  }
  isEditing.value = false;
  applyStyles();
};

const copyPrompt = () => {
  const text = `请帮我写一个简洁的 HTML/CSS 卡片组件。
功能：[在此输入你的需求，如：显示当前日期和一句名言]
要求：
1. 容器宽高自适应，内容居中。
2. 风格现代简约，圆角设计。
3. 请分别提供 HTML 和 CSS 代码。`;
  navigator.clipboard.writeText(text).then(() => {
    alert("提示词已复制到剪贴板，快去发送给 AI 吧！");
  });
};

const toggleEdit = () => {
  isEditing.value = !isEditing.value;
};

// File Upload Handling
const handleFileUpload = (event: Event) => {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = (e) => {
    try {
      const content = e.target?.result as string;
      // Assume JSON format { "html": "...", "css": "..." }
      // Or maybe just a text file with sections?
      // Let's support JSON for now as it's structured.
      try {
        const json = JSON.parse(content);
        if (json.html) htmlContent.value = json.html;
        if (json.css) cssContent.value = json.css;
      } catch {
        // If not JSON, maybe just treat it as HTML?
        htmlContent.value = content;
      }
    } catch (err) {
      console.error("Failed to read file", err);
    }
  };
  reader.readAsText(file);
};

onMounted(() => {
  applyStyles();
});

watch([htmlContent, cssContent], () => {
  // Optional: Live preview if we want, but might be heavy.
  // applyStyles();
});
</script>

<template>
  <div
    :id="`widget-${widget.id}`"
    class="w-full h-full relative group overflow-hidden bg-transparent rounded-xl"
  >
    <!-- View Mode -->
    <div
      v-if="!isEditing"
      class="w-full h-full overflow-auto custom-content"
      v-html="htmlContent"
    ></div>

    <!-- Edit Overlay Button -->
    <button
      @click="toggleEdit"
      class="absolute top-2 right-2 z-50 p-1.5 bg-gray-100 hover:bg-gray-200 rounded-lg opacity-0 group-hover:opacity-100 transition-opacity text-gray-600"
      title="编辑组件"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
      </svg>
    </button>

    <!-- Edit Mode -->
    <div
      v-if="isEditing"
      class="absolute inset-0 z-40 bg-white flex flex-col p-4 gap-2 overflow-auto"
    >
      <div class="flex items-center justify-between mb-2">
        <h3 class="font-bold text-gray-700">编辑自定义组件</h3>
        <div class="flex gap-2">
          <label
            class="cursor-pointer px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded text-xs flex items-center gap-1 text-gray-900"
          >
            <span>📂 导入</span>
            <input type="file" accept=".json,.txt" class="hidden" @change="handleFileUpload" />
          </label>
          <button
            @click="save"
            class="px-3 py-1 bg-blue-500 text-white rounded text-xs hover:bg-blue-600"
          >
            保存
          </button>
          <button
            @click="toggleEdit"
            class="px-2 py-1 bg-gray-200 text-gray-700 rounded text-xs hover:bg-gray-300"
          >
            取消
          </button>
        </div>
      </div>

      <div class="flex-1 flex flex-col gap-1 min-h-0">
        <label class="text-xs font-semibold text-gray-500">HTML</label>
        <textarea
          v-model="htmlContent"
          class="flex-1 p-2 border rounded font-mono text-xs resize-none focus:border-blue-500 outline-none text-gray-900"
          placeholder="<div>Content</div>"
        ></textarea>
      </div>

      <div class="flex-1 flex flex-col gap-1 min-h-0">
        <label class="text-xs font-semibold text-gray-500">CSS</label>
        <textarea
          v-model="cssContent"
          class="flex-1 p-2 border rounded font-mono text-xs resize-none focus:border-blue-500 outline-none text-gray-900"
          placeholder=".class { color: red; }"
        ></textarea>
      </div>
      <p class="text-[10px] text-gray-400 mt-1">
        提示: CSS 选择器会自动加上前缀以避免污染全局样式，但建议使用独特的类名。
      </p>

      <!-- AI Helper -->
      <div
        class="mt-2 p-2 bg-gradient-to-r from-purple-50 to-blue-50 rounded-lg border border-purple-100"
      >
        <div class="flex items-center gap-1 mb-1.5">
          <span class="text-sm">🤖</span>
          <span class="text-xs font-bold text-purple-700">AI 辅助生成</span>
        </div>
        <p class="text-[10px] text-gray-600 mb-1.5">
          复制下方提示词给 AI (如 ChatGPT/DeepSeek)，即可快速生成组件代码：
        </p>
        <div
          class="bg-white p-2 rounded border border-purple-100 text-[10px] text-gray-500 font-mono break-all cursor-pointer hover:border-purple-300 transition-colors relative group"
          @click="copyPrompt"
          title="点击复制提示词"
        >
          <div
            class="hidden group-hover:block absolute right-1 top-1 bg-purple-100 text-purple-600 px-1 py-0.5 rounded text-[10px]"
          >
            点击复制
          </div>
          <span class="text-purple-600 select-none">Prompt: </span>请帮我写一个简洁的 HTML/CSS
          卡片组件...
        </div>
      </div>
    </div>
  </div>
</template>
