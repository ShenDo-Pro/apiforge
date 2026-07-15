import {
  Globe2,
  Radio,
  Send,
  Network,
  Hexagon,
  Cable,
  Binary,
  type Component,
} from "lucide-vue-next";

// 协议清单：与后端收件箱、侧边栏入口、请求树图标保持一致。
export interface ProtocolMeta {
  key: string;
  label: string;
  icon: Component;
  // 是否为保存请求支持的协议（流水线引用也复用此清单）
  savable: boolean;
}

export const PROTOCOLS: ProtocolMeta[] = [
  { key: "http", label: "HTTP / HTTP2", icon: Globe2, savable: true },
  { key: "ws", label: "WebSocket", icon: Radio, savable: true },
  { key: "mqtt", label: "MQTT", icon: Send, savable: true },
  { key: "socket", label: "TCP / UDP", icon: Network, savable: true },
  { key: "graphql", label: "GraphQL", icon: Hexagon, savable: true },
  { key: "socketio", label: "Socket.IO", icon: Cable, savable: true },
  { key: "grpc", label: "gRPC", icon: Binary, savable: true },
];

export const PROTOCOL_MAP: Record<string, ProtocolMeta> = Object.fromEntries(
  PROTOCOLS.map((p) => [p.key, p]),
);

export function protocolMeta(key: string): ProtocolMeta {
  return PROTOCOL_MAP[key] || PROTOCOLS[0];
}
