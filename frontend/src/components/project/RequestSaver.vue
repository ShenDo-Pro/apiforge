<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { useRequestSaver, type RequestPayload } from "@/composables/useRequestSaver";
import CollectionPickerDialog from "./CollectionPickerDialog.vue";
import Button from "@/components/ui/Button.vue";
import { Save } from "lucide-vue-next";
import type { SavedRequest } from "@/types/project";

// 各协议客户端共用的「保存到集合」按钮 + 嵌套文件夹选择器。
// buildPayload: 由客户端提供，把当前表单序列化为待保存的请求体。
const props = defineProps<{
  projectId: number;
  request: SavedRequest | null;
  defaultCollectionId?: number | null;
  defaultName?: string;
  buildPayload: (name: string) => RequestPayload;
}>();
const emit = defineEmits<{ (e: "saved", r: SavedRequest): void }>();

const { t } = useI18n();
const { pickerOpen, defaultCollectionId, openSave, confirmSave } = useRequestSaver(
  () => props.projectId,
  (r) => emit("saved", r),
);

function onSave() {
  openSave(props.defaultCollectionId ?? null);
}
async function onPick(collectionId: number, name: string) {
  await confirmSave(collectionId, props.request, props.buildPayload(name));
}
</script>

<template>
  <Button variant="secondary" @click="onSave">
    <Save :size="15" /> {{ t("common.collectionSaveTo") }}
  </Button>
  <CollectionPickerDialog
    :open="pickerOpen"
    :project-id="projectId"
    :model-value="defaultCollectionId"
    :default-name="defaultName"
    @close="pickerOpen = false"
    @confirm="onPick"
  />
</template>
