import http, { type ApiResp } from "./index";
import type { Project, ProjectMember } from "@/types/project";

export function listProjects() {
  return http.get<ApiResp<Project[]>>("/project").then((r) => r.data.data);
}
export function createProject(name: string, description: string) {
  return http
    .post<ApiResp<Project>>("/project", { name, description })
    .then((r) => r.data.data);
}
export function getProject(id: number) {
  return http.get<ApiResp<Project>>(`/project/${id}`).then((r) => r.data.data);
}
export function updateProject(id: number, name: string, description: string) {
  return http
    .put<ApiResp<null>>(`/project/${id}`, { name, description })
    .then((r) => r.data.data);
}
export function deleteProject(id: number) {
  return http.delete<ApiResp<null>>(`/project/${id}`).then((r) => r.data.data);
}
export function listMembers(projectId: number) {
  return http
    .get<ApiResp<ProjectMember[]>>(`/project/${projectId}/members`)
    .then((r) => r.data.data);
}
export function addMember(
  projectId: number,
  userId: number,
  role: string,
  permissions: Record<string, boolean>
) {
  return http
    .post<ApiResp<null>>(`/project/${projectId}/members`, {
      userId,
      role,
      permissions,
    })
    .then((r) => r.data.data);
}
export function updateMember(
  projectId: number,
  userId: number,
  role: string,
  permissions: Record<string, boolean>
) {
  return http
    .put<ApiResp<null>>(`/project/${projectId}/members/${userId}`, {
      userId,
      role,
      permissions,
    })
    .then((r) => r.data.data);
}
export function removeMember(projectId: number, userId: number) {
  return http
    .delete<ApiResp<null>>(`/project/${projectId}/members/${userId}`)
    .then((r) => r.data.data);
}
