<script setup lang="ts">
import { onMounted, ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useProjectStore } from "@/stores/project";
import { useAuthStore } from "@/stores/auth";
import { useToast } from "@/composables/useToast";
import AppHeader from "@/components/layout/AppHeader.vue";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import Dialog from "@/components/ui/Dialog.vue";
import type { Project } from "@/types/project";
import { FolderKanban, Plus, ArrowRight, Pencil, Trash2 } from "lucide-vue-next";

const { t } = useI18n();
const router = useRouter();
const project = useProjectStore();
const toast = useToast();

const creating = ref(false);
const name = ref("");
const description = ref("");

onMounted(() => project.fetchProjects(1));

const auth = useAuthStore();

function canManage(p: Project) {
  return auth.user?.id === p.ownerId;
}

// 重命名项目
const renaming = ref(false);
const editId = ref<number | null>(null);
const editName = ref("");
const editDesc = ref("");
function startRename(p: Project) {
  editId.value = p.id;
  editName.value = p.name;
  editDesc.value = p.description || "";
  renaming.value = true;
}
async function doRename() {
  if (editId.value == null || !editName.value.trim()) return toast.error(t("common.name"));
  await project.update(editId.value, editName.value.trim(), editDesc.value);
  toast.success(t("common.success"));
  renaming.value = false;
}

// 删除项目
const deleting = ref(false);
const deleteId = ref<number | null>(null);
function startDelete(p: Project) {
  deleteId.value = p.id;
  deleting.value = true;
}
async function doDelete() {
  if (deleteId.value == null) return;
  await project.remove(deleteId.value);
  toast.success(t("common.delete"));
  deleting.value = false;
  await project.fetchProjects(project.page);
}

async function create() {
  if (!name.value) return toast.error(t("common.name"));
  const p = await project.create(name.value, description.value);
  toast.success(t("common.success"));
  creating.value = false;
  name.value = "";
  description.value = "";
  router.push(`/project/${p.id}`);
}

// 简单分页（M15）
const totalPages = computed(() =>
  project.perPage > 0 ? Math.max(1, Math.ceil(project.total / project.perPage)) : 1,
);
function gotoPage(p: number) {
  if (p < 1 || p > totalPages.value) return;
  project.fetchProjects(p);
}

function openProject(id: number) {
  router.push(`/project/${id}`);
}
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />
    <main class="mx-auto max-w-6xl px-4 pt-20 pb-10">
      <div class="mb-6 flex items-center justify-between">
        <h1 class="text-2xl font-semibold text-foreground">{{ t("project.listTitle") }}</h1>
        <Button @click="creating = true"><Plus :size="15" /> {{ t("project.newProject") }}</Button>
      </div>

      <div v-if="project.projects.length === 0" class="rounded-xl border border-dashed border-border p-12 text-center text-muted">
        {{ t("project.noProjects") }}
      </div>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card
          v-for="p in project.projects"
          :key="p.id"
          class="group cursor-pointer p-5 transition-all hover:border-primary/50 hover:shadow-glow"
          @click="openProject(p.id)"
        >
          <div class="flex items-start justify-between">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-primary/30 to-primary-3/20 text-primary">
              <FolderKanban :size="20" />
            </div>
            <div class="flex items-center gap-1" @click.stop>
              <template v-if="canManage(p)">
                <button class="rounded p-1 text-muted hover:text-foreground" :title="t('project.renameProject')" @click="startRename(p)"><Pencil :size="15" /></button>
                <button class="rounded p-1 text-muted hover:text-danger" :title="t('project.deleteProject')" @click="startDelete(p)"><Trash2 :size="15" /></button>
              </template>
              <ArrowRight v-else :size="16" class="text-muted opacity-0 transition-opacity group-hover:opacity-100" />
            </div>
          </div>
          <h3 class="mt-3 truncate text-base font-semibold text-foreground">{{ p.name }}</h3>
          <p class="mt-1 line-clamp-2 h-10 text-sm text-muted">{{ p.description || "—" }}</p>
          <div class="mt-3 text-xs text-muted/70">{{ p.createdAt.slice(0, 10) }}</div>
        </Card>
      </div>

      <div v-if="totalPages > 1" class="mt-6 flex items-center justify-center gap-3 text-sm">
        <Button variant="ghost" :disabled="project.page <= 1" @click="gotoPage(project.page - 1)">{{ t("common.prev") }}</Button>
        <span class="text-muted">{{ project.page }} / {{ totalPages }} ({{ project.total }})</span>
        <Button variant="ghost" :disabled="project.page >= totalPages" @click="gotoPage(project.page + 1)">{{ t("common.next") }}</Button>
      </div>
    </main>

    <Dialog :open="creating" :title="t('project.newProjectTitle')" @close="creating = false">
      <div class="space-y-3">
        <div>
          <label class="mb-1 block text-xs font-medium text-muted">{{ t("common.name") }}</label>
          <Input v-model="name" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-muted">{{ t("common.description") }}</label>
          <Input v-model="description" />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="ghost" @click="creating = false">{{ t("common.cancel") }}</Button>
          <Button @click="create">{{ t("project.createBtn") }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- 重命名项目 -->
    <Dialog :open="renaming" :title="t('project.renameProject')" @close="renaming = false">
      <div class="space-y-3">
        <div>
          <label class="mb-1 block text-xs font-medium text-muted">{{ t("common.name") }}</label>
          <Input v-model="editName" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-muted">{{ t("common.description") }}</label>
          <Input v-model="editDesc" />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="ghost" @click="renaming = false">{{ t("common.cancel") }}</Button>
          <Button @click="doRename">{{ t("common.save") }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- 删除项目确认 -->
    <Dialog :open="deleting" :title="t('project.deleteProject')" @close="deleting = false">
      <p class="text-sm text-muted">{{ t("project.confirmDeleteProject") }}</p>
      <div class="flex justify-end gap-2 pt-4">
        <Button variant="ghost" @click="deleting = false">{{ t("common.cancel") }}</Button>
        <Button variant="danger" @click="doDelete">{{ t("common.delete") }}</Button>
      </div>
    </Dialog>
  </div>
</template>
