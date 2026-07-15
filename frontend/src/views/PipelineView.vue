<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useToast } from "@/composables/useToast";
import { Plus, Play, Copy, RefreshCw, Trash2, Eye } from "lucide-vue-next";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import Textarea from "@/components/ui/Textarea.vue";
import Select from "@/components/ui/Select.vue";
import Badge from "@/components/ui/Badge.vue";
import Dialog from "@/components/ui/Dialog.vue";
import Label from "@/components/ui/Label.vue";
import {
  listPipelines,
  getPipeline,
  createPipeline,
  updatePipeline,
  deletePipeline,
  runPipeline,
  listRuns,
  getRun,
  regenerateToken,
  listProjectRequests,
} from "@/api/pipeline";
import type { Pipeline, PipelineRun, PipelineStepResult, Assertion, AssertionType } from "@/types/pipeline";
import type { SavedRequest } from "@/types/project";

const props = defineProps<{ projectId: number }>();
const { t } = useI18n();
const toast = useToast();

const methods = ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"];
const assertTypes: AssertionType[] = ["status", "body_contains", "header_equals", "max_duration_ms"];

interface EditStep {
  id?: number;
  mode: "ref" | "inline";
  savedRequestId: number | null;
  name: string;
  enabled: boolean;
  method: string;
  url: string;
  headers: string;
  body: string;
  assertList: Assertion[];
}

const pipelines = ref<Pipeline[]>([]);
const selectedId = ref<number | null>(null);
const pipeline = ref<Pipeline | null>(null);
const editSteps = ref<EditStep[]>([]);
const name = ref("");
const description = ref("");
const webhookURL = ref("");
const projectRequests = ref<SavedRequest[]>([]);
const runs = ref<PipelineRun[]>([]);
const selectedRun = ref<PipelineRun | null>(null);
const responseStep = ref<PipelineStepResult | null>(null);
const loading = ref(false);
const running = ref(false);

function parseAssertions(raw: string): Assertion[] {
  if (!raw) return [];
  try {
    return JSON.parse(raw);
  } catch {
    return [];
  }
}
function serializeAssertions(list: Assertion[]): string {
  return JSON.stringify(list ?? []);
}

async function loadPipelines() {
  pipelines.value = await listPipelines(props.projectId);
}

async function selectPipeline(id: number) {
  selectedId.value = id;
  const p = await getPipeline(props.projectId, id);
  pipeline.value = p;
  name.value = p.name;
  description.value = p.description;
  webhookURL.value = p.webhookURL || "";
  editSteps.value = (p.steps || []).map((s) => ({
    id: s.id,
    mode: s.savedRequestId ? "ref" : "inline",
    savedRequestId: s.savedRequestId ?? null,
    name: s.name,
    enabled: s.enabled,
    method: s.method,
    url: s.url,
    headers: s.headers,
    body: s.body,
    assertList: parseAssertions(s.assertions),
  }));
  if (projectRequests.value.length === 0) {
    projectRequests.value = await listProjectRequests(props.projectId);
  }
  await loadRuns();
}

async function loadRuns() {
  if (!selectedId.value) return;
  runs.value = await listRuns(props.projectId, selectedId.value);
}

async function newPipeline() {
  const p = await createPipeline(props.projectId, "New Pipeline");
  await loadPipelines();
  await selectPipeline(p.id);
}

async function savePipeline() {
  if (!selectedId.value) return;
  loading.value = true;
  try {
    const steps = editSteps.value.map((s) => ({
      id: s.id,
      savedRequestId: s.mode === "ref" ? s.savedRequestId : null,
      name: s.name,
      enabled: s.enabled,
      method: s.method,
      url: s.url,
      headers: s.headers,
      body: s.body,
      assertions: serializeAssertions(s.assertList),
    }));
    await updatePipeline(props.projectId, selectedId.value, {
      name: name.value,
      description: description.value,
      steps,
    });
    toast.success(t("pipeline.saved"));
    await loadPipelines();
  } catch (e) {
    toast.error(String(e));
  } finally {
    loading.value = false;
  }
}

async function runNow() {
  if (!selectedId.value) return;
  running.value = true;
  try {
    const run = await runPipeline(props.projectId, selectedId.value);
    selectedRun.value = run;
    await loadRuns();
    toast.success(run.status === "passed" ? t("pipeline.passed") : t("pipeline.failed"));
  } catch (e) {
    toast.error(String(e));
  } finally {
    running.value = false;
  }
}

async function viewRun(id: number) {
  if (!selectedId.value) return;
  selectedRun.value = await getRun(props.projectId, selectedId.value, id);
}

