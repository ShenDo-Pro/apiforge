import { createI18n } from "vue-i18n";
import zhCNCommon from "./zh-CN/common.json";
import zhCNAuth from "./zh-CN/auth.json";
import zhCNProject from "./zh-CN/project.json";
import zhCNHttp from "./zh-CN/http.json";
import zhCNWs from "./zh-CN/ws.json";
import zhCNMqtt from "./zh-CN/mqtt.json";
import zhCNSocket from "./zh-CN/socket.json";
import zhCNGraphql from "./zh-CN/graphql.json";
import zhCNSocketio from "./zh-CN/socketio.json";
import zhCNGrpc from "./zh-CN/grpc.json";
import zhCNAudit from "./zh-CN/audit.json";
import zhCNAccount from "./zh-CN/account.json";
import zhCNPipeline from "./zh-CN/pipeline.json";
import enCommon from "./en-US/common.json";
import enAudit from "./en-US/audit.json";
import enAccount from "./en-US/account.json";
import enAuth from "./en-US/auth.json";
import enProject from "./en-US/project.json";
import enHttp from "./en-US/http.json";
import enWs from "./en-US/ws.json";
import enMqtt from "./en-US/mqtt.json";
import enGraphql from "./en-US/graphql.json";
import enSocket from "./en-US/socket.json";
import enSocketio from "./en-US/socketio.json";
import enGrpc from "./en-US/grpc.json";
import enPipeline from "./en-US/pipeline.json";

// 按模块聚合，便于后续扩展更多语言时不改动组件引用方式。
function pack(m: Record<string, unknown>) {
  return m;
}

export const messages = {
  "zh-CN": {
    common: pack(zhCNCommon),
    auth: pack(zhCNAuth),
    project: pack(zhCNProject),
    http: pack(zhCNHttp),
    ws: pack(zhCNWs),
    mqtt: pack(zhCNMqtt),
    socket: pack(zhCNSocket),
    graphql: pack(zhCNGraphql),
    socketio: pack(zhCNSocketio),
    grpc: pack(zhCNGrpc),
    pipeline: pack(zhCNPipeline),
    audit: pack(zhCNAudit),
    account: pack(zhCNAccount),
  },
  "en-US": {
    common: pack(enCommon),
    auth: pack(enAuth),
    project: pack(enProject),
    http: pack(enHttp),
    ws: pack(enWs),
    mqtt: pack(enMqtt),
    socket: pack(enSocket),
    graphql: pack(enGraphql),
    socketio: pack(enSocketio),
    grpc: pack(enGrpc),
    pipeline: pack(enPipeline),
    audit: pack(enAudit),
    account: pack(enAccount),
  },
};

// 语言偏好持久化，切换即时生效
const saved = localStorage.getItem("locale") || "zh-CN";

export const i18n = createI18n({
  legacy: false,
  locale: saved,
  fallbackLocale: "en-US",
  messages,
});

export function setLocale(loc: string) {
  i18n.global.locale.value = loc as "zh-CN" | "en-US";
  localStorage.setItem("locale", loc);
}
