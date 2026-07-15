package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"time"
)

// cookieJar 在全局维护按域的 Cookie，使代理具备类浏览器的 Cookie 管理能力：
// 自动随请求回送、随 Set-Cookie 更新，并在响应中返回供前端展示。
var cookieJar, _ = cookiejar.New(nil)

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

// Do 执行一次 HTTP/HTTP2 请求并采集协议版本与各阶段耗时。
// forceHttp2 时强制只协商 h2，用于验证目标服务的 HTTP/2 能力。
func Do(req *ProxyRequest, maxBody int64) *ProxyResponse {
	resp := &ProxyResponse{Headers: map[string]string{}, Timings: Timing{}}

	// 累计各阶段耗时（纳秒转毫秒）
	var dnsStart, tlsStart, connStart, ttfbStart time.Time
	var dnsDur, tlsDur, connDur, ttfbDur time.Duration

	trace := &httptrace.ClientTrace{
		DNSStart:  func(httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:   func(httptrace.DNSDoneInfo) { dnsDur = time.Since(dnsStart) },
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone:  func(tls.ConnectionState, error) { tlsDur = time.Since(tlsStart) },
		ConnectStart:      func(network, addr string) { connStart = time.Now() },
		ConnectDone:       func(network, addr string, err error) { connDur = time.Since(connStart) },
		GotFirstResponseByte: func() { ttfbDur = time.Since(ttfbStart) },
	}

	start := time.Now()
	ctx := context.Background()
	if req.TimeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(req.TimeoutMs)*time.Millisecond)
		defer cancel()
	}
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

	// followRedirect 为 false 时禁止跟随（便于观察真实 3xx 状态码）
	checkRedirect := func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	if req.FollowRedirect {
		checkRedirect = nil
	}
	client := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: !req.SslVerify},
		},
		CheckRedirect: checkRedirect,
		Jar:           cookieJar,
	}
	// forceHttp2：限定 ALPN 仅 h2，避免协商到 HTTP/1.1
	if req.ForceHttp2 {
		client.Transport.(*http.Transport).TLSClientConfig.NextProtos = []string{"h2"}
	}

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
	// 解析 Set-Cookie 供前端 Cookie 管理器展示（Jar 已自动完成存储与回送）
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
