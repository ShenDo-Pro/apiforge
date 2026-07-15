package handler

import (
	"encoding/json"
	"net/http"

	"apiforge/backend/internal/service"
	"apiforge/backend/pkg/config"
	"apiforge/backend/pkg/response"
)

// AuthHandler 处理登录、注册、刷新三类认证接口。
type AuthHandler struct {
	svc *service.AuthService
	cfg *config.JWTConfig
}

func NewAuthHandler(svc *service.AuthService, cfg *config.JWTConfig) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login 校验凭据并返回双 token 与用户信息。
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in loginReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	u, access, refresh, err := h.svc.Login(in.Username, in.Password)
	if err != nil {
		response.Fail(w, http.StatusUnauthorized, 401, "invalid username or password")
		return
	}
	response.OK(w, map[string]interface{}{
		"access_token":  access,
		"refresh_token": refresh,
		"user":          u,
	})
}

// Register 注册普通用户，成功后直接签发 token 以便前端自动登录。
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in loginReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	if in.Username == "" || in.Password == "" {
		response.Fail(w, http.StatusBadRequest, 400, "username and password required")
		return
	}
	if _, err := h.svc.Register(in.Username, in.Password); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, err.Error())
		return
	}
	u, access, refresh, err := h.svc.Login(in.Username, in.Password)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	response.OK(w, map[string]interface{}{
		"access_token":  access,
		"refresh_token": refresh,
		"user":          u,
	})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh 用 refresh token 换取新的 access token。
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var in refreshReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.RefreshToken == "" {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	access, err := h.svc.Refresh(in.RefreshToken)
	if err != nil {
		response.Fail(w, http.StatusUnauthorized, 401, "invalid refresh token")
		return
	}
	response.OK(w, map[string]string{"access_token": access})
}
