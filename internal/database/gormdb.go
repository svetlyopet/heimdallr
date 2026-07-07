package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// GormDB wraps *gorm.DB and implements graceful shutdown for the DI container.
type GormDB struct {
	*gorm.DB
}

func (g *GormDB) Shutdown(ctx context.Context) error {
	if g == nil || g.DB == nil {
		return nil
	}

	sqlDB, err := g.DB.DB()
	if err != nil {
		return fmt.Errorf("get sql db for shutdown: %w", err)
	}

	return sqlDB.Close()
}

func OpenGormDB(cfg Config, migrator Migrator) (*GormDB, error) {
	db, err := Open(cfg, migrator)
	if err != nil {
		return nil, err
	}

	return &GormDB{DB: db}, nil
}
