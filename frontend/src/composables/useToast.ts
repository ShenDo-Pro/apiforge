import { reactive } from "vue";

// 极简全局 toast：全局单例数组，任意组件调用 useToast() 即可。
export interface Toast {
  id: number;
  type: "success" | "error" | "info";
  message: string;
}

export const toasts = reactive<Toast[]>([]);
let seq = 0;

function push(type: Toast["type"], message: string, timeout = 3200) {
  const id = ++seq;
  toasts.push({ id, type, message });
  setTimeout(() => {
    const i = toasts.findIndex((t) => t.id === id);
    if (i >= 0) toasts.splice(i, 1);
  }, timeout);
}

export function useToast() {
  return {
    success: (m: string) => push("success", m),
    error: (m: string) => push("error", m),
    info: (m: string) => push("info", m),
  };
}
