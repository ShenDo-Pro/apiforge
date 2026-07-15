<script setup lang="ts">
import { toasts } from "@/composables/useToast";
import { cn } from "@/lib/utils";

const toneCls: Record<string, string> = {
  success: "border-success/40 text-success",
  error: "border-danger/40 text-danger",
  info: "border-info/40 text-info",
};
</script>

<template>
  <div class="fixed bottom-6 right-6 z-[60] flex flex-col gap-2">
    <TransitionGroup name="slide">
      <div
        v-for="t in toasts"
        :key="t.id"
        :class="
          cn(
            'rounded-lg border bg-card/90 glass px-4 py-2 text-sm shadow-glow backdrop-blur',
            toneCls[t.type] || 'text-foreground'
          )
        "
      >
        {{ t.message }}
      </div>
    </TransitionGroup>
  </div>
</template>
