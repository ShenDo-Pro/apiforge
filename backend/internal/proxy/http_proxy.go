package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"time"

	"apiforge/backend/internal/netutil"
)

// 安全配置由 main 在启动时注入（见 Configure）。
var (
	allowPrivateTargets bool
	skipTLSVerify       bool
	defaultTimeoutMs    int64 = 30000
)

// Configure 注入代理全局安全配置（SSRF/TLS/超时）。
func Configure(allowPrivate bool, skipTLS bool, timeoutMs int64) {
	allowPrivateTargets = allowPrivate
	skipTLSVerify = skipTLS
	if timeoutMs > 0 {
		defaultTimeoutMs = timeoutMs
	}
}

// sharedTransport 被所有请求复用，避免每次新建 Transport 导致的连接/goroutine 泄漏（M10）。
var sharedTransport = &http.Transport{
	ForceAttemptHTTP2:  true,
	MaxIdleConns:       200,
	MaxIdleConnsPerHost: 20,
	IdleConnTimeout:    90 * time.Second,
}

// h2Transport 专用于强制 h2 协商的场景（复用，不每次新建）。
var h2Transport = &http.Transport{
	ForceAttemptHTTP2:  true,
	MaxIdleConns:       50,
	MaxIdleConnsPerHost: 10,
	IdleConnTimeout:    90 * time.Second,
	TLSClientConfig:    &tls.Config{NextProtos: []string{"h2"}},
}

// Do 执行一次 HTTP/HTTP2 请求并采集协议版本与各阶段耗时。
// forceHttp2 时强制只协商 h2，用于验证目标服务的 HTTP/2 能力。
func Do(req *ProxyRequest, maxBody int64) *ProxyResponse {
	resp := &ProxyResponse{Headers: map[string]string{}, Timings: Timing{}}

	// SSRF 防护：默认禁止访问私有/内网/云元数据地址（H4）
	if !allowPrivateTargets {
		if err := netutil.ValidateOutboundURL(req.URL, false); err != nil {
			resp.Error = "target blocked: " + err.Error()
			resp.Timings.Total = 0
			return resp
		}
	}

	// 每次请求使用独立 cookie jar，避免不同用户/请求间 Cookie 串号（H3）
	jar, _ := cookiejar.New(nil)

	var dnsStart, tlsStart, connStart, ttfbStart time.Time
	var dnsDur, tlsDur, connDur, ttfbDur time.Duration

	trace := &httptrace.ClientTrace{
		DNSStart:           func(httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:            func(httptrace.DNSDoneInfo) { dnsDur = time.Since(dnsStart) },
		TLSHandshakeStart:  func() { tlsStart = time.Now() },
		TLSHandshakeDone:   func(tls.ConnectionState, error) { tlsDur = time.Since(tlsStart) },
		ConnectStart:       func(network, addr string) { connStart = time.Now() },
		ConnectDone:        func(network, addr string, err error) { connDur = time.Since(connStart) },
		GotFirstResponseByte: func() { ttfbDur = time.Since(ttfbStart) },
	}

	start := time.Now()
	var timeout int64 = defaultTimeoutMs
	if req.TimeoutMs > 0 {
		timeout = int64(req.TimeoutMs)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(ctx, trace),
		req.Method, req.URL, bytes.NewReader([]byte(req.Body)),
	)
	if err != nil {
		resp.Error = err.Error()
		resp.Timings.Total = time.Since(start).Milliseconds()
		return resp
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	ttfbStart = time.Now()

	// 重定向策略（H4）：
	// - FollowRedirect 为 false：禁止跟随，返回 ErrUseLastResponse 让调用方拿到真实 3xx；
	// - FollowRedirect 为 true：跟随，但**对每个重定向目标重新做 SSRF 出口校验**，
	//   禁止跳转到私有/内网/云元数据地址（初始 URL 已在上文校验，重定向目标不可遗漏）。
	checkRedirect := func(redir *http.Request, via []*http.Request) error {
		if !req.FollowRedirect {
			return http.ErrUseLastResponse
		}
		if !allowPrivateTargets {
			if err := netutil.ValidateOutboundURL(redir.URL.String(), false); err != nil {
				return fmt.Errorf("redirect target blocked: %w", err)
			}
		}
		return nil
	}

	// TLS：默认校验证书；仅当服务端允许跳过(skipTLSVerify)且客户端显式要求不校验时跳过（H7）
	tlsCfg := &tls.Config{InsecureSkipVerify: !req.SslVerify && skipTLSVerify}
	transport := sharedTransport
	if req.ForceHttp2 {
		transport = h2Transport
		tlsCfg.NextProtos = []string{"h2"}
	}
	client := &http.Client{
		Transport:     transport,
		CheckRedirect: checkRedirect,
		Jar:           jar,
	}
	// 为本次请求设定独立的 TLS 配置（不可变共享 transport 的字段，需克隆）
	transportClone := transport.Clone()
	transportClone.TLSClientConfig = tlsCfg
	client.Transport = transportClone

	httpResp, err := client.Do(httpReq)
	resp.Timings.Total = time.Since(start).Milliseconds()
	if err != nil {
		resp.Error = err.Error()
		return resp
	}
	defer httpResp.Body.Close()

	resp.Proto = httpResp.Proto
	resp.Status = httpResp.StatusCode
	for k, vv := range httpResp.Header {
		if len(vv) > 0 {
			resp.Headers[k] = vv[0]
		}
	}
	// 解析 Set-Cookie 供前端 Cookie 管理器展示（本请求的 jar 已自动完成存储与回送）
	for _, ck := range httpResp.Cookies() {
		exp := ""
		if !ck.Expires.IsZero() {
			exp = ck.Expires.Format(time.RFC1123)
		}
		resp.Cookies = append(resp.Cookies, RespCookie{
			Name:     ck.Name,
			Value:    ck.Value,
			Domain:   ck.Domain,
			Path:     ck.Path,
			Expires:  exp,
			HttpOnly: ck.HttpOnly,
			Secure:   ck.Secure,
			SameSite: sameSiteString(ck.SameSite),
		})
	}

	// 限制响应体大小，防止大响应撑爆内存
	limited := io.LimitReader(httpResp.Body, maxBody)
	bodyBytes, _ := io.ReadAll(limited)
	resp.Body = string(bodyBytes)
	resp.Timings.DNS = dnsDur.Milliseconds()
	resp.Timings.TLS = tlsDur.Milliseconds()
	resp.Timings.Connect = connDur.Milliseconds()
	resp.Timings.TTFB = ttfbDur.Milliseconds()
	return resp
}

// sameSiteString 将 http.SameSite 映射为前端可读的字符串（标准库无 String 方法）。
func sameSiteString(s http.SameSite) string {
	switch s {
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteNoneMode:
		return "None"
	default:
		return "Default"
	}
}
