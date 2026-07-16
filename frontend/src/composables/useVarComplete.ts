import { ref, computed } from "vue";
import { useEnvironmentStore } from "@/stores/environment";
import type { VarSuggestion } from "@/stores/environment";

// 变量自动补全：在任意 textarea/input 中输入 `{{` 时弹出可用变量，
// 带「作用域来源徽标 + 解析值预览」，键盘上下/回车/Tab 插入，Esc/失焦关闭。

// 用镜像 div 估算输入框内光标像素坐标（相对元素左上角）。参考 textarea-caret-position 思路。
function getCaretCoordinates(
  el: HTMLTextAreaElement | HTMLInputElement,
  position: number,
): { top: number; left: number } {
  const style = getComputedStyle(el);
  const div = document.createElement("div");
  const props = [
    "boxSizing", "width", "height", "overflowX", "overflowY",
    "borderTopWidth", "borderRightWidth", "borderBottomWidth", "borderLeftWidth",
    "paddingTop", "paddingRight", "paddingBottom", "paddingLeft",
    "fontStyle", "fontVariant", "fontWeight", "fontStretch", "fontSize",
    "fontFamily", "lineHeight", "textAlign", "textTransform", "textIndent",
    "letterSpacing", "wordSpacing", "tabSize",
  ] as const;
  for (const p of props) {
    // 部分属性在 div 上无效果但拷贝无害，统一设置
    (div.style as any)[p] = (style as any)[p];
  }
  div.style.position = "absolute";
  div.style.visibility = "hidden";
  div.style.whiteSpace = "pre-wrap";
  div.style.wordWrap = "break-word";
  div.style.top = "0";
  div.style.left = "0";
  // 镜像容器宽度与元素内容区一致，保证换行点相同
  div.style.width = el.clientWidth + "px";
  div.textContent = el.value.substring(0, position);
  const span = document.createElement("span");
  // 用零宽内容定位光标边界；空内容时以「.」兜底避免 offset 为 0
  span.textContent = el.value.substring(position) || ".";
  div.appendChild(span);
  document.body.appendChild(div);
  const coords = {
    top: span.offsetTop + parseInt(style.borderTopWidth || "0", 10) - (el.scrollTop || 0),
    left: span.offsetLeft + parseInt(style.borderLeftWidth || "0", 10) - (el.scrollLeft || 0),
  };
  document.body.removeChild(div);
  return coords;
}

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
    // 弹层定位到光标处（L12）：用镜像 div 估算 textarea/input 内光标像素坐标，
    // 再叠加元素在视口的偏移，避免「固定贴在输入框下方」的体验问题。
    const node = el.value as HTMLTextAreaElement | HTMLInputElement;
    const rect = node.getBoundingClientRect();
    let caretTop = rect.bottom + 6;
    let caretLeft = rect.left + 8;
    try {
      const caret = getCaretCoordinates(node, node.selectionStart ?? 0);
      caretTop = rect.top + caret.top + 20;
      caretLeft = rect.left + caret.left;
    } catch {
      /* 估算失败退化为输入框下方 */
    }
    // 钳制到视口内，避免溢出右/下边缘
    caretTop = Math.min(caretTop, window.innerHeight - 200);
    caretLeft = Math.min(caretLeft, window.innerWidth - 248);
    top.value = Math.max(8, caretTop);
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
