<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/lib/utils";

type Tone = "default" | "success" | "danger" | "warning" | "info" | "primary";
const props = withDefaults(defineProps<{ tone?: Tone }>(), { tone: "default" });

const tones: Record<Tone, string> = {
  default: "bg-border/40 text-muted",
  success: "bg-success/15 text-success",
  danger: "bg-danger/15 text-danger",
  warning: "bg-warning/15 text-warning",
  info: "bg-info/15 text-info",
  primary: "bg-primary/15 text-primary",
};

const cls = computed(() => cn("inline-flex items-center rounded-md px-2 py-0.5 text-xs font-semibold", tones[props.tone]));
</script>

<template>
  <span :class="cls"><slot /></span>
</template>
