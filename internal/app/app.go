package app

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/analytics"
	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/job"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/provider"
	"github.com/svetlyopet/heimdallr/internal/release"
	"github.com/svetlyopet/heimdallr/internal/report"
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
}

func New(db *gorm.DB, appLogger *logger.Logger) (*App, error) {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	providerRepository := provider.NewRepository(db)
	providerService := provider.NewService(providerRepository, appLogger)
	providerHandler, err := provider.NewHandler(providerService)
	if err != nil {
		return nil, err
	}

	authRepository := auth.NewRepository(db)
	authService := auth.NewService(authRepository, appLogger)
	authHandler, err := auth.NewHandler(authService)
	if err != nil {
		return nil, err
	}

	applicationRepository := application.NewRepository(db)
	applicationService := application.NewService(applicationRepository, appLogger)
	applicationHandler, err := application.NewHandler(applicationService)
	if err != nil {
		return nil, err
	}

	releaseRepository := release.NewRepository(db)
	releaseService := release.NewService(releaseRepository, applicationService, appLogger)
	releaseHandler, err := release.NewHandler(releaseService)
	if err != nil {
		return nil, err
	}

	reportRepository := report.NewRepository(db)
	reportService := report.NewService(reportRepository, releaseService, appLogger)
	reportHandler, err := report.NewHandler(reportService)
	if err != nil {
		return nil, err
	}

	automationRepository := automation.NewRepository(db)
	automationService := automation.NewService(automationRepository, providerService, appLogger)
	automationHandler, err := automation.NewHandler(automationService)
	if err != nil {
		return nil, err
	}

	jobRepository := job.NewRepository(db)
	jobService := job.NewService(jobRepository, automationService, appLogger)
	jobHandler, err := job.NewHandler(jobService)
	if err != nil {
		return nil, err
	}

	analyticsRepository := analytics.NewRepository(db)
	analyticsService := analytics.NewService(analyticsRepository, appLogger)
	analyticsHandler, err := analytics.NewHandler(analyticsService)
	if err != nil {
		return nil, err
	}

	return &App{
		db:     db,
		logger: appLogger,

		providerService: providerService,
		providerHandler: providerHandler,

		automationService: automationService,
		automationHandler: automationHandler,

		jobService: jobService,
		jobHandler: jobHandler,

		applicationService: applicationService,
		applicationHandler: applicationHandler,

		releaseService: releaseService,
		releaseHandler: releaseHandler,

		reportService: reportService,
		reportHandler: reportHandler,

		analyticsService: analyticsService,
		analyticsHandler: analyticsHandler,

		authService: authService,
		authHandler: authHandler,
	}, nil
}

func (a *App) RegisterRoutes(rg *gin.RouterGroup) {
	provider.RegisterRoutes(rg, a.providerHandler)
	automation.RegisterRoutes(rg, a.automationHandler)
	job.RegisterRoutes(rg, a.jobHandler)
	application.RegisterRoutes(rg, a.applicationHandler)
	release.RegisterRoutes(rg, a.releaseHandler)
	report.RegisterRoutes(rg, a.reportHandler)
	analytics.RegisterRoutes(rg, a.analyticsHandler)
	auth.RegisterRoutes(rg, a.authHandler, a.authService)
}

func (a *App) AuthService() auth.Service {
	return a.authService
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
