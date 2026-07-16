<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useEnvironmentStore } from "@/stores/environment";
import { useToast } from "@/composables/useToast";
import { parseEnvVars } from "@/lib/vars";
import type { EnvVar } from "@/types/project";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import { Plus, Copy, Trash2, Star, Download, Upload, Eye, EyeOff, GripVertical, Check } from "lucide-vue-next";

const props = defineProps<{ projectId: number }>();
const { t } = useI18n();
const envStore = useEnvironmentStore();
const toast = useToast();

const envs = computed(() => envStore.environments.filter((e) => e.kind === "env"));
const globalEnv = computed(() => envStore.globalEnv);

// 当前编辑对象（global 或某个 env）；默认选中 global
const selectedId = ref<number | null>(null);
const selected = computed(() => {
  if (selectedId.value == null) return globalEnv.value;
  return envStore.environments.find((e) => e.id === selectedId.value) ?? null;
});

// 工作副本：直接编辑后防抖落库
const working = ref<EnvVar[]>([]);
watch(
  selected,
  (s) => {
    // 切换环境时取消上一个尚未触发的持久化，避免把旧环境的 working 落到新环境（L17）
    clearTimeout(timer);
    working.value = s ? parseEnvVars(s.values).map((v) => ({ ...v })) : [];
  },
  { immediate: true }
);

onMounted(() => {
  selectedId.value = globalEnv.value?.id ?? envs.value[0]?.id ?? null;
});

// 防抖持久化（避免每次按键都打接口）
let timer: any = null;
function schedulePersist() {
  const target = selected.value;
  if (!target) return;
  const targetId = target.id;
  clearTimeout(timer);
  timer = setTimeout(() => {
    // 若 400ms 内切换了环境，丢弃本次过期写入（L17 防抖竞态）
    const cur = selected.value;
    if (!cur || cur.id !== targetId) return;
    envStore.persistEnvironment(cur.id, cur.name, working.value);
  }, 400);
}

function selectEnv(id: number | null) {
  selectedId.value = id;
}

function setActive(id: number) {
  envStore.setActive(id);
}

// 变量行操作
function addVar() {
  working.value.push({ key: "", value: "", enabled: true, secret: false });
  schedulePersist();
}
function removeVar(i: number) {
  working.value.splice(i, 1);
  schedulePersist();
}
function toggleSecret(i: number) {
  working.value[i].secret = !working.value[i].secret;
  schedulePersist();
}
const revealed = ref<Record<number, boolean>>({});
function reveal(i: number) {
  revealed.value[i] = !revealed.value[i];
}

// 新建/复制/删除环境
async function newEnv() {
  const env = await envStore.addEnvironment("新环境", props.projectId);
  selectedId.value = env.id;
}
async function dupEnv(id: number) {
  const c = await envStore.duplicateEnvironment(id);
  if (c) selectedId.value = c.id;
}
async function delEnv(id: number) {
  await envStore.removeEnvironment(id);
  if (selectedId.value === id) selectedId.value = globalEnv.value?.id ?? envs.value[0]?.id ?? null;
}

// 拖拽排序（列表项）
const dragId = ref<number | null>(null);
function onDragStart(id: number) {
  dragId.value = id;
}
function onDrop(targetId: number) {
  if (dragId.value == null || dragId.value === targetId) return;
  const ids = envs.value.map((e) => e.id);
  const from = ids.indexOf(dragId.value);
  const to = ids.indexOf(targetId);
  ids.splice(to, 0, ids.splice(from, 1)[0]);
  envStore.reorder(ids, props.projectId);
  dragId.value = null;
}

