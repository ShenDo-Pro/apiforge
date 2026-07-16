<script setup lang="ts">
import { ref, onUnmounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import mqtt from "mqtt";
import { useToast } from "@/composables/useToast";
import Card from "@/components/ui/Card.vue";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import Textarea from "@/components/ui/Textarea.vue";
import Select from "@/components/ui/Select.vue";
import { Plug, PlugZap, Send, Rss, Server } from "lucide-vue-next";
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
    protocol: "mqtt",
    name,
    method: "",
    url: connMode.value === "browser" ? brokerUrl.value : `${brokerHost.value}:${brokerPort.value}`,
    headers: "{}",
    body: JSON.stringify({
      connMode: connMode.value,
      brokerUrl: brokerUrl.value,
      brokerHost: brokerHost.value,
      brokerPort: brokerPort.value,
      clientId: clientId.value,
      username: username.value,
      password: password.value,
      topic: topic.value,
      qos: qos.value,
      message: message.value,
    }),
  };
}
function loadRequest(r: SavedRequest) {
  try {
    const o = JSON.parse(r.body || "{}");
    if (o.connMode) connMode.value = o.connMode;
    if (o.brokerUrl) brokerUrl.value = o.brokerUrl;
    if (o.brokerHost) brokerHost.value = o.brokerHost;
    if (o.brokerPort) brokerPort.value = o.brokerPort;
    clientId.value = o.clientId || clientId.value;
    username.value = o.username || "";
    password.value = o.password || "";
    topic.value = o.topic || "";
    qos.value = o.qos ?? 0;
    message.value = o.message || "";
  } catch {}
}
watch(
  () => props.request,
  (r) => {
    if (r) loadRequest(r);
  },
  { immediate: true },
);

// 连接模式：浏览器直连 Broker 的 WebSocket 端口，或经后端 TCP 代理（仅暴露 1883 的 Broker 也能连）
const connMode = ref<"browser" | "backend">("browser");
const brokerUrl = ref("ws://broker.emqx.io:8083/mqtt");
const brokerHost = ref("broker.emqx.io");
const brokerPort = ref("1883");
const clientId = ref("apiforge-" + Math.random().toString(16).slice(2, 8));
const username = ref("");
const password = ref("");
const topic = ref("test/topic");
const qos = ref(0);
const message = ref("");
const connected = ref(false);
const connecting = ref(false);
const subscribed = ref<string[]>([]);
const messages = ref<{ topic: string; payload: string; time: string }[]>([]);
// 长会话内存上限：仅保留最近 N 条消息，避免无限增长（L14）
const MAX_MESSAGES = 2000;
function pushMessage(m: { topic: string; payload: string; time: string }) {
  messages.value.unshift(m);
  if (messages.value.length > MAX_MESSAGES) messages.value.splice(MAX_MESSAGES);
}

let client: mqtt.MqttClient | null = null;

function now() {
  return new Date().toLocaleTimeString();
}

// 后端 TCP 代理：经 /ws/relay 以 MQTT 二进制帧透传到 Broker 的 TCP 端口
function buildBackendUrlWith(host: string, port: string): string {
  const token = localStorage.getItem("access_token") || "";
  const scheme = location.protocol === "https:" ? "wss:" : "ws:";
  const params = new URLSearchParams({
    proto: "mqtt",
    host,
    port,
    token,
  });
  return `${scheme}//${location.host}/ws/relay?${params.toString()}`;
}

function connect() {
  if (connMode.value === "browser") {
    if (!brokerUrl.value) return toast.error(t("mqtt.brokerUrl"));
  } else {
    if (!brokerHost.value || !brokerPort.value) return toast.error(t("mqtt.brokerAddress"));
  }
  connecting.value = true;
  // 发送前替换 {{变量}}
  const resolvedBrokerUrl = vr.resolve(brokerUrl.value);
  const resolvedHost = vr.resolve(brokerHost.value);
  const resolvedPort = vr.resolve(brokerPort.value);
  const url =
    connMode.value === "browser"
      ? resolvedBrokerUrl
      : buildBackendUrlWith(resolvedHost, resolvedPort);
  // 后端代理失败后若自动重连会反复拨号 Broker，故关闭自动重连，由用户手动重连
  const opts: mqtt.IClientOptions = {
    clientId: vr.resolve(clientId.value),
    username: vr.resolve(username.value) || undefined,
    password: vr.resolve(password.value) || undefined,
    reconnectPeriod: connMode.value === "browser" ? 4000 : 0,
  };
  client = mqtt.connect(url, opts);
  client.on("connect", () => {
    connected.value = true;
    connecting.value = false;
    toast.success(t("mqtt.connected"));
  });
  client.on("error", () => {
    connecting.value = false;
    toast.error(t("common.error"));
  });
  client.on("message", (tp, payload) => {
    pushMessage({ topic: tp, payload: payload.toString(), time: now() });
  });
  client.on("close", () => (connected.value = false));
}

function disconnect() {
  client?.end();
  client = null;
}

