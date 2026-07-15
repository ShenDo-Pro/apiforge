<script setup lang="ts">
import { cn } from "@/lib/utils";

// 轻量受控标签页：tabs 为 [{key,label}]，v-model 绑定当前 key
defineProps<{ tabs: { key: string; label: string }[]; modelValue: string }>();
const emit = defineEmits<{ (e: "update:modelValue", v: string): void }>();
</script>

<template>
  <div class="flex items-center gap-1 border-b border-border px-2">
    <button
      v-for="t in tabs"
      :key="t.key"
      class="relative px-3 py-2 text-sm transition-colors"
      :class="
        cn(
          modelValue === t.key
            ? 'text-foreground'
            : 'text-muted hover:text-foreground'
        )
      "
      @click="emit('update:modelValue', t.key)"
    >
      {{ t.label }}
      <span
        v-if="modelValue === t.key"
        class="absolute inset-x-2 -bottom-px h-0.5 rounded-full bg-gradient-to-r from-primary to-primary-3"
      />
    </button>
  </div>
</template>
