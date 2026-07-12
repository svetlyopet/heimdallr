package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	Database fileDatabaseConfig `yaml:"database"`
	Logger   fileLoggerConfig   `yaml:"logger"`
	Server   fileServerConfig   `yaml:"server"`
	Auth     fileAuthConfig     `yaml:"auth"`
}

type fileDatabaseConfig struct {
	URL string `yaml:"url"`
}

type fileLoggerConfig struct {
	Format string `yaml:"format"`
	Level  string `yaml:"level"`
}

type fileServerConfig struct {
	Host                  string        `yaml:"host"`
	Port                  string        `yaml:"port"`
	ReadHeaderTimeout     time.Duration `yaml:"read_header_timeout"`
	ReadTimeout           time.Duration `yaml:"read_timeout"`
	WriteTimeout          time.Duration `yaml:"write_timeout"`
	IdleTimeout           time.Duration `yaml:"idle_timeout"`
	MaxHeaderBytes        int           `yaml:"max_header_bytes"`
	MaxRequestBodyBytes   int64         `yaml:"max_request_body_bytes"`
	MaxDecodedOutputBytes int64         `yaml:"max_decoded_output_bytes"`
	MaxPaginationLimit    int           `yaml:"max_pagination_limit"`
}

type fileAuthConfig struct {
	BootstrapRootPassword string        `yaml:"bootstrap_root_password"`
	LoginRateLimitMax     int           `yaml:"login_rate_limit_max"`
	LoginRateLimitWindow  time.Duration `yaml:"login_rate_limit_window"`
	LoginRateLimitMaxKeys int           `yaml:"login_rate_limit_max_keys"`
	APIRateLimitIPRate    float64       `yaml:"api_rate_limit_ip_rate"`
	APIRateLimitIPBurst   float64       `yaml:"api_rate_limit_ip_burst"`
	APIRateLimitUserRate  float64       `yaml:"api_rate_limit_user_rate"`
	APIRateLimitUserBurst float64       `yaml:"api_rate_limit_user_burst"`
	APIRateLimitMaxKeys   int           `yaml:"api_rate_limit_max_keys"`
	APIRateLimitStaleTTL  time.Duration `yaml:"api_rate_limit_stale_ttl"`
	SessionTokenTTL       time.Duration `yaml:"session_token_ttl"`
	DefaultAPITokenTTL    time.Duration `yaml:"api_token_default_ttl"`
	MaxAPITokenTTL        time.Duration `yaml:"api_token_max_ttl"`
	SessionCookieName     string        `yaml:"session_cookie_name"`
	CSRFCookieName        string        `yaml:"csrf_cookie_name"`
	CookieSecure          *bool         `yaml:"cookie_secure"`
}

func loadFileConfig(path string) (fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, fmt.Errorf("parse config file %q: %w", path, err)
	}

	return cfg, nil
}
