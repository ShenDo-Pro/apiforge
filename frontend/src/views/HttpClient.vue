<script setup lang="ts">
import { ref, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useCollectionStore } from "@/stores/collection";
import { useToast } from "@/composables/useToast";
import { useRequestSaver } from "@/composables/useRequestSaver";
import { proxySend } from "@/api/proxy";
import { addHistory } from "@/api/request";
import { useEnvironmentStore } from "@/stores/environment";
import { useRequestRuntime, type ExtractRuleResolved } from "@/composables/useRequestRuntime";
import { unresolvedTokens } from "@/lib/vars";
import { resolveAuth, defaultAuth, type AuthConfig } from "@/composables/useAuthHeader";
import type { SavedRequest, RequestHistory, ExtractRule } from "@/types/project";
import type { ProxyResponse } from "@/types/protocol";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";
import Tabs from "@/components/ui/Tabs.vue";
import Badge from "@/components/ui/Badge.vue";
import Dialog from "@/components/ui/Dialog.vue";
import Select from "@/components/ui/Select.vue";
import VarInput from "@/components/ui/VarInput.vue";
import VarTextarea from "@/components/ui/VarTextarea.vue";
import ScriptEditor from "@/components/ui/ScriptEditor.vue";
import TestResultsPanel from "@/components/ui/TestResultsPanel.vue";
import CodeSnippetDialog from "@/components/ui/CodeSnippetDialog.vue";
import CollectionPickerDialog from "@/components/project/CollectionPickerDialog.vue";
import ResponseViewer from "@/components/ui/ResponseViewer.vue";
import CookieManager from "@/components/ui/CookieManager.vue";
import { Play, Save, Clock, Plus, Trash2, Code, Cookie } from "lucide-vue-next";
import type { RespCookie } from "@/types/protocol";

const props = defineProps<{
  projectId: number;
  protocol: string;
  request: SavedRequest | null;
  defaultCollectionId?: number | null;
  defaultName?: string;
}>();
const emit = defineEmits<{ (e: "saved", r: SavedRequest): void }>();
const { t } = useI18n();
const store = useCollectionStore();
const toast = useToast();
const envStore = useEnvironmentStore();

const methods = ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"];
const method = ref("GET");
const url = ref("");
interface HeaderRow {
  key: string;
  value: string;
  enabled: boolean;
}
const headersRows = ref<HeaderRow[]>([]);
const headerMode = ref<"grid" | "bulk">("grid");
const bulkHeaders = ref("");
// Body 多模式：none / raw（含子语言）/ formdata / urlencoded
type BodyMode = "none" | "raw" | "formdata" | "urlencoded";
interface ParamRow { key: string; value: string; description: string; enabled: boolean; }
interface FormItem { key: string; value: string; type: "text" | "file"; fileName: string; enabled: boolean; file?: File; }
interface UrlItem { key: string; value: string; enabled: boolean; }
const bodyMode = ref<BodyMode>("none");
const rawLang = ref<"json" | "text" | "xml" | "javascript" | "html">("json");
const rawBody = ref("");
const formItems = ref<FormItem[]>([]);
const urlencodedItems = ref<UrlItem[]>([]);
const paramsRows = ref<ParamRow[]>([]);

// 文档与设置：按请求 id 存 localStorage，避免新增后端字段
const docs = ref("");
interface ReqSettings { timeoutMs: number; followRedirect: boolean; sslVerify: boolean; http2: boolean; }
const settings = ref<ReqSettings>({ timeoutMs: 0, followRedirect: false, sslVerify: true, http2: false });
const currentReqId = ref<number | null>(null);
const fileTextCache = new Map<FormItem, string>();

function lsKey(prefix: string, id: number | null) {
  return id ? `${prefix}:${id}` : `${prefix}:draft`;
}
function loadReqDocs(id: number | null) {
  docs.value = localStorage.getItem(lsKey("reqdocs", id)) || "";
}
function loadReqSettings(id: number | null) {
  const raw = localStorage.getItem(lsKey("reqsettings", id));
  if (raw) {
    try {
      settings.value = { timeoutMs: 0, followRedirect: false, sslVerify: true, http2: false, ...JSON.parse(raw) };
    } catch { /* 忽略损坏值 */ }
  } else {
    settings.value = { timeoutMs: 0, followRedirect: false, sslVerify: true, http2: false };
  }
}
watch([docs], () => localStorage.setItem(lsKey("reqdocs", currentReqId.value), docs.value));
watch([settings], () => localStorage.setItem(lsKey("reqsettings", currentReqId.value), JSON.stringify(settings.value)), { deep: true });

