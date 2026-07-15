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
	MaxBodyBytes int64 `yaml:"max_body_bytes"`
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
	if cfg.JWT.Secret == "" {
		cfg.JWT.Secret = "dev-secret"
	}
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
}

// applyEnv 仅覆盖敏感或部署期必须可配的字段。
func applyEnv(cfg *Config) {
	if v := os.Getenv("APIFORGE_JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
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
