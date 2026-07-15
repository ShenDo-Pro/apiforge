package proxy

// ProxyRequest 是前端发往 /api/proxy 的请求体，与前端 DTO 严格对齐。
type ProxyRequest struct {
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	ForceHttp2 bool              `json:"forceHttp2"`
	TimeoutMs  int               `json:"timeoutMs"`  // 请求超时（毫秒），0 表示不限制
	FollowRedirect bool          `json:"followRedirect"` // 是否跟随 3xx 重定向
	SslVerify  bool              `json:"sslVerify"`  // 是否验证 TLS 证书，false 跳过验证
}

// ProxyResponse 是代理返回给前端的响应，含协议版本与分阶段耗时。
type ProxyResponse struct {
	Proto    string            `json:"proto"`
	Status   int               `json:"status"`
	Headers  map[string]string `json:"headers"`
	Body     string            `json:"body"`
	Cookies  []RespCookie      `json:"cookies,omitempty"`
	Timings  Timing            `json:"timings"`
	Error    string            `json:"error,omitempty"`
}

// RespCookie 是响应 Set-Cookie 解析后的结构化表示，供前端 Cookie 管理器展示与编辑。
type RespCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Expires  string `json:"expires,omitempty"`
	HttpOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
	SameSite string `json:"sameSite,omitempty"`
}

// Timing 记录一次请求各阶段耗时（毫秒）。
type Timing struct {
	DNS    int64 `json:"dns"`
	TLS    int64 `json:"tls"`
	Connect int64 `json:"connect"`
	TTFB   int64 `json:"ttfb"`
	Total  int64 `json:"total"`
}
