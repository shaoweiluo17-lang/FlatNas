<script setup lang="ts">
import { ref, watch, nextTick } from "vue";
import { useMainStore } from "../stores/main";
import { useToast } from "../composables/useToast";

const props = defineProps<{ show: boolean }>();
const emit = defineEmits(["update:show"]);
const store = useMainStore();
const toast = useToast();

const username = ref("");
const password = ref("");
const inviteCode = ref("");
const isRegister = ref(false);
const inputRef = ref<HTMLInputElement | null>(null);

// 表单错误提示
const formError = ref("");

// 监听打开：一旦打开，自动聚焦输入框，并清空旧密码
watch(
  () => props.show,
  (newVal) => {
    if (newVal) {
      username.value = "";
      password.value = "";
      isRegister.value = false;
      formError.value = "";
      nextTick(() => {
        // Focus username input if visible, else password
        if (store.systemConfig.authMode === "multi") {
          const input = document.querySelector('input[placeholder="用户名"]') as HTMLInputElement;
          if (input) input.focus();
          else inputRef.value?.focus();
        } else {
          inputRef.value?.focus();
        }
      });
    }
  },
);

const close = () => emit("update:show", false);

const handleSubmit = async () => {
  formError.value = "";

  // If single user mode, username can be empty (defaults to admin on server)
  if (store.systemConfig.authMode === "multi" && !username.value.trim()) {
    formError.value = "请输入用户名";
    return;
  }
  if (!password.value) {
    formError.value = "请输入密码";
    return;
  }

  try {
    if (isRegister.value) {
      // Check if registration is allowed
      if (!store.systemConfig.allowRegistration && !inviteCode.value.trim()) {
        formError.value = "注册功能已关闭，请输入邀请码或联系管理员";
        return;
      }
      await store.register(username.value, password.value, inviteCode.value);
      toast.success("注册成功，请登录");
      isRegister.value = false;
      password.value = "";
      inviteCode.value = "";
      formError.value = "";
    } else {
      const success = await store.login(username.value, password.value);
      if (success) {
        close();
      }
    }
  } catch (e: unknown) {
    const err = e as Error;
    const errorMsg = err.message || "操作失败！";

    // 根据错误类型显示不同提示
    if (errorMsg.includes("用户名") || errorMsg.includes("密码") || errorMsg.includes("邀请码") || errorMsg.includes("注册")) {
      formError.value = errorMsg;
    } else {
      toast.error(errorMsg);
    }
    password.value = "";
  }
};
</script>

<template>
  <div
    v-if="show"
    class="fixed inset-0 bg-black/40 backdrop-blur-sm z-50 flex items-center justify-center p-4"
  >
    <div
      class="bg-white rounded-2xl shadow-2xl w-full max-w-sm overflow-hidden transform transition-all scale-100"
    >
      <div
        class="px-6 py-4 border-b border-gray-100 flex justify-between items-center bg-gray-50/50"
      >
        <h3 class="text-lg font-bold text-gray-800 flex items-center gap-2">
          <span v-if="isRegister">👤 新用户注册</span>
          <template v-else>
            <img src="/ICON.PNG" class="w-6 h-6 object-contain" alt="lock" />
            <span>
              {{
                store.systemConfig.authMode === "single"
                  ? "管理员登录"
                  : "用户登录"
              }}
            </span>
          </template>
        </h3>
        <button @click="close" class="text-gray-400 hover:text-gray-600 text-2xl leading-none">
          &times;
        </button>
      </div>

      <div class="p-6">
        <!-- 内联错误提示 -->
        <Transition
          enter-active-class="transition-all duration-200 ease-out"
          enter-from-class="opacity-0 -translate-y-2"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition-all duration-200 ease-in"
          leave-from-class="opacity-100 translate-y-0"
          leave-to-class="opacity-0 -translate-y-2"
        >
          <div
            v-if="formError"
            class="mb-4 px-4 py-3 bg-red-50 border border-red-200 rounded-xl text-red-600 text-sm flex items-center gap-2"
          >
            <span class="text-red-500">⚠</span>
            <span>{{ formError }}</span>
          </div>
        </Transition>

        <div class="mb-5 space-y-4">
          <div v-if="store.systemConfig.authMode === 'multi'">
            <input
              v-model="username"
              type="text"
              placeholder="用户名"
              class="w-full px-4 py-3 rounded-xl border border-gray-200 focus:border-blue-500 focus:ring-4 focus:ring-blue-100 outline-none transition-all text-center text-lg tracking-widest"
              @keyup.enter="handleSubmit"
            />
          </div>
          <div>
            <input
              ref="inputRef"
              v-model="password"
              type="password"
              placeholder="密码"
              class="w-full px-4 py-3 rounded-xl border border-gray-200 focus:border-blue-500 focus:ring-4 focus:ring-blue-100 outline-none transition-all text-center text-lg tracking-widest"
              @keyup.enter="handleSubmit"
            />
          </div>
          <div v-if="isRegister && !store.systemConfig.allowRegistration">
            <input
              v-model="inviteCode"
              type="text"
              placeholder="邀请码 (注册需要)"
              class="w-full px-4 py-3 rounded-xl border border-gray-200 focus:border-blue-500 focus:ring-4 focus:ring-blue-100 outline-none transition-all text-center text-lg tracking-widest"
              @keyup.enter="handleSubmit"
            />
          </div>
        </div>

        <button
          @click="handleSubmit"
          class="w-full bg-gray-800 text-white py-3 rounded-xl font-bold hover:bg-black active:scale-95 transition-all shadow-lg"
        >
          {{ isRegister ? "注 册" : "登 录" }}
        </button>

        <div class="mt-4 text-center" v-if="store.systemConfig.authMode === 'multi'">
          <button
            @click="isRegister = !isRegister; formError = ''"
            class="text-sm text-gray-500 hover:text-gray-800 hover:underline transition-colors"
          >
            {{ isRegister ? "已有账号？去登录" : "没有账号？去注册" }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
