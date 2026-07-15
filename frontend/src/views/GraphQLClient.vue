<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useToast } from "@/composables/useToast";
import { proxySend } from "@/api/proxy";
import type { ProxyResponse } from "@/types/protocol";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import Textarea from "@/components/ui/Textarea.vue";
import Tabs from "@/components/ui/Tabs.vue";
import Badge from "@/components/ui/Badge.vue";
import { Play, Hexagon, Clock } from "lucide-vue-next";
import type { SavedRequest } from "@/types/project";
import RequestSaver from "@/components/project/RequestSaver.vue";
import { type RequestPayload } from "@/composables/useRequestSaver";
import { useVarResolver } from "@/composables/useVarResolver";

const props = defineProps<{
  projectId: number;
  request: SavedRequest | null;
  defaultCollectionId?: number | null;
  defaultName?: string;
}>();
const emit = defineEmits<{ (e: "saved", r: SavedRequest): void }>();
const { t } = useI18n();
const toast = useToast();
const vr = useVarResolver();

function buildPayload(name: string): RequestPayload {
  return {
    protocol: "graphql",
    name,
    method: "",
    url: url.value,
    headers: "{}",
    body: JSON.stringify({
      query: query.value,
      variablesText: variablesText.value,
      headersText: headersText.value,
    }),
  };
}
function loadRequest(r: SavedRequest) {
  url.value = r.url || "";
  try {
    const o = JSON.parse(r.body || "{}");
    if (o.query) query.value = o.query;
    if (o.variablesText !== undefined) variablesText.value = o.variablesText;
    if (o.headersText !== undefined) headersText.value = o.headersText;
  } catch {}
}
watch(
  () => props.request,
  (r) => {
    if (r) loadRequest(r);
  },
  { immediate: true },
);

const url = ref("");
const query = ref("query {\n  __typename\n}");
const variablesText = ref("");
const headersText = ref("");
const reqTab = ref("query");
const response = ref<ProxyResponse | null>(null);
const loading = ref(false);

const timingKeys = ["dns", "tls", "connect", "ttfb", "total"] as const;

function parseHeaders(text: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const line of text.split("\n")) {
    const idx = line.indexOf(":");
    if (idx > 0) {
      const k = line.slice(0, idx).trim();
      const v = line.slice(idx + 1).trim();
      if (k) out[k] = v;
    }
  }
  return out;
}

const statusTone = computed(() => {
  const s = response.value?.status ?? 0;
  if (s >= 200 && s < 300) return "success";
  if (s >= 400 && s < 500) return "warning";
  if (s >= 500) return "danger";
  return "info";
});

const maxTiming = computed(() => {
  const tm = response.value?.timings;
  if (!tm) return 1;
  return Math.max(...timingKeys.map((k) => tm[k]), 1);
});

