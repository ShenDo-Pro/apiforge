package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"apiforge/backend/internal/middleware"
	"apiforge/backend/internal/service"
	"apiforge/backend/internal/server"
	"apiforge/backend/pkg/response"
)

// ProjectHandler 处理项目与成员的 REST 接口。
type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

type projectCreateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create 新建项目，调用方身份取自 JWT 上下文。
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ContextUser(r)
	var in projectCreateReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
		response.Fail(w, http.StatusBadRequest, 400, "name required")
		return
	}
	p, err := h.svc.Create(claims.UserID, in.Name, in.Description)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.OK(w, p)
}

// List 返回当前用户可见项目。
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ContextUser(r)
	ps, err := h.svc.ListForUser(claims.UserID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.OK(w, ps)
}

// Get 获取单个项目详情。
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	p, err := h.svc.Get(uint(id))
	if err != nil {
		response.Fail(w, http.StatusNotFound, 404, "project not found")
		return
	}
	response.OK(w, p)
}

type projectUpdateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Update 更新项目元信息。
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	var in projectUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	if err := h.svc.Update(uint(id), in.Name, in.Description); err != nil {
		response.Fail(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.OK(w, nil)
}

// Delete 删除项目（需 owner 或 admin，由 RBAC 中间件保证）。
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	if err := h.svc.Delete(uint(id)); err != nil {
		response.Fail(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.OK(w, nil)
}

// ListMembers 返回项目成员（含用户名）。
func (h *ProjectHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	ms, err := h.svc.ListMembers(uint(id))
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.OK(w, ms)
}

type memberReq struct {
	UserID uint            `json:"userId"`
	Role   string          `json:"role"`
	Perms  map[string]bool `json:"permissions"`
}

// AddMember 邀请成员（仅 owner/admin，由 RBAC 保证）。
func (h *ProjectHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	var in memberReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.UserID == 0 {
		response.Fail(w, http.StatusBadRequest, 400, "userId required")
		return
	}
	if err := h.svc.AddMember(uint(id), in.UserID, in.Role, in.Perms); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, err.Error())
		return
	}
	response.OK(w, nil)
}

// UpdateMember 调整成员角色与权限（仅 owner/admin）。
func (h *ProjectHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	var in memberReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.UserID == 0 {
		response.Fail(w, http.StatusBadRequest, 400, "userId required")
		return
	}
	if err := h.svc.UpdateMember(uint(id), in.UserID, in.Role, in.Perms); err != nil {
		response.Fail(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.OK(w, nil)
}

// RemoveMember 移除成员（仅 owner/admin）。
func (h *ProjectHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
	uid, _ := strconv.ParseUint(server.Param(r, "userID"), 10, 64)
	if err := h.svc.RemoveMember(uint(id), uint(uid)); err != nil {
		response.Fail(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.OK(w, nil)
}
