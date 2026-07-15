<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/lib/utils";

const props = defineProps<{ modelValue?: string; class?: string }>();
const emit = defineEmits<{ (e: "update:modelValue", v: string): void }>();

const cls = computed(() =>
  cn(
    "h-10 rounded-lg border border-border bg-surface px-3 text-sm text-foreground outline-none focus:border-primary/60 focus:ring-2 focus:ring-primary/30",
    props.class
  )
);
</script>

<template>
  <select
    :value="props.modelValue"
    :class="cls"
    @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
  >
    <slot />
  </select>
</template>
