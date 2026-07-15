<script setup lang="ts">
import { ref, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useCollectionStore } from "@/stores/collection";
import { PROTOCOLS } from "@/lib/protocols";
import CollectionPickerNode from "./CollectionPickerNode.vue";
import Dialog from "@/components/ui/Dialog.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";

const props = defineProps<{ open: boolean; projectId: number }>();
const emit = defineEmits<{
  (e: "close"): void;
  (e: "confirm", payload: { collectionId: number; protocol: string; name: string }): void;
}>();

const { t } = useI18n();
const store = useCollectionStore();

const expanded = ref<Record<number, boolean>>({});
const selectedId = ref<number | null>(null);
const protocol = ref("http");
const name = ref("");

watch(
  () => props.open,
  (v) => {
    if (v) {
      store.fetchCollections(props.projectId);
      selectedId.value = null;
      protocol.value = "http";
      name.value = "";
      if (Object.keys(expanded.value).length === 0 && store.collections.length) {
        store.collections
          .filter((c) => c.parentId === null)
          .forEach((c) => (expanded.value[c.id] = true));
      }
    }
  },
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
  if (!selectedId.value || !name.value.trim()) return;
  emit("confirm", {
    collectionId: selectedId.value,
    protocol: protocol.value,
    name: name.value.trim(),
  });
}
</script>

<template>
  <Dialog :open="open" :title="t('common.newRequestTitle')" @close="emit('close')">
    <div class="space-y-4">
      <div>
        <label class="mb-2 block text-xs font-medium text-muted">{{ t("common.protocol") }}</label>
        <div class="grid grid-cols-3 gap-2">
          <button
            v-for="p in PROTOCOLS"
            :key="p.key"
            type="button"
            class="flex flex-col items-center gap-1 rounded-lg border px-2 py-2.5 text-xs transition-colors"
            :class="
              protocol === p.key
                ? 'border-primary bg-primary/10 text-foreground'
                : 'border-border text-muted hover:border-primary/40 hover:text-foreground'
            "
            @click="protocol = p.key"
          >
            <component :is="p.icon" :size="18" />
            <span class="text-center leading-tight">{{ p.label }}</span>
          </button>
        </div>
      </div>

      <div>
        <label class="mb-2 block text-xs font-medium text-muted">{{ t("common.targetCollection") }}</label>
        <div class="max-h-48 overflow-y-auto scroll-thin rounded-lg border border-border bg-surface p-2">
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

      <div>
        <label class="mb-1 block text-xs font-medium text-muted">{{ t("common.name") }}</label>
        <Input
          v-model="name"
          :placeholder="t('common.requestNamePlaceholder')"
          @keyup.enter="confirm"
        />
      </div>
    </div>
    <div class="mt-4 flex justify-end gap-2">
      <Button variant="ghost" @click="emit('close')">{{ t("common.cancel") }}</Button>
      <Button :disabled="!selectedId || !name.trim()" @click="confirm">{{ t("common.confirm") }}</Button>
    </div>
  </Dialog>
</template>