// Body 序列化 / 反序列化（结构化存进 SavedRequest.Body 列，旧纯文本兼容为 raw）
function restoreBody(rawStr: string) {
  if (!rawStr) { bodyMode.value = "none"; rawBody.value = ""; formItems.value = []; urlencodedItems.value = []; return; }
  try {
    const o = JSON.parse(rawStr);
    if (o && o._v === 1) {
      bodyMode.value = o.mode || "raw";
      rawLang.value = o.rawLang || "json";
      rawBody.value = o.raw || "";
      formItems.value = (o.form || []).map((f: any) => ({ key: f.key || "", value: f.value || "", type: f.type || "text", fileName: f.fileName || "", enabled: f.enabled !== false }));
      urlencodedItems.value = (o.urlencoded || []).map((u: any) => ({ key: u.key || "", value: u.value || "", enabled: u.enabled !== false }));
      return;
    }
  } catch { /* 旧版纯文本 body，当作 raw */ }
  bodyMode.value = "raw";
  rawLang.value = "json";
  rawBody.value = rawStr;
  formItems.value = [];
  urlencodedItems.value = [];
}
function serializeBody(): string {
  return JSON.stringify({
    _v: 1,
    mode: bodyMode.value,
    rawLang: rawLang.value,
    raw: rawBody.value,
    // 文件项不持久化内容，仅保留文件名提示（文件需重选）
    form: formItems.value.map((f) => ({ key: f.key, value: f.value, type: f.type, fileName: f.fileName, enabled: f.enabled })),
    urlencoded: urlencodedItems.value.map((u) => ({ key: u.key, value: u.value, enabled: u.enabled })),
  });
}
// URL 与查询参数双向同步：从 url 拆出 params，或把 params 合并回 url
function parseURLParams(input: string): { url: string; params: ParamRow[] } {
  try {
    const u = new URL(input);
    const params: ParamRow[] = [];
    u.searchParams.forEach((v, k) => params.push({ key: k, value: v, description: "", enabled: true }));
    u.search = "";
    return { url: u.toString(), params };
  } catch {
    return { url: input, params: [] };
  }
}
function mergeParams(input: string, params: ParamRow[]): string {
  let base: string;
  try { base = parseURLParams(input).url; } catch { base = input; }
  try {
    const u = new URL(base);
    for (const p of params) {
      if (p.enabled && p.key.trim()) u.searchParams.set(p.key.trim(), p.value);
    }
    return u.toString();
  } catch {
    const q = params.filter((p) => p.enabled && p.key.trim()).map((p) => `${encodeURIComponent(p.key.trim())}=${encodeURIComponent(p.value)}`).join("&");
    return q ? `${base}?${q}` : base;
  }
}
// 按当前 body 模式构造发送体，并返回应注入的 Content-Type（null 表示不加）
function buildBody(): { body: string; contentType: string | null } {
  if (bodyMode.value === "none") return { body: "", contentType: null };
  if (bodyMode.value === "raw") {
    const map: Record<string, string> = { json: "application/json", text: "text/plain", xml: "application/xml", javascript: "application/javascript", html: "text/html" };
    return { body: rawBody.value, contentType: rawLang.value === "text" ? null : map[rawLang.value] };
  }
  if (bodyMode.value === "urlencoded") {
    const q = urlencodedItems.value.filter((u) => u.enabled && u.key.trim()).map((u) => `${encodeURIComponent(u.key.trim())}=${encodeURIComponent(u.value)}`).join("&");
    return { body: q, contentType: "application/x-www-form-urlencoded" };
  }
  // formdata：文本值直接写；文件项以文本方式读取写入（二进制文件经字符串通道可能损坏，详见说明）
  const boundary = "----apiforgeBoundary" + Date.now().toString(36);
  const parts: string[] = [];
  for (const f of formItems.value) {
    if (!f.enabled || !f.key.trim()) continue;
    if (f.type === "file" && f.file) {
      const text = fileTextCache.get(f) ?? "";
      parts.push(`--${boundary}\r\nContent-Disposition: form-data; name="${f.key.trim()}"; filename="${f.fileName}"\r\nContent-Type: application/octet-stream\r\n\r\n${text}\r\n`);
    } else {
      parts.push(`--${boundary}\r\nContent-Disposition: form-data; name="${f.key.trim()}"\r\n\r\n${f.value}\r\n`);
    }
  }
  return { body: parts.join("") + `--${boundary}--\r\n`, contentType: `multipart/form-data; boundary=${boundary}` };
}
function onFormFile(item: FormItem, e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  item.file = file;
  item.fileName = file.name;
  const reader = new FileReader();
  reader.onload = () => fileTextCache.set(item, String(reader.result));
  reader.readAsText(file);
}

// 预请求 / 测试脚本与 GUI 提取规则
const preScript = ref("");
const testScript = ref("");
const extractRules = ref<ExtractRule[]>([]);

// 鉴权配置（None/Bearer/Basic/API Key/OAuth2）
const auth = ref<AuthConfig>(defaultAuth());

const DEFAULT_HEADERS: HeaderRow[] = [
  { key: "User-Agent", value: "apiforge", enabled: true },
  { key: "Accept", value: "*/*", enabled: true },
];

const response = ref<ProxyResponse | null>(null);
const showCookies = ref(false);
const loading = ref(false);
const reqTab = ref("docs");
const history = ref<ProxyResponse[]>([]);
const savedHistories = ref<RequestHistory[]>([]);

