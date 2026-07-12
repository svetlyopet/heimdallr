package token

import "time"

type ServiceConfig struct {
	SessionTokenTTL    time.Duration
	DefaultAPITokenTTL time.Duration
	MaxAPITokenTTL     time.Duration
}

func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		SessionTokenTTL:    24 * time.Hour,
		DefaultAPITokenTTL: 90 * 24 * time.Hour,
		MaxAPITokenTTL:     365 * 24 * time.Hour,
	}
}
