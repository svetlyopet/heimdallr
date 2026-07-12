package release

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/modules/release/api"
	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context, applicationID string, limit int, offset int) ([]Release, int64, error)
	FindById(ctx context.Context, releaseID string, applicationID string) (Release, error)
	FindByApplicationAndVersion(ctx context.Context, applicationID uuid.UUID, version string) (Release, error)
	Create(ctx context.Context, release Release) (Release, error)
	Upsert(ctx context.Context, release Release) (Release, error)
	GetComplianceSummary(ctx context.Context, releaseID uuid.UUID) (api.ComplianceSummary, error)
	GetComplianceSummariesForReleases(ctx context.Context, releaseIDs []uuid.UUID) (map[uuid.UUID]api.ComplianceSummary, error)
}

type repository struct {
	db *gorm.DB
}

func (r repository) FindAll(ctx context.Context, applicationID string, limit int, offset int) ([]Release, int64, error) {
	var releases []Release
	var total int64

	query := r.db.WithContext(ctx).
		Model(&Release{}).
		Where("application_id = ?", applicationID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.Order("created_at DESC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Find(&releases).Error; err != nil {
		return nil, 0, err
	}

	return releases, total, nil
}

func (r repository) FindById(ctx context.Context, releaseID string, applicationID string) (Release, error) {
	var release Release

	if err := r.db.WithContext(ctx).
		Where("id = ? AND application_id = ?", releaseID, applicationID).
		First(&release).Error; err != nil {
		return Release{}, err
	}

	return release, nil
}

func (r repository) FindByApplicationAndVersion(ctx context.Context, applicationID uuid.UUID, version string) (Release, error) {
	var release Release

	if err := r.db.WithContext(ctx).
		Where("application_id = ? AND version = ?", applicationID, version).
		First(&release).Error; err != nil {
		return Release{}, err
	}

	return release, nil
}

func (r repository) Create(ctx context.Context, release Release) (Release, error) {
	if err := r.db.WithContext(ctx).Create(&release).Error; err != nil {
		return Release{}, err
	}

	return release, nil
}

func (r repository) Upsert(ctx context.Context, release Release) (Release, error) {
	release.Version = strings.TrimSpace(release.Version)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing Release
		findErr := tx.
			Where("application_id = ? AND version = ?", release.ApplicationID, release.Version).
			First(&existing).Error

		if findErr == nil {
			updates := map[string]any{}
			if release.CommitSHA != "" {
				updates["commit_sha"] = release.CommitSHA
			}
			if release.PipelineURL != "" {
				updates["pipeline_url"] = release.PipelineURL
			}
			if release.Branch != "" {
				updates["branch"] = release.Branch
			}

			if len(updates) > 0 {
				if err := tx.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
			}

			release = existing
			return nil
		}

		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		if err := tx.Create(&release).Error; err != nil {
			if !errors.Is(err, gorm.ErrDuplicatedKey) {
				return err
			}

			if err := tx.
				Where("application_id = ? AND version = ?", release.ApplicationID, release.Version).
				First(&existing).Error; err != nil {
				return err
			}

			updates := map[string]any{}
			if release.CommitSHA != "" {
				updates["commit_sha"] = release.CommitSHA
			}
			if release.PipelineURL != "" {
				updates["pipeline_url"] = release.PipelineURL
			}
			if release.Branch != "" {
				updates["branch"] = release.Branch
			}

			if len(updates) > 0 {
				if err := tx.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
			}

			release = existing
			return nil
		}

		return nil
	})

	if err != nil {
		return Release{}, err
	}

	return release, nil
}

func (r repository) GetComplianceSummary(ctx context.Context, releaseID uuid.UUID) (api.ComplianceSummary, error) {
	type row struct {
		Type   string
		Status string
		Count  int64
	}

	var rows []row
	if err := r.db.WithContext(ctx).
		Table("reports").
		Select("type, status, COUNT(*) AS count").
		Where("release_id = ?", releaseID).
		Group("type, status").
		Scan(&rows).Error; err != nil {
		return api.ComplianceSummary{}, err
	}

	summary := api.ComplianceSummary{}

	for _, row := range rows {
		summary.TotalReports += int(row.Count)

		switch row.Status {
		case "success":
			summary.SuccessfulReports += int(row.Count)
		case "failed":
			summary.FailedReports += int(row.Count)
		case "started":
			summary.StartedReports += int(row.Count)
		}

		summary.ByType = append(summary.ByType, api.ReportSummary{
			Type:   row.Type,
			Status: row.Status,
			Count:  int(row.Count),
		})
	}

	if summary.TotalReports > 0 {
		summary.SuccessRate = float64(summary.SuccessfulReports) / float64(summary.TotalReports) * 100
	}

	return summary, nil
}

func (r repository) GetComplianceSummariesForReleases(ctx context.Context, releaseIDs []uuid.UUID) (map[uuid.UUID]api.ComplianceSummary, error) {
	summaries := make(map[uuid.UUID]api.ComplianceSummary, len(releaseIDs))
	if len(releaseIDs) == 0 {
		return summaries, nil
	}

	type row struct {
		ReleaseID uuid.UUID
		Type      string
		Status    string
		Count     int64
	}

	var rows []row
	if err := r.db.WithContext(ctx).
		Table("reports").
		Select("release_id, type, status, COUNT(*) AS count").
		Where("release_id IN ?", releaseIDs).
		Group("release_id, type, status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		summary := summaries[row.ReleaseID]
		summary.TotalReports += int(row.Count)

		switch row.Status {
		case "success":
			summary.SuccessfulReports += int(row.Count)
		case "failed":
			summary.FailedReports += int(row.Count)
		case "started":
			summary.StartedReports += int(row.Count)
		}

		summary.ByType = append(summary.ByType, api.ReportSummary{
			Type:   row.Type,
			Status: row.Status,
			Count:  int(row.Count),
		})

		summaries[row.ReleaseID] = summary
	}

	for releaseID, summary := range summaries {
		if summary.TotalReports > 0 {
			summary.SuccessRate = float64(summary.SuccessfulReports) / float64(summary.TotalReports) * 100
		}

		summaries[releaseID] = summary
	}

	return summaries, nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}
