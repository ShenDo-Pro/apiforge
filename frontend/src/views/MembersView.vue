<script setup lang="ts">
import { ref, reactive, computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useProjectStore } from "@/stores/project";
import { useToast } from "@/composables/useToast";
import Input from "@/components/ui/Input.vue";
import Button from "@/components/ui/Button.vue";
import Select from "@/components/ui/Select.vue";
import Badge from "@/components/ui/Badge.vue";
import Dialog from "@/components/ui/Dialog.vue";
import type { ProjectMember } from "@/types/project";
import { Trash2, UserPlus, Search, Pencil } from "lucide-vue-next";

const props = defineProps<{ projectId: number }>();
const { t } = useI18n();
const project = useProjectStore();
const toast = useToast();

const query = ref("");
const newUserId = ref("");
const newRole = ref("developer");
const newPerms = ref({ add: true, edit: true, delete: true });

const roleTone: Record<string, string> = {
  owner: "primary",
  maintainer: "info",
  developer: "default",
};

function roleLabel(r: string) {
  return r === "owner" ? t("project.owner") : r === "maintainer" ? t("project.maintainer") : t("project.developer");
}

onMounted(() => project.fetchMembers(props.projectId));

// 按用户名或用户 ID 检索
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return project.members;
  return project.members.filter(
    (m) =>
      (m.username || "").toLowerCase().includes(q) || String(m.userId).includes(q),
  );
});

async function add() {
  const id = Number(newUserId.value);
  if (!id) {
    toast.error(t("project.memberName"));
    return;
  }
  await project.addMember(props.projectId, id, newRole.value, { ...newPerms.value });
  toast.success(t("common.success"));
  newUserId.value = "";
}

async function remove(userId: number) {
  await project.removeMember(props.projectId, userId);
  toast.info(t("common.delete"));
}

// 编辑成员角色与权限（E5）
const editingUserId = ref<number | null>(null);
const editRole = ref("developer");
const editPerms = reactive({ add: true, edit: true, delete: true });

function roleEditable(m: ProjectMember) {
  return m.role !== "owner";
}
function startEditRole(m: ProjectMember) {
  editingUserId.value = m.userId;
  editRole.value = m.role;
  try {
    const p = JSON.parse(m.permissions || "{}");
    editPerms.add = !!p.add;
    editPerms.edit = !!p.edit;
    editPerms.delete = !!p.delete;
  } catch {
    editPerms.add = editPerms.edit = editPerms.delete = true;
  }
}
async function saveRole() {
  if (editingUserId.value == null) return;
  await project.updateMember(props.projectId, editingUserId.value, editRole.value, {
    add: editPerms.add,
    edit: editPerms.edit,
    delete: editPerms.delete,
  });
  toast.success(t("common.success"));
  editingUserId.value = null;
  await project.fetchMembers(props.projectId);
}
</script>

<template>
  <div class="h-full overflow-y-auto scroll-thin">
    <div class="mx-auto max-w-3xl space-y-6 p-6">
      <h2 class="text-lg font-semibold text-foreground">{{ t("project.membersTitle") }}</h2>

      <!-- 检索 -->
      <div class="relative">
        <Search :size="15" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
        <Input v-model="query" :placeholder="t('project.membersSearchPlaceholder')" class="pl-9" />
      </div>

      <!-- 成员列表 -->
      <div class="space-y-2">
        <div
          v-for="m in filtered"
          :key="m.id"
          class="flex items-center justify-between rounded-xl border border-border bg-surface px-4 py-3"
        >
          <div class="flex items-center gap-3">
            <span class="flex h-9 w-9 items-center justify-center rounded-full bg-primary/15 text-sm font-medium text-primary">
              {{ (m.username || String(m.userId)).charAt(0).toUpperCase() }}
            </span>
            <div class="text-sm">
              <div class="text-foreground">{{ m.username || ("#" + m.userId) }}</div>
              <div class="text-xs text-muted">#{{ m.userId }}</div>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <Badge :tone="(roleTone[m.role] as any)">{{ roleLabel(m.role) }}</Badge>
            <button v-if="roleEditable(m)" class="text-muted hover:text-foreground" :title="t('project.editRole')" @click="startEditRole(m)">
              <Pencil :size="15" />
            </button>
            <button class="text-muted hover:text-danger" @click="remove(m.userId)">
              <Trash2 :size="16" />
            </button>
          </div>
        </div>
        <div v-if="filtered.length === 0" class="rounded-xl border border-dashed border-border px-4 py-8 text-center text-sm text-muted">
          {{ t("project.noMembers") }}
        </div>
      </div>

      <!-- 添加成员 -->
      <div class="space-y-3 rounded-xl border border-border bg-surface p-4">
        <div class="text-sm font-medium text-muted">{{ t("project.addMember") }}</div>
        <div class="flex flex-wrap gap-2">
          <Input v-model="newUserId" :placeholder="t('project.memberName')" class="w-40" />
          <Select v-model="newRole" class="w-36">
            <option value="maintainer">{{ t("project.maintainer") }}</option>
            <option value="developer">{{ t("project.developer") }}</option>
          </Select>
          <Button @click="add"><UserPlus :size="15" /> {{ t("common.add") }}</Button>
        </div>
        <div class="flex gap-4 text-xs text-muted">
          <label class="flex items-center gap-1"><input type="checkbox" v-model="newPerms.add" />{{ t("project.addPerm") }}</label>
          <label class="flex items-center gap-1"><input type="checkbox" v-model="newPerms.edit" />{{ t("project.editPerm") }}</label>
          <label class="flex items-center gap-1"><input type="checkbox" v-model="newPerms.delete" />{{ t("project.deletePerm") }}</label>
        </div>
      </div>

      <!-- 编辑成员角色与权限 -->
      <Dialog :open="editingUserId !== null" :title="t('project.editRole')" @close="editingUserId = null">
        <div class="space-y-4">
          <div>
            <label class="mb-1 block text-xs font-medium text-muted">{{ t("project.role") }}</label>
            <Select v-model="editRole" class="w-40">
              <option value="maintainer">{{ t("project.maintainer") }}</option>
              <option value="developer">{{ t("project.developer") }}</option>
            </Select>
          </div>
          <div class="flex gap-4 text-xs text-muted">
            <label class="flex items-center gap-1"><input type="checkbox" v-model="editPerms.add" />{{ t("project.addPerm") }}</label>
            <label class="flex items-center gap-1"><input type="checkbox" v-model="editPerms.edit" />{{ t("project.editPerm") }}</label>
            <label class="flex items-center gap-1"><input type="checkbox" v-model="editPerms.delete" />{{ t("project.deletePerm") }}</label>
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <Button variant="ghost" @click="editingUserId = null">{{ t("common.cancel") }}</Button>
            <Button @click="saveRole">{{ t("common.save") }}</Button>
          </div>
        </div>
      </Dialog>
    </div>
  </div>
</template>
