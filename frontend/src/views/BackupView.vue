<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useProjectStore } from "@/stores/project";
import { useCollectionStore } from "@/stores/collection";
import { useEnvironmentStore } from "@/stores/environment";
import { useToast } from "@/composables/useToast";
import {
  createCollection,
  saveRequest,
  updateCollection,
} from "@/api/collection";
import { createEnvironment, upsertGlobal } from "@/api/environment";
import { parseEnvVars } from "@/lib/vars";
import type { Collection, SavedRequest } from "@/types/project";

const props = defineProps<{ projectId: number }>();
const { t } = useI18n();
const project = useProjectStore();
const collectionStore = useCollectionStore();
const envStore = useEnvironmentStore();
const toast = useToast();

const fileInput = ref<HTMLInputElement | null>(null);
const exporting = ref(false);
const importing = ref(false);

interface BackupEnvelope {
  type: "apiforge-backup";
  version: number;
  exportedAt: string;
  project: { id: number; name?: string };
  collections: Collection[];
  requests: SavedRequest[];
  environments: { name: string; kind: "env" | "global"; values: string }[];
}

function download(text: string, filename: string) {
  const blob = new Blob([text], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

async function exportBackup() {
  exporting.value = true;
  try {
    // 确保所有集合下的请求都已加载，备份才完整
    for (const c of collectionStore.collections) {
      if (!collectionStore.requestsByCollection[c.id]) {
        await collectionStore.fetchRequests(props.projectId, c.id);
      }
    }
    const requests: SavedRequest[] = [];
    for (const list of Object.values(collectionStore.requestsByCollection)) {
      for (const r of list) requests.push(r);
    }
    const env = envStore.environments.map((e) => ({
      name: e.name,
      kind: e.kind,
      values: e.values,
    }));
    const backup: BackupEnvelope = {
      type: "apiforge-backup",
      version: 1,
      exportedAt: new Date().toISOString(),
      project: { id: props.projectId, name: project.current?.name },
      collections: collectionStore.collections,
      requests,
      environments: env,
    };
    const date = new Date().toISOString().slice(0, 10);
    download(
      JSON.stringify(backup, null, 2),
      `apiforge-backup-${project.current?.name || props.projectId}-${date}.json`,
    );
    toast.success(t("common.backupExported"));
  } catch {
    toast.error(t("common.backupImportFail"));
  } finally {
    exporting.value = false;
  }
}

function triggerImport() {
  fileInput.value?.click();
}

async function importBackup(file: File) {
  importing.value = true;
  try {
    const raw = JSON.parse(await file.text()) as BackupEnvelope;
    if (raw?.type !== "apiforge-backup" || !Array.isArray(raw.collections)) {
      toast.error(t("common.backupImportFail"));
      return;
    }
    // 逐层重建集合（任意深度），建立旧 id → 新 id 映射
    const idMap = new Map<number, number>();
    let remaining = [...raw.collections];
    let progress = true;
    while (remaining.length && progress) {
      progress = false;
      const next: Collection[] = [];
      for (const c of remaining) {
        const parentNew = c.parentId == null ? null : idMap.get(c.parentId);
        if (c.parentId != null && parentNew === undefined) {
          next.push(c);
          continue;
        }
        const created = await createCollection(props.projectId, {
          parentId: parentNew,
          name: c.name,
          sortOrder: c.sortOrder,
        });
        idMap.set(c.id, created.id);
        if (c.variables) {
          await updateCollection(props.projectId, created.id, { variables: c.variables });
        }
        progress = true;
      }
      remaining = next;
    }
    // 恢复请求（按新集合 id 归属）
    for (const r of raw.requests) {
      const newCol = idMap.get(r.collectionId);
      if (newCol === undefined) continue;
      const payload = {
        protocol: r.protocol,
        name: r.name,
        method: r.method,
        url: r.url,
        headers: r.headers,
        body: r.body,
        auth: r.auth,
        preRequestScript: r.preRequestScript,
        testScript: r.testScript,
        extractRules: r.extractRules,
      };
      await saveRequest(props.projectId, newCol, payload);
    }
    // 恢复环境（global 走 upsert 覆盖单例，其余新建）
    for (const e of raw.environments || []) {
      if (e.kind === "global") {
        await upsertGlobal(props.projectId, parseEnvVars(e.values || "[]"));
      } else {
        await createEnvironment(props.projectId, {
          name: e.name,
          values: parseEnvVars(e.values || "[]"),
        });
      }
    }
    await collectionStore.fetchCollections(props.projectId);
    await envStore.fetchEnvironments(props.projectId);
    toast.success(t("common.backupImported"));
  } catch {
    toast.error(t("common.backupImportFail"));
  } finally {
    importing.value = false;
  }
}

async function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  await importBackup(file);
}
</script>

<template>
  <div class="h-full overflow-y-auto scroll-thin">
    <div class="mx-auto max-w-3xl space-y-6 p-6">
      <h2 class="text-lg font-semibold text-foreground">{{ t("common.backup") }}</h2>

      <div class="rounded-xl border border-border bg-surface p-4">
        <p class="mb-4 text-sm text-muted">{{ t("common.backupHint") }}</p>
        <div class="flex flex-wrap gap-3">
          <button
            class="flex items-center gap-2 rounded-lg bg-gradient-to-r from-primary to-primary-3 px-4 py-2 text-sm font-medium text-white shadow-glow transition-opacity hover:opacity-90 disabled:opacity-60"
            :disabled="exporting"
            @click="exportBackup"
          >
            {{ exporting ? t("common.loading") : t("common.exportBackup") }}
          </button>
          <button
            class="flex items-center gap-2 rounded-lg border border-border bg-surface px-4 py-2 text-sm text-foreground transition-colors hover:border-primary/40 disabled:opacity-60"
            :disabled="importing"
            @click="triggerImport"
          >
            {{ importing ? t("common.loading") : t("common.importBackup") }}
          </button>
          <input
            ref="fileInput"
            type="file"
            accept=".json,application/json"
            class="hidden"
            @change="onFileChange"
          />
        </div>
      </div>
    </div>
  </div>
</template>