// 导入/导出
function exportEnv() {
  const s = selected.value;
  if (!s) return;
  const payload = {
    name: s.name,
    _postman_variable_scope: s.kind === "global" ? "globals" : "environment",
    values: working.value.map((v) => ({
      key: v.key,
      value: v.value,
      enabled: v.enabled,
      type: v.secret ? "secret" : "default",
    })),
  };
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: "application/json" });
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = `${s.name || "environment"}.json`;
  a.click();
  URL.revokeObjectURL(a.href);
}
function onImport(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0];
  if (!file) return;
  const reader = new FileReader();
  reader.onload = async () => {
    try {
      const data = JSON.parse(String(reader.result));
      const values: EnvVar[] = (data.values || []).map((v: any) => ({
        key: v.key ?? "",
        value: v.value ?? "",
        enabled: v.enabled !== false,
        secret: v.type === "secret" || v.secret === true,
      }));
      const name = data.name || "导入的环境";
      const env = await envStore.importEnvironment({ name, values }, props.projectId);
      selectedId.value = env.id;
      toast.success(t("common.imported"));
    } catch {
      toast.error(t("common.importFail"));
    }
  };
  reader.readAsText(file);
  (e.target as HTMLInputElement).value = "";
}
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- 工具条 -->
    <div class="flex items-center gap-2 border-b border-border px-4 py-2.5">
      <h2 class="text-sm font-semibold text-foreground">{{ t("common.envManage") }}</h2>
      <div class="ml-auto flex items-center gap-2">
        <Button variant="ghost" size="sm" @click="() => (($refs.fileInput as HTMLInputElement)?.click())">
          <Upload :size="14" /> {{ t("common.import") }}
        </Button>
        <input ref="fileInput" type="file" accept="application/json" class="hidden" @change="onImport" />
        <Button variant="ghost" size="sm" @click="exportEnv"><Download :size="14" /> {{ t("common.export") }}</Button>
        <Button size="sm" @click="newEnv"><Plus :size="14" /> {{ t("common.envNew") }}</Button>
      </div>
    </div>

    <div class="grid flex-1 grid-cols-[260px_1fr] overflow-hidden">
      <!-- 左：环境列表 -->
      <div class="overflow-y-auto scroll-thin border-r border-border p-2">
        <!-- 全局变量单例 -->
        <button
          v-if="globalEnv"
          class="mb-1 flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm transition-colors"
          :class="selectedId == null ? 'bg-primary/15 text-foreground' : 'text-muted hover:bg-border/30'"
          @click="selectEnv(null)"
        >
          <Star :size="15" class="text-amber-400" />
          <span class="truncate">{{ t("common.globals") }}</span>
        </button>

        <div class="mb-1 px-1 text-[11px] text-muted/60">{{ t("common.envList") }}</div>
        <div
          v-for="e in envs"
          :key="e.id"
          class="group mb-1 flex items-center gap-1.5 rounded-lg px-2 py-2 text-sm transition-colors"
          :class="selectedId === e.id ? 'bg-primary/15 text-foreground' : 'text-muted hover:bg-border/30'"
          draggable="true"
          @dragstart="onDragStart(e.id)"
          @dragover.prevent
          @drop="onDrop(e.id)"
          @click="selectEnv(e.id)"
        >
          <GripVertical :size="13" class="cursor-grab text-muted/40" />
          <span class="flex-1 truncate">{{ e.name }}</span>
          <button
            v-if="envStore.activeId === String(e.id)"
            class="text-emerald-400"
            :title="t('common.envActive')"
            @click.stop="setActive(e.id)"
          >
            <Check :size="14" />
          </button>
          <button
            class="opacity-0 transition-opacity group-hover:opacity-100 hover:text-foreground"
            :title="t('common.duplicate')"
            @click.stop="dupEnv(e.id)"
          >
            <Copy :size="13" />
          </button>
          <button
            class="opacity-0 transition-opacity group-hover:opacity-100 hover:text-danger"
            :title="t('common.delete')"
            @click.stop="delEnv(e.id)"
          >
            <Trash2 :size="13" />
          </button>
        </div>
        <div v-if="envs.length === 0" class="px-2 py-3 text-xs text-muted/60">{{ t("common.envEmpty") }}</div>
      </div>

      <!-- 右：编辑 -->
      <div v-if="selected" class="flex flex-col overflow-hidden p-4">
        <div class="mb-3 flex items-center gap-2">
          <Input v-model="selected.name" class="max-w-xs" @update:model-value="schedulePersist" />
          <span
            v-if="selected.kind === 'env'"
            class="rounded-full px-2 py-0.5 text-[11px]"
            :class="envStore.activeId === String(selected.id) ? 'bg-emerald-500/20 text-emerald-300' : 'bg-border/40 text-muted'"
          >
            {{ envStore.activeId === String(selected.id) ? t("common.envActive") : t("common.envInactive") }}
          </span>
          <Button
            v-if="selected.kind === 'env'"
            variant="ghost"
            size="sm"
            class="ml-auto"
            @click="setActive(selected.id)"
          >
            <Check :size="14" /> {{ t("common.envSetActive") }}
          </Button>
        </div>

        <!-- 变量表头 -->
        <div class="grid grid-cols-[auto_1fr_1.4fr_auto_auto_auto] gap-2 px-1 pb-1 text-xs text-muted">
          <span></span>
          <span>{{ t("common.varKey") }}</span>
          <span>{{ t("common.varValue") }}</span>
          <span>{{ t("common.varSecret") }}</span>
          <span></span>
          <span></span>
        </div>
        <div class="flex-1 overflow-y-auto scroll-thin space-y-1.5">
          <div
            v-for="(v, i) in working"
            :key="i"
            class="grid grid-cols-[auto_1fr_1.4fr_auto_auto_auto] items-center gap-2"
          >
            <input type="checkbox" v-model="v.enabled" :title="t('common.varEnabled')" @change="schedulePersist" />
            <Input v-model="v.key" :placeholder="t('common.varKey')" @update:model-value="schedulePersist" />
            <Input
              v-if="!v.secret || revealed[i]"
              v-model="v.value"
              :placeholder="t('common.varValue')"
              @update:model-value="schedulePersist"
            />
            <div v-else class="flex items-center gap-1">
              <Input v-model="v.value" type="password" :placeholder="t('common.varValue')" @update:model-value="schedulePersist" />
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
        <button class="mt-3 flex w-fit items-center gap-1 text-xs text-primary hover:underline" @click="addVar">
          <Plus :size="13" /> {{ t("common.varAdd") }}
        </button>
      </div>
      <div v-else class="flex items-center justify-center text-sm text-muted/60">{{ t("common.envEmpty") }}</div>
    </div>
  </div>
</template>
