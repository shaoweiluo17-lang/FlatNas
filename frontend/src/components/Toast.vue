<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";

export interface ToastProps {
  message: string;
  type?: "success" | "error" | "info" | "warning";
  duration?: number;
}

const props = withDefaults(defineProps<ToastProps>(), {
  type: "info",
  duration: 3000,
});

const emit = defineEmits(["close"]);

const visible = ref(false);
let timer: number | null = null;

onMounted(() => {
  visible.value = true;
  if (props.duration > 0) {
    timer = setTimeout(() => {
      visible.value = false;
      setTimeout(() => emit("close"), 300);
    }, props.duration);
  }
});

onUnmounted(() => {
  if (timer) clearTimeout(timer);
});

const colors = {
  success: "bg-green-500",
  error: "bg-red-500",
  info: "bg-blue-500",
  warning: "bg-yellow-500",
};

const icons = {
  success: "✓",
  error: "✕",
  info: "ℹ",
  warning: "⚠",
};
</script>

<template>
  <Transition
    enter-active-class="transition-all duration-300 ease-out"
    enter-from-class="opacity-0 translate-y-4"
    enter-to-class="opacity-100 translate-y-0"
    leave-active-class="transition-all duration-300 ease-in"
    leave-from-class="opacity-100 translate-y-0"
    leave-to-class="opacity-0 translate-y-4"
  >
    <div
      v-if="visible"
      :class="[
        'fixed top-4 left-1/2 -translate-x-1/2 px-6 py-3 rounded-xl shadow-lg text-white font-medium text-sm flex items-center gap-2 z-[100]',
        colors[type]
      ]"
    >
      <span class="text-lg">{{ icons[type] }}</span>
      <span>{{ message }}</span>
    </div>
  </Transition>
</template>