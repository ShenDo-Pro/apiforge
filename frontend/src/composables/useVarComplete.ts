import { ref, computed } from "vue";
import { useEnvironmentStore } from "@/stores/environment";
import type { VarSuggestion } from "@/stores/environment";

// 变量自动补全：在任意 textarea/input 中输入 `{{` 时弹出可用变量，
// 带「作用域来源徽标 + 解析值预览」，键盘上下/回车/Tab 插入，Esc/失焦关闭。
export function useVarComplete() {
  const envStore = useEnvironmentStore();
  const el = ref<HTMLTextAreaElement | HTMLInputElement | null>(null);
  const show = ref(false);
  const top = ref(0);
  const left = ref(0);
  const activeIndex = ref(0);
  const query = ref("");

  const items = computed<VarSuggestion[]>(() => {
    const q = query.value.toLowerCase();
    const list = envStore.suggestions.filter(
      (s) => s.key.toLowerCase().includes(q) || s.value.toLowerCase().includes(q)
    );
    return list.slice(0, 12);
  });

  // 取光标前最近的 {{ 片段，返回 { token, start } 或 null
  function beforeCaret(): { token: string; start: number } | null {
    const node = el.value;
    if (!node) return null;
    const pos = node.selectionStart ?? 0;
    const text = node.value.slice(0, pos);
    const open = text.lastIndexOf("{{");
    if (open < 0) return null;
    // 之间不能有关闭 }}，否则已结束
    const between = text.slice(open + 2);
    if (between.includes("}}")) return null;
    return { token: between, start: open };
  }

  function update() {
    const bc = beforeCaret();
    if (!bc) {
      show.value = false;
      return;
    }
    query.value = bc.token;
    activeIndex.value = 0;
    // 计算弹层位置（行高近似）
    const node = el.value as HTMLTextAreaElement;
    const rect = node.getBoundingClientRect();
    let caretTop = rect.bottom + 4;
    let caretLeft = rect.left + 8;
    try {
      // textarea 用镜像 div 估算，这里退化为固定在输入框下方，足够可用
      caretTop = rect.bottom + 6;
      caretLeft = Math.min(rect.left + 8, rect.right - 240);
    } catch {
      /* ignore */
    }
    top.value = caretTop;
    left.value = Math.max(8, caretLeft);
    show.value = items.value.length > 0;
  }

  function apply(item: VarSuggestion) {
    const node = el.value;
    if (!node) return;
    const pos = node.selectionStart ?? 0;
    const bc = beforeCaret();
    if (!bc) return;
    const before = node.value.slice(0, bc.start);
    const after = node.value.slice(pos);
    const insert = `{{${item.key}}}`;
    node.value = before + insert + after;
    const newPos = (before + insert).length;
    node.setSelectionRange(newPos, newPos);
    // 触发 input 事件让 v-model 同步
    node.dispatchEvent(new Event("input", { bubbles: true }));
    show.value = false;
  }

  function onInput() {
    update();
  }
  function onClick() {
    update();
  }
  function onKeyup() {
    update();
  }
  function onKeydown(e: KeyboardEvent) {
    if (!show.value) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      activeIndex.value = Math.min(activeIndex.value + 1, items.value.length - 1);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      activeIndex.value = Math.max(activeIndex.value - 1, 0);
    } else if (e.key === "Enter" || e.key === "Tab") {
      e.preventDefault();
      if (items.value[activeIndex.value]) apply(items.value[activeIndex.value]);
    } else if (e.key === "Escape") {
      show.value = false;
    }
  }
  function onBlur() {
    // 延迟关闭，给点击选项留时间
    setTimeout(() => (show.value = false), 120);
  }

  return {
    el,
    show,
    top,
    left,
    activeIndex,
    items,
    onInput,
    onClick,
    onKeyup,
    onKeydown,
    onBlur,
    apply,
  };
}
