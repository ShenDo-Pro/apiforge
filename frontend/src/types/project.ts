export interface Project {
  id: number;
  name: string;
  description: string;
  ownerId: number;
  createdAt: string;
}

export interface ProjectMember {
  id: number;
  projectId: number;
  userId: number;
  username: string;
  role: "owner" | "maintainer" | "developer";
  permissions: string; // JSON: {"add":bool,"edit":bool,"delete":bool}
}

export interface Collection {
  id: number;
  projectId: number;
  parentId: number | null;
  name: string;
  sortOrder: number;
  variables: string; // 集合级变量 JSON（[]EnvVar），默认生效
}

export interface EnvVar {
  key: string;
  value: string;
  enabled: boolean; // 是否参与替换（集合/全局变量恒为 true）
  secret: boolean; // 机密变量：界面掩码显示
}

export type EnvScope = "global" | "collection" | "environment" | "local";

export interface Environment {
  id: number;
  projectId: number;
  kind: "env" | "global"; // global 为全局变量单例
  name: string;
  values: string; // JSON: EnvVar[]（后端返回字符串，前端解析）
  sortOrder: number;
}

// 解析后的环境（values 已转为 EnvVar[]）
export interface EnvironmentParsed extends Omit<Environment, "values"> {
  vars: EnvVar[];
}

export interface ExtractRule {
  source: "body" | "header"; // 从响应体(JSONPath) 或 响应头
  expr: string; // JSONPath（如 $.token）或 header 名
  targetVar: string; // 写入的变量名
  scope: EnvScope; // 写回目标作用域
  enabled: boolean;
}

export interface SavedRequest {
  id: number;
  collectionId: number;
  name: string;
  protocol: string; // http / ws / mqtt / ...，空值按 http 处理
  method: string;
  url: string;
  headers: string; // JSON 字符串
  body: string;
  preRequestScript: string; // 预请求脚本
  testScript: string; // 测试脚本
  extractRules: string; // GUI 提取规则 JSON
  auth: string; // 鉴权配置 JSON（None/Bearer/Basic/APIKey/OAuth2）
}

export interface RequestHistory {
  id: number;
  savedRequestId: number;
  method: string;
  url: string;
  statusCode: number;
  proto: string;
  responseHeaders: string;
  responseBody: string;
  timings: string; // JSON
  createdAt: string;
}
