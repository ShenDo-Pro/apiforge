import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useToast } from "./useToast";
import { saveRequest } from "@/api/collection";
import { updateRequest } from "@/api/request";
import type { SavedRequest } from "@/types/project";

export interface RequestPayload {
  protocol?: string;
  name: string;
  method: string;
  url: string;
  headers: string;
  body: string;
  // 鉴权与脚本透传到后端持久化（导入 Postman 时需完整保留）
  auth?: string;
  preRequestScript?: string;
  testScript?: string;
  extractRules?: string;
}

// 统一的「保存到集合」逻辑：弹出嵌套文件夹选择器，新增或更新请求。
// getProjectId: 取当前项目 id；onSaved: 保存成功后回调（用于刷新树 / 回填）。
export function useRequestSaver(
  getProjectId: () => number,
  onSaved: (r: SavedRequest) => void,
) {
  const { t } = useI18n();
  const toast = useToast();
  const pickerOpen = ref(false);
  const defaultCollectionId = ref<number | null>(null);
  const saving = ref(false);

  function openSave(collectionId?: number | null) {
    defaultCollectionId.value = collectionId ?? null;
    pickerOpen.value = true;
  }

  async function confirmSave(
    collectionId: number,
    existing: SavedRequest | null,
    payload: RequestPayload,
  ) {
    saving.value = true;
    try {
      const saved = existing?.id
        ? await updateRequest(getProjectId(), existing.id, payload)
        : await saveRequest(getProjectId(), collectionId, payload);
      toast.success(t("common.success"));
      pickerOpen.value = false;
      onSaved(saved);
      return saved;
    } catch (e: any) {
      toast.error(e?.response?.data?.message || t("common.error"));
    } finally {
      saving.value = false;
    }
  }

  return { pickerOpen, defaultCollectionId, openSave, confirmSave, saving };
}
