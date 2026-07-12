package token

import "time"

type ServiceConfig struct {
	SessionTokenTTL time.Duration
}

func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		SessionTokenTTL: 24 * time.Hour,
	}
}
