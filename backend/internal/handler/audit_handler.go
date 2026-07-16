package handler

import (
	"net/http"
	"strconv"

	"apiforge/backend/internal/service"
	"apiforge/backend/pkg/response"
)

// AuditHandler 提供审计日志查询接口（C9）。
type AuditHandler struct {
	svc *service.AuditService
}

func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// List 返回分页的审计日志，供管理员追溯操作。
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("perPage"))
	logs, total, err := h.svc.List(page, perPage)
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, map[string]any{"logs": logs, "total": total})
}
