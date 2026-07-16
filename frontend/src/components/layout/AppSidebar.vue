<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useContextMenu } from "@/composables/useContextMenu";
import {
  Users,
  Settings,
  Workflow,
  FolderPlus,
  FilePlus2,
  ArrowLeft,
  Layers,
  Upload,
  ShieldCheck,
  Database,
} from "lucide-vue-next";
import { useProjectStore } from "@/stores/project";
import { useCollectionStore } from "@/stores/collection";
import { useAuthStore } from "@/stores/auth";
import { useToast } from "@/composables/useToast";
import { createCollection, saveRequest } from "@/api/collection";
import { fromPostman } from "@/composables/usePostmanCollection";
import CollectionTree from "@/components/project/CollectionTree.vue";

const props = defineProps<{ projectId: number }>();
const emit = defineEmits<{
  (e: "select-request", req: unknown): void;
  (e: "new-request"): void;
  (e: "new-folder", parentId: number | null): void;
  (e: "edit-vars", node: unknown): void;
  (e: "open-view", view: "members" | "pipeline" | "settings" | "environments" | "audit" | "backup"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const project = useProjectStore();
const collectionStore = useCollectionStore();
const auth = useAuthStore();
const toast = useToast();

// 导入 Postman v2.1：选文件后解析为导入计划，先建根集合再按 DFS 顺序递归建文件夹与请求。
const fileInput = ref<HTMLInputElement | null>(null);
function triggerImport() {
  fileInput.value?.click();
}
async function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = ""; // 允许重复选择同一文件
  if (!file) return;
  let json: any;
  try {
    json = JSON.parse(await file.text());
  } catch {
    toast.error(t("common.importFail"));
    return;
  }
  const plan = fromPostman(json);
  if (!plan) {
    toast.error(t("common.importFail"));
    return;
  }
  try {
    const root = await createCollection(props.projectId, { parentId: null, name: plan.name });
    const idMap = new Map<string, number>();
    idMap.set("", root.id); // parentTempId 为空表示挂在根集合下
    for (const op of plan.ops) {
      const parentId = op.parentTempId ? idMap.get(op.parentTempId) ?? root.id : root.id;
      if (op.kind === "folder") {
        const c = await createCollection(props.projectId, { parentId, name: op.name });
        idMap.set(op.tempId, c.id);
      } else if (op.payload) {
        await saveRequest(props.projectId, parentId, op.payload);
      }
    }
    await collectionStore.fetchCollections(props.projectId);
    toast.success(t("common.imported"));
    if (plan.unsupportedAuth) toast.error(t("common.importAuthDowngraded"));
  } catch {
    toast.error(t("common.importFail"));
  }
}

// 新建根目录弹框收到来自树节点的「新建目录」事件后，统一上抛
function onTreeNewFolder(parentId: number | null) {
  emit("new-folder", parentId);
}

// 集合列表及下方空白区的右键菜单：默认新增顶层目录（全局单例，只保留一个菜单）
const emptyMenuEl = ref<HTMLElement | null>(null);
const { isOpen: emptyMenuOpen, open: openEmptyMenu, close: closeEmptyMenu } = useContextMenu(
  "sidebar-empty",
  emptyMenuEl,
);
const emptyMenuPos = ref({ x: 0, y: 0 });

function onEmptyContextMenu(e: MouseEvent) {
  e.preventDefault();
  // 钳制到视口内，避免贴在屏幕右下边缘时菜单溢出
  const w = 176; // w-44
  const h = 92; // 菜单大致高度（两项）
  const px = Math.min(e.clientX, window.innerWidth - w - 8);
  const py = Math.min(e.clientY, window.innerHeight - h - 8);
  emptyMenuPos.value = { x: Math.max(8, px), y: Math.max(8, py) };
  openEmptyMenu();
}
function emptyCreateFolder() {
  closeEmptyMenu();
  emit("new-folder", null);
}
function emptyCreateRequest() {
  closeEmptyMenu();
  emit("new-request");
}
</script>

<template>
  <aside class="flex w-64 shrink-0 flex-col border-r border-border glass">
    <div class="border-b border-border px-4 py-3">
      <button
        class="mb-2 flex items-center gap-1 text-xs text-muted transition-colors hover:text-foreground"
        @click="router.push('/projects')"
      >
        <ArrowLeft :size="13" /> {{ t("common.backToProjects") }}
      </button>
      <div class="truncate text-sm font-semibold text-foreground">{{ project.current?.name }}</div>
      <div class="truncate text-xs text-muted">{{ project.current?.description }}</div>
    </div>

    <!-- 顶部操作：新建请求 / 新建目录 -->
    <div class="flex gap-2 px-3 py-3">
      <button
        class="flex flex-1 items-center justify-center gap-1.5 rounded-lg bg-gradient-to-r from-primary to-primary-3 px-3 py-2 text-sm font-medium text-white shadow-glow transition-opacity hover:opacity-90"
        @click="emit('new-request')"
      >
        <FilePlus2 :size="15" /> {{ t("common.collectionNewRequest") }}
      </button>
      <button
        class="flex items-center justify-center rounded-lg border border-border bg-surface px-2.5 py-2 text-sm text-muted transition-colors hover:border-primary/40 hover:text-foreground"
        :title="t('common.collectionNewFolder')"
        @click="emit('new-folder', null)"
      >
        <FolderPlus :size="16" />
      </button>
      <button
        class="flex items-center justify-center rounded-lg border border-border bg-surface px-2.5 py-2 text-sm text-muted transition-colors hover:border-primary/40 hover:text-foreground"
        :title="t('common.import')"
        @click="triggerImport"
      >
        <Upload :size="16" />
      </button>
      <input
        ref="fileInput"
        type="file"
        accept=".json,application/json"
        class="hidden"
        @change="onFileChange"
      />
    </div>

    <!-- 集合树（置顶主体） -->
    <div class="flex items-center justify-between px-4 pb-1 text-xs font-medium text-muted">
      <span>{{ t("common.collections") }}</span>
    </div>
    <div class="flex-1 overflow-y-auto scroll-thin px-2 pb-3" @contextmenu.prevent="onEmptyContextMenu">
      <CollectionTree
        :project-id="projectId"
        @select-request="(r) => emit('select-request', r)"
        @new-request="() => emit('new-request')"
        @new-folder="onTreeNewFolder"
        @edit-vars="(n) => emit('edit-vars', n)"
      />
    </div>

    <!-- 集合列表下方空白区右键菜单：Teleport 到 body，脱离 .glass 的
         backdrop-filter 包含块，使 fixed + clientX/clientY 精确定位到鼠标点击处；
         全局单例保证只保留一个菜单 -->
    <Teleport to="body">
      <div
        v-if="emptyMenuOpen"
        ref="emptyMenuEl"
        class="fixed z-[100] w-44 rounded-lg border border-border bg-surface p-1 shadow-glow"
        :style="{ left: emptyMenuPos.x + 'px', top: emptyMenuPos.y + 'px' }"
        @contextmenu.prevent.stop
      >
        <button
          class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-foreground hover:bg-border/30"
          @click="emptyCreateFolder"
        >
          <FolderPlus :size="15" /> {{ t("common.collectionNewFolder") }}
        </button>
        <button
          class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-foreground hover:bg-border/30"
          @click="emptyCreateRequest"
        >
          <FilePlus2 :size="15" /> {{ t("common.collectionNewRequest") }}
        </button>
      </div>
    </Teleport>

    <!-- 底部导航：环境 / 成员 / 流水线 / 设置 -->
    <div class="border-t border-border p-2">
      <button
        class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted transition-colors hover:bg-border/30 hover:text-foreground"
        @click="emit('open-view', 'environments')"
      >
        <Layers :size="16" /> {{ t("common.navEnvironments") }}
      </button>
      <button
        class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted transition-colors hover:bg-border/30 hover:text-foreground"
        @click="emit('open-view', 'members')"
      >
        <Users :size="16" /> {{ t("common.members") }}
      </button>
      <button
        class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted transition-colors hover:bg-border/30 hover:text-foreground"
        @click="emit('open-view', 'pipeline')"
      >
        <Workflow :size="16" /> {{ t("common.navPipeline") }}
      </button>
      <button
        v-if="auth.user?.role === 'admin'"
        class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted transition-colors hover:bg-border/30 hover:text-foreground"
        @click="emit('open-view', 'audit')"
      >
        <ShieldCheck :size="16" /> {{ t("audit.title") }}
      </button>
      <button
        class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted transition-colors hover:bg-border/30 hover:text-foreground"
        @click="emit('open-view', 'backup')"
      >
        <Database :size="16" /> {{ t("common.backup") }}
      </button>
      <button
        class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-muted transition-colors hover:bg-border/30 hover:text-foreground"
        @click="emit('open-view', 'settings')"
      >
        <Settings :size="16" /> {{ t("common.navSettings") }}
      </button>
    </div>
  </aside>
</template>
