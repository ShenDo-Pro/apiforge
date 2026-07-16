package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"apiforge/backend/internal/model"
	"apiforge/backend/internal/service"
	"apiforge/backend/internal/server"
	"apiforge/backend/pkg/response"
)

// RequestHandler 处理保存请求与发送历史的 CRUD。
type RequestHandler struct {
	svc *service.RequestService
}

func NewRequestHandler(svc *service.RequestService) *RequestHandler {
	return &RequestHandler{svc: svc}
}

type requestSaveReq struct {
	Protocol         string `json:"protocol"`
	Name             string `json:"name"`
	Method           string `json:"method"`
	URL              string `json:"url"`
	Headers          string `json:"headers"`
	Body             string `json:"body"`
	PreRequestScript string `json:"preRequestScript"`
	TestScript       string `json:"testScript"`
	ExtractRules     string `json:"extractRules"`
	Auth             string `json:"auth"`
}

// ListByCollection 返回集合下全部保存请求。
func (h *RequestHandler) ListByCollection(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	cid, _ := strconv.ParseUint(server.Param(r, "collectionID"), 10, 64)
	rs, err := h.svc.ListByCollection(uint(pid), uint(cid))
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, rs)
}

// ListAllByProject 返回项目下全部保存请求（跨集合），供流水线引用选择。
func (h *RequestHandler) ListAllByProject(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	rs, err := h.svc.ListAllByProject(uint(pid))
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, rs)
}

// Save 在集合下新增一条保存请求。
func (h *RequestHandler) Save(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	cid, _ := strconv.ParseUint(server.Param(r, "collectionID"), 10, 64)
	var in requestSaveReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.URL == "" {
		response.Fail(w, http.StatusBadRequest, 400, "url required")
		return
	}
	req, err := h.svc.Save(uint(pid), uint(cid), in.Protocol, in.Name, in.Method, in.URL, in.Headers, in.Body, in.PreRequestScript, in.TestScript, in.ExtractRules, in.Auth)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, req)
}

// Get 获取单条保存请求。
func (h *RequestHandler) Get(w http.ResponseWriter, r *http.Request) {
	rid, _ := strconv.ParseUint(server.Param(r, "requestID"), 10, 64)
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	req, err := h.svc.Get(uint(rid), uint(pid))
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.Fail(w, http.StatusNotFound, 404, "request not found")
		return
	}
	response.OK(w, req)
}

// Update 修改保存请求内容。
func (h *RequestHandler) Update(w http.ResponseWriter, r *http.Request) {
	rid, _ := strconv.ParseUint(server.Param(r, "requestID"), 10, 64)
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	var in requestSaveReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	if err := h.svc.Update(uint(rid), uint(pid), in.Protocol, in.Name, in.Method, in.URL, in.Headers, in.Body, in.PreRequestScript, in.TestScript, in.ExtractRules, in.Auth); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, nil)
}

// Delete 删除保存请求及其历史。
func (h *RequestHandler) Delete(w http.ResponseWriter, r *http.Request) {
	rid, _ := strconv.ParseUint(server.Param(r, "requestID"), 10, 64)
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	if err := h.svc.Delete(uint(rid), uint(pid)); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, nil)
}

type historyReq struct {
	Method          string `json:"method"`
	URL             string `json:"url"`
	StatusCode      int    `json:"statusCode"`
	Proto           string `json:"proto"`
	ResponseHeaders string `json:"responseHeaders"`
	ResponseBody    string `json:"responseBody"`
	Timings         string `json:"timings"`
}

// AddHistory 追加一条发送历史（由前端在代理返回后调用）。
func (h *RequestHandler) AddHistory(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	rid, _ := strconv.ParseUint(server.Param(r, "requestID"), 10, 64)
	var in historyReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	h2 := &model.RequestHistory{
		Method:          in.Method,
		URL:             in.URL,
		StatusCode:      in.StatusCode,
		Proto:           in.Proto,
		ResponseHeaders: in.ResponseHeaders,
		ResponseBody:    in.ResponseBody,
		Timings:         in.Timings,
	}
	if err := h.svc.AddHistory(uint(rid), uint(pid), h2); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, h2)
}

// ListHistory 返回请求的历史快照。
func (h *RequestHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	rid, _ := strconv.ParseUint(server.Param(r, "requestID"), 10, 64)
	hs, err := h.svc.ListHistory(uint(rid), uint(pid))
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, hs)
}
