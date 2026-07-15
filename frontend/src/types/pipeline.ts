// 测试流水线相关前端类型，与后端 model/pipeline.go 对齐。

export type AssertionType = "status" | "body_contains" | "header_equals" | "max_duration_ms";

export interface Assertion {
  type: AssertionType;
  expected: string;
  header?: string;
  invert?: boolean;
}

export interface PipelineStep {
  id?: number;
  savedRequestId?: number | null;
  name: string;
  enabled: boolean;
  method: string;
  url: string;
  headers: string; // JSON 字符串
  body: string;
  assertions: string; // JSON 字符串
}

export interface Pipeline {
  id: number;
  projectId: number;
  name: string;
  description: string;
  createdAt: string;
  updatedAt?: string;
  steps?: PipelineStep[];
  runCount?: number;
  webhookURL?: string;
}

export interface AssertionResult {
  type: string;
  expected: string;
  header?: string;
  actual: string;
  passed: boolean;
}

export interface PipelineStepResult {
  id: number;
  runId: number;
  stepId: number;
  stepName: string;
  status: "passed" | "failed" | "error";
  method: string;
  url: string;
  statusCode: number;
  durationMs: number;
  responseHeaders: string;
  responseBody: string;
  error: string;
  assertionResults: string; // JSON 字符串
}

export interface PipelineRun {
  id: number;
  pipelineId: number;
  trigger: "manual" | "webhook";
  status: "running" | "passed" | "failed";
  startedAt: string;
  finishedAt: string;
  summary: string;
  results?: PipelineStepResult[];
}
