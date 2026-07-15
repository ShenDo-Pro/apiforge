<script setup lang="ts">
import { computed } from "vue";
import { ChevronRight, Folder, FolderOpen } from "lucide-vue-next";
import type { Collection } from "@/types/project";

const props = defineProps<{
  node: Collection;
  all: Collection[];
  selectedId: number | null;
  expanded: Record<number, boolean>;
}>();
const emit = defineEmits<{
  (e: "select", id: number): void;
  (e: "toggle", id: number): void;
}>();

const children = computed(() =>
  props.all
    .filter((c) => c.parentId === props.node.id)
    .sort((a, b) => a.sortOrder - b.sortOrder),
);
const isOpen = computed(() => !!props.expanded[props.node.id]);
const isSelected = computed(() => props.selectedId === props.node.id);
</script>

<template>
  <div>
    <div
      class="flex cursor-pointer items-center gap-1 rounded-md px-2 py-1.5 hover:bg-border/30"
      :class="isSelected ? 'bg-primary/15 text-primary' : 'text-foreground'"
      @click="emit('select', node.id)"
    >
      <button class="shrink-0" @click.stop="emit('toggle', node.id)">
        <ChevronRight
          :size="14"
          class="transition-transform"
          :class="isOpen ? 'rotate-90' : ''"
        />
      </button>
      <component
        :is="isOpen ? FolderOpen : Folder"
        :size="15"
        class="shrink-0 text-primary-3"
      />
      <span class="truncate text-sm">{{ node.name }}</span>
    </div>
    <div v-if="isOpen" class="ml-4 border-l border-border pl-2">
      <CollectionPickerNode
        v-for="c in children"
        :key="c.id"
        :node="c"
        :all="all"
        :selected-id="selectedId"
        :expanded="expanded"
        @select="(id) => emit('select', id)"
        @toggle="(id) => emit('toggle', id)"
      />
    </div>
  </div>
</template>
