<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useCollectionStore } from "@/stores/collection";
import CollectionPickerNode from "./CollectionPickerNode.vue";
import Dialog from "@/components/ui/Dialog.vue";
import Button from "@/components/ui/Button.vue";

const props = defineProps<{
  open: boolean;
  projectId: number;
  modelValue: number | null;
  defaultName?: string;
}>();
const emit = defineEmits<{
  (e: "close"): void;
  (e: "confirm", collectionId: number, name: string): void;
}>();

const { t } = useI18n();
const store = useCollectionStore();

// 打开时确保集合已加载，默认展开根节点
watch(
  () => props.open,
  (v) => {
    if (v) {
      store.fetchCollections(props.projectId);
      name.value = props.defaultName || "";
      if (Object.keys(expanded.value).length === 0 && store.collections.length) {
        const roots = store.collections.filter((c) => c.parentId === null);
        roots.forEach((c) => (expanded.value[c.id] = true));
      }
    }
  },
);

const expanded = ref<Record<number, boolean>>({});
const selectedId = ref<number | null>(props.modelValue);
const name = ref("");

watch(
  () => props.modelValue,
  (v) => (selectedId.value = v),
);

const roots = computed(() =>
  store.collections
    .filter((c) => c.parentId === null)
    .sort((a, b) => a.sortOrder - b.sortOrder),
);

function toggle(id: number) {
  expanded.value[id] = !expanded.value[id];
}
function confirm() {
  if (!selectedId.value) return;
  emit("confirm", selectedId.value);
}
</script>

<template>
  <Dialog :open="open" :title="t('common.collectionSelectTarget')" @close="emit('close')">
    <div class="space-y-3">
      <div>
        <label class="mb-1 block text-xs text-muted">{{ t("common.collectionRequestName") }}</label>
        <Input v-model="name" @keyup.enter="confirm" />
      </div>
      <div class="max-h-64 overflow-y-auto scroll-thin">
        <div v-if="roots.length === 0" class="px-2 py-3 text-xs text-muted">
          {{ t("common.collectionEmptyHint") }}
        </div>
        <CollectionPickerNode
          v-for="c in roots"
          :key="c.id"
          :node="c"
          :all="store.collections"
          :selected-id="selectedId"
          :expanded="expanded"
          @select="(id) => (selectedId = id)"
          @toggle="toggle"
        />
      </div>
    </div>
    <div class="mt-3 flex justify-end gap-2">
      <Button variant="ghost" @click="emit('close')">{{ t("common.cancel") }}</Button>
      <Button :disabled="!selectedId" @click="confirm">{{ t("common.confirm") }}</Button>
    </div>
  </Dialog>
</template>
