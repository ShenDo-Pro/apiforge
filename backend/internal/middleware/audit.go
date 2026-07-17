package middleware

import (
	"net/http"
	"strings"

	"apitoolx/backend/internal/model"
	"apitoolx/backend/internal/service"
	"apitoolx/backend/internal/token"
)

// statusRecorder 捕获响应状态码，供审计记录（C9）。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Audit 记录所有会改变状态的写操作（非 GET/HEAD/OPTIONS），用于安全审计追溯（C9）。
// 放在 JWT 外层：handler 处理完成后（JWT 已注入用户到上下文）再读取用户身份。
func Audit(secret string, svc *service.AuditService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: 200}
			next.ServeHTTP(rec, r)
			// 仅记录会改变状态的写操作
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				return
			}
			userID := uint(0)
			username := ""
			if claims := ContextUser(r); claims != nil {
				userID = claims.UserID
				username = claims.Username
			} else if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				// 审计在最外层、JWT 尚未注入 context 时，直接从 token 解析用户
				tk := strings.TrimPrefix(auth, "Bearer ")
				c := &token.Claims{}
				if parsed, err := token.ParseToken(tk, secret, c); err == nil && parsed.Valid {
					userID = c.UserID
					username = c.Username
				}
			}
			go func() {
				_ = svc.Append(&model.AuditLog{
					UserID:   userID,
					Username: username,
					Method:   r.Method,
					Path:     r.URL.Path,
					Status:   rec.status,
				})
			}()
		})
	}
}
