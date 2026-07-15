import http, { type ApiResp } from "./index";
import type { SavedRequest, RequestHistory } from "@/types/project";

export function getRequest(projectId: number, requestId: number) {
  return http
    .get<ApiResp<SavedRequest>>(`/project/${projectId}/request/${requestId}`)
    .then((r) => r.data.data);
}
export function updateRequest(
  projectId: number,
  requestId: number,
  payload: {
    protocol?: string;
    name: string;
    method: string;
    url: string;
    headers: string;
    body: string;
  }
) {
  return http
    .put<ApiResp<null>>(`/project/${projectId}/request/${requestId}`, payload)
    .then((r) => r.data.data);
}
export function deleteRequest(projectId: number, requestId: number) {
  return http
    .delete<ApiResp<null>>(`/project/${projectId}/request/${requestId}`)
    .then((r) => r.data.data);
}
export function listHistory(projectId: number, requestId: number) {
  return http
    .get<ApiResp<RequestHistory[]>>(
      `/project/${projectId}/request/${requestId}/history`
    )
    .then((r) => r.data.data);
}
export function addHistory(
  projectId: number,
  requestId: number,
  payload: {
    method: string;
    url: string;
    statusCode: number;
    proto: string;
    responseHeaders: string;
    responseBody: string;
    timings: string;
  }
) {
  return http
    .post<ApiResp<RequestHistory>>(
      `/project/${projectId}/request/${requestId}/history`,
      payload
    )
    .then((r) => r.data.data);
}
