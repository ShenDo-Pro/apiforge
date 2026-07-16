<script setup lang="ts">
import { ref, onUnmounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useToast } from "@/composables/useToast";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import Textarea from "@/components/ui/Textarea.vue";
import { Plug, PlugZap, Send, Server } from "lucide-vue-next";
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
    protocol: "ws",
    name,
    method: "",
    url: wsUrl.value,
    headers: "{}",
    body: JSON.stringify({
      subprotocol: subprotocol.value,
      message: message.value,
      headers: headersText.value,
    }),
  };
}
function loadRequest(r: SavedRequest) {
  wsUrl.value = r.url || "";
  try {
    const o = JSON.parse(r.body || "{}");
    subprotocol.value = o.subprotocol || "";
    message.value = o.message || "";
    headersText.value = o.headers || "";
  } catch {}
}
watch(
  () => props.request,
  (r) => {
    if (r) loadRequest(r);
  },
  { immediate: true },
);

const wsUrl = ref("wss://echo.websocket.org");
const subprotocol = ref("");
const headersText = ref("");
const message = ref("");
const connected = ref(false);
const connecting = ref(false);
type WsFrame = {
  dir: "sent" | "recv";
  opcode: "text" | "binary";
  text: string;
  size: number;
  time: string;
};
const frames = ref<WsFrame[]>([]);
// 长会话内存上限：仅保留最近 N 条帧，避免无限增长（L14）
const MAX_FRAMES = 2000;

function utf8Len(s: string): number {
  return new TextEncoder().encode(s).length;
}
function pushFrame(dir: "sent" | "recv", opcode: "text" | "binary", text: string, size: number) {
  frames.value.push({ dir, opcode, text, size, time: now() });
  if (frames.value.length > MAX_FRAMES) frames.value.splice(0, frames.value.length - MAX_FRAMES);
}

let ws: WebSocket | null = null;

function now() {
  return new Date().toLocaleTimeString();
}

// 经后端 /ws/relay 中继：后端作为 WS 客户端连往目标，使前端能设置浏览器不允许的自定义请求头，
// 并可访问内网/需服务端代理的目标。自定义头以 JSON 经 query 传递。
function relayWsUrl(resolvedUrl: string): string {
  const u = new URL(location.origin);
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  u.pathname = "/ws/relay";
  u.searchParams.set("proto", "ws");
  u.searchParams.set("url", resolvedUrl);
  // 携带鉴权 token，与 gRPC/Socket/TCP/MQTT 客户端保持一致，避免后端中继拒绝连接（M28）
  const token = localStorage.getItem("access_token");
  if (token) u.searchParams.set("token", token);
  if (subprotocol.value) u.searchParams.set("sub", vr.resolve(subprotocol.value));
  if (headersText.value.trim()) {
    try {
      u.searchParams.set("headers", vr.resolve(headersText.value));
    } catch {
      toast.error(t("common.error"));
      return "";
    }
  }
  return u.toString();
}

function connect() {
  if (!wsUrl.value) return toast.error(t("common.url"));
  const resolvedUrl = vr.resolve(wsUrl.value);
  const relay = relayWsUrl(resolvedUrl);
  if (!relay) return;
  connecting.value = true;
  try {
    ws = new WebSocket(relay);
  } catch {
    connecting.value = false;
    return toast.error(t("common.error"));
  }
  ws.onmessage = (ev) => {
    const data = ev.data;
    // 后端中继的控制帧（JSON 文本）：connected / error / closed
    if (typeof data === "string") {
      try {
        const j = JSON.parse(data);
        if (j && j.type) {
          if (j.type === "connected") {
            connected.value = true;
            connecting.value = false;
            toast.success(t("ws.connected"));
            return;
          }
          if (j.type === "error") {
            connected.value = false;
            connecting.value = false;
            toast.error(j.message || t("common.error"));
            return;
          }
          if (j.type === "closed") {
            connected.value = false;
            return;
          }
        }
      } catch {}
    }
    if (typeof data === "string") {
      pushFrame("recv", "text", data, utf8Len(data));
    } else {
      const blob = data as Blob;
      pushFrame("recv", "binary", `[${t("ws.binary")} ${blob.size}B]`, blob.size);
    }
  };
  ws.onclose = () => {
    connected.value = false;
    connecting.value = false;
  };
  ws.onerror = () => toast.error(t("common.error"));
}

