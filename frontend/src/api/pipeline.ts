import http, { type ApiResp } from "./index";
import type { Pipeline, PipelineRun, PipelineStep } from "@/types/pipeline";
import type { SavedRequest as SavedReq } from "@/types/project";

export function listPipelines(projectId: number) {
  return http.get<ApiResp<Pipeline[]>>(`/project/${projectId}/pipelines`).then((r) => r.data.data);
}

export function getPipeline(projectId: number, pipelineId: number) {
  return http
    .get<ApiResp<Pipeline>>(`/project/${projectId}/pipeline/${pipelineId}`)
    .then((r) => r.data.data);
}

export function createPipeline(projectId: number, name: string, description = "") {
  return http
    .post<ApiResp<Pipeline>>(`/project/${projectId}/pipelines`, { name, description })
    .then((r) => r.data.data);
}

export function updatePipeline(
  projectId: number,
  pipelineId: number,
  payload: { name: string; description: string; steps: PipelineStep[] }
) {
  return http
    .put<ApiResp<Pipeline>>(`/project/${projectId}/pipeline/${pipelineId}`, payload)
    .then((r) => r.data.data);
}

export function deletePipeline(projectId: number, pipelineId: number) {
  return http
    .delete<ApiResp<null>>(`/project/${projectId}/pipeline/${pipelineId}`)
    .then((r) => r.data.data);
}

export function runPipeline(projectId: number, pipelineId: number) {
  return http
    .post<ApiResp<PipelineRun>>(`/project/${projectId}/pipeline/${pipelineId}/run`)
    .then((r) => r.data.data);
}

export function listRuns(projectId: number, pipelineId: number) {
  return http
    .get<ApiResp<PipelineRun[]>>(`/project/${projectId}/pipeline/${pipelineId}/runs`)
    .then((r) => r.data.data);
}

export function getRun(projectId: number, pipelineId: number, runId: number) {
  return http
    .get<ApiResp<PipelineRun>>(`/project/${projectId}/pipeline/${pipelineId}/run/${runId}`)
    .then((r) => r.data.data);
}

export function regenerateToken(projectId: number, pipelineId: number) {
  return http
    .post<ApiResp<{ webhookToken: string; webhookURL: string }>>(
      `/project/${projectId}/pipeline/${pipelineId}/regenerate-token`
    )
    .then((r) => r.data.data);
}

// 列出项目下全部保存请求，供流水线步骤「引用已有请求」选择。
export function listProjectRequests(projectId: number) {
  return http
    .get<ApiResp<SavedReq[]>>(`/project/${projectId}/requests`)
    .then((r) => r.data.data);
}
