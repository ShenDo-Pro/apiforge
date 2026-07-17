package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"apitoolx/backend/internal/middleware"
	"apitoolx/backend/internal/service"
	"apitoolx/backend/pkg/config"
	"apitoolx/backend/pkg/response"
)

// maxAuthBody 限制认证接口请求体大小，防止超大请求体耗尽内存（M12）。
const maxAuthBody = 1 << 20 // 1MB

// registerLimit 对注册接口做按 IP 的滑动窗口限流，缓解批量注册/爆破（M5）。
// 每个来源 IP 每分钟最多 10 次注册尝试。
var (
	regMu          sync.Mutex
	regHits        = map[string][]time.Time{}
	regMaxPerMin   = 10
)

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 取第一个（最原始）客户端地址
		if i := indexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	return r.RemoteAddr
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func registerAllowed(ip string) bool {
	regMu.Lock()
	defer regMu.Unlock()
	now := time.Now()
	hits := regHits[ip]
	recent := hits[:0]
	for _, t := range hits {
		if now.Sub(t) < time.Minute {
			recent = append(recent, t)
		}
	}
	if len(recent) >= regMaxPerMin {
		regHits[ip] = recent
		return false
	}
	recent = append(recent, now)
	regHits[ip] = recent
	return true
}

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
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBody)
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
	// 注册限流：按来源 IP 滑动窗口（M5）
	if !registerAllowed(clientIP(r)) {
		response.Fail(w, http.StatusTooManyRequests, 429, "too many registration attempts")
		return
	}
	var in loginReq
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBody)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	if in.Username == "" || in.Password == "" {
		response.Fail(w, http.StatusBadRequest, 400, "username and password required")
		return
	}
	// 基础密码强度策略：至少 8 位（M5）。更严格的策略可随后端策略扩展。
	if len(in.Password) < 8 {
		response.Fail(w, http.StatusBadRequest, 400, "password must be at least 8 characters")
		return
	}
	if _, err := h.svc.Register(in.Username, in.Password); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, err.Error())
		return
	}
	u, access, refresh, err := h.svc.Login(in.Username, in.Password)
	if err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
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

// Logout 注销 refresh token：将其 JTI 加入黑名单，使其立即失效（M2）。
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var in refreshReq
	_ = json.NewDecoder(r.Body).Decode(&in)
	if in.RefreshToken == "" {
		// 无 refresh token 时视为已登出，直接成功
		response.OK(w, nil)
		return
	}
	if err := h.svc.Logout(in.RefreshToken); err != nil {
		response.FailSafe(w, http.StatusInternalServerError, 500, "internal error", err)
		return
	}
	response.OK(w, nil)
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

type resetReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ResetPassword 修改当前登录用户密码，需有效 JWT（H6）。
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ContextUser(r)
	if claims == nil {
		response.Fail(w, http.StatusUnauthorized, 401, "unauthorized")
		return
	}
	var in resetReq
	// 改密属敏感操作，同样限制请求体大小（M12）
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBody)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.NewPassword == "" {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	if err := h.svc.ResetPassword(claims.UserID, in.OldPassword, in.NewPassword); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, err.Error())
		return
	}
	response.OK(w, nil)
}
