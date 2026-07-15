<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useProjectStore } from "@/stores/project";
import { useToast } from "@/composables/useToast";
import AppHeader from "@/components/layout/AppHeader.vue";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import Dialog from "@/components/ui/Dialog.vue";
import { FolderKanban, Plus, ArrowRight } from "lucide-vue-next";

const { t } = useI18n();
const router = useRouter();
const project = useProjectStore();
const toast = useToast();

const creating = ref(false);
const name = ref("");
const description = ref("");

onMounted(() => project.fetchProjects());

async function create() {
  if (!name.value) return toast.error(t("common.name"));
  const p = await project.create(name.value, description.value);
  toast.success(t("common.success"));
  creating.value = false;
  name.value = "";
  description.value = "";
  router.push(`/project/${p.id}`);
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
            <ArrowRight :size="16" class="text-muted opacity-0 transition-opacity group-hover:opacity-100" />
          </div>
          <h3 class="mt-3 truncate text-base font-semibold text-foreground">{{ p.name }}</h3>
          <p class="mt-1 line-clamp-2 h-10 text-sm text-muted">{{ p.description || "—" }}</p>
          <div class="mt-3 text-xs text-muted/70">{{ p.createdAt.slice(0, 10) }}</div>
        </Card>
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
  </div>
</template>
