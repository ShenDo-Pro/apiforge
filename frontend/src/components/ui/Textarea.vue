<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/lib/utils";

const props = defineProps<{ modelValue?: string; class?: string }>();
const emit = defineEmits<{ (e: "update:modelValue", v: string): void }>();

const cls = computed(() =>
  cn(
    "w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-foreground placeholder:text-muted/60 outline-none transition-colors focus:border-primary/60 focus:ring-2 focus:ring-primary/30 font-mono scroll-thin",
    props.class
  )
);
</script>

<template>
  <textarea
    :value="props.modelValue"
    :class="cls"
    @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
  />
</template>
