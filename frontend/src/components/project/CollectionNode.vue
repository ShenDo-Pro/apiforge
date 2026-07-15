<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useContextMenu } from "@/composables/useContextMenu";
import {
  ChevronRight,
  Folder,
  FolderOpen,
  FileCode,
  MoreVertical,
  FolderPlus,
  FilePlus2,
  Pencil,
  Trash2,
  Variable,
  Download,
} from "lucide-vue-next";
import { useCollectionStore } from "@/stores/collection";
import { protocolMeta } from "@/lib/protocols";
import { exportCollection } from "@/composables/usePostmanCollection";
import type { Collection, SavedRequest } from "@/types/project";

const props = defineProps<{
  node: Collection;
  all: Collection[];
  expanded: Record<number, boolean>;
  projectId: number;
}>();

const emit = defineEmits<{
  (e: "toggle", c: Collection): void;
  (e: "select-request", r: SavedRequest): void;
  (e: "new-request", collectionId: number | null): void;
  (e: "new-folder", parentId: number): void;
  (e: "edit-vars", node: Collection): void;
}>();

const { t } = useI18n();
const store = useCollectionStore();

const children = computed(() =>
  props.all
    .filter((c) => c.parentId === props.node.id)
    .sort((a, b) => a.sortOrder - b.sortOrder),
);
const isOpen = computed(() => !!props.expanded[props.node.id]);
const requests = computed(() => store.requestsByCollection[props.node.id] || []);

// 节点操作菜单：全局单例，保证同一时刻只有一个右键菜单打开
const menuEl = ref<HTMLElement | null>(null);
const { isOpen: menuOpen, open: openMenuState, close: closeMenu } = useContextMenu(
  "node-" + props.node.id,
  menuEl,
);
const menuPos = ref({ x: 0, y: 0 });
const renaming = ref(false);
const renameValue = ref("");

function openMenu(x: number, y: number) {
  // 钳制到视口内，避免贴在屏幕右下边缘时菜单溢出
  const w = 176; // w-44
  const h = 184; // 菜单大致高度
  const px = Math.min(x, window.innerWidth - w - 8);
  const py = Math.min(y, window.innerHeight - h - 8);
  menuPos.value = { x: Math.max(8, px), y: Math.max(8, py) };
  openMenuState();
}

function onMenuClick(e: MouseEvent) {
  e.stopPropagation();
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
  openMenu(rect.left, rect.bottom + 4);
}

function onContextMenu(e: MouseEvent) {
  e.preventDefault();
  e.stopPropagation();
  openMenu(e.clientX, e.clientY);
}

async function onNewRequest() {
  closeMenu();
  if (!isOpen.value) emit("toggle", props.node);
  emit("new-request", props.node.id);
}

function onAddChild() {
  closeMenu();
  emit("new-folder", props.node.id);
}

function onEditVars() {
  closeMenu();
  emit("edit-vars", props.node);
}

function onRename() {
  closeMenu();
  renaming.value = true;
  renameValue.value = props.node.name;
}

async function doRename() {
  const name = renameValue.value.trim();
  if (name) await store.renameCollection(props.projectId, props.node.id, name);
  renaming.value = false;
}

async function onDelete() {
  closeMenu();
  if (!confirm(t("common.collectionDeleteConfirm"))) return;
  await store.removeCollection(props.projectId, props.node.id);
}

