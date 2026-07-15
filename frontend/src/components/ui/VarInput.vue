<script setup lang="ts">
import { ref, watch } from "vue";
import { useVarComplete } from "@/composables/useVarComplete";
import VarSuggestPopup from "./VarSuggestPopup.vue";
import { cn } from "@/lib/utils";

const props = defineProps<{
  modelValue: string;
  class?: string;
  placeholder?: string;
  type?: string;
}>();
const emit = defineEmits<{ (e: "update:modelValue", v: string): void }>();

const inputEl = ref<HTMLInputElement | null>(null);
const { el, show, top, left, activeIndex, items, onInput, onClick, onKeyup, onKeydown, onBlur, apply } =
  useVarComplete();
watch(inputEl, (v) => (el.value = v), { immediate: true });

function update(e: Event) {
  emit("update:modelValue", (e.target as HTMLInputElement).value);
  onInput();
}
</script>

<template>
  <div class="relative">
    <input
      ref="inputEl"
      :value="modelValue"
      :type="type || 'text'"
      :placeholder="placeholder"
      :class="cn('h-10 w-full rounded-lg border border-border bg-surface px-3 text-sm text-foreground placeholder:text-muted/60 outline-none transition-colors focus:border-primary/60 focus:ring-2 focus:ring-primary/30', $props.class)"
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
