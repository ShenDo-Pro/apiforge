package handler

import (
	"encoding/json"
	"net/http"

	"apiforge/backend/internal/proxy"
	"apiforge/backend/pkg/response"
)

// ProxyHandler 处理 /api/proxy：转发 HTTP/HTTP2 请求并回传协议细节与计时。
type ProxyHandler struct {
	maxBody int64
}

func NewProxyHandler(maxBody int64) *ProxyHandler {
	return &ProxyHandler{maxBody: maxBody}
}

// ServeHTTP 解析请求体后调用代理核心，原样回写响应结构。
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var in proxy.ProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Fail(w, http.StatusBadRequest, 400, "invalid body")
		return
	}
	if in.URL == "" {
		response.Fail(w, http.StatusBadRequest, 400, "url required")
		return
	}
	resp := proxy.Do(&in, h.maxBody)
	response.OK(w, resp)
}
