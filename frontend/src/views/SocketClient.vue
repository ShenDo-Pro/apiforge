<script setup lang="ts">
import { ref, computed, onUnmounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useToast } from "@/composables/useToast";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import Textarea from "@/components/ui/Textarea.vue";
import { Plug, PlugZap, Send, Trash2, Network, RefreshCw } from "lucide-vue-next";
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
    protocol: "socket",
    name,
    method: "",
    url: `${host.value}:${port.value}`,
    headers: "{}",
    body: JSON.stringify({
      mode: mode.value,
      sendMode: sendMode.value,
      viewMode: viewMode.value,
      sendInput: sendInput.value,
    }),
  };
}
function loadRequest(r: SavedRequest) {
  try {
    const [h, p] = (r.url || "").split(":");
    if (h) host.value = h;
    if (p) port.value = p;
    const o = JSON.parse(r.body || "{}");
    if (o.mode) mode.value = o.mode;
    if (o.sendMode) sendMode.value = o.sendMode;
    if (o.viewMode) viewMode.value = o.viewMode;
    if (o.sendInput) sendInput.value = o.sendInput;
  } catch {}
}
watch(
  () => props.request,
  (r) => {
    if (r) loadRequest(r);
  },
  { immediate: true },
);

const mode = ref<"tcp" | "udp">("tcp");
const host = ref("127.0.0.1");
const port = ref("9000");
const sendInput = ref("");
const sendMode = ref<"hex" | "ascii">("hex");
const viewMode = ref<"hex" | "ascii">("hex");
const connected = ref(false);
const connecting = ref(false);
const sentBytes = ref(0);
const recvBytes = ref(0);

// 接收缓冲：累积远端返回的所有字节，统一渲染为 HEX 转储（16 字节/行）
const MAX_BUF = 256 * 1024;
const recvBuf = ref<number[]>([]);
const logs = ref<{ time: string; level: "info" | "error" | "success"; text: string }[]>([]);
let reconnectTimer: number | null = null;
let manualClose = false;
let ws: WebSocket | null = null;

const hexDump = computed(() => {
  const bytes = recvBuf.value;
  if (bytes.length === 0) return "";
  const lines: string[] = [];
  for (let i = 0; i < bytes.length; i += 16) {
    const chunk = bytes.slice(i, i + 16);
    const hex = chunk.map((b) => b.toString(16).padStart(2, "0")).join(" ");
    const ascii = chunk
      .map((b) => (b >= 32 && b < 127 ? String.fromCharCode(b) : "."))
      .join("");
    const offset = i.toString(16).padStart(8, "0");
    lines.push(`${offset}  ${hex.padEnd(47, " ")}  ${ascii}`);
  }
  return lines.join("\n");
});

const asciiView = computed(() => {
  if (recvBuf.value.length === 0) return "";
  const u8 = Uint8Array.from(recvBuf.value);
  try {
    return new TextDecoder("utf-8", { fatal: false }).decode(u8);
  } catch {
    return u8.map((b) => String.fromCharCode(b)).join("");
  }
});

function now() {
  return new Date().toLocaleTimeString();
}

function log(level: "info" | "error" | "success", text: string) {
  logs.value.push({ time: now(), level, text });
}

function hexToBytes(hex: string): Uint8Array {
  const clean = hex.replace(/[\s,]/g, "");
  if (clean.length === 0) return new Uint8Array(0);
  if (clean.length % 2 !== 0 || !/^[0-9a-fA-F]+$/.test(clean)) {
    throw new Error("invalid hex");
  }
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(clean.substr(i * 2, 2), 16);
  }
  return out;
}

function buildUrl(): string {
  const token = localStorage.getItem("access_token") || "";
  const scheme = location.protocol === "https:" ? "wss:" : "ws:";
  const params = new URLSearchParams({
    proto: mode.value,
    host: vr.resolve(host.value),
    port: vr.resolve(port.value),
    token,
  });
  return `${scheme}//${location.host}/ws/relay?${params.toString()}`;
}

function clearReconnect() {
  if (reconnectTimer !== null) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
}

