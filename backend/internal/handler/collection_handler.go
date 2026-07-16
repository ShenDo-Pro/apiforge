package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"apiforge/backend/internal/service"
	"apiforge/backend/internal/server"
	"apiforge/backend/pkg/response"
)

// CollectionHandler 处理请求集合的树形 CRUD。
type CollectionHandler struct {
	svc *service.CollectionService
}

func NewCollectionHandler(svc *service.CollectionService) *CollectionHandler {
	return &CollectionHandler{svc: svc}
}

type collectionCreateReq struct {
	ParentID  *uint  `json:"parentId"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
}

// List 返回项目下全部集合（扁平）。
func (h *CollectionHandler) List(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	cs, err := h.svc.List(uint(pid))
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, cs)
}

// Create 新建集合，可指定父节点。
func (h *CollectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	var in collectionCreateReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
		response.Fail(w, http.StatusBadRequest, 400, "name required")
		return
	}
	c, err := h.svc.Create(uint(pid), in.ParentID, in.Name, in.SortOrder)
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, c)
}

type collectionUpdateReq struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
	Variables string `json:"variables"` // 集合级变量 JSON（[]EnvVar）
}

// Update 重命名、调整排序或保存集合变量。
func (h *CollectionHandler) Update(w http.ResponseWriter, r *http.Request) {
	cid, _ := strconv.ParseUint(server.Param(r, "collectionID"), 10, 64)
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	var in collectionUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	if err := h.svc.Update(uint(cid), uint(pid), in.Name, in.SortOrder, in.Variables); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, nil)
}

// Delete 递归删除集合及其子节点。
func (h *CollectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cid, _ := strconv.ParseUint(server.Param(r, "collectionID"), 10, 64)
	pid, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	if err := h.svc.Delete(uint(cid), uint(pid)); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, nil)
}
