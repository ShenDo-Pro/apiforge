import http, { type ApiResp } from "./index";
import type { Collection, SavedRequest } from "@/types/project";

export function listCollections(projectId: number) {
  return http
    .get<ApiResp<Collection[]>>(`/project/${projectId}/collections`)
    .then((r) => r.data.data);
}
export function createCollection(
  projectId: number,
  payload: { parentId?: number | null; name: string; sortOrder?: number }
) {
  return http
    .post<ApiResp<Collection>>(`/project/${projectId}/collections`, payload)
    .then((r) => r.data.data);
}
export function updateCollection(
  projectId: number,
  collectionId: number,
  payload: { name?: string; sortOrder?: number; variables?: string }
) {
  return http
    .put<ApiResp<null>>(`/project/${projectId}/collection/${collectionId}`, payload)
    .then((r) => r.data.data);
}
export function deleteCollection(projectId: number, collectionId: number) {
  return http
    .delete<ApiResp<null>>(`/project/${projectId}/collection/${collectionId}`)
    .then((r) => r.data.data);
}
export function listRequests(projectId: number, collectionId: number) {
  return http
    .get<ApiResp<SavedRequest[]>>(
      `/project/${projectId}/collection/${collectionId}/requests`
    )
    .then((r) => r.data.data);
}
export function saveRequest(
  projectId: number,
  collectionId: number,
  payload: {
    protocol?: string;
    name: string;
    method: string;
    url: string;
    headers: string;
    body: string;
    // 鉴权与脚本透传（导入 Postman 时一并落库）
    auth?: string;
    preRequestScript?: string;
    testScript?: string;
    extractRules?: string;
  }
) {
  return http
    .post<ApiResp<SavedRequest>>(
      `/project/${projectId}/collection/${collectionId}/requests`,
      payload
    )
    .then((r) => r.data.data);
}
