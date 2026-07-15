<script setup lang="ts">
import { ref, watch, nextTick } from "vue";
import { useVarComplete } from "@/composables/useVarComplete";
import VarSuggestPopup from "./VarSuggestPopup.vue";

const props = defineProps<{
  modelValue: string;
  placeholder?: string;
  rows?: number;
}>();
const emit = defineEmits<{ (e: "update:modelValue", v: string): void }>();

const ta = ref<HTMLTextAreaElement | null>(null);
const { el, show, top, left, activeIndex, items, onInput, onClick, onKeyup, onKeydown, onBlur, apply } =
  useVarComplete();

watch(ta, (v) => (el.value = v), { immediate: true });

const lineCount = ref(1);
watch(
  () => props.modelValue,
  (v) => {
    lineCount.value = (v || "").split("\n").length;
  },
  { immediate: true }
);

function update(e: Event) {
  emit("update:modelValue", (e.target as HTMLTextAreaElement).value);
  onInput();
}
</script>

<template>
  <div class="relative flex rounded-lg border border-border bg-black/30 font-mono text-xs">
    <!-- 行号 -->
    <div
      class="select-none border-r border-border/60 px-2 py-2 text-right text-muted/50"
      style="min-width: 2.5rem"
    >
      <div v-for="n in Math.max(lineCount, rows || 6)" :key="n">{{ n }}</div>
    </div>
    <!-- 编辑区 -->
    <textarea
      ref="ta"
      :value="modelValue"
      :placeholder="placeholder"
      :rows="rows || 8"
      class="flex-1 resize-y bg-transparent px-3 py-2 leading-relaxed text-foreground outline-none scroll-thin"
      @input="update"
      @click="onClick"
      @keyup="onKeyup"
      @keydown="onKeydown"
      @blur="onBlur"
    />
    <VarSuggestPopup
      v-if="show"
      :items="items"
      :top="top"
      :left="left"
      :active-index="activeIndex"
      @pick="apply"
    />
  </div>
</template>
