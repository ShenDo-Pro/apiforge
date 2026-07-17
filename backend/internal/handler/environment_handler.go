package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"apitoolx/backend/internal/model"
	"apitoolx/backend/internal/service"
	"apitoolx/backend/internal/server"
	"apitoolx/backend/pkg/response"
)

// EnvironmentHandler 处理环境与全局变量的 REST 接口。
type EnvironmentHandler struct {
	svc *service.EnvironmentService
}

func NewEnvironmentHandler(svc *service.EnvironmentService) *EnvironmentHandler {
	return &EnvironmentHandler{svc: svc}
}

type envMutateReq struct {
	Name   string         `json:"name"`
	Values []model.EnvVar `json:"values"`
}

// List 返回项目下全部环境（含 global 单例）。
func (h *EnvironmentHandler) List(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	envs, err := h.svc.List(uint(pid))
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, envs)
}

// Create 新建普通环境。
func (h *EnvironmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	var in envMutateReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
		response.Fail(w, http.StatusBadRequest, 400, "name required")
		return
	}
	env, err := h.svc.Create(uint(pid), in.Name, in.Values)
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, env)
}

// Update 修改环境名称与变量。
func (h *EnvironmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	eid, _ := strconv.ParseUint(server.Param(r, "envID"), 10, 64)
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	var in envMutateReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	if err := h.svc.Update(uint(eid), uint(pid), in.Name, in.Values); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, nil)
}

// Delete 删除环境（global 不可删）。
func (h *EnvironmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	eid, _ := strconv.ParseUint(server.Param(r, "envID"), 10, 64)
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	if err := h.svc.Delete(uint(eid), uint(pid)); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, nil)
}

// UpsertGlobal 覆盖写入全局变量（前端增量编辑后整体提交）。
func (h *EnvironmentHandler) UpsertGlobal(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	var in envMutateReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	if err := h.svc.UpsertGlobal(uint(pid), in.Values); err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, nil)
}

// Reorder 按前端提交的有序 id 重排环境顺序（拖拽）。
func (h *EnvironmentHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var ids []uint
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	if err := h.svc.Reorder(ids); err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, nil)
}
