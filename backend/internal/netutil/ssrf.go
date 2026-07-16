// Package netutil 提供出站网络目标的安全校验，防止 SSRF。
package netutil

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// ValidateOutboundURL 校验出站 URL 是否允许被后端代理/中继访问。
// allowPrivate 为 true 时放行私有/内网地址（仅限受信任的内网调试场景）。
// 默认禁止回环、链路本地、私有网段与云元数据地址，防止 SSRF 访问内网或云元数据。
func ValidateOutboundURL(raw string, allowPrivate bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	switch u.Scheme {
	case "http", "https", "ws", "wss":
	default:
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
	return validateHost(u.Hostname(), allowPrivate)
}

// ValidateHostPort 校验 tcp/udp 类出站目标的 host:port。
func ValidateHostPort(host, port string, allowPrivate bool) error {
	if host == "" {
		return fmt.Errorf("empty host")
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("invalid port: %s", port)
	}
	return validateHost(host, allowPrivate)
}

// validateHost 解析主机名为 IP 并逐一对每个 IP 做私有/受限判定。
// 解析失败按失败关闭处理，避免 DNS 重绑定绕过。
func validateHost(host string, allowPrivate bool) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("empty host")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("host %q resolved to no address", host)
	}
	for _, ip := range ips {
		if !allowPrivate && isRestricted(ip) {
			return fmt.Errorf("target %s (%s) is blocked: private or restricted network", host, ip.String())
		}
	}
	return nil
}

// isRestricted 判定 IP 是否属于回环/链路本地/私有网段或云元数据地址。
func isRestricted(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if ip.String() == "169.254.169.254" {
		return true
	}
	return false
}
