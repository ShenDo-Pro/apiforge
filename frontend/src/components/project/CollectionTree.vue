<script setup lang="ts">
import { onMounted, computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useCollectionStore } from "@/stores/collection";
import CollectionNode from "./CollectionNode.vue";
import type { Collection, SavedRequest } from "@/types/project";

const props = defineProps<{ projectId: number }>();
const emit = defineEmits<{
  (e: "select-request", req: SavedRequest): void;
  (e: "new-request", collectionId: number | null): void;
  (e: "new-folder", parentId: number | null): void;
  (e: "edit-vars", node: Collection): void;
}>();
const { t } = useI18n();
const store = useCollectionStore();
const expanded = ref<Record<number, boolean>>({});

onMounted(() => store.fetchCollections(props.projectId));

const roots = computed(() =>
  store.collections
    .filter((c) => c.parentId === null)
    .sort((a, b) => a.sortOrder - b.sortOrder),
);

async function toggle(c: Collection) {
  if (expanded.value[c.id]) {
    expanded.value[c.id] = false;
    return;
  }
  expanded.value[c.id] = true;
  if (!store.requestsByCollection[c.id]) {
    await store.fetchRequests(props.projectId, c.id);
  }
}
</script>

<template>
  <div class="flex flex-col gap-0.5">
    <CollectionNode
      v-for="c in roots"
      :key="c.id"
      :node="c"
      :all="store.collections"
      :expanded="expanded"
      :project-id="projectId"
      @toggle="toggle"
      @select-request="(r) => emit('select-request', r)"
      @new-request="(id) => emit('new-request', id)"
      @new-folder="(id) => emit('new-folder', id)"
      @edit-vars="(n) => emit('edit-vars', n)"
    />
    <div v-if="roots.length === 0" class="px-3 py-4 text-xs text-muted">
      {{ t("common.collectionEmptyHint") }}
    </div>
  </div>
</template>
