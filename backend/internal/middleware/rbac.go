package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"

	"apitoolx/backend/internal/model"
	"apitoolx/backend/internal/server"
	"apitoolx/backend/pkg/response"
	"gorm.io/gorm"
)

// 项目级权限标识。
const (
	PermAdd          = "add"          // 新增集合/请求
	PermEdit         = "edit"         // 修改集合/请求
	PermDelete       = "delete"       // 删除集合/请求
	PermDeleteProject = "delete_project" // 删除整个项目
	PermManageMembers = "manage_members" // 成员管理
)

// RequireAuth 仅校验请求已携带有效 JWT（用户已登录），用于无需项目级权限的接口，
// 例如「创建项目」这种尚未涉及 projectID 的入口（M1）。
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ContextUser(r) == nil {
			response.Fail(w, http.StatusUnauthorized, 401, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireProjectPerm 校验当前登录用户对 :projectID 项目是否拥有指定权限。
// 授权顺序：系统管理员 > 项目创建者 > 成员 owner > maintainer(受限) > developer(按 JSON 细粒度)。
func RequireProjectPerm(perm string, db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ContextUser(r)
			if claims == nil {
				response.Fail(w, http.StatusUnauthorized, 401, "unauthorized")
				return
			}
			// 系统管理员对所有项目拥有全部权限
			if claims.Role == "admin" {
				next.ServeHTTP(w, r)
				return
			}
			pid, err := strconv.ParseUint(server.Param(r, "projectID"), 10, 64)
			if err != nil {
				response.Fail(w, http.StatusBadRequest, 400, "invalid project id")
				return
			}
			var proj model.Project
			if err := db.First(&proj, pid).Error; err != nil {
				response.Fail(w, http.StatusNotFound, 404, "project not found")
				return
			}
			// 项目创建者天然全权限
			if proj.OwnerID == claims.UserID {
				next.ServeHTTP(w, r)
				return
			}
			var m model.ProjectMember
			if err := db.Where("project_id = ? AND user_id = ?", pid, claims.UserID).First(&m).Error; err != nil {
				response.Fail(w, http.StatusForbidden, 403, "no project permission")
				return
			}
			if m.Role == "owner" {
				next.ServeHTTP(w, r)
				return
			}
			if m.Role == "maintainer" {
				// maintainer 可管理集合/请求，但不可删除项目或管理成员
				if perm == PermDeleteProject || perm == PermManageMembers {
					response.Fail(w, http.StatusForbidden, 403, "forbidden")
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			// developer：按 JSON 细粒度校验 add/edit/delete
			var perms map[string]bool
			_ = json.Unmarshal([]byte(m.Permissions), &perms)
			if perms[perm] {
				next.ServeHTTP(w, r)
				return
			}
			response.Fail(w, http.StatusForbidden, 403, "permission denied")
		})
	}
}

// RequireAdmin 仅允许系统管理员访问（用于审计日志等全局管理接口）。
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := ContextUser(r)
		if c == nil || c.Role != "admin" {
			response.Fail(w, http.StatusForbidden, 403, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}
