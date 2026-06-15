package database

import (
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/job"
	"github.com/svetlyopet/heimdallr/internal/provider"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewSQLiteDatabase(databasePath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err = db.AutoMigrate(
		&provider.Provider{},
		&automation.Automation{},
		&job.Job{},
	); err != nil {
		return nil, err
	}

	return db, nil
}
