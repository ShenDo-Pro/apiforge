<script setup lang="ts">
import { ref, onUnmounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { io, type Socket } from "socket.io-client";
import { useToast } from "@/composables/useToast";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import Textarea from "@/components/ui/Textarea.vue";
import { Plug, PlugZap, Send, Cable } from "lucide-vue-next";
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
    protocol: "socketio",
    name,
    method: "",
    url: url.value,
    headers: "{}",
    body: JSON.stringify({
      path: path.value,
      transports: transports.value,
      eventName: eventName.value,
      payload: payload.value,
    }),
  };
}
function loadRequest(r: SavedRequest) {
  url.value = r.url || "";
  try {
    const o = JSON.parse(r.body || "{}");
    path.value = o.path || "/socket.io";
    transports.value = o.transports || "websocket,polling";
    eventName.value = o.eventName || "message";
    payload.value = o.payload || "";
  } catch {}
}
watch(
  () => props.request,
  (r) => {
    if (r) loadRequest(r);
  },
  { immediate: true },
);

const url = ref("http://localhost:3000");
const path = ref("/socket.io");
const transports = ref("websocket,polling");
const eventName = ref("message");
const payload = ref("");
const connected = ref(false);
const connecting = ref(false);
const logs = ref<{ dir: "sent" | "recv"; text: string; time: string }[]>([]);

let socket: Socket | null = null;

function now() {
  return new Date().toLocaleTimeString();
}

function connect() {
  if (!url.value) return toast.error(t("common.url"));
  connecting.value = true;
  socket = io(vr.resolve(url.value), {
    path: vr.resolve(path.value) || undefined,
    transports: vr.resolve(transports.value) ? vr.resolve(transports.value).split(",") : undefined,
    // 连接超时上限，避免目标不可达时无限期挂起（M32 一致性）
    timeout: 20000,
  });
  socket.on("connect", () => {
    connected.value = true;
    connecting.value = false;
    toast.success(t("socketio.connected"));
  });
  socket.on("connect_error", (e) => {
    connecting.value = false;
    toast.error(t("socketio.connectFailed") + ": " + e.message);
  });
  socket.on("disconnect", () => (connected.value = false));
  socket.onAny((event, ...args) => {
    logs.value.unshift({ dir: "recv", text: event + " " + JSON.stringify(args), time: now() });
  });
}

function disconnect() {
  socket?.disconnect();
  socket = null;
}

function sendEvent() {
  if (!socket || !socket.connected) return;
  const raw = vr.resolve(payload.value);
  let data: unknown = raw;
  try {
    data = JSON.parse(raw);
  } catch {
    /* 保持字符串 */
  }
  const ev = vr.resolve(eventName.value);
  socket.emit(ev, data);
  logs.value.unshift({ dir: "sent", text: ev + " " + JSON.stringify(data), time: now() });
  payload.value = "";
}

onUnmounted(() => socket?.disconnect());
</script>

<template>
  <div class="flex h-full flex-col gap-3 p-3">
    <Card class="p-3">
      <div class="flex flex-wrap items-center gap-2">
        <Input v-model="url" :placeholder="t('socketio.urlPlaceholder')" class="min-w-[200px] flex-1" />
        <Input v-model="path" :placeholder="t('socketio.path')" class="w-32" />
        <Input v-model="transports" :placeholder="t('socketio.transports')" class="w-40" />
        <Button v-if="!connected" :disabled="connecting" @click="connect">
          <PlugZap :size="15" /> {{ t("socketio.connect") }}
        </Button>
        <Button v-else variant="danger" @click="disconnect">
          <Plug :size="15" /> {{ t("socketio.disconnect") }}
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
    </Card>

    <div class="grid flex-1 grid-cols-1 gap-3 overflow-hidden lg:grid-cols-2">
      <Card class="flex flex-col p-3">
        <div class="mb-2 text-sm font-medium text-foreground">{{ t("socketio.emit") }}</div>
        <Input v-model="eventName" :placeholder="t('socketio.eventName')" class="mb-2" />
        <Textarea v-model="payload" :rows="8" :placeholder="t('socketio.payloadPlaceholder')" class="flex-1" />
        <Button class="mt-2 self-end" :disabled="!connected" @click="sendEvent">
          <Send :size="15" /> {{ t("socketio.emit") }}
        </Button>
      </Card>

      <Card class="flex flex-col overflow-hidden p-3">
        <div class="mb-2 flex items-center gap-1 text-sm font-medium text-foreground">
          <Cable :size="14" /> {{ t("socketio.logs") }}
        </div>
        <div class="flex-1 space-y-1 overflow-y-auto scroll-thin">
          <div
            v-for="(l, i) in logs"
            :key="i"
            class="flex animate-slide-in items-start gap-2 rounded-md px-2 py-1 text-xs"
            :class="l.dir === 'sent' ? 'bg-primary/10' : 'bg-surface'"
          >
            <span class="shrink-0 text-muted">{{ l.time }}</span>
            <span :class="l.dir === 'sent' ? 'text-primary' : 'text-success'">{{ l.dir === "sent" ? "↑" : "↓" }}</span>
            <span class="break-all text-foreground">{{ l.text }}</span>
          </div>
          <div v-if="logs.length === 0" class="text-xs text-muted/60">{{ t("socketio.noLogs") }}</div>
        </div>
      </Card>
    </div>
  </div>
</template>
