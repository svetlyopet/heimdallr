package config

import (
	"os"

	"github.com/samber/do/v2"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/sqlitemigrate"
	"github.com/svetlyopet/heimdallr/internal/token"
	"gorm.io/gorm"
)

var InfrastructurePackage = do.Package(
	do.Lazy(provideAuthServiceConfig),
	do.Lazy(provideTokenServiceConfig),
	do.Lazy(provideLogger),
	do.Lazy(provideGormDB),
	do.Lazy(provideDB),
)

var Package = do.Package(
	do.Lazy(provideConfig),
	do.Lazy(provideAuthServiceConfig),
	do.Lazy(provideTokenServiceConfig),
	do.Lazy(provideLogger),
	do.Lazy(provideGormDB),
	do.Lazy(provideDB),
)

func provideConfig(i do.Injector) (*AppConfig, error) {
	cfg, err := LoadFromFlags(os.Args[1:], os.Getenv)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func provideAuthServiceConfig(i do.Injector) (auth.ServiceConfig, error) {
	cfg := do.MustInvoke[*AppConfig](i)
	return auth.ServiceConfig{BootstrapRootPassword: cfg.Auth.BootstrapRootPassword}, nil
}

func provideTokenServiceConfig(i do.Injector) (token.ServiceConfig, error) {
	cfg := do.MustInvoke[*AppConfig](i)
	return token.ServiceConfig{
		SessionTokenTTL:    cfg.Auth.SessionTokenTTL,
		DefaultAPITokenTTL: cfg.Auth.DefaultAPITokenTTL,
		MaxAPITokenTTL:     cfg.Auth.MaxAPITokenTTL,
	}, nil
}

func provideLogger(i do.Injector) (*logger.Logger, error) {
	cfg := do.MustInvoke[*AppConfig](i)
	return logger.New(cfg.Logger), nil
}

func provideGormDB(i do.Injector) (*database.GormDB, error) {
	cfg := do.MustInvoke[*AppConfig](i)
	return database.OpenGormDB(cfg.Database, database.NewMigrator(sqlitemigrate.AutoMigrate))
}

func provideDB(i do.Injector) (*gorm.DB, error) {
	return do.MustInvoke[*database.GormDB](i).DB, nil
}
