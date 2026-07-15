<script setup lang="ts">
import { onMounted, ref, computed } from "vue";
import { useRoute } from "vue-router";
import { useProjectStore } from "@/stores/project";
import { useCollectionStore } from "@/stores/collection";
import AppHeader from "@/components/layout/AppHeader.vue";
import AppSidebar from "@/components/layout/AppSidebar.vue";
import NewRequestDialog from "@/components/project/NewRequestDialog.vue";
import MembersView from "@/views/MembersView.vue";
import SettingsView from "@/views/SettingsView.vue";
import HttpClient from "@/views/HttpClient.vue";
import WebSocketClient from "@/views/WebSocketClient.vue";
import MqttClient from "@/views/MqttClient.vue";
import SocketClient from "@/views/SocketClient.vue";
import GraphQLClient from "@/views/GraphQLClient.vue";
import SocketIOClient from "@/views/SocketIOClient.vue";
import GrpcClient from "@/views/GrpcClient.vue";
import PipelineView from "@/views/PipelineView.vue";
import EnvironmentsView from "@/views/EnvironmentsView.vue";
import Dialog from "@/components/ui/Dialog.vue";
import Input from "@/components/ui/Input.vue";
import Button from "@/components/ui/Button.vue";
import Select from "@/components/ui/Select.vue";
import CollectionVarsDialog from "@/components/project/CollectionVarsDialog.vue";
import { useEnvironmentStore } from "@/stores/environment";
import { useI18n } from "vue-i18n";
import { PROTOCOLS } from "@/lib/protocols";
import { Settings } from "lucide-vue-next";
import type { SavedRequest, Collection } from "@/types/project";

const route = useRoute();
const project = useProjectStore();
const collectionStore = useCollectionStore();
const envStore = useEnvironmentStore();
const { t } = useI18n();

// 活动环境选择器（全局，与请求编辑器共享）：仅列出用户环境，不含全局单例
const envList = computed(() => envStore.environments.filter((e) => e.kind === "env"));
const activeEnvId = computed<string>({
  get: () => envStore.activeId ?? "",
  set: (v) => envStore.setActive(v ? Number(v) : null),
});

const projectId = Number(route.params.projectId);
const protocol = ref("http");
const selectedRequest = ref<SavedRequest | null>(null);
// 新建请求时预选的目标集合与名称（来自「新建请求」弹框）
const draftCollectionId = ref<number | null>(null);
const draftName = ref("");
// 主区域视图：编辑器 / 成员 / 流水线 / 设置 / 环境
const view = ref<"editor" | "members" | "pipeline" | "settings" | "environments">("editor");

const newRequestOpen = ref(false);
const folderOpen = ref(false);
const folderParentId = ref<number | null>(null);
const folderName = ref("");

onMounted(async () => {
  await project.fetchProjects();
  const p = project.projects.find((x) => x.id === projectId);
  if (p) project.setCurrent(p);
  // 加载环境（服务端持久化），驱动活动环境选择器与变量替换
  await envStore.fetchEnvironments(projectId);
});

// 打开已保存请求：按协议路由到对应客户端
function onSelectRequest(r: SavedRequest) {
  selectedRequest.value = r;
  protocol.value = r.protocol || "http";
  draftCollectionId.value = null;
  draftName.value = "";
  view.value = "editor";
  // 注入当前请求所属集合（含父文件夹）的变量，供四层覆盖解析
  if (r.collectionId) {
    envStore.setActiveCollection(r.collectionId, collectionStore.mergedVarsOf(r.collectionId));
  } else {
    envStore.setActiveCollection(null, {});
  }
}

// 弹框确认：定位到目标集合与协议，预填名称，打开编辑器
function onNewRequestConfirm(payload: { collectionId: number; protocol: string; name: string }) {
  selectedRequest.value = null;
  protocol.value = payload.protocol;
  draftCollectionId.value = payload.collectionId;
  draftName.value = payload.name;
  view.value = "editor";
  newRequestOpen.value = false;
}

// 新建目录弹框：来自侧边栏（parentId=null）或集合节点（parentId=节点id）
function onNewFolder(parentId: number | null) {
  folderParentId.value = parentId;
  folderName.value = "";
  folderOpen.value = true;
}
async function confirmFolder() {
  const name = folderName.value.trim();
  if (!name) return;
  await collectionStore.createCollection(projectId, { parentId: folderParentId.value, name });
  folderOpen.value = false;
}

// 集合变量编辑
const varsEditOpen = ref(false);
const varsEditNode = ref<Collection | null>(null);
function onEditVars(node: Collection) {
  varsEditNode.value = node;
  varsEditOpen.value = true;
}
</script>

