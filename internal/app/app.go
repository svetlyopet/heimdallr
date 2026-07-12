package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/agent"
	"github.com/svetlyopet/heimdallr/internal/analytics"
	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/job"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/provider"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/release"
	"github.com/svetlyopet/heimdallr/internal/report"
	"github.com/svetlyopet/heimdallr/internal/server"
	"github.com/svetlyopet/heimdallr/internal/token"
	"gorm.io/gorm"
)

type App struct {
	db     *gorm.DB
	logger *logger.Logger

	providerService provider.Service
	providerHandler provider.Handler

	automationService automation.Service
	automationHandler automation.Handler

	jobService job.Service
	jobHandler job.Handler

	applicationService application.Service
	applicationHandler application.Handler

	releaseService release.Service
	releaseHandler release.Handler

	reportService report.Service
	reportHandler report.Handler

	analyticsService analytics.Service
	analyticsHandler analytics.Handler

	authService      auth.Service
	authHandler      auth.Handler
	loginRateLimiter *auth.LoginRateLimiter
	authCookieConfig auth.CookieConfig
	authorizer       rbac.Authorizer

	tokenService token.Service
	tokenHandler token.Handler

	serverService server.Service
	serverHandler server.Handler

	agentService agent.Service
	agentHandler agent.Handler
}

func (a *App) RegisterRoutes(rg *gin.RouterGroup) {
	provider.RegisterRoutes(rg, a.providerHandler, a.authorizer, a.logger)
	automation.RegisterRoutes(rg, a.automationHandler, a.authorizer, a.logger)
	job.RegisterRoutes(rg, a.jobHandler, a.authorizer, a.logger)
	application.RegisterRoutes(rg, a.applicationHandler, a.authorizer, a.logger)
	release.RegisterRoutes(rg, a.releaseHandler, a.authorizer, a.logger)
	report.RegisterRoutes(rg, a.reportHandler, a.authorizer, a.logger)
	analytics.RegisterRoutes(rg, a.analyticsHandler, a.authorizer, a.logger)
	server.RegisterRoutes(rg, a.serverHandler, a.authorizer, a.logger)
	agent.RegisterRoutes(rg, a.agentHandler, a.authorizer, a.logger)
	token.RegisterRoutes(rg, a.tokenHandler, a.authorizer, a.logger)
}

func (a *App) RegisterPublicRoutes(rg *gin.RouterGroup) {
	auth.RegisterPublicRoutes(rg, a.authHandler, a.loginRateLimiter, a.authCookieConfig)
}

func (a *App) RegisterProtectedAuthRoutes(rg *gin.RouterGroup) {
	auth.RegisterProtectedRoutes(rg, a.authHandler, a.authorizer, a.logger, a.authCookieConfig)
}

func (a *App) AuthService() auth.Service {
	return a.authService
}

func (a *App) TokenService() token.Service {
	return a.tokenService
}

func (a *App) Bootstrap(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	rootPassword, err := a.authService.EnsureRootUser(ctx)
	if err != nil {
		return err
	}

	if rootPassword == "" {
		a.logger.Info(ctx, "root user bootstrap skipped; root user already exists")
		return nil
	}

	a.logger.Warn(ctx, "root user created on first bootstrap; retrieve credentials from HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD or first-run stderr output",
		slog.String("username", "root"),
	)

	if os.Getenv("HEIMDALLR_BOOTSTRAP_ROOT_PASSWORD") == "" {
		_, _ = fmt.Fprintf(os.Stderr, "\n=== Heimdallr root user bootstrap ===\nusername: root\npassword: %s\n===================================\n\n", rootPassword)
	}

	return nil
}
