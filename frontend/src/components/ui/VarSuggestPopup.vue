<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { VarSuggestion } from "@/stores/environment";

defineProps<{
  items: VarSuggestion[];
  top: number;
  left: number;
  activeIndex: number;
}>();
const emit = defineEmits<{ (e: "pick", item: VarSuggestion): void }>();
const { t } = useI18n();

const scopeLabel: Record<string, string> = {
  global: "G",
  collection: "C",
  environment: "E",
  local: "L",
};
const scopeClass: Record<string, string> = {
  global: "bg-amber-500/20 text-amber-300",
  collection: "bg-violet-500/20 text-violet-300",
  environment: "bg-sky-500/20 text-sky-300",
  local: "bg-emerald-500/20 text-emerald-300",
};
</script>

<template>
  <Teleport to="body">
    <div
      class="fixed z-[1000] w-64 overflow-hidden rounded-lg border border-border bg-popover/95 shadow-2xl backdrop-blur"
      :style="{ top: top + 'px', left: left + 'px' }"
    >
      <div class="border-b border-border px-2 py-1 text-[11px] text-muted">
        {{ t("common.script.varSuggest") }}
      </div>
      <div class="max-h-56 overflow-y-auto scroll-thin py-1">
        <button
          v-for="(it, i) in items"
          :key="it.key + it.scope"
          class="flex w-full items-center gap-2 px-2 py-1.5 text-left text-xs hover:bg-primary/15"
          :class="i === activeIndex ? 'bg-primary/15' : ''"
          @mousedown.prevent="emit('pick', it)"
        >
          <span
            class="rounded px-1 text-[10px] font-bold"
            :class="scopeClass[it.scope]"
            >{{ scopeLabel[it.scope] }}</span
          >
          <code class="text-foreground">{{ it.key }}</code>
          <span class="ml-auto truncate text-muted">{{
            it.secret ? "••••" : it.value || "—"
          }}</span>
        </button>
        <div v-if="items.length === 0" class="px-2 py-2 text-xs text-muted/60">
          {{ t("common.script.noVar") }}
        </div>
      </div>
    </div>
  </Teleport>
</template>