<template>
  <div class="flex h-screen flex-col bg-background">
    <AppHeader />
    <div class="flex flex-1 pt-14">
      <AppSidebar
        :project-id="projectId"
      @select-request="onSelectRequest"
      @new-request="newRequestOpen = true"
      @new-folder="onNewFolder"
      @edit-vars="onEditVars"
      @open-view="(v) => (view = v)"
    />
      <main class="flex-1 overflow-hidden">
        <!-- 请求编辑器 -->
        <template v-if="view === 'editor'">
          <div class="flex items-center gap-2 border-b border-border px-3 py-2">
            <Select v-model="protocol" class="w-40" :title="t('common.request')">
              <option v-for="p in PROTOCOLS" :key="p.key" :value="p.key">{{ p.label }}</option>
            </Select>

            <!-- 活动环境选择 + 管理入口 -->
            <Select v-model="activeEnvId" class="w-48" :title="t('common.envManage')">
              <option value="">{{ t("common.envNone") }}</option>
              <option v-for="e in envList" :key="e.id" :value="e.id">
                {{ e.name || t("common.envNone") }}
              </option>
            </Select>
            <button
              class="flex h-10 w-10 items-center justify-center rounded-lg border border-border bg-surface text-muted hover:border-primary/60 hover:text-foreground"
              :title="t('common.envManage')"
              @click="view = 'environments'"
            >
              <Settings :size="16" />
            </button>
          </div>
          <HttpClient
            v-if="protocol === 'http'"
            :project-id="projectId"
            :protocol="protocol"
            :request="selectedRequest"
            :default-collection-id="draftCollectionId"
            :default-name="draftName"
            @saved="(r) => (selectedRequest = r)"
          />
          <WebSocketClient
            v-else-if="protocol === 'ws'"
            :project-id="projectId"
            :request="selectedRequest"
            :default-collection-id="draftCollectionId"
            :default-name="draftName"
            @saved="(r) => (selectedRequest = r)"
          />
          <MqttClient
            v-else-if="protocol === 'mqtt'"
            :project-id="projectId"
            :request="selectedRequest"
            :default-collection-id="draftCollectionId"
            :default-name="draftName"
            @saved="(r) => (selectedRequest = r)"
          />
          <SocketClient
            v-else-if="protocol === 'socket'"
            :project-id="projectId"
            :request="selectedRequest"
            :default-collection-id="draftCollectionId"
            :default-name="draftName"
            @saved="(r) => (selectedRequest = r)"
          />
          <GraphQLClient
            v-else-if="protocol === 'graphql'"
            :project-id="projectId"
            :request="selectedRequest"
            :default-collection-id="draftCollectionId"
            :default-name="draftName"
            @saved="(r) => (selectedRequest = r)"
          />
          <SocketIOClient
            v-else-if="protocol === 'socketio'"
            :project-id="projectId"
            :request="selectedRequest"
            :default-collection-id="draftCollectionId"
            :default-name="draftName"
            @saved="(r) => (selectedRequest = r)"
          />
          <GrpcClient
            v-else-if="protocol === 'grpc'"
            :project-id="projectId"
            :request="selectedRequest"
            :default-collection-id="draftCollectionId"
            :default-name="draftName"
            @saved="(r) => (selectedRequest = r)"
          />
          <div v-else class="p-6 text-sm text-muted">{{ t("common.comingSoon") }}</div>
        </template>

        <!-- 成员 -->
        <MembersView v-else-if="view === 'members'" :project-id="projectId" />
        <!-- 流水线 -->
        <PipelineView v-else-if="view === 'pipeline'" :project-id="projectId" />
        <!-- 设置 -->
        <SettingsView v-else-if="view === 'settings'" :project-id="projectId" />
        <!-- 环境整页 -->
        <EnvironmentsView v-else-if="view === 'environments'" :project-id="projectId" />
      </main>
    </div>

    <!-- 新建请求弹框（居中） -->
    <NewRequestDialog
      :open="newRequestOpen"
      :project-id="projectId"
      @close="newRequestOpen = false"
      @confirm="onNewRequestConfirm"
    />

    <!-- 新建目录弹框（居中） -->
    <Dialog :open="folderOpen" :title="t('common.collectionNewFolder')" @close="folderOpen = false">
      <div class="space-y-3">
        <div>
          <label class="mb-1 block text-xs text-muted">{{ t("common.collectionFolderName") }}</label>
          <Input v-model="folderName" @keyup.enter="confirmFolder" />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="ghost" @click="folderOpen = false">{{ t("common.cancel") }}</Button>
          <Button @click="confirmFolder">{{ t("common.confirm") }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- 集合变量编辑弹框 -->
    <CollectionVarsDialog
      :open="varsEditOpen"
      :collection="varsEditNode"
      :project-id="projectId"
      @close="varsEditOpen = false"
    />
  </div>
</template>
