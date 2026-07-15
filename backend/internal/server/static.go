package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// SpaHandler 托管前端构建产物（dist）。
// 命中静态文件直接返回；其余非 API/WS 路径回退到 index.html，支撑前端路由。
func SpaHandler(distDir string) http.Handler {
	fs := http.FileServer(http.Dir(distDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API 与 WS 交由路由层，不在此处理
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}
		clean := filepath.Clean(r.URL.Path)
		p := filepath.Join(distDir, clean)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/"
		fs.ServeHTTP(w, r)
	})
}
