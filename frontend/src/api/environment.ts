import http, { type ApiResp } from "./index";
import type { EnvVar, Environment } from "@/types/project";

// 环境与环境变量（跟项目走）的 REST 封装。
// 后端返回的 Environment.values 为 JSON 字符串，前端按需解析为 EnvVar[]。

export function listEnvironments(projectId: number) {
  return http
    .get<ApiResp<Environment[]>>(`/project/${projectId}/environments`)
    .then((r) => r.data.data);
}

export function createEnvironment(
  projectId: number,
  payload: { name: string; values?: EnvVar[] }
) {
  return http
    .post<ApiResp<Environment>>(`/project/${projectId}/environments`, payload)
    .then((r) => r.data.data);
}

export function updateEnvironment(
  projectId: number,
  envId: number,
  payload: { name: string; values: EnvVar[] }
) {
  return http
    .put<ApiResp<null>>(`/project/${projectId}/environment/${envId}`, payload)
    .then((r) => r.data.data);
}

export function deleteEnvironment(projectId: number, envId: number) {
  return http
    .delete<ApiResp<null>>(`/project/${projectId}/environment/${envId}`)
    .then((r) => r.data.data);
}

// upsertGlobal 整体覆盖写入全局变量（前端增量编辑后整体提交）。
export function upsertGlobal(projectId: number, values: EnvVar[]) {
  return http
    .put<ApiResp<null>>(`/project/${projectId}/environment/global`, { values })
    .then((r) => r.data.data);
}

// reorderEnvironments 按给定 id 顺序重排（拖拽）。
export function reorderEnvironments(projectId: number, ids: number[]) {
  return http
    .post<ApiResp<null>>(`/project/${projectId}/environments/reorder`, ids)
    .then((r) => r.data.data);
}
