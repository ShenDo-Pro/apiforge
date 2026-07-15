<script setup lang="ts">
import { ref, onUnmounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useToast } from "@/composables/useToast";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import Textarea from "@/components/ui/Textarea.vue";
import { Plug, PlugZap, RefreshCw, Send, Server } from "lucide-vue-next";
import type { SavedRequest } from "@/types/project";
import RequestSaver from "@/components/project/RequestSaver.vue";
import { type RequestPayload } from "@/composables/useRequestSaver";
import { useVarResolver } from "@/composables/useVarResolver";

const { t } = useI18n();
const toast = useToast();
const vr = useVarResolver();

const props = defineProps<{
  projectId: number;
  request: SavedRequest | null;
  defaultCollectionId?: number | null;
  defaultName?: string;
}>();
const emit = defineEmits<{ (e: "saved", r: SavedRequest): void }>();

function buildPayload(name: string): RequestPayload {
  return {
    protocol: "grpc",
    name,
    method: "",
    url: target.value,
    headers: "{}",
    body: requestBody.value,
  };
}
function loadRequest(r: SavedRequest) {
  target.value = r.url || "";
  if (r.body) requestBody.value = r.body;
}
watch(
  () => props.request,
  (r) => {
    if (r) loadRequest(r);
  },
  { immediate: true },
);

const target = ref("localhost:50051");
const connected = ref(false);
const connecting = ref(false);
const services = ref<{ name: string; methods: { name: string; input: string; output: string }[] }[]>([]);
const selected = ref<{ service: string; method: string } | null>(null);
const requestBody = ref("{}");
const response = ref<any>(null);
const errorText = ref("");
const invoking = ref(false);

let ws: WebSocket | null = null;

function buildUrl(): string {
  const token = localStorage.getItem("access_token") || "";
  const scheme = location.protocol === "https:" ? "wss:" : "ws:";
  const params = new URLSearchParams({ token });
  return `${scheme}//${location.host}/ws/grpc?${params.toString()}`;
}

function sendMsg(obj: Record<string, unknown>) {
  if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(obj));
}

function connect() {
  if (!target.value) return toast.error(t("grpc.target"));
  connecting.value = true;
  try {
    ws = new WebSocket(buildUrl());
  } catch {
    connecting.value = false;
    return toast.error(t("common.error"));
  }
  ws.onopen = () => {
    sendMsg({ type: "connect", target: vr.resolve(target.value) });
  };
  ws.onmessage = (ev) => {
    let m: any;
    try {
      m = JSON.parse(ev.data);
    } catch {
      return;
    }
    if (m.type === "connected") {
      connected.value = true;
      connecting.value = false;
      toast.success(t("grpc.connected"));
      sendMsg({ type: "list" });
    } else if (m.type === "error") {
      toast.error(t("grpc.connectFailed") + ": " + m.message);
      connecting.value = false;
    } else if (m.type === "list") {
      services.value = m.services || [];
      if (!services.value.length) toast.error(t("grpc.noServices"));
    } else if (m.type === "result") {
      invoking.value = false;
      errorText.value = "";
      response.value = m.data;
    }
  };
  ws.onclose = () => {
    ws = null;
    connected.value = false;
    connecting.value = false;
  };
  ws.onerror = () => toast.error(t("common.error"));
}

function disconnect() {
  ws?.close();
  ws = null;
  connected.value = false;
  services.value = [];
  selected.value = null;
  response.value = null;
}

function refreshList() {
  if (!connected.value) return toast.error(t("grpc.notConnected"));
  sendMsg({ type: "list" });
}

function selectMethod(service: string, method: string) {
  selected.value = { service, method };
  requestBody.value = "{}";
  response.value = null;
  errorText.value = "";
}

function invoke() {
  if (!selected.value) return toast.error(t("grpc.methods"));
  let data: unknown = {};
  try {
    data = JSON.parse(vr.resolve(requestBody.value));
  } catch {
    return toast.error(t("grpc.invalidJson"));
  }
  invoking.value = true;
  errorText.value = "";
  response.value = null;
  sendMsg({ type: "invoke", service: selected.value.service, method: selected.value.method, data });
}

