<script setup lang="ts">
import { computed } from "vue";
import { cn } from "@/lib/utils";

type Variant = "default" | "secondary" | "ghost" | "danger" | "outline";
type Size = "sm" | "md" | "icon";

const props = withDefaults(
  defineProps<{ variant?: Variant; size?: Size }>(),
  { variant: "default", size: "md" }
);

const base =
  "inline-flex items-center justify-center gap-2 rounded-lg font-medium transition-all duration-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/60 disabled:opacity-50 disabled:cursor-not-allowed";

const variants: Record<Variant, string> = {
  default:
    "bg-gradient-to-r from-primary to-primary-3 text-white shadow-glow hover:brightness-110 hover:scale-[1.02] active:scale-95",
  secondary: "bg-surface text-foreground border border-border hover:bg-border/40",
  ghost: "text-muted hover:text-foreground hover:bg-border/30",
  danger: "bg-danger/90 text-white hover:bg-danger",
  outline: "border border-border text-foreground hover:border-primary/60 hover:text-primary",
};

const sizes: Record<Size, string> = {
  sm: "h-8 px-3 text-xs",
  md: "h-10 px-4 text-sm",
  icon: "h-9 w-9",
};

const cls = computed(() => cn(base, variants[props.variant], sizes[props.size]));
</script>

<template>
  <button :class="cls">
    <slot />
  </button>
</template>
