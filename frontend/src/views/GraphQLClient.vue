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
import { Play, Hexagon, Clock, BookOpen } from "lucide-vue-next";
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
      // 与其他协议客户端对齐：显式透传超时/重定向/TLS 校验（M32）
      timeoutMs: 30000,
      followRedirect: true,
      sslVerify: true,
    });
    response.value = res;
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t("common.error"));
  } finally {
    loading.value = false;
  }
}

// A1 · GraphQL Schema exploration：通过 introspection 拉取并浏览 schema。
const schema = ref<any>(null);
const schemaLoading = ref(false);
const schemaError = ref("");
const viewTab = ref<"response" | "schema">("response");
const expanded = ref<Record<string, boolean>>({});

// 标准 introspection 查询（含递归 type 引用，供前端解析类型树）
const INTROSPECTION_QUERY = `query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      kind
      name
      description
      fields(includeDeprecated: true) {
        name
        description
        args { name type { name kind ofType { name kind ofType { name kind } } } }
        type { name kind ofType { name kind ofType { name kind } } }
      }
      inputFields { name type { name kind ofType { name kind } } }
      enumValues { name description }
    }
  }
}`;

// 将递归 type 引用渲染为可读字符串，如 [User!]!
function typeRefToString(type: any): string {
  if (!type) return "";
  const inner = type.name || "";
  if (type.ofType) {
    const child = typeRefToString(type.ofType);
    if (type.kind === "LIST") return `[${child}]`;
    if (type.kind === "NON_NULL") return `${child}!`;
    return child;
  }
  return inner;
}

const kindTone: Record<string, string> = {
  OBJECT: "primary",
  INPUT_OBJECT: "info",
  ENUM: "warning",
  SCALAR: "success",
  INTERFACE: "default",
  UNION: "default",
};

// 过滤掉 GraphQL 内部类型（__ 前缀），按名称排序，便于浏览
const visibleTypes = computed(() => {
  if (!schema.value?.types) return [];
  return schema.value.types
    .filter((tp: any) => tp.name && !tp.name.startsWith("__"))
    .sort((a: any, b: any) => a.name.localeCompare(b.name));
});

function toggleType(name: string) {
  expanded.value[name] = !expanded.value[name];
}

// 模板内联箭头不支持类型注解，抽取为函数（A1 schema 字段参数/枚举渲染）
function argsToString(args: any[]): string {
  if (!args || !args.length) return "";
  return args.map((a) => `${a.name}: ${typeRefToString(a.type)}`).join(", ");
}
function enumToString(values: any[]): string {
  if (!values || !values.length) return "";
  return values.map((e) => e.name).join(" | ");
}

async function loadSchema() {
  if (!url.value) return toast.error(t("common.url"));
  schemaLoading.value = true;
  schemaError.value = "";
  try {
    const res = await proxySend({
      method: "POST",
      url: vr.resolve(url.value),
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ query: INTROSPECTION_QUERY }),
      forceHttp2: false,
      timeoutMs: 30000,
      followRedirect: true,
      sslVerify: true,
    });
    if (res.error) {
      schemaError.value = res.error;
      return;
    }
    const json = JSON.parse(res.body || "{}");
    if (json.errors && json.errors.length) {
      schemaError.value = json.errors.map((e: any) => e.message).join("; ");
      return;
    }
    schema.value = json.data?.__schema || null;
    if (!schema.value) schemaError.value = t("graphql.noSchema");
  } catch (e: any) {
    schemaError.value = e?.response?.data?.message || t("common.error");
  } finally {
    schemaLoading.value = false;
  }
}

async function loadSchemaAndShow() {
  await loadSchema();
  viewTab.value = "schema";
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
        <Button :disabled="schemaLoading" variant="ghost" @click="loadSchemaAndShow">
          <BookOpen :size="15" /> {{ t("graphql.loadSchema") }}
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

      <!-- 响应 / Schema 浏览（A1） -->
      <Card class="flex flex-col overflow-hidden">
        <div class="flex items-center gap-2 border-b border-border px-3 py-2">
          <button
            class="rounded px-2 py-0.5 text-xs"
            :class="viewTab === 'response' ? 'bg-surface text-foreground' : 'text-muted hover:text-foreground'"
            @click="viewTab = 'response'"
          >{{ t("graphql.responseTab") }}</button>
          <button
            class="rounded px-2 py-0.5 text-xs"
            :class="viewTab === 'schema' ? 'bg-surface text-foreground' : 'text-muted hover:text-foreground'"
            @click="loadSchemaAndShow"
          >{{ t("graphql.schema") }}</button>
          <template v-if="viewTab === 'response'">
            <Badge v-if="response" :tone="statusTone">{{ response.status }}</Badge>
            <Badge v-if="response" tone="primary">{{ response.proto }}</Badge>
            <span v-if="response?.error" class="text-xs text-danger">{{ response.error }}</span>
          </template>
          <span v-else-if="!schema && !schemaLoading && !schemaError" class="text-xs text-muted">—</span>
        </div>

        <div v-if="viewTab === 'response'" class="flex-1 overflow-y-auto scroll-thin p-3 space-y-3">
          <template v-if="response">
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
          </template>
          <div v-else class="flex flex-1 items-center justify-center text-xs text-muted/60">
            {{ t("graphql.noResponse") }}
          </div>
        </div>

        <div v-else-if="viewTab === 'schema'" class="flex-1 overflow-y-auto scroll-thin p-3">
          <div v-if="schemaLoading" class="text-xs text-muted">{{ t("graphql.schemaLoading") }}</div>
          <div v-else-if="schemaError" class="text-xs text-danger">{{ schemaError }}</div>
          <div v-else-if="!schema" class="text-xs text-muted">{{ t("graphql.noSchema") }}</div>
          <div v-else class="space-y-1">
            <div v-for="tp in visibleTypes" :key="tp.name" class="rounded border border-border/60">
              <button class="flex w-full items-center gap-2 px-2 py-1 text-left hover:bg-surface/60" @click="toggleType(tp.name)">
                <span class="w-3 text-muted">{{ expanded[tp.name] ? '▾' : '▸' }}</span>
                <Badge :tone="(kindTone[tp.kind] || 'default') as any">{{ tp.kind }}</Badge>
                <span class="font-medium text-foreground">{{ tp.name }}</span>
              </button>
              <div v-if="expanded[tp.name]" class="space-y-1 border-t border-border/40 px-3 py-2 text-xs">
                <div v-if="tp.fields && tp.fields.length" class="space-y-0.5">
                  <div v-for="f in tp.fields" :key="f.name" class="flex flex-wrap gap-x-2 gap-y-0.5">
                    <span class="text-primary">{{ f.name }}</span>
                    <span class="text-muted">: {{ typeRefToString(f.type) }}</span>
                    <span v-if="f.args && f.args.length" class="text-muted/70">({{ argsToString(f.args) }})</span>
                  </div>
                </div>
                <div v-else-if="tp.enumValues && tp.enumValues.length" class="text-muted">
                  {{ enumToString(tp.enumValues) }}
                </div>
                <div v-else class="text-muted/60">—</div>
              </div>
            </div>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>