// 请求运行时：变量替换 → 预请求脚本 → 发送 → 提取/测试脚本
const runtime = useRequestRuntime(props.projectId, () => ({
  pre: preScript.value,
  test: testScript.value,
  rules: extractRules.value as ExtractRuleResolved[],
}));

const { pickerOpen, defaultCollectionId, openSave, confirmSave } = useRequestSaver(
  () => props.projectId,
  (r) => emit("saved", r),
);

// 选中的保存请求变化时回填表单（含脚本与提取规则）
watch(
  () => props.request,
  (r) => {
    currentReqId.value = r?.id ?? null;
    if (!r) {
      loadReqDocs(null);
      loadReqSettings(null);
      return;
    }
    method.value = r.method || "GET";
    const { url: baseUrl, params } = parseURLParams(r.url || "");
    url.value = baseUrl;
    paramsRows.value = params.length ? params : [];
    headersRows.value = parseHeadersToRows(r.headers || "{}");
    if (headersRows.value.length === 0) headersRows.value = DEFAULT_HEADERS.map((h) => ({ ...h }));
    restoreBody(r.body || "");
    preScript.value = r.preRequestScript || "";
    testScript.value = r.testScript || "";
    extractRules.value = r.extractRules ? safeParseRules(r.extractRules) : [];
    auth.value = r.auth ? safeParseAuth(r.auth) : defaultAuth();
    loadReqDocs(r.id);
    loadReqSettings(r.id);
    store.fetchHistory(props.projectId, r.id).then((h) => (savedHistories.value = h));
  },
  { immediate: true }
);

if (headersRows.value.length === 0) headersRows.value = DEFAULT_HEADERS.map((h) => ({ ...h }));

function safeParseRules(s: string): ExtractRule[] {
  try {
    const arr = JSON.parse(s);
    return Array.isArray(arr) ? arr : [];
  } catch {
    return [];
  }
}

function safeParseAuth(s: string): AuthConfig {
  try {
    return { ...defaultAuth(), ...JSON.parse(s) };
  } catch {
    return defaultAuth();
  }
}

function enterBulk() {
  bulkHeaders.value = headersRows.value
    .map((h) => `${h.enabled ? "" : "# "}${h.key}: ${h.value}`)
    .join("\n");
  headerMode.value = "bulk";
}
function applyBulk() {
  headersRows.value = parseHeadersToRows(bulkHeaders.value, true);
  headerMode.value = "grid";
}

function parseHeadersToRows(text: string, keepDisabled = false): HeaderRow[] {
  const out: HeaderRow[] = [];
  let obj: Record<string, string> = {};
  try {
    obj = JSON.parse(text);
  } catch {
    for (const line of text.split("\n")) {
      const idx = line.indexOf(":");
      if (idx > 0) {
        const k = line.slice(0, idx).trim();
        const v = line.slice(idx + 1).trim();
        if (k) obj[k] = v;
      }
    }
    return Object.entries(obj).map(([k, v]) => ({ key: k, value: v, enabled: true }));
  }
  return Object.entries(obj).map(([k, v]) => ({ key: k, value: v, enabled: true }));
}

function parseHeaders(rows: HeaderRow[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const h of rows) {
    if (h.enabled && h.key.trim()) out[h.key.trim()] = h.value;
  }
  return out;
}

function addHeader() {
  headersRows.value.push({ key: "", value: "", enabled: true });
}
function removeHeader(i: number) {
  headersRows.value.splice(i, 1);
}

// 提取规则操作
function addRule() {
  extractRules.value.push({ source: "body", expr: "$.token", targetVar: "", scope: "environment", enabled: true });
}
function removeRule(i: number) {
  extractRules.value.splice(i, 1);
}

// 未定义变量高亮：发送前汇总 URL/Headers/Body 中引用但未定义的 token
const unresolved = computed(() => {
  const v = envStore.mergedVars;
  const set = new Set<string>();
  for (const tok of unresolvedTokens(url.value, v)) set.add(tok);
  for (const h of headersRows.value) {
    if (!h.key.trim()) continue;
    for (const tok of unresolvedTokens(`${h.key} ${h.value}`, v)) set.add(tok);
  }
  for (const tok of unresolvedTokens(rawBody.value, v)) set.add(tok);
  return [...set];
});

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
  return Math.max(tm.dns, tm.tls, tm.connect, tm.ttfb, tm.total, 1);
});

