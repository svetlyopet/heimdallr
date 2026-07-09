package app

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/agent"
	"github.com/svetlyopet/heimdallr/internal/analytics"
	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/job"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/provider"
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

	authService auth.Service
	authHandler auth.Handler

	tokenService token.Service
	tokenHandler token.Handler

	serverService server.Service
	serverHandler server.Handler

	agentService agent.Service
	agentHandler agent.Handler
}

func (a *App) RegisterRoutes(rg *gin.RouterGroup) {
	provider.RegisterRoutes(rg, a.providerHandler)
	automation.RegisterRoutes(rg, a.automationHandler)
	job.RegisterRoutes(rg, a.jobHandler)
	application.RegisterRoutes(rg, a.applicationHandler)
	release.RegisterRoutes(rg, a.releaseHandler)
	report.RegisterRoutes(rg, a.reportHandler)
	analytics.RegisterRoutes(rg, a.analyticsHandler)
	server.RegisterRoutes(rg, a.serverHandler)
	agent.RegisterRoutes(rg, a.agentHandler)
	token.RegisterRoutes(rg, a.tokenHandler, a.authService)
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

	a.logger.Warn(ctx, "root user bootstrapped with generated credentials",
		slog.String("username", "root"),
		slog.String("password", rootPassword),
	)

	return nil
}
