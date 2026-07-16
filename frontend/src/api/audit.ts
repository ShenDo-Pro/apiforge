import http, { type ApiResp } from "./index";

export interface AuditLog {
  id: number;
  userId: number;
  username: string;
  method: string;
  path: string;
  status: number;
  createdAt: string;
}

// 取分页审计日志（仅管理员）
export function listAudit(page = 1, perPage = 20) {
  return http
    .get<ApiResp<{ logs: AuditLog[]; total: number }>>(
      `/audit?page=${page}&perPage=${perPage}`,
    )
    .then((r) => r.data.data);
}