function disconnect() {
  ws?.close();
  ws = null;
}

function send() {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  const resolvedMsg = vr.resolve(message.value);
  ws.send(resolvedMsg);
  pushFrame("sent", "text", resolvedMsg, utf8Len(resolvedMsg));
  message.value = "";
}

onUnmounted(() => ws?.close());
</script>

<template>
  <div class="flex h-full flex-col gap-3 p-3">
    <Card class="p-3">
      <div class="flex flex-wrap items-center gap-2">
        <Input v-model="wsUrl" :placeholder="t('ws.urlPlaceholder')" class="min-w-[240px] flex-1" />
        <Input v-model="subprotocol" :placeholder="t('ws.subprotocol')" class="w-40" />
        <Button v-if="!connected" :disabled="connecting" @click="connect">
          <PlugZap :size="15" /> {{ t("ws.connect") }}
        </Button>
        <Button v-else variant="danger" @click="disconnect">
          <Plug :size="15" /> {{ t("ws.disconnect") }}
        </Button>
        <RequestSaver
          :project-id="projectId"
          :request="request"
          :default-collection-id="defaultCollectionId"
          :default-name="defaultName || wsUrl"
          :build-payload="buildPayload"
          @saved="(r) => emit('saved', r)"
        />
      </div>
      <div class="mt-2">
        <div class="mb-1 flex items-center gap-1.5 text-xs text-muted">
          <Server :size="13" /> 自定义请求头（JSON，经后端中继发送，浏览器直连不支持）
        </div>
        <Textarea v-model="headersText" :rows="2" placeholder='{"Authorization":"Bearer ..."}' />
      </div>
    </Card>

    <div class="grid flex-1 grid-cols-1 gap-3 overflow-hidden lg:grid-cols-2">
      <Card class="flex flex-col p-3">
        <div class="mb-2 text-sm font-medium text-foreground">{{ t("ws.message") }}</div>
        <Textarea v-model="message" class="flex-1" :rows="8" :placeholder="t('ws.message')" />
        <Button class="mt-2 self-end" :disabled="!connected" @click="send">
          <Send :size="15" /> {{ t("ws.send") }}
        </Button>
      </Card>

      <Card class="flex flex-col overflow-hidden p-3">
        <div class="mb-2 text-sm font-medium text-foreground">{{ t("ws.frames") }}</div>
        <div class="flex-1 space-y-1 overflow-y-auto scroll-thin">
          <div
            v-for="(f, i) in frames"
            :key="i"
            class="flex animate-slide-in items-start gap-2 rounded-md px-2 py-1 text-xs"
            :class="f.dir === 'sent' ? 'bg-primary/10' : 'bg-surface'"
          >
            <span class="shrink-0 text-muted">{{ f.time }}</span>
            <span :class="f.dir === 'sent' ? 'text-primary' : 'text-success'">
              {{ f.dir === "sent" ? "↑" : "↓" }}
            </span>
            <span
              class="shrink-0 rounded px-1 text-[10px] font-semibold"
              :class="f.opcode === 'binary' ? 'bg-amber-500/20 text-amber-400' : 'bg-sky-500/20 text-sky-400'"
            >{{ f.opcode === "binary" ? "BIN 0x2" : "TXT 0x1" }}</span>
            <span class="shrink-0 text-muted/70">{{ f.size }}B</span>
            <span class="break-all whitespace-pre-wrap text-foreground">{{ f.text }}</span>
          </div>
          <div v-if="frames.length === 0" class="text-xs text-muted/60">{{ t("ws.noFrames") }}</div>
        </div>
      </Card>
    </div>
  </div>
</template>