async function send() {
  if (!url.value) return toast.error(t("common.url"));
  runtime.resetRuntime();
  loading.value = true;
  try {
    // 0) 鉴权解析（OAuth2 异步取 token）→ 注入请求头 / 查询参数
    const authRes = await resolveAuth(auth.value);
    // 1) 合并 params 表格 + auth query 到 url，构造 body 与 Content-Type
    const finalUrl = mergeParams(
      Object.entries(authRes.query).reduce(
        (u, [k, v]) => {
          try {
            const nu = new URL(u);
            nu.searchParams.set(k, v);
            return nu.toString();
          } catch {
            return u;
          }
        },
        url.value
      ),
      paramsRows.value
    );
    const { body: builtBody, contentType } = buildBody();
    const baseHeaders = { ...parseHeaders(headersRows.value), ...authRes.headers };
    if (contentType && !Object.keys(baseHeaders).some((k) => k.toLowerCase() === "content-type")) {
      baseHeaders["Content-Type"] = contentType;
    }
    // 2) 变量替换 + 预请求脚本（可改写请求、写变量）
    const prepared = await runtime.prepare({
      url: finalUrl,
      method: method.value,
      headers: baseHeaders,
      body: builtBody,
    });
    // 3) 发送（用脚本可能改写后的报文）
    const res = await proxySend({
      method: prepared.method,
      url: prepared.url,
      headers: prepared.headers,
      body: prepared.body,
      forceHttp2: settings.value.http2,
      timeoutMs: settings.value.timeoutMs > 0 ? settings.value.timeoutMs : 0,
      followRedirect: settings.value.followRedirect,
      sslVerify: settings.value.sslVerify,
    });
    response.value = res;
    history.value.unshift(res);
    // 3) 响应提取规则 + 测试脚本（写回变量、收集断言）
    if (!res.error) {
      const resp = {
        code: res.status,
        status: String(res.status),
        responseTime: res.timings?.total ?? 0,
        headers: res.headers,
        json: () => {
          try {
            return JSON.parse(res.body);
          } catch {
            return {};
          }
        },
        text: () => res.body,
      };
      await runtime.finalize(resp);
    }
    if (props.request) {
      await addHistory(props.projectId, props.request.id, {
        method: method.value,
        url: url.value,
        statusCode: res.status,
        proto: res.proto,
        responseHeaders: JSON.stringify(res.headers),
        responseBody: res.body,
        timings: JSON.stringify(res.timings),
      });
    }
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t("common.error"));
  } finally {
    loading.value = false;
  }
}

function buildPayload(name: string) {
  return {
    protocol: props.request?.protocol || props.protocol,
    name,
    method: method.value,
    url: mergeParams(url.value, paramsRows.value),
    headers: JSON.stringify(parseHeaders(headersRows.value)),
    body: serializeBody(),
    preRequestScript: preScript.value,
    testScript: testScript.value,
    extractRules: JSON.stringify(extractRules.value),
    auth: JSON.stringify(auth.value),
  };
}

function onSave() {
  openSave(props.defaultCollectionId ?? null);
}

// 代码片段对话框：基于当前填写的请求（含变量）生成多语言调用代码
const snippetOpen = ref(false);
const currentSnippetReq = computed(() => ({
  method: method.value,
  url: url.value,
  headers: parseHeaders(headersRows.value),
  body: buildBody().body,
}));

// 把未定义变量名包成 {{key}} 用于提示显示（避免模板里出现字面 {{ 干扰编译）
function braces(k: string): string {
  return "{{" + k + "}}";
}

