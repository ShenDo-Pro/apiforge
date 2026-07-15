<script setup lang="ts">
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useCollectionStore } from "@/stores/collection";
import { useToast } from "@/composables/useToast";
import { parseEnvVars } from "@/lib/vars";
import type { Collection, EnvVar } from "@/types/project";
import Dialog from "@/components/ui/Dialog.vue";
import Input from "@/components/ui/Input.vue";
import Button from "@/components/ui/Button.vue";
import { Plus, Trash2, Eye, EyeOff } from "lucide-vue-next";

const props = defineProps<{ open: boolean; collection: Collection | null; projectId: number }>();
const emit = defineEmits<{ (e: "close"): void }>();
const { t } = useI18n();
const store = useCollectionStore();
const toast = useToast();

const working = ref<EnvVar[]>([]);
const revealed = ref<Record<number, boolean>>({});

watch(
  () => [props.open, props.collection],
  () => {
    working.value = props.collection?.variables
      ? parseEnvVars(props.collection.variables).map((v) => ({ ...v }))
      : [];
  },
  { immediate: true }
);

function addVar() {
  working.value.push({ key: "", value: "", enabled: true, secret: false });
}
function removeVar(i: number) {
  working.value.splice(i, 1);
}
function toggleSecret(i: number) {
  working.value[i].secret = !working.value[i].secret;
}
function reveal(i: number) {
  revealed.value[i] = !revealed.value[i];
}

async function save() {
  if (!props.collection) return;
  await store.saveCollectionVariables(props.projectId, props.collection.id, working.value);
  toast.success(t("common.saved"));
  emit("close");
}
</script>

<template>
  <Dialog :open="open" :title="t('common.collectionVars')" @close="emit('close')">
    <div class="space-y-2">
      <p class="text-xs text-muted">{{ t("common.collectionVarsHint") }}</p>
      <div class="grid grid-cols-[1fr_1.4fr_auto_auto] gap-2 px-1 pb-1 text-xs text-muted">
        <span>{{ t("common.varKey") }}</span>
        <span>{{ t("common.varValue") }}</span>
        <span>{{ t("common.varSecret") }}</span>
        <span></span>
      </div>
      <div class="max-h-80 space-y-1.5 overflow-y-auto scroll-thin">
        <div
          v-for="(v, i) in working"
          :key="i"
          class="grid grid-cols-[1fr_1.4fr_auto_auto] items-center gap-2"
        >
          <Input v-model="v.key" :placeholder="t('common.varKey')" />
          <Input
            v-if="!v.secret || revealed[i]"
            v-model="v.value"
            :placeholder="t('common.varValue')"
          />
          <div v-else class="flex items-center gap-1">
            <Input v-model="v.value" type="password" :placeholder="t('common.varValue')" />
            <button class="text-muted hover:text-foreground" @click="reveal(i)"><EyeOff :size="14" /></button>
          </div>
          <button
            class="text-muted hover:text-amber-300"
            :class="v.secret ? 'text-amber-400' : ''"
            :title="t('common.varSecret')"
            @click="toggleSecret(i)"
          >
            <Eye :size="14" />
          </button>
          <button class="text-muted hover:text-danger" :title="t('common.delete')" @click="removeVar(i)">
            <Trash2 :size="14" />
          </button>
        </div>
        <div v-if="working.length === 0" class="px-1 py-3 text-xs text-muted/60">—</div>
      </div>
      <button class="flex items-center gap-1 text-xs text-primary hover:underline" @click="addVar">
        <Plus :size="13" /> {{ t("common.varAdd") }}
      </button>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <Button variant="ghost" @click="emit('close')">{{ t("common.cancel") }}</Button>
        <Button @click="save">{{ t("common.save") }}</Button>
      </div>
    </template>
  </Dialog>
</template>