function subscribe() {
  if (!client || !topic.value) return;
  const rt = vr.resolve(topic.value);
  client.subscribe(rt, { qos: qos.value as 0 | 1 | 2 }, () => {
    if (!subscribed.value.includes(rt)) subscribed.value.push(rt);
    toast.success(t("common.subscribe"));
  });
}

function publish() {
  if (!client || !topic.value) return;
  const rt = vr.resolve(topic.value);
  const payload = vr.resolve(message.value);
  client.publish(rt, payload, { qos: qos.value as 0 | 1 | 2 });
  pushMessage({ topic: rt, payload, time: now() });
  message.value = "";
}

onUnmounted(() => client?.end());
</script>

<template>
  <div class="flex h-full flex-col gap-3 overflow-y-auto scroll-thin p-3">
    <Card class="p-3">
      <div class="mb-2 flex items-center gap-2">
        <div class="flex overflow-hidden rounded-lg border border-border text-sm">
          <button
            type="button"
            class="px-3 py-1.5 transition-colors"
            :class="connMode === 'browser' ? 'bg-primary text-white' : 'text-muted hover:text-foreground'"
            @click="connMode = 'browser'"
          >
            {{ t("mqtt.modeBrowser") }}
          </button>
          <button
            type="button"
            class="flex items-center gap-1 px-3 py-1.5 transition-colors"
            :class="connMode === 'backend' ? 'bg-primary text-white' : 'text-muted hover:text-foreground'"
            @click="connMode = 'backend'"
          >
            <Server :size="14" /> {{ t("mqtt.modeBackend") }}
          </button>
        </div>
        <span class="text-xs text-muted">{{ t("mqtt.modeHint") }}</span>
      </div>

      <div class="grid grid-cols-1 gap-2 md:grid-cols-2">
        <div class="md:col-span-2">
          <Input
            v-if="connMode === 'browser'"
            v-model="brokerUrl"
            :placeholder="t('mqtt.brokerPlaceholder')"
          />
          <div v-else class="flex gap-2">
            <Input v-model="brokerHost" :placeholder="t('mqtt.brokerHost')" class="flex-1" />
            <Input v-model="brokerPort" :placeholder="t('mqtt.brokerPort')" class="w-24" />
          </div>
        </div>
        <Input v-model="clientId" :placeholder="t('mqtt.clientId')" />
        <Input v-model="username" :placeholder="t('mqtt.username')" />
        <Input v-model="password" type="password" :placeholder="t('mqtt.password')" />
        <div class="flex items-center gap-2">
          <Button v-if="!connected" :disabled="connecting" @click="connect">
            <PlugZap :size="15" /> {{ t("mqtt.connect") }}
          </Button>
          <Button v-else variant="danger" @click="disconnect">
            <Plug :size="15" /> {{ t("mqtt.disconnect") }}
          </Button>
          <RequestSaver
            :project-id="projectId"
            :request="request"
            :default-collection-id="defaultCollectionId"
            :default-name="defaultName || (connMode === 'browser' ? brokerUrl : `${brokerHost}:${brokerPort}`)"
            :build-payload="buildPayload"
            @saved="(r) => emit('saved', r)"
          />
        </div>
      </div>
    </Card>

    <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
      <Card class="p-3 space-y-2">
        <div class="text-sm font-medium text-foreground">{{ t("mqtt.subscribe") }}</div>
        <div class="flex gap-2">
          <Input v-model="topic" :placeholder="t('mqtt.topic')" class="flex-1" />
          <Select v-model="qos" class="w-20">
            <option :value="0">0</option>
            <option :value="1">1</option>
            <option :value="2">2</option>
          </Select>
          <Button :disabled="!connected" @click="subscribe"><Rss :size="15" /></Button>
        </div>
        <div class="flex flex-wrap gap-1">
          <span
            v-for="s in subscribed"
            :key="s"
            class="rounded-md bg-primary/15 px-2 py-0.5 text-xs text-primary"
          >{{ s }}</span>
        </div>
      </Card>

      <Card class="flex flex-col p-3">
        <div class="mb-2 text-sm font-medium text-foreground">{{ t("mqtt.publish") }}</div>
        <Textarea v-model="message" :rows="3" :placeholder="t('mqtt.message')" />
        <Button class="mt-2 self-end" :disabled="!connected" @click="publish">
          <Send :size="15" /> {{ t("mqtt.publish") }}
        </Button>
      </Card>
    </div>

    <Card class="flex flex-col overflow-hidden p-3">
      <div class="mb-2 text-sm font-medium text-foreground">{{ t("mqtt.messages") }}</div>
      <div class="space-y-1">
        <div
          v-for="(m, i) in messages"
          :key="i"
          class="flex animate-slide-in items-start gap-2 rounded-md bg-surface px-2 py-1 text-xs"
        >
          <span class="shrink-0 text-muted">{{ m.time }}</span>
          <span class="shrink-0 text-primary"># {{ m.topic }}</span>
          <span class="break-all text-foreground">{{ m.payload }}</span>
        </div>
        <div v-if="messages.length === 0" class="text-xs text-muted/60">{{ t("mqtt.noMessages") }}</div>
      </div>
    </Card>
  </div>
</template>