async function removePipeline() {
  if (!selectedId.value) return;
  if (!confirm(t("pipeline.confirmDelete"))) return;
  await deletePipeline(props.projectId, selectedId.value);
  selectedId.value = null;
  pipeline.value = null;
  await loadPipelines();
}

async function regen() {
  if (!selectedId.value) return;
  const r = await regenerateToken(props.projectId, selectedId.value);
  webhookURL.value = r.webhookURL;
  toast.success(t("pipeline.regenerated"));
}

function copyWebhook() {
  navigator.clipboard.writeText(webhookURL.value);
  toast.success(t("pipeline.copied"));
}

function addStep() {
  editSteps.value.push({
    mode: "inline",
    savedRequestId: null,
    name: "Step " + (editSteps.value.length + 1),
    enabled: true,
    method: "GET",
    url: "",
    headers: "{}",
    body: "",
    assertList: [],
  });
}
function removeStep(i: number) {
  editSteps.value.splice(i, 1);
}
function addAssertion(step: EditStep) {
  step.assertList.push({ type: "status", expected: "200" });
}
function removeAssertion(step: EditStep, i: number) {
  step.assertList.splice(i, 1);
}
function onRefIdChange(step: EditStep, v: string) {
  step.savedRequestId = v ? Number(v) : null;
  if (step.savedRequestId) {
    const r = projectRequests.value.find((x) => x.id === step.savedRequestId);
    if (r) {
      step.method = r.method;
      step.url = r.url;
      step.headers = r.headers;
      step.body = r.body;
      step.name = r.name;
    }
  }
}

function selectedRequestLabel(step: EditStep): string {
  const r = projectRequests.value.find((x) => x.id === step.savedRequestId);
  return r ? `${r.method} ${r.name} · ${r.url}` : "";
}

function parseResHeaders(raw: string): Record<string, string> {
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch {
    return {};
  }
}
function parseAssertResults(raw: string): AssertionResult[] {
  if (!raw) return [];
  try {
    return JSON.parse(raw);
  } catch {
    return [];
  }
}
interface AssertionResult {
  type: string;
  expected: string;
  header?: string;
  actual: string;
  passed: boolean;
}

function assertLabel(type: string): string {
  return t("pipeline.assertType_" + type);
}

function statusTone(s: string): "success" | "danger" | "warning" {
  if (s === "passed") return "success";
  if (s === "failed") return "danger";
  return "warning";
}

onMounted(loadPipelines);
</script>