onUnmounted(() => ws?.close());
</script>

<template>
  <div class="flex h-full flex-col gap-3 overflow-hidden p-3">
    <Card class="p-3">
      <div class="flex flex-wrap items-center gap-2">
        <Input v-model="target" :placeholder="t('grpc.targetPlaceholder')" class="min-w-[200px] flex-1" />
        <Button v-if="!connected" :disabled="connecting" @click="connect">
          <PlugZap :size="15" /> {{ t("grpc.connect") }}
        </Button>
        <Button v-else variant="danger" @click="disconnect">
          <Plug :size="15" /> {{ t("grpc.disconnect") }}
        </Button>
        <RequestSaver
          :project-id="projectId"
          :request="request"
          :default-collection-id="defaultCollectionId"
          :default-name="defaultName || target"
          :build-payload="buildPayload"
          @saved="(r) => emit('saved', r)"
        />
        <Button variant="secondary" :disabled="!connected" @click="refreshList">
          <RefreshCw :size="15" /> {{ t("grpc.list") }}
        </Button>
      </div>
    </Card>

    <div class="grid flex-1 grid-cols-1 gap-3 overflow-hidden lg:grid-cols-[260px_1fr]">
      <!-- 服务/方法树 -->
      <Card class="flex flex-col overflow-hidden p-3">
        <div class="mb-2 flex items-center gap-1 text-sm font-medium text-foreground">
          <Server :size="14" /> {{ t("grpc.services") }}
        </div>
        <div class="flex-1 space-y-1 overflow-y-auto scroll-thin">
          <div v-for="svc in services" :key="svc.name" class="rounded-md bg-surface/50 p-1">
            <div class="px-1 text-xs font-semibold text-primary">{{ svc.name }}</div>
            <button
              v-for="m in svc.methods"
              :key="m.name"
              class="block w-full truncate rounded px-2 py-0.5 text-left text-xs text-foreground hover:bg-border/30"
              :class="selected && selected.service === svc.name && selected.method === m.name ? 'bg-primary/15 text-primary' : ''"
              :title="m.name"
              @click="selectMethod(svc.name, m.name)"
            >
              {{ m.name }}
            </button>
          </div>
          <div v-if="services.length === 0" class="text-xs text-muted/60">{{ t("grpc.noServices") }}</div>
        </div>
      </Card>

      <!-- 调用 + 响应 -->
      <div class="grid grid-rows-2 gap-3 overflow-hidden">
        <Card class="flex flex-col overflow-hidden p-3">
          <div class="mb-2 flex items-center justify-between">
            <span class="text-sm font-medium text-foreground">
              {{ selected ? selected.service + " / " + selected.method : t("grpc.requestBody") }}
            </span>
            <Button size="sm" :disabled="!selected || invoking" @click="invoke">
              <Send :size="14" /> {{ invoking ? t("grpc.invoking") : t("grpc.invoke") }}
            </Button>
          </div>
          <Textarea v-model="requestBody" :rows="6" :placeholder="t('grpc.requestPlaceholder')" class="flex-1 font-mono text-xs" />
        </Card>

        <Card class="flex flex-col overflow-hidden p-3">
          <div class="mb-2 text-sm font-medium text-foreground">{{ t("grpc.response") }}</div>
          <div class="flex-1 overflow-y-auto scroll-thin">
            <pre v-if="errorText" class="whitespace-pre-wrap rounded-lg bg-danger/10 p-2 text-xs text-danger">{{ errorText }}</pre>
            <pre v-else-if="response !== null" class="whitespace-pre-wrap rounded-lg bg-surface p-2 text-xs text-foreground">{{ JSON.stringify(response, null, 2) }}</pre>
            <div v-else class="text-xs text-muted/60">{{ t("grpc.noResponse") }}</div>
          </div>
        </Card>
      </div>
    </div>
  </div>
</template>