function connect() {
  if (!host.value || !port.value) return toast.error(t("socket.addrRequired"));
  clearReconnect();
  manualClose = false;
  connecting.value = true;
  try {
    ws = new WebSocket(buildUrl());
    ws.binaryType = "arraybuffer";
  } catch {
    connecting.value = false;
    return toast.error(t("common.error"));
  }
  ws.onopen = () => {
    connected.value = true;
    connecting.value = false;
    log("success", t("socket.connected"));
  };
  ws.onmessage = (ev) => {
    if (typeof ev.data === "string") {
      // 后端文本控制帧（connected/status/error/closed）
      try {
        const m = JSON.parse(ev.data);
        if (m.type === "status") {
          log("info", `${t("socket.local")}: ${m.local}  ·  ${t("socket.remote")}: ${m.remote}`);
        } else if (m.type === "error") {
          log("error", m.message || t("common.error"));
        } else if (m.type === "closed") {
          log("info", t("socket.disconnected"));
        }
      } catch {
        /* 非 JSON 文本忽略 */
      }
      return;
    }
    // 二进制数据帧：追加到接收缓冲
    const arr = new Uint8Array(ev.data as ArrayBuffer);
    const cur = recvBuf.value;
    const next = cur.concat(Array.from(arr));
    if (next.length > MAX_BUF) next.splice(0, next.length - MAX_BUF);
    recvBuf.value = next;
    recvBytes.value += arr.length;
  };
  ws.onclose = () => {
    ws = null;
    connected.value = false;
    connecting.value = false;
    if (manualClose) {
      log("info", t("socket.disconnected"));
      return;
    }
    // 非主动断开：自动重连
    log("info", t("socket.reconnecting"));
    if (reconnectTimer === null) {
      reconnectTimer = window.setTimeout(() => {
        reconnectTimer = null;
        if (!manualClose && !connected.value) connect();
      }, 2000);
    }
  };
  ws.onerror = () => toast.error(t("common.error"));
}

function disconnect() {
  manualClose = true;
  clearReconnect();
  ws?.close();
  ws = null;
  log("info", t("socket.disconnected"));
}

function send() {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  const raw = vr.resolve(sendInput.value);
  let bytes: Uint8Array;
  try {
    bytes =
      sendMode.value === "hex"
        ? hexToBytes(raw)
        : new TextEncoder().encode(raw);
  } catch {
    return toast.error(t("socket.invalidHex"));
  }
  if (bytes.length === 0) return toast.error(t("socket.invalidInput"));
  ws.send(bytes);
  sentBytes.value += bytes.length;
  sendInput.value = "";
}

function clearRecv() {
  recvBuf.value = [];
  recvBytes.value = 0;
}

function clearLog() {
  logs.value = [];
}

onUnmounted(() => {
  manualClose = true;
  clearReconnect();
  ws?.close();
});
</script>

