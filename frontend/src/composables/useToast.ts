import { ref, h, render } from "vue";
import ToastNotification from "../components/ToastNotification.vue";

export type ToastType = "success" | "error" | "info" | "warning";

const toastContainer = ref<HTMLElement | null>(null);

function getContainer() {
  if (!toastContainer.value) {
    toastContainer.value = document.createElement("div");
    toastContainer.value.id = "toast-container";
    document.body.appendChild(toastContainer.value);
  }
  return toastContainer.value;
}

export function useToast() {
  const show = (
    message: string,
    type: ToastType = "info",
    duration: number = 3000
  ) => {
    const container = getContainer();
    const toastWrapper = document.createElement("div");
    container.appendChild(toastWrapper);

    const close = () => {
      render(null, toastWrapper);
      if (container.contains(toastWrapper)) {
        container.removeChild(toastWrapper);
      }
    };

    const vnode = h(ToastNotification, {
      message,
      type,
      duration,
      onClose: close,
    });

    render(vnode, toastWrapper);
  };

  return {
    show,
    success: (message: string, duration?: number) => show(message, "success", duration),
    error: (message: string, duration?: number) => show(message, "error", duration),
    info: (message: string, duration?: number) => show(message, "info", duration),
    warning: (message: string, duration?: number) => show(message, "warning", duration),
  };
}