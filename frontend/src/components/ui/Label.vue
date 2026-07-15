<script setup lang="ts">
import { cn } from "@/lib/utils";
defineProps<{ class?: string }>();
</script>

<template>
  <label :class="cn('text-xs font-medium text-muted', $props.class)">
    <slot />
  </label>
</template>