async function send() {
  if (!url.value) return toast.error(t("common.url"));
  if (!query.value.trim()) return toast.error(t("graphql.queryRequired"));

  let variables: Record<string, unknown> | undefined;
  if (variablesText.value.trim()) {
    try {
      variables = JSON.parse(vr.resolve(variablesText.value));
    } catch {
      return toast.error(t("graphql.variablesInvalid"));
    }
  }

  const headers = parseHeaders(vr.resolve(headersText.value));
  if (!Object.keys(headers).some((k) => k.toLowerCase() === "content-type")) {
    headers["Content-Type"] = "application/json";
  }

  const payload = {
    query: vr.resolve(query.value),
    ...(variables ? { variables } : {}),
  };

  loading.value = true;
  try {
    const res = await proxySend({
      method: "POST",
      url: vr.resolve(url.value),
      headers,
      body: JSON.stringify(payload),
      forceHttp2: false,
    });
    response.value = res;
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t("common.error"));
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="border-b border-border p-3">
      <div class="flex items-center gap-2">
        <Hexagon :size="16" class="text-primary" />
        <Input v-model="url" :placeholder="t('graphql.urlPlaceholder')" class="flex-1" @keyup.enter="send" />
        <Button :disabled="loading" @click="send">
          <Play :size="15" /> {{ t("graphql.send") }}
        </Button>
        <RequestSaver
          :project-id="projectId"
          :request="request"
          :default-collection-id="defaultCollectionId"
          :default-name="defaultName || url"
          :build-payload="buildPayload"
          @saved="(r) => emit('saved', r)"
        />
      </div>
    </div>

    <div class="grid flex-1 grid-cols-1 gap-3 overflow-hidden p-3 lg:grid-cols-2">
      <!-- 请求构造 -->
      <Card class="flex flex-col overflow-hidden">
        <Tabs
          :tabs="[
            { key: 'query', label: t('graphql.query') },
            { key: 'variables', label: t('graphql.variables') },
            { key: 'headers', label: t('common.headers') },
          ]"
          v-model="reqTab"
        />
        <div class="flex-1 overflow-y-auto scroll-thin p-3">
          <template v-if="reqTab === 'query'">
            <label class="mb-1 block text-xs text-muted">{{ t("graphql.queryHint") }}</label>
            <Textarea v-model="query" :rows="16" :placeholder="t('graphql.queryPlaceholder')" class="font-mono" />
          </template>
          <template v-else-if="reqTab === 'variables'">
            <label class="mb-1 block text-xs text-muted">{{ t("graphql.variablesHint") }}</label>
            <Textarea v-model="variablesText" :rows="16" :placeholder="t('graphql.variablesPlaceholder')" class="font-mono" />
          </template>
          <template v-else>
            <label class="mb-1 block text-xs text-muted">{{ t("graphql.headersHint") }}</label>
            <Textarea v-model="headersText" :rows="16" placeholder="Authorization: Bearer xxx" />
          </template>
        </div>
      </Card>

      <!-- 响应 -->
      <Card class="flex flex-col overflow-hidden">
        <div class="flex items-center gap-3 border-b border-border px-3 py-2">
          <Badge v-if="response" :tone="statusTone">{{ response.status }}</Badge>
          <Badge v-if="response" tone="primary">{{ response.proto }}</Badge>
          <span v-if="response?.error" class="text-xs text-danger">{{ response.error }}</span>
          <span v-if="!response" class="text-xs text-muted">—</span>
        </div>

        <div v-if="response" class="flex-1 overflow-y-auto scroll-thin p-3 space-y-3">
          <div>
            <div class="mb-1 flex items-center gap-1 text-xs text-muted">
              <Clock :size="13" />{{ t("graphql.timingsTitle") }}
            </div>
            <div class="space-y-1">
              <div v-for="k in timingKeys" :key="k" class="flex items-center gap-2 text-xs">
                <span class="w-14 text-muted">{{ t("graphql." + k) }}</span>
                <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-border/40">
                  <div
                    class="h-full rounded-full bg-gradient-to-r from-primary to-primary-3"
                    :style="{ width: (response.timings[k] / maxTiming) * 100 + '%' }"
                  />
                </div>
                <span class="w-14 text-right text-foreground">{{ response.timings[k] }}ms</span>
              </div>
            </div>
          </div>

          <div>
            <div class="mb-1 text-xs font-medium text-muted">{{ t("graphql.responseHeaders") }}</div>
            <pre class="rounded-lg bg-surface p-2 text-xs text-foreground">{{ JSON.stringify(response.headers, null, 2) }}</pre>
          </div>
          <div>
            <div class="mb-1 text-xs font-medium text-muted">{{ t("graphql.responseBody") }}</div>
            <pre class="max-h-72 overflow-auto scroll-thin rounded-lg bg-surface p-2 text-xs text-foreground">{{ response.body }}</pre>
          </div>
        </div>
        <div v-else class="flex flex-1 items-center justify-center text-xs text-muted/60">
          {{ t("graphql.noResponse") }}
        </div>
      </Card>
    </div>
  </div>
</template>
