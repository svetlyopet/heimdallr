package app

import (
	"github.com/samber/do/v2"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/config"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/modules/agent"
	"github.com/svetlyopet/heimdallr/internal/modules/analytics"
	"github.com/svetlyopet/heimdallr/internal/modules/application"
	"github.com/svetlyopet/heimdallr/internal/modules/automation"
	"github.com/svetlyopet/heimdallr/internal/modules/job"
	"github.com/svetlyopet/heimdallr/internal/modules/provider"
	"github.com/svetlyopet/heimdallr/internal/modules/release"
	"github.com/svetlyopet/heimdallr/internal/modules/report"
	"github.com/svetlyopet/heimdallr/internal/modules/server"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/token"
	"gorm.io/gorm"
)

var Package = do.Package(
	rbac.Package,
	provider.Package,
	auth.Package,
	token.Package,
	application.Package,
	release.Package,
	report.Package,
	automation.Package,
	job.Package,
	analytics.Package,
	agent.Package,
	server.Package,
	do.Lazy(provideLoginRateLimiter),
	do.Lazy(provideApp),
)

func provideLoginRateLimiter(i do.Injector) (*auth.LoginRateLimiter, error) {
	cfg := do.MustInvoke[*config.AppConfig](i)
	return auth.NewLoginRateLimiter(
		cfg.Auth.LoginRateLimitMax,
		cfg.Auth.LoginRateLimitWindow,
		cfg.Auth.LoginRateLimitMaxKeys,
	), nil
}

func provideApp(i do.Injector) (*App, error) {
	if err := ValidatePolicies(); err != nil {
		return nil, err
	}

	cfg := do.MustInvoke[*config.AppConfig](i)

	return &App{
		db:     do.MustInvoke[*gorm.DB](i),
		logger: do.MustInvoke[*logger.Logger](i),

		providerService: do.MustInvoke[provider.Service](i),
		providerHandler: do.MustInvoke[provider.Handler](i),

		automationService: do.MustInvoke[automation.Service](i),
		automationHandler: do.MustInvoke[automation.Handler](i),

		jobService: do.MustInvoke[job.Service](i),
		jobHandler: do.MustInvoke[job.Handler](i),

		applicationService: do.MustInvoke[application.Service](i),
		applicationHandler: do.MustInvoke[application.Handler](i),

		releaseService: do.MustInvoke[release.Service](i),
		releaseHandler: do.MustInvoke[release.Handler](i),

		reportService: do.MustInvoke[report.Service](i),
		reportHandler: do.MustInvoke[report.Handler](i),

		analyticsService: do.MustInvoke[analytics.Service](i),
		analyticsHandler: do.MustInvoke[analytics.Handler](i),

		authService:      do.MustInvoke[auth.Service](i),
		authHandler:      do.MustInvoke[auth.Handler](i),
		loginRateLimiter: do.MustInvoke[*auth.LoginRateLimiter](i),
		authCookieConfig: auth.CookieConfig{
			SessionCookieName: cfg.Auth.SessionCookieName,
			CSRFCookieName:    cfg.Auth.CSRFCookieName,
			Secure:            cfg.Auth.CookieSecure,
			SessionTTL:        cfg.Auth.SessionTokenTTL,
		},
		authorizer: do.MustInvoke[rbac.Authorizer](i),

		tokenService: do.MustInvoke[token.Service](i),
		tokenHandler: do.MustInvoke[token.Handler](i),

		serverService: do.MustInvoke[server.Service](i),
		serverHandler: do.MustInvoke[server.Handler](i),

		agentService: do.MustInvoke[agent.Service](i),
		agentHandler: do.MustInvoke[agent.Handler](i),
	}, nil
}