<template>
  <div class="flex h-full">
    <!-- 流水线列表 -->
    <aside class="flex w-64 shrink-0 flex-col border-r border-border glass">
      <div class="flex items-center justify-between border-b border-border px-4 py-3">
        <span class="text-sm font-semibold text-foreground">{{ t("pipeline.title") }}</span>
        <Button size="sm" variant="ghost" @click="newPipeline">
          <Plus :size="14" /> {{ t("pipeline.new") }}
        </Button>
      </div>
      <div class="flex-1 space-y-1 overflow-y-auto scroll-thin p-2">
        <button
          v-for="p in pipelines"
          :key="p.id"
          class="w-full rounded-lg px-3 py-2 text-left text-sm transition-colors"
          :class="p.id === selectedId ? 'bg-primary/15 text-primary' : 'text-muted hover:bg-border/30'"
          @click="selectPipeline(p.id)"
        >
          <div class="truncate font-medium">{{ p.name }}</div>
          <div class="truncate text-xs opacity-60">{{ p.description || "—" }}</div>
        </button>
        <p v-if="pipelines.length === 0" class="px-3 py-6 text-center text-xs text-muted">
          {{ t("pipeline.noPipeline") }}
        </p>
      </div>
    </aside>

    <!-- 主区 -->
    <main class="flex-1 overflow-y-auto scroll-thin p-5" v-if="pipeline">
      <!-- 元信息与操作 -->
      <Card class="mb-4 p-4">
        <div class="flex flex-wrap items-end gap-3">
          <div class="flex-1 min-w-[200px]">
            <Label>{{ t("pipeline.name") }}</Label>
            <Input v-model="name" class="mt-1" />
          </div>
          <div class="flex-1 min-w-[200px]">
            <Label>{{ t("pipeline.description") }}</Label>
            <Input v-model="description" class="mt-1" />
          </div>
          <Button variant="default" :disabled="loading" @click="savePipeline">
            {{ t("pipeline.save") }}
          </Button>
          <Button variant="secondary" :disabled="running" @click="runNow">
            <Play :size="14" /> {{ running ? t("pipeline.running") : t("pipeline.run") }}
          </Button>
          <Button variant="danger" @click="removePipeline">
            <Trash2 :size="14" /> {{ t("pipeline.delete") }}
          </Button>
        </div>
      </Card>

      <!-- Webhook -->
      <Card class="mb-4 p-4">
        <div class="mb-2 flex items-center gap-2 text-sm font-semibold text-foreground">
          <RefreshCw :size="14" /> {{ t("pipeline.webhook") }}
        </div>
        <p class="mb-2 text-xs text-muted">{{ t("pipeline.subtitle") }}</p>
        <div class="flex items-center gap-2">
          <Input :model-value="webhookURL" readonly class="flex-1 font-mono text-xs" />
          <Button size="sm" variant="outline" @click="copyWebhook">
            <Copy :size="14" /> {{ t("pipeline.copy") }}
          </Button>
          <Button size="sm" variant="ghost" @click="regen">
            <RefreshCw :size="14" /> {{ t("pipeline.regenerateToken") }}
          </Button>
        </div>
      </Card>

      <!-- 步骤 -->
      <Card class="mb-4 p-4">
        <div class="mb-3 flex items-center justify-between">
          <span class="text-sm font-semibold text-foreground">{{ t("pipeline.steps") }}</span>
          <Button size="sm" variant="outline" @click="addStep">
            <Plus :size="14" /> {{ t("pipeline.addStep") }}
          </Button>
        </div>

        <p v-if="editSteps.length === 0" class="py-6 text-center text-xs text-muted">
          {{ t("pipeline.noSteps") }}
        </p>

        <div v-for="(step, i) in editSteps" :key="i" class="mb-3 rounded-xl border border-border bg-surface/50 p-3">
          <div class="flex flex-wrap items-center gap-2">
            <input type="checkbox" v-model="step.enabled" class="accent-primary" :title="t('pipeline.enabled')" />
            <Input v-model="step.name" class="w-48" />
            <Select v-model="step.mode" class="w-40">
              <option value="inline">{{ t("pipeline.inlineMode") }}</option>
              <option value="ref">{{ t("pipeline.refMode") }}</option>
            </Select>
            <Button size="sm" variant="ghost" class="ml-auto" @click="removeStep(i)">
              <Trash2 :size="14" />
            </Button>
          </div>

          <!-- 引用已有请求 -->
          <div v-if="step.mode === 'ref'" class="mt-3">
            <Label>{{ t("pipeline.savedRequest") }}</Label>
            <Select
              :model-value="String(step.savedRequestId ?? '')"
              class="mt-1 w-full"
              @update:model-value="(v: string) => onRefIdChange(step, v)"
            >
              <option value="">—</option>
              <option v-for="r in projectRequests" :key="r.id" :value="r.id">
                {{ r.method }} · {{ r.name }}
              </option>
            </Select>
            <p v-if="step.savedRequestId" class="mt-1 truncate font-mono text-xs text-muted">
              {{ selectedRequestLabel(step) }}
            </p>
          </div>

          <!-- 内联定义 -->
          <div v-else class="mt-3 grid grid-cols-1 gap-2 md:grid-cols-[120px_1fr]">
            <Select v-model="step.method" class="w-full">
              <option v-for="m in methods" :key="m" :value="m">{{ m }}</option>
            </Select>
            <Input v-model="step.url" :placeholder="t('pipeline.url')" />
            <Label class="md:mt-2">{{ t("pipeline.headers") }}</Label>
            <Textarea v-model="step.headers" rows="3" class="w-full" />
            <Label class="md:mt-2">{{ t("pipeline.body") }}</Label>
            <Textarea v-model="step.body" rows="3" class="w-full" />
          </div>

          <!-- 断言 -->
          <div class="mt-3 border-t border-border pt-3">
            <div class="mb-2 flex items-center justify-between">
              <span class="text-xs font-semibold text-muted">{{ t("pipeline.assertions") }}</span>
              <Button size="sm" variant="ghost" @click="addAssertion(step)">
                <Plus :size="12" /> {{ t("pipeline.addAssertion") }}
              </Button>
            </div>
            <p v-if="step.assertList.length === 0" class="text-xs text-muted">
              {{ t("pipeline.noAssertion") }}
            </p>
            <div
              v-for="(a, ai) in step.assertList"
              :key="ai"
              class="mb-2 flex flex-wrap items-center gap-2 rounded-lg bg-surface/60 p-2"
            >
              <Select
                :model-value="a.type"
                class="w-44"
                @update:model-value="(v: string) => (a.type = v as AssertionType)"
              >
                <option v-for="at in assertTypes" :key="at" :value="at">{{ assertLabel(at) }}</option>
              </Select>
              <Input
                v-if="a.type === 'header_equals'"
                v-model="a.header"
                :placeholder="t('pipeline.headerName')"
                class="w-40"
              />
              <Input v-model="a.expected" :placeholder="t('pipeline.expected')" class="w-44" />
              <label class="flex items-center gap-1 text-xs text-muted">
                <input type="checkbox" v-model="a.invert" class="accent-primary" />
                {{ t("pipeline.invert") }}
              </label>
              <Button size="sm" variant="ghost" @click="removeAssertion(step, ai)">
                <Trash2 :size="12" />
              </Button>
            </div>
          </div>
        </div>
      </Card>

      <!-- 运行历史 -->
      <Card class="p-4">
        <div class="mb-3 text-sm font-semibold text-foreground">{{ t("pipeline.history") }}</div>
        <p v-if="runs.length === 0" class="py-4 text-center text-xs text-muted">
          {{ t("pipeline.noRuns") }}
        </p>
        <div v-for="run in runs" :key="run.id" class="mb-2 rounded-lg border border-border bg-surface/40 p-3">
          <div class="flex cursor-pointer items-center gap-3" @click="viewRun(run.id)">
            <Badge :tone="statusTone(run.status)">{{ t("pipeline." + run.status) }}</Badge>
            <span class="text-xs text-muted">{{ t("pipeline.trigger") }}: {{ run.trigger }}</span>
            <span class="text-xs text-muted">{{ run.summary }}</span>
            <span class="ml-auto text-xs text-muted">{{ run.finishedAt }}</span>
          </div>

          <!-- 运行详情 -->
          <div v-if="selectedRun && selectedRun.id === run.id" class="mt-3 space-y-2 border-t border-border pt-3">
            <div
              v-for="res in selectedRun.results"
              :key="res.id"
              class="flex flex-wrap items-center gap-3 rounded-lg bg-surface/60 p-2 text-xs"
            >
              <Badge :tone="statusTone(res.status)">{{ t("pipeline." + res.status) }}</Badge>
              <span class="font-medium text-foreground">{{ res.stepName }}</span>
              <span class="text-muted">{{ res.method }} {{ res.statusCode }}</span>
              <span class="text-muted">{{ res.durationMs }}ms</span>
              <Button size="sm" variant="ghost" class="ml-auto" @click="responseStep = res">
                <Eye :size="12" /> {{ t("pipeline.viewResponse") }}
              </Button>
            </div>
          </div>
        </div>
      </Card>
    </main>

    <main v-else class="flex flex-1 items-center justify-center">
      <div class="text-center">
        <p class="mb-3 text-sm text-muted">{{ t("pipeline.noPipeline") }}</p>
        <Button @click="newPipeline"><Plus :size="14" /> {{ t("pipeline.new") }}</Button>
      </div>
    </main>

    <!-- 响应详情弹窗 -->
    <Dialog :open="!!responseStep" :title="t('pipeline.viewResponse')" @close="responseStep = null">
      <div v-if="responseStep" class="space-y-3 text-xs">
        <div class="flex items-center gap-2">
          <Badge :tone="statusTone(responseStep.status)">{{ t("pipeline." + responseStep.status) }}</Badge>
          <span class="text-muted">{{ responseStep.method }} {{ responseStep.statusCode }} · {{ responseStep.durationMs }}ms</span>
        </div>
        <div>
          <div class="mb-1 font-semibold text-foreground">{{ t("pipeline.assertions") }}</div>
          <div v-if="parseAssertResults(responseStep.assertionResults).length === 0" class="text-muted">
            {{ t("pipeline.noAssertion") }}
          </div>
          <div
            v-for="(ar, i) in parseAssertResults(responseStep.assertionResults)"
            :key="i"
            class="flex items-center gap-2 rounded bg-surface/60 p-1.5"
          >
            <Badge :tone="ar.passed ? 'success' : 'danger'">{{ ar.passed ? "✓" : "✗" }}</Badge>
            <span class="text-foreground">{{ assertLabel(ar.type) }}</span>
            <span class="text-muted">exp={{ ar.expected }} act={{ ar.actual }}</span>
          </div>
        </div>
        <div>
          <div class="mb-1 font-semibold text-foreground">{{ t("pipeline.headers") }}</div>
          <pre class="max-h-32 overflow-auto rounded bg-surface/60 p-2 font-mono text-muted scroll-thin">{{
            JSON.stringify(parseResHeaders(responseStep.responseHeaders), null, 2)
          }}</pre>
        </div>
        <div>
          <div class="mb-1 font-semibold text-foreground">{{ t("pipeline.body") }}</div>
          <pre class="max-h-48 overflow-auto rounded bg-surface/60 p-2 font-mono text-muted scroll-thin">{{
            responseStep.responseBody
          }}</pre>
        </div>
        <div v-if="responseStep.error" class="text-danger">{{ responseStep.error }}</div>
      </div>
    </Dialog>
  </div>
</template>
