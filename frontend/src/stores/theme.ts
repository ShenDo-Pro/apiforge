import { defineStore } from "pinia";
import { ref } from "vue";

export type Theme = "dark" | "light";

const STORAGE_KEY = "theme";

// 默认暗色（保持历史观感）；用户一旦显式切换则持久化并优先
function resolveInitial(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY);
  return stored === "light" ? "light" : "dark";
}

function applyTheme(theme: Theme) {
  const root = document.documentElement;
  root.classList.remove("light", "dark");
  root.classList.add(theme);
  root.style.colorScheme = theme;
}

export const useThemeStore = defineStore("theme", () => {
  const theme = ref<Theme>(resolveInitial());

  function setTheme(next: Theme) {
    theme.value = next;
    localStorage.setItem(STORAGE_KEY, next);
    applyTheme(next);
  }

  function toggle() {
    setTheme(theme.value === "dark" ? "light" : "dark");
  }

  // 在挂载前调用，使 DOM class 与 store 状态一致（配合 index.html 内联脚本避免闪烁）
  function init() {
    applyTheme(theme.value);
  }

  return { theme, setTheme, toggle, init };
});
