package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"apitoolx/backend/internal/model"
	"apitoolx/backend/internal/server"
	"apitoolx/backend/internal/service"
	"apitoolx/backend/pkg/response"
)

// webhookLimit 对免鉴权的 Webhook 触发做简单的滑动窗口限流，防止被滥用耗尽资源（M13）。
// 每个 token 每分钟最多 20 次触发。
var (
	webhookMu       sync.Mutex
	webhookHits     = map[string][]time.Time{}
	webhookMaxPerMin = 20
)

func webhookAllowed(token string) bool {
	webhookMu.Lock()
	defer webhookMu.Unlock()
	now := time.Now()
	hits := webhookHits[token]
	// 仅保留最近一分钟内的记录
	recent := hits[:0]
	for _, t := range hits {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	if len(recent) >= webhookMaxPerMin {
		webhookHits[token] = recent
		return false
	}
	recent = append(recent, now)
	webhookHits[token] = recent
	return true
}

// PipelineHandler 暴露流水线 REST 与 Webhook 触发端点。
type PipelineHandler struct {
	svc *service.PipelineService
	// baseURL 用于拼接 Webhook 完整地址回传给前端；为空时由前端据自身 host 拼接。
	baseURL string
}

func NewPipelineHandler(svc *service.PipelineService, baseURL string) *PipelineHandler {
	return &PipelineHandler{svc: svc, baseURL: baseURL}
}

// List 项目下流水线列表。
func (h *PipelineHandler) List(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	ps, err := h.svc.List(uint(pid))
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, ps)
}

type pipelineCreateBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create 新建流水线。
func (h *PipelineHandler) Create(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	var in pipelineCreateBody
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
		response.Fail(w, http.StatusBadRequest, 400, "name required")
		return
	}
	p, err := h.svc.Create(uint(pid), service.PipelineCreateReq{Name: in.Name, Description: in.Description})
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, h.withWebhookURL(p))
}

// Get 返回流水线含步骤。
func (h *PipelineHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "pipelineID"), 10, 64)
	p, err := h.svc.Get(uint(id))
	if err != nil {
		response.Fail(w, http.StatusNotFound, 404, "pipeline not found")
		return
	}
	response.OK(w, h.withWebhookURL(p))
}

// Update 全量更新流水线含步骤。
func (h *PipelineHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "pipelineID"), 10, 64)
	var in service.PipelineUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	p, err := h.svc.Update(uint(id), in)
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, h.withWebhookURL(p))
}

// Delete 删除流水线。
func (h *PipelineHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "pipelineID"), 10, 64)
	if err := h.svc.Delete(uint(id)); err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, nil)
}

// RegenerateToken 重置 Webhook token。
func (h *PipelineHandler) RegenerateToken(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "pipelineID"), 10, 64)
	token, err := h.svc.RegenerateToken(uint(id))
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, map[string]string{"webhookToken": token, "webhookURL": h.buildURL(token)})
}

// Run 手动触发一次运行并返回完整结果。
func (h *PipelineHandler) Run(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "pipelineID"), 10, 64)
	run, err := h.svc.Run(uint(id), "manual")
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, run)
}

// ListRuns 运行历史。
func (h *PipelineHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "pipelineID"), 10, 64)
	runs, err := h.svc.ListRuns(uint(id))
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, runs)
}

// GetRun 单次运行详情。
func (h *PipelineHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	rid, _ := strconv.ParseUint(server.Param(r, "runID"), 10, 64)
	run, err := h.svc.GetRun(uint(rid))
	if err != nil {
		response.Fail(w, http.StatusNotFound, 404, "run not found")
		return
	}
	response.OK(w, run)
}

// Webhook 免鉴权触发端点：按 token 定位流水线并运行。
// 为避免慢流水线长时间占用 HTTP 连接，运行放到 goroutine 异步执行，
// 接口立即返回 202（M13）。运行错误由 service 层经 slog 记录，不再阻塞响应。
func (h *PipelineHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	token := server.Param(r, "token")
	// 免鉴权端点：先做限流，避免被滥用耗尽资源（M13）
	if !webhookAllowed(token) {
		response.Fail(w, http.StatusTooManyRequests, 429, "too many requests")
		return
	}
	p, err := h.svc.FindByToken(token)
	if err != nil {
		response.Fail(w, http.StatusNotFound, 404, "invalid webhook token")
		return
	}
	pid := p.ID
	go func() {
		if _, e := h.svc.Run(pid, "webhook"); e != nil {
			// 异步运行错误不回传客户端，仅记录（Run 内部已 slog）
			slog.Warn("webhook pipeline run failed", "pipelineID", pid, "err", e)
		}
	}()
	response.JSON(w, http.StatusAccepted, 0, "accepted", map[string]interface{}{
		"pipelineId": pid,
		"status":     "queued",
	})
}

// withWebhookURL 在返回对象上附加可调用 Webhook 地址（token 不随对象返回，仅在 URL 中体现）。
func (h *PipelineHandler) withWebhookURL(p *model.Pipeline) map[string]interface{} {
	return map[string]interface{}{
		"id":          p.ID,
		"projectId":   p.ProjectID,
		"name":        p.Name,
		"description": p.Description,
		"createdAt":   p.CreatedAt,
		"updatedAt":   p.UpdatedAt,
		"steps":       p.Steps,
		"runCount":    len(p.Runs),
		"webhookURL":  h.buildURL(p.WebhookToken),
	}
}

func (h *PipelineHandler) buildURL(token string) string {
	if h.baseURL != "" {
		return h.baseURL + "/api/webhook/" + token
	}
	return "/api/webhook/" + token
}
