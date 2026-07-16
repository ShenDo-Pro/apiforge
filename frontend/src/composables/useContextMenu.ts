import { computed, onUnmounted, ref, watch, type Ref } from "vue";

// 全局单例：同一时刻只允许一个右键菜单处于打开状态，
// 解决「在左侧不同位置连续右键时旧菜单不关闭」的问题。
const activeKey = ref<string | null>(null);

export function useContextMenu(key: string, menuEl: Ref<HTMLElement | null>) {
  const isOpen = computed(() => activeKey.value === key);

  function open() {
    activeKey.value = key;
  }

  function close() {
    if (activeKey.value === key) activeKey.value = null;
  }

  function onDocPointer(e: MouseEvent) {
    // 点击/右键发生在当前菜单内部：交给菜单自身处理，不关闭
    const el = menuEl.value;
    if (el && el.contains(e.target as Node)) return;
    // 命中菜单外部：先关闭当前菜单。
    // 右键场景下面由目标自身的 @contextmenu 去打开新菜单。
    close();
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === "Escape") close();
  }

  watch(isOpen, (openState) => {
    if (openState) {
      window.addEventListener("click", onDocPointer, true);
      window.addEventListener("contextmenu", onDocPointer, true);
      window.addEventListener("keydown", onKey, true);
    } else {
      window.removeEventListener("click", onDocPointer, true);
      window.removeEventListener("contextmenu", onDocPointer, true);
      window.removeEventListener("keydown", onKey, true);
    }
  });

  onUnmounted(() => {
    window.removeEventListener("click", onDocPointer, true);
    window.removeEventListener("contextmenu", onDocPointer, true);
    window.removeEventListener("keydown", onKey, true);
    // 组件卸载时若本节点菜单正打开，清掉全局单例状态，避免残留「某菜单已开」影响后续（L20）
    if (activeKey.value === key) activeKey.value = null;
  });

  return { isOpen, open, close };
}
