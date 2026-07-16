package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 是服务的全局运行配置，从 config.yaml 与环境变量合并加载。
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	JWT      JWTConfig      `yaml:"jwt"`
	Database DatabaseConfig `yaml:"database"`
	Proxy    ProxyConfig    `yaml:"proxy"`
	CORS     CORSConfig     `yaml:"cors"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	PublicURL string `yaml:"public_url"`
}

type JWTConfig struct {
	Secret           string `yaml:"secret"`
	AccessTTLMinutes int    `yaml:"access_ttl_minutes"`
	RefreshTTLHours  int    `yaml:"refresh_ttl_hours"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type ProxyConfig struct {
	// MaxBodyBytes 单次响应体上限（字节），防止大响应撑爆内存。
	MaxBodyBytes int64 `yaml:"max_body_bytes"`
	// AllowPrivateTargets 允许代理/中继访问私有/内网地址（SSRF 防护开关，默认 false）。
	AllowPrivateTargets bool `yaml:"allow_private_targets"`
	// SkipTLSVerify 允许客户端主动跳过 TLS 证书校验（默认 false，即默认校验证书）。
	SkipTLSVerify bool `yaml:"skip_tls_verify"`
	// DefaultTimeoutMs 代理/请求默认超时（毫秒），避免无限挂起（默认 30000）。
	DefaultTimeoutMs int64 `yaml:"default_timeout_ms"`
	// RequireHTTPS 强制中继/WS 握手走 TLS，避免 token 经 query 在非加密信道泄露（默认 false）。
	RequireHTTPS bool `yaml:"require_https"`
	// MaxConns 中继（/ws/relay）并发连接上限，0 表示不限制（默认 512）。
	MaxConns int64 `yaml:"max_conns"`
}

type CORSConfig struct {
	AllowOrigins []string `yaml:"allow_origins"`
}

// Load 读取配置文件并叠加环境变量覆盖。
// 环境变量优先级高于配置文件，便于容器化部署时注入敏感信息。
func Load(path string) (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(path)
	if err == nil {
		// 配置文件缺失时退回默认值，不阻断启动
		if e := yaml.Unmarshal(data, cfg); e != nil {
			return nil, e
		}
	}
	applyDefaults(cfg)
	applyEnv(cfg)
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	// 注意：JWT 密钥不再提供内置默认值。若配置与环境变量均未提供，则保持为空，
	// 由 main 在启动时告警并拒绝用公开串签发票（H2）。
	if cfg.JWT.AccessTTLMinutes == 0 {
		cfg.JWT.AccessTTLMinutes = 15
	}
	if cfg.JWT.RefreshTTLHours == 0 {
		cfg.JWT.RefreshTTLHours = 168
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.DSN == "" {
		cfg.Database.DSN = "./data/apiforge.db"
	}
	if cfg.Proxy.MaxBodyBytes == 0 {
		cfg.Proxy.MaxBodyBytes = 10 << 20
	}
	if cfg.Proxy.DefaultTimeoutMs == 0 {
		cfg.Proxy.DefaultTimeoutMs = 30000
	}
	if cfg.Proxy.MaxConns == 0 {
		cfg.Proxy.MaxConns = 512
	}
}

// applyEnv 仅覆盖敏感或部署期必须可配的字段。
func applyEnv(cfg *Config) {
	if v := os.Getenv("APIFORGE_JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("APIFORGE_ALLOW_PRIVATE_TARGETS"); v == "true" || v == "1" {
		cfg.Proxy.AllowPrivateTargets = true
	}
	if v := os.Getenv("APIFORGE_SKIP_TLS_VERIFY"); v == "true" || v == "1" {
		cfg.Proxy.SkipTLSVerify = true
	}
	if v := os.Getenv("APIFORGE_REQUIRE_HTTPS"); v == "true" || v == "1" {
		cfg.Proxy.RequireHTTPS = true
	}
	if v := os.Getenv("DB_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
}
