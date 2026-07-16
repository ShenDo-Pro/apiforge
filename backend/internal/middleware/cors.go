package middleware

import (
	"net/http"
)

// CORS 处理跨域，开发期前端跑在 5173，生产期同源。
// 同时放行 WebSocket 升级所需的头，避免握手被浏览器拦截。
func CORS(allowOrigins []string) func(http.Handler) http.Handler {
	allow := map[string]bool{}
	for _, o := range allowOrigins {
		allow[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allow[origin] {
			// 精确匹配的源：允许携带凭据
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if origin != "" && allow["*"] {
			// 通配符配置：返回 * 但不带凭据，避免任意源携带凭据的安全风险（M18）
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