// 导出为 Postman v2.1：递归收集节点及子孙集合的请求（按需补齐缓存），序列化后下载。
function descendantIds(id: number): number[] {
  const out: number[] = [];
  const stack = [id];
  while (stack.length) {
    const cur = stack.pop()!;
    for (const c of store.collections) {
      if (c.parentId === cur) {
        out.push(c.id);
        stack.push(c.id);
      }
    }
  }
  return out;
}
async function onExport() {
  closeMenu();
  const ids = [props.node.id, ...descendantIds(props.node.id)];
  for (const id of ids) {
    if (!store.requestsByCollection[id]) {
      try {
        await store.fetchRequests(props.projectId, id);
      } catch {
        /* 单个集合请求拉取失败不应中断整体导出 */
      }
    }
  }
  const pm = exportCollection(props.node, store.collections, store.requestsByCollection);
  const blob = new Blob([JSON.stringify(pm, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `${props.node.name}.postman_collection.json`;
  a.click();
  URL.revokeObjectURL(url);
}
</script>

<template>
  <div>
    <div
      class="group relative flex items-center gap-1 rounded-md px-2 py-1.5 hover:bg-border/30"
      @contextmenu="onContextMenu"
    >
      <button
        class="flex min-w-0 flex-1 items-center gap-1 text-left text-sm text-foreground"
        @click="emit('toggle', node)"
      >
        <ChevronRight :size="14" class="shrink-0 transition-transform" :class="isOpen ? 'rotate-90' : ''" />
        <component :is="isOpen ? FolderOpen : Folder" :size="15" class="shrink-0 text-primary-3" />
        <input
          v-if="renaming"
          v-model="renameValue"
          class="min-w-0 flex-1 rounded border border-border bg-surface px-1 text-sm outline-none focus:border-primary/60"
          @click.stop
          @keyup.enter="doRename"
          @keyup.esc="renaming = false"
        />
        <span v-else class="truncate">{{ node.name }}</span>
      </button>

      <button
        class="shrink-0 rounded p-1 text-muted opacity-0 hover:text-foreground group-hover:opacity-100"
        :title="t('common.collectionNew')"
        @click="onMenuClick"
      >
        <MoreVertical :size="14" />
      </button>

      <!-- 节点操作菜单：Teleport 到 body，脱离 .glass 的 backdrop-filter 包含块，
           使 fixed + clientX/clientY 精确定位到鼠标点击处；全局单例保证只存在一个菜单 -->
      <Teleport to="body">
        <div
          v-if="menuOpen"
          ref="menuEl"
          class="fixed z-[100] w-44 rounded-lg border border-border bg-surface p-1 shadow-glow"
          :style="{ left: menuPos.x + 'px', top: menuPos.y + 'px' }"
          @contextmenu.prevent.stop
        >
          <button
            class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-foreground hover:bg-border/30"
            @click="onNewRequest"
          >
            <FilePlus2 :size="15" /> {{ t("common.collectionNewRequest") }}
          </button>
          <button
            class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-foreground hover:bg-border/30"
            @click="onAddChild"
          >
            <FolderPlus :size="15" /> {{ t("common.collectionNewFolder") }}
          </button>
          <button
            class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-foreground hover:bg-border/30"
            @click="onRename"
          >
            <Pencil :size="15" /> {{ t("common.collectionRename") }}
          </button>
          <button
            class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-foreground hover:bg-border/30"
            @click="onEditVars"
          >
            <Variable :size="15" /> {{ t("common.collectionVars") }}
          </button>
          <button
            class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-foreground hover:bg-border/30"
            @click="onExport"
          >
            <Download :size="15" /> {{ t("common.exportPostman") }}
          </button>
          <button
            class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-danger hover:bg-border/30"
            @click="onDelete"
          >
            <Trash2 :size="15" /> {{ t("common.delete") }}
          </button>
        </div>
      </Teleport>
    </div>

    <div v-if="isOpen" class="ml-4 border-l border-border pl-2">
      <CollectionNode
        v-for="c in children"
        :key="c.id"
        :node="c"
        :all="all"
        :expanded="expanded"
        :project-id="projectId"
        @toggle="(c2) => emit('toggle', c2)"
        @select-request="(r) => emit('select-request', r)"
        @new-request="(id) => emit('new-request', id)"
        @new-folder="(id) => emit('new-folder', id)"
      />

      <button
        v-for="r in requests"
        :key="r.id"
        class="flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-left text-sm text-muted hover:bg-border/30 hover:text-foreground"
        @click="emit('select-request', r)"
      >
        <component
          :is="protocolMeta(r.protocol || 'http').icon"
          :size="14"
          class="shrink-0 text-success"
        />
        <span class="truncate">{{ r.name || r.method + " " + r.url || t("common.collectionUntitled") }}</span>
      </button>

      <div
        v-if="children.length === 0 && requests.length === 0"
        class="px-2 py-1 text-xs text-muted/60"
      >
        empty
      </div>
    </div>
  </div>
</template>
