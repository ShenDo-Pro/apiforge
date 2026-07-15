<script setup lang="ts">
import { cn } from "@/lib/utils";
defineProps<{ open: boolean; title?: string }>();
const emit = defineEmits<{ (e: "close"): void }>();
</script>

<template>
  <Transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      @click.self="emit('close')"
    >
      <div :class="cn('w-full max-w-md rounded-2xl border border-border glass p-5 animate-fade-in-up shadow-glow')">
        <div class="mb-4 flex items-center justify-between">
          <h3 class="text-base font-semibold text-foreground">{{ title }}</h3>
          <button class="text-muted hover:text-foreground" @click="emit('close')">✕</button>
        </div>
        <slot />
      </div>
    </div>
  </Transition>
</template>
