package middleware

import (
	"context"
	"net/http"
	"strings"

	"apiforge/backend/internal/token"
	"apiforge/backend/pkg/config"
	"apiforge/backend/pkg/response"
)

// contextKey 用独立类型避免与其他包键冲突。
type contextKey string

const userKey contextKey = "claims"

// Claims 是 JWT 载荷，从 token 包复用（避免 service 层反向依赖 middleware）。
type Claims = token.Claims

// ContextUser 从请求上下文取出已认证的 Claims。
func ContextUser(r *http.Request) *Claims {
	if c, ok := r.Context().Value(userKey).(*Claims); ok {
		return c
	}
	return nil
}

// JWT 校验 Authorization: Bearer <token>，解析后注入上下文。
// 失败一律返回 401，由前端统一处理刷新或跳转登录。
func JWT(cfg *config.JWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				// 浏览器 WebSocket 无法设置自定义请求头，允许通过查询参数携带 token
				if q := r.URL.Query().Get("token"); q != "" {
					auth = "Bearer " + q
				}
			}
			if !strings.HasPrefix(auth, "Bearer ") {
				response.Fail(w, http.StatusUnauthorized, 401, "missing token")
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			claims := &token.Claims{}
			parsed, err := token.ParseToken(tokenStr, cfg.Secret, claims)
			if err != nil || !parsed.Valid {
				response.Fail(w, http.StatusUnauthorized, 401, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), userKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