<template>
  <div class="flex h-full flex-col gap-3 overflow-y-auto scroll-thin p-3">
    <Card class="p-3">
      <div class="flex flex-wrap items-center gap-2">
        <!-- 协议切换 -->
        <div class="flex overflow-hidden rounded-lg border border-border">
          <button
            class="px-3 py-1.5 text-sm transition-colors"
            :class="mode === 'tcp' ? 'bg-primary text-white' : 'text-muted hover:text-foreground'"
            @click="mode = 'tcp'"
          >
            {{ t("socket.tcp") }}
          </button>
          <button
            class="px-3 py-1.5 text-sm transition-colors"
            :class="mode === 'udp' ? 'bg-primary text-white' : 'text-muted hover:text-foreground'"
            @click="mode = 'udp'"
          >
            {{ t("socket.udp") }}
          </button>
        </div>
        <Input v-model="host" :placeholder="t('socket.hostPlaceholder')" class="w-48" />
        <Input v-model="port" :placeholder="t('socket.portPlaceholder')" class="w-28" />
        <div class="ml-auto flex items-center gap-2">
          <span class="text-xs text-muted">{{ t("socket.sentBytes") }}: {{ sentBytes }}</span>
          <span class="text-xs text-muted">{{ t("socket.recvBytes") }}: {{ recvBytes }}</span>
          <Button v-if="!connected" :disabled="connecting" @click="connect">
            <PlugZap :size="15" /> {{ t("socket.connect") }}
          </Button>
          <Button v-else variant="danger" @click="disconnect">
            <Plug :size="15" /> {{ t("socket.disconnect") }}
          </Button>
          <RequestSaver
            :project-id="projectId"
            :request="request"
            :default-collection-id="defaultCollectionId"
            :default-name="defaultName || `${host}:${port}`"
            :build-payload="buildPayload"
            @saved="(r) => emit('saved', r)"
          />
        </div>
      </div>
    </Card>

    <div class="grid flex-1 grid-cols-1 gap-3 lg:grid-cols-2">
      <!-- 发送区 -->
      <Card class="flex flex-col p-3">
        <div class="mb-2 flex items-center justify-between">
          <div class="flex items-center gap-2 text-sm font-medium text-foreground">
            <Network :size="15" /> {{ t("socket.send") }}
          </div>
          <!-- 发送格式：HEX / ASCII -->
          <div class="flex overflow-hidden rounded border border-border text-xs">
            <button
              class="px-2 py-1 transition-colors"
              :class="sendMode === 'hex' ? 'bg-primary text-white' : 'text-muted hover:text-foreground'"
              @click="sendMode = 'hex'"
            >
              {{ t("socket.hex") }}
            </button>
            <button
              class="px-2 py-1 transition-colors"
              :class="sendMode === 'ascii' ? 'bg-primary text-white' : 'text-muted hover:text-foreground'"
              @click="sendMode = 'ascii'"
            >
              {{ t("socket.ascii") }}
            </button>
          </div>
        </div>
        <Textarea
          v-model="sendInput"
          class="flex-1 font-mono text-xs"
          :rows="10"
          :placeholder="sendMode === 'hex' ? t('socket.sendPlaceholderHex') : t('socket.sendPlaceholderAscii')"
        />
        <Button class="mt-2 self-end" :disabled="!connected" @click="send">
          <Send :size="15" /> {{ t("socket.send") }}
        </Button>
      </Card>

      <!-- 接收区 -->
      <Card class="flex flex-col overflow-hidden p-3">
        <div class="mb-2 flex items-center justify-between">
          <span class="text-sm font-medium text-foreground">
            {{ viewMode === "hex" ? t("socket.receivedHex") : t("socket.receivedAscii") }}
          </span>
          <div class="flex items-center gap-2">
            <!-- 显示格式：HEX / ASCII -->
            <div class="flex overflow-hidden rounded border border-border text-xs">
              <button
                class="px-2 py-1 transition-colors"
                :class="viewMode === 'hex' ? 'bg-primary text-white' : 'text-muted hover:text-foreground'"
                @click="viewMode = 'hex'"
              >
                {{ t("socket.hex") }}
              </button>
              <button
                class="px-2 py-1 transition-colors"
                :class="viewMode === 'ascii' ? 'bg-primary text-white' : 'text-muted hover:text-foreground'"
                @click="viewMode = 'ascii'"
              >
                {{ t("socket.ascii") }}
              </button>
            </div>
            <button class="text-xs text-muted hover:text-foreground" @click="clearRecv">
              <Trash2 :size="13" class="inline" /> {{ t("socket.clear") }}
            </button>
          </div>
        </div>
        <div
          class="flex-1 overflow-auto scroll-thin rounded-md bg-surface p-2 font-mono text-xs leading-relaxed text-foreground"
        >
          <pre v-if="recvBuf.length && viewMode === 'hex'" class="whitespace-pre">{{ hexDump }}</pre>
          <pre v-else-if="recvBuf.length && viewMode === 'ascii'" class="whitespace-pre break-all">{{ asciiView }}</pre>
          <div v-else class="text-muted/60">{{ t("socket.noData") }}</div>
        </div>
      </Card>
    </div>

    <!-- 连接日志 -->
    <Card class="flex flex-col p-3">
      <div class="mb-2 flex items-center justify-between text-sm font-medium text-foreground">
        <span class="flex items-center gap-2">
          <RefreshCw v-if="!connected && !manualClose" :size="14" class="animate-spin text-muted" />
          {{ t("socket.connectionLog") }}
        </span>
        <button class="text-xs text-muted hover:text-foreground" @click="clearLog">
          <Trash2 :size="13" class="inline" /> {{ t("socket.clear") }}
        </button>
      </div>
      <div class="max-h-40 space-y-1 overflow-y-auto scroll-thin">
        <div
          v-for="(l, i) in logs"
          :key="i"
          class="flex animate-slide-in items-start gap-2 text-xs"
          :class="{
            'text-success': l.level === 'success',
            'text-danger': l.level === 'error',
            'text-foreground': l.level === 'info',
          }"
        >
          <span class="shrink-0 text-muted">{{ l.time }}</span>
          <span class="break-all">{{ l.text }}</span>
        </div>
        <div v-if="logs.length === 0" class="text-xs text-muted/60">{{ t("socket.noData") }}</div>
      </div>
    </Card>
  </div>
</template>
