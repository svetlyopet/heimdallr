package sqlitemigrate

import (
	"fmt"

	"github.com/svetlyopet/heimdallr/internal/agent"
	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/job"
	"github.com/svetlyopet/heimdallr/internal/provider"
	"github.com/svetlyopet/heimdallr/internal/release"
	"github.com/svetlyopet/heimdallr/internal/report"
	"github.com/svetlyopet/heimdallr/internal/server"
	"github.com/svetlyopet/heimdallr/internal/token"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	if err := prepareTokenExpiration(db); err != nil {
		return err
	}

	if err := db.AutoMigrate(
		&auth.User{},
		&provider.Provider{},
		&automation.Automation{},
		&job.Job{},
		&application.Application{},
		&release.Release{},
		&report.Report{},
		&token.APIToken{},
		&server.Server{},
		&agent.Agent{},
		&agent.ServerAgent{},
		&server.ServerJob{},
		&server.ServerRelease{},
	); err != nil {
		return err
	}

	if err := db.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_automations_name ON automations (name) WHERE deleted_at IS NULL",
	).Error; err != nil {
		return fmt.Errorf("create sqlite automation name index: %w", err)
	}

	return nil
}

func prepareTokenExpiration(db *gorm.DB) error {
	migrator := db.Migrator()
	if !migrator.HasTable(&token.APIToken{}) {
		return nil
	}

	if !migrator.HasColumn(&token.APIToken{}, "Kind") {
		if err := db.Exec(
			"ALTER TABLE api_tokens ADD COLUMN kind VARCHAR(32) NOT NULL DEFAULT 'api'",
		).Error; err != nil {
			return fmt.Errorf("add sqlite api token kind: %w", err)
		}
	}
	if !migrator.HasColumn(&token.APIToken{}, "ExpiresAt") {
		if err := db.Exec(
			"ALTER TABLE api_tokens ADD COLUMN expires_at DATETIME NULL",
		).Error; err != nil {
			return fmt.Errorf("add sqlite api token expiration: %w", err)
		}
	}

	if err := db.Exec(
		"UPDATE api_tokens SET kind = 'session' WHERE name LIKE 'session-%' AND kind = 'api'",
	).Error; err != nil {
		return fmt.Errorf("classify sqlite session tokens: %w", err)
	}
	if err := db.Exec(`
		UPDATE api_tokens
		SET expires_at = CASE
			WHEN kind = 'session' THEN datetime('now', '+24 hours')
			ELSE datetime('now', '+90 days')
		END
		WHERE expires_at IS NULL
	`).Error; err != nil {
		return fmt.Errorf("backfill sqlite api token expiration: %w", err)
	}

	return nil
}
