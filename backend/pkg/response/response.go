package response

import (
	"encoding/json"
	"net/http"
)

// APIResponse 是全站统一的 JSON 响应结构。
// code 取 0 表示成功，非 0 表示业务错误，便于前端统一拦截。
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON 以统一结构写出响应体，并固定 Content-Type 为 application/json。
func JSON(w http.ResponseWriter, status int, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{Code: code, Message: message, Data: data})
}

// OK 成功响应。
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, 0, "ok", data)
}

// Fail 失败响应，调用方自行决定 HTTP 状态码。
func Fail(w http.ResponseWriter, status int, code int, message string) {
	JSON(w, status, code, message, nil)
}