async function onPick(collectionId: number, name: string) {
  const saved = await confirmSave(collectionId, props.request, buildPayload(name));
  if (saved && response.value) {
    await addHistory(props.projectId, saved.id, {
      method: method.value,
      url: url.value,
      statusCode: response.value.status,
      proto: response.value.proto,
      responseHeaders: JSON.stringify(response.value.headers),
      responseBody: response.value.body,
      timings: JSON.stringify(response.value.timings),
    });
  }
}
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- 请求栏 -->
    <div class="border-b border-border p-3">
      <div class="flex items-center gap-2">
        <Select v-model="method" class="w-32">
          <option v-for="m in methods" :key="m" :value="m">{{ m }}</option>
        </Select>
        <VarInput v-model="url" :placeholder="t('http.urlPlaceholder')" class="flex-1" @keyup.enter="send" />
        <Button :disabled="loading" @click="send">
          <Play :size="15" /> {{ t("http.send") }}
        </Button>
        <Button variant="secondary" @click="onSave"><Save :size="15" /> {{ t("common.collectionSaveTo") }}</Button>
        <Button variant="ghost" @click="snippetOpen = true"><Code :size="15" /> {{ t("common.codeSnippet") }}</Button>
      </div>
    </div>

    <!-- 未定义变量高亮提示 -->
    <div
      v-if="unresolved.length"
      class="flex items-center gap-2 border-b border-amber-500/30 bg-amber-500/10 px-3 py-1.5 text-xs text-amber-300"
    >
      <span>{{ t("common.script.unresolvedHint") }}</span>
      <code v-for="u in unresolved" :key="u" class="rounded bg-black/30 px-1.5 py-0.5">{{ braces(u) }}</code>
    </div>

    <!-- 主体：左请求构造 / 右响应 -->
    <div class="grid flex-1 grid-cols-1 gap-3 overflow-hidden p-3 lg:grid-cols-2">
      <!-- 请求构造 -->
      <Card class="flex flex-col overflow-hidden">
        <Tabs
          :tabs="[
            { key: 'docs', label: t('common.docs') },
            { key: 'params', label: t('common.params') },
            { key: 'auth', label: t('common.auth') },
            { key: 'headers', label: t('common.headers') },
            { key: 'body', label: t('common.body') },
            { key: 'scripts', label: t('common.scripts') },
            { key: 'settings', label: t('common.settings') },
          ]"
          v-model="reqTab"
        />
        <div class="flex-1 overflow-y-auto scroll-thin p-3">
          <!-- Docs -->
          <template v-if="reqTab === 'docs'">
            <div class="mb-1 text-xs text-muted">{{ t("common.docsHint") }}</div>
            <VarTextarea v-model="docs" :rows="14" :placeholder="t('common.docsPlaceholder')" />
          </template>

          <!-- Params -->
          <template v-else-if="reqTab === 'params'">
            <div class="grid grid-cols-[auto_1fr_1.4fr_1fr_auto] items-center gap-2 px-1 pb-1 text-xs text-muted">
              <span></span>
              <span>{{ t("http.headerKey") }}</span>
              <span>{{ t("http.headerValue") }}</span>
              <span>{{ t("common.description") }}</span>
              <span></span>
            </div>
            <div class="space-y-1.5">
              <div
                v-for="(p, i) in paramsRows"
                :key="i"
                class="grid grid-cols-[auto_1fr_1.4fr_1fr_auto] items-center gap-2"
              >
                <input type="checkbox" v-model="p.enabled" :title="t('common.enabled')" />
                <VarInput v-model="p.key" :placeholder="t('http.headerKey')" />
                <VarInput v-model="p.value" :placeholder="t('http.headerValue')" />
                <VarInput v-model="p.description" :placeholder="t('common.description')" />
                <button class="rounded p-1 text-muted hover:text-danger" :title="t('common.delete')" @click="paramsRows.splice(i, 1)"><Trash2 :size="14" /></button>
              </div>
              <div v-if="paramsRows.length === 0" class="px-1 py-2 text-xs text-muted/60">—</div>
            </div>
            <button class="mt-2 flex items-center gap-1 text-xs text-primary hover:underline" @click="paramsRows.push({ key: '', value: '', description: '', enabled: true })"><Plus :size="13" /> {{ t("common.addParam") }}</button>
          </template>

          <template v-else-if="reqTab === 'headers'">
            <div class="mb-2 flex items-center justify-between">
              <span class="text-xs text-muted">{{ t("http.headerHint") }}</span>
              <Button
                v-if="headerMode === 'grid'"
                variant="ghost"
                size="sm"
                @click="enterBulk"
              >{{ t("http.bulkEdit") }}</Button>
              <Button v-else variant="ghost" size="sm" @click="applyBulk">{{ t("common.confirm") }}</Button>
            </div>

            <template v-if="headerMode === 'grid'">
              <div class="grid grid-cols-[auto_1fr_1.4fr_auto] items-center gap-2 px-1 text-xs text-muted">
                <span></span>
                <span>{{ t("http.headerKey") }}</span>
                <span>{{ t("http.headerValue") }}</span>
                <span></span>
              </div>
              <div class="space-y-1.5">
                <div
                  v-for="(h, i) in headersRows"
                  :key="i"
                  class="grid grid-cols-[auto_1fr_1.4fr_auto] items-center gap-2"
                >
                  <input type="checkbox" v-model="h.enabled" :title="t('http.headerEnabled')" />
                  <VarInput v-model="h.key" :placeholder="t('http.headerKey')" />
                  <VarInput v-model="h.value" :placeholder="t('http.headerValue')" />
                  <button
                    class="rounded p-1 text-muted hover:text-danger"
                    :title="t('common.delete')"
                    @click="removeHeader(i)"
                  >
                    <Trash2 :size="14" />
                  </button>
                </div>
                <div v-if="headersRows.length === 0" class="px-1 py-2 text-xs text-muted/60">—</div>
              </div>
              <button
                class="mt-2 flex items-center gap-1 text-xs text-primary hover:underline"
                @click="addHeader"
              >
                <Plus :size="13" /> {{ t("http.addHeader") }}
              </button>
            </template>

            <template v-else>
              <VarTextarea v-model="bulkHeaders" :rows="12" placeholder="Authorization: Bearer xxx" />
              <p class="mt-1 text-xs text-muted/60"># 开头的行为禁用</p>
            </template>
          </template>

          <template v-else-if="reqTab === 'body'">
            <div class="mb-2 flex flex-wrap items-center gap-2">
              <Select v-model="bodyMode" class="w-40">
                <option value="none">{{ t("common.bodyNone") }}</option>
                <option value="raw">{{ t("common.bodyRaw") }}</option>
                <option value="formdata">{{ t("common.bodyFormData") }}</option>
                <option value="urlencoded">{{ t("common.bodyUrlEncoded") }}</option>
              </Select>
              <Select v-if="bodyMode === 'raw'" v-model="rawLang" class="w-36">
                <option value="json">JSON</option>
                <option value="text">Text</option>
                <option value="xml">XML</option>
                <option value="javascript">JavaScript</option>
                <option value="html">HTML</option>
              </Select>
            </div>

            <template v-if="bodyMode === 'raw'">
              <VarTextarea v-model="rawBody" :rows="14" :placeholder="t('common.bodyRawPlaceholder')" />
            </template>

            <template v-else-if="bodyMode === 'urlencoded'">
              <div class="space-y-1.5">
                <div
                  v-for="(u, i) in urlencodedItems"
                  :key="i"
                  class="grid grid-cols-[auto_1fr_1.4fr_auto] items-center gap-2"
                >
                  <input type="checkbox" v-model="u.enabled" />
                  <VarInput v-model="u.key" :placeholder="t('http.headerKey')" />
                  <VarInput v-model="u.value" :placeholder="t('http.headerValue')" />
                  <button class="rounded p-1 text-muted hover:text-danger" @click="urlencodedItems.splice(i, 1)"><Trash2 :size="14" /></button>
                </div>
                <div v-if="urlencodedItems.length === 0" class="px-1 py-2 text-xs text-muted/60">—</div>
              </div>
              <button class="mt-2 flex items-center gap-1 text-xs text-primary hover:underline" @click="urlencodedItems.push({ key: '', value: '', enabled: true })"><Plus :size="13" /> {{ t("common.addParam") }}</button>
            </template>

            <template v-else-if="bodyMode === 'formdata'">
              <div class="space-y-1.5">
                <div
                  v-for="(f, i) in formItems"
                  :key="i"
                  class="grid grid-cols-[auto_1fr_1.4fr_auto_auto] items-center gap-2"
                >
                  <input type="checkbox" v-model="f.enabled" />
                  <VarInput v-model="f.key" :placeholder="t('http.headerKey')" />
                  <template v-if="f.type === 'file'">
                    <label class="flex items-center">
                      <input type="file" class="hidden" @change="onFormFile(f, $event)" />
                      <span class="cursor-pointer rounded border border-border px-2 py-1 text-xs text-muted hover:border-primary/60">{{ f.fileName || t("common.bodyChooseFile") }}</span>
                    </label>
                  </template>
                  <template v-else>
                    <VarInput v-model="f.value" :placeholder="t('http.headerValue')" />
                  </template>
                  <Select v-model="f.type" class="w-24">
                    <option value="text">{{ t("common.bodyText") }}</option>
                    <option value="file">{{ t("common.bodyFile") }}</option>
                  </Select>
                  <button class="rounded p-1 text-muted hover:text-danger" @click="formItems.splice(i, 1)"><Trash2 :size="14" /></button>
                </div>
                <div v-if="formItems.length === 0" class="px-1 py-2 text-xs text-muted/60">—</div>
              </div>
              <button class="mt-2 flex items-center gap-1 text-xs text-primary hover:underline" @click="formItems.push({ key: '', value: '', type: 'text', fileName: '', enabled: true })"><Plus :size="13" /> {{ t("common.addParam") }}</button>
              <p class="mt-1 text-xs text-muted/60">{{ t("common.bodyFormDataHint") }}</p>
            </template>

            <div v-if="bodyMode === 'none'" class="py-6 text-center text-xs text-muted/60">{{ t("common.bodyNoneHint") }}</div>
          </template>

          <template v-else-if="reqTab === 'scripts'">
            <div class="mb-1 text-xs font-medium text-muted">{{ t("common.script.preRequest") }}</div>
            <div class="mb-1 text-xs text-muted">{{ t("common.script.preHint") }}</div>
            <ScriptEditor v-model="preScript" :placeholder="t('common.script.prePlaceholder')" :rows="8" />
            <div class="mb-1 mt-3 text-xs font-medium text-muted">{{ t("common.script.test") }}</div>
            <!-- 提取规则 -->
            <div class="mb-3 mt-2">
              <div class="mb-1 flex items-center justify-between text-xs text-muted">
                <span>{{ t("common.script.extractTitle") }}</span>
                <Button variant="ghost" size="sm" @click="addRule"><Plus :size="13" /> {{ t("common.script.addRule") }}</Button>
              </div>
              <div
                v-for="(r, i) in extractRules"
                :key="i"
                class="mb-1.5 flex items-center gap-2"
              >
                <Select v-model="r.source" class="w-24">
                  <option value="body">{{ t("common.script.fromBody") }}</option>
                  <option value="header">{{ t("common.script.fromHeader") }}</option>
                </Select>
                <VarInput v-model="r.expr" :placeholder="r.source === 'header' ? 'Header-Name' : '$.token'" class="flex-1" />
                <Select v-model="r.scope" class="w-28">
                  <option value="environment">{{ t("common.envShort") }}</option>
                  <option value="global">{{ t("common.globalShort") }}</option>
                  <option value="collection">{{ t("common.collectionShort") }}</option>
                  <option value="local">{{ t("common.localShort") }}</option>
                </Select>
                <VarInput v-model="r.targetVar" :placeholder="t('common.script.targetVar')" class="w-32" />
                <input type="checkbox" v-model="r.enabled" :title="t('common.enabled')" />
                <button class="text-muted hover:text-danger" @click="removeRule(i)"><Trash2 :size="13" /></button>
              </div>
              <div v-if="extractRules.length === 0" class="text-xs text-muted/60">—</div>
            </div>
            <div class="mb-1 text-xs text-muted">{{ t("common.script.testHint") }}</div>
            <ScriptEditor v-model="testScript" :placeholder="t('common.script.testPlaceholder')" :rows="8" />
          </template>

          <template v-else-if="reqTab === 'auth'">
            <div class="space-y-2.5">
              <div>
                <label class="mb-1 block text-xs text-muted">{{ t("common.authType") }}</label>
                <Select v-model="auth.type" class="w-56">
                  <option value="none">{{ t("common.authNone") }}</option>
                  <option value="bearer">{{ t("common.authBearer") }}</option>
                  <option value="basic">{{ t("common.authBasic") }}</option>
                  <option value="apikey">{{ t("common.authApiKey") }}</option>
                  <option value="oauth2">{{ t("common.authOAuth2") }}</option>
                </Select>
              </div>

              <template v-if="auth.type === 'bearer'">
                <label class="mb-1 block text-xs text-muted">{{ t("common.authToken") }}</label>
                <VarInput v-model="auth.token" :placeholder="t('common.authTokenPh')" />
              </template>

              <template v-else-if="auth.type === 'basic'">
                <label class="mb-1 block text-xs text-muted">{{ t("common.authUsername") }}</label>
                <VarInput v-model="auth.username" />
                <label class="mb-1 block text-xs text-muted">{{ t("common.authPassword") }}</label>
                <input type="password" v-model="auth.password" class="w-full rounded-lg border border-border bg-surface px-2 py-1.5 text-sm text-foreground outline-none focus:border-primary/60" />
              </template>

              <template v-else-if="auth.type === 'apikey'">
                <label class="mb-1 block text-xs text-muted">{{ t("common.authKeyName") }}</label>
                <VarInput v-model="auth.keyName" />
                <label class="mb-1 block text-xs text-muted">{{ t("common.authKeyValue") }}</label>
                <VarInput v-model="auth.keyValue" />
                <Select v-model="auth.keyIn" class="w-44">
                  <option value="header">{{ t("common.authInHeader") }}</option>
                  <option value="query">{{ t("common.authInQuery") }}</option>
                </Select>
              </template>

              <template v-else-if="auth.type === 'oauth2'">
                <Select v-model="auth.grantType" class="w-60">
                  <option value="client_credentials">{{ t("common.authCC") }}</option>
                  <option value="password">{{ t("common.authPasswordCred") }}</option>
                </Select>
                <label class="mb-1 block text-xs text-muted">{{ t("common.authTokenUrl") }}</label>
                <VarInput v-model="auth.tokenUrl" :placeholder="t('common.authTokenUrlPh')" />
                <label class="mb-1 block text-xs text-muted">{{ t("common.authClientId") }}</label>
                <VarInput v-model="auth.clientId" />
                <label class="mb-1 block text-xs text-muted">{{ t("common.authClientSecret") }}</label>
                <input type="password" v-model="auth.clientSecret" class="w-full rounded-lg border border-border bg-surface px-2 py-1.5 text-sm text-foreground outline-none focus:border-primary/60" />
                <label class="mb-1 block text-xs text-muted">{{ t("common.authScope") }}</label>
                <VarInput v-model="auth.scope" />
                <template v-if="auth.grantType === 'password'">
                  <label class="mb-1 block text-xs text-muted">{{ t("common.authUsername") }}</label>
                  <VarInput v-model="auth.username2" />
                  <label class="mb-1 block text-xs text-muted">{{ t("common.authPassword") }}</label>
                  <input type="password" v-model="auth.password2" class="w-full rounded-lg border border-border bg-surface px-2 py-1.5 text-sm text-foreground outline-none focus:border-primary/60" />
                </template>
                <label class="mb-1 block text-xs text-muted">{{ t("common.authVarName") }}</label>
                <VarInput v-model="auth.varName" :placeholder="t('common.authVarNamePh')" />
                <p class="rounded-lg bg-surface/60 px-2 py-1.5 text-xs text-muted/70">{{ t("common.authOAuthHint") }}</p>
              </template>
            </div>
          </template>

          <!-- Settings -->
          <template v-else-if="reqTab === 'settings'">
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <label class="text-xs text-muted">{{ t("common.settingTimeout") }}</label>
                <div class="flex items-center gap-1">
                  <input
                    type="number"
                    min="0"
                    v-model.number="settings.timeoutMs"
                    class="w-24 rounded-lg border border-border bg-surface px-2 py-1 text-sm text-foreground outline-none focus:border-primary/60"
                  />
                  <span class="text-xs text-muted">ms</span>
                </div>
              </div>
              <label class="flex items-center justify-between text-xs text-muted">
                <span>{{ t("common.settingFollowRedirect") }}</span>
                <input type="checkbox" v-model="settings.followRedirect" />
              </label>
              <label class="flex items-center justify-between text-xs text-muted">
                <span>{{ t("common.settingSslVerify") }}</span>
                <input type="checkbox" v-model="settings.sslVerify" />
              </label>
              <label class="flex items-center justify-between text-xs text-muted">
                <span>{{ t("common.settingHttp2") }}</span>
                <input type="checkbox" v-model="settings.http2" />
              </label>
              <p class="rounded-lg bg-surface/60 px-2 py-1.5 text-xs text-muted/70">{{ t("common.settingHint") }}</p>
            </div>
          </template>
        </div>
      </Card>

      <!-- 响应 -->
      <Card class="flex flex-col overflow-hidden">
        <div class="flex items-center gap-3 border-b border-border px-3 py-2">
          <Badge v-if="response" :tone="statusTone">{{ response.status }}</Badge>
          <Badge v-if="response" tone="primary">{{ response.proto }}</Badge>
          <button
            v-if="response?.cookies?.length"
            class="flex items-center gap-1 rounded-md px-2 py-0.5 text-xs text-muted transition-colors hover:bg-border/30 hover:text-foreground"
            @click="showCookies = !showCookies"
          >
            <Cookie :size="13" /> Cookies ({{ response.cookies.length }})
          </button>
          <span v-if="response?.error" class="text-xs text-danger">{{ response.error }}</span>
          <span v-if="!response" class="text-xs text-muted">—</span>
        </div>

        <div v-if="response" class="flex-1 overflow-y-auto scroll-thin p-3 space-y-3">
          <!-- 计时 -->
          <div>
            <div class="mb-1 flex items-center gap-1 text-xs text-muted"><Clock :size="13" />{{ t("http.timingsTitle") }}</div>
            <div class="space-y-1">
              <div v-for="k in (['dns','tls','connect','ttfb','total'] as const)" :key="k" class="flex items-center gap-2 text-xs">
                <span class="w-14 text-muted">{{ t('http.' + k) }}</span>
                <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-border/40">
                  <div class="h-full rounded-full bg-gradient-to-r from-primary to-primary-3" :style="{ width: (response.timings[k] / maxTiming) * 100 + '%' }" />
                </div>
                <span class="w-14 text-right text-foreground">{{ response.timings[k] }}ms</span>
              </div>
            </div>
          </div>

          <!-- 脚本结果 / 提取结果 / 断言 -->
          <TestResultsPanel
            :assertions="runtime.assertions.value"
            :extracted="runtime.extracted.value"
            :logs="runtime.scriptLogs.value"
            :error="runtime.scriptError.value"
          />

          <div>
            <div class="mb-1 text-xs font-medium text-muted">{{ t("http.responseHeaders") }}</div>
            <pre class="rounded-lg bg-surface p-2 text-xs text-foreground">{{ JSON.stringify(response.headers, null, 2) }}</pre>
          </div>
          <div>
            <div class="mb-1 text-xs font-medium text-muted">{{ t("http.responseBody") }}</div>
            <div class="h-72 overflow-hidden rounded-lg border border-border">
              <ResponseViewer :body="response.body" :content-type="response.headers['Content-Type']" />
            </div>
          </div>
          <div v-if="showCookies && response?.cookies?.length" class="rounded-lg border border-border p-2">
            <CookieManager :cookies="(response.cookies as RespCookie[])" />
          </div>
        </div>

        <!-- 历史 -->
        <div class="border-t border-border p-2">
          <div class="mb-1 text-xs font-medium text-muted">{{ t("http.historyTitle") }}</div>
          <div v-if="history.length === 0" class="text-xs text-muted/60">{{ t("http.noHistory") }}</div>
          <div class="flex flex-col gap-1">
            <button
              v-for="(h, i) in history"
              :key="i"
              class="flex items-center gap-2 rounded-md px-2 py-1 text-left text-xs hover:bg-border/30"
              @click="response = h"
            >
              <Badge :tone="h.status >= 200 && h.status < 300 ? 'success' : h.status >= 500 ? 'danger' : 'warning'">{{ h.status }}</Badge>
              <span class="text-muted">{{ h.proto }}</span>
              <span class="truncate text-foreground">{{ h.body.slice(0, 40) }}</span>
            </button>
          </div>
        </div>
      </Card>
    </div>

    <CollectionPickerDialog
      :open="pickerOpen"
      :project-id="projectId"
      :model-value="defaultCollectionId"
      :default-name="(defaultName || props.request?.name || method + ' ' + (url || '')).trim()"
      @close="pickerOpen = false"
      @confirm="onPick"
    />

    <CodeSnippetDialog :open="snippetOpen" :req="currentSnippetReq" @close="snippetOpen = false" />
  </div>
</template>
