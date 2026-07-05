package report

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context, releaseID string, applicationID string, limit int, offset int) ([]Report, int64, error)
	FindAllGlobal(ctx context.Context, filters ListFilters, limit int, offset int) ([]Report, int64, error)
	FindById(ctx context.Context, reportID string, releaseID string, applicationID string) (Report, error)
	Create(ctx context.Context, report Report) (Report, error)
	Update(ctx context.Context, report Report) (Report, error)
}

type repository struct {
	db *gorm.DB
}

type releaseRelation struct {
	ReleaseID     uuid.UUID
	ApplicationID uuid.UUID
	Application   string
	Version       string
}

func (r repository) FindAll(ctx context.Context, releaseID string, applicationID string, limit int, offset int) ([]Report, int64, error) {
	var reports []Report
	var total int64

	query := r.db.WithContext(ctx).
		Table("reports").
		Where("release_id = ? AND application_id = ?", releaseID, applicationID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.
		Select(`
			reports.id,
			reports.release_id,
			reports.application_id,
			reports.application,
			reports.version,
			reports.type,
			reports.status,
			reports.location,
			reports.url,
			reports.created_at,
			reports.updated_at
		`).
		Order("reports.created_at DESC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Find(&reports).Error; err != nil {
		return nil, 0, err
	}

	return reports, total, nil
}

func (r repository) FindAllGlobal(ctx context.Context, filters ListFilters, limit int, offset int) ([]Report, int64, error) {
	var reports []Report
	var total int64

	query := r.db.WithContext(ctx).Table("reports")

	if filters.ApplicationID != "" {
		query = query.Where("application_id = ?", filters.ApplicationID)
	}

	if filters.ReleaseID != "" {
		query = query.Where("release_id = ?", filters.ReleaseID)
	}

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.
		Select(`
			reports.id,
			reports.release_id,
			reports.application_id,
			reports.application,
			reports.version,
			reports.type,
			reports.status,
			reports.location,
			reports.url,
			reports.created_at,
			reports.updated_at
		`).
		Order("reports.created_at DESC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Find(&reports).Error; err != nil {
		return nil, 0, err
	}

	return reports, total, nil
}

func (r repository) FindById(ctx context.Context, reportID string, releaseID string, applicationID string) (Report, error) {
	return findReportById(ctx, r.db, reportID, releaseID, applicationID)
}

func (r repository) Create(ctx context.Context, report Report) (Report, error) {
	report.ID = strings.TrimSpace(report.ID)
	report.Type = strings.TrimSpace(report.Type)
	report.Status = strings.TrimSpace(report.Status)
	report.Location = strings.TrimSpace(report.Location)
	report.URL = strings.TrimSpace(report.URL)
	report.Output = strings.TrimSpace(report.Output)

	if report.ID == "" || report.ReleaseID == uuid.Nil {
		return Report{}, gorm.ErrRecordNotFound
	}

	var returned Report

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		relation, err := findReleaseRelation(ctx, tx, report.ReleaseID, report.ApplicationID)
		if err != nil {
			return err
		}

		report.ReleaseID = relation.ReleaseID
		report.ApplicationID = relation.ApplicationID
		report.Application = relation.Application
		report.Version = relation.Version

		if err := tx.Create(&report).Error; err != nil {
			return err
		}

		created, err := findReportById(ctx, tx, report.ID, report.ReleaseID.String(), report.ApplicationID.String())
		if err != nil {
			return err
		}

		returned = created
		return nil
	})

	if err != nil {
		return Report{}, err
	}

	return returned, nil
}

func (r repository) Update(ctx context.Context, report Report) (Report, error) {
	report.ID = strings.TrimSpace(report.ID)
	report.Status = strings.TrimSpace(report.Status)
	report.Output = strings.TrimSpace(report.Output)

	if report.ID == "" || report.ReleaseID == uuid.Nil {
		return Report{}, gorm.ErrRecordNotFound
	}

	var returned Report

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Model(&Report{}).
			Where("id = ? AND release_id = ? AND application_id = ?", report.ID, report.ReleaseID, report.ApplicationID).
			Select("status", "metadata", "output").
			Updates(Report{
				Status:   report.Status,
				Metadata: report.Metadata,
				Output:   report.Output,
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		updated, err := findReportById(ctx, tx, report.ID, report.ReleaseID.String(), report.ApplicationID.String())
		if err != nil {
			return err
		}

		returned = updated
		return nil
	})

	if err != nil {
		return Report{}, err
	}

	return returned, nil
}

func findReleaseRelation(ctx context.Context, db *gorm.DB, releaseID uuid.UUID, applicationID uuid.UUID) (releaseRelation, error) {
	if releaseID == uuid.Nil || applicationID == uuid.Nil {
		return releaseRelation{}, gorm.ErrRecordNotFound
	}

	var relation releaseRelation

	if err := db.WithContext(ctx).
		Table("releases").
		Select(`
			releases.id AS release_id,
			releases.application_id AS application_id,
			releases.application AS application,
			releases.version AS version
		`).
		Where("releases.id = ? AND releases.application_id = ?", releaseID, applicationID).
		Take(&relation).Error; err != nil {
		return releaseRelation{}, err
	}

	return relation, nil
}

func findReportById(ctx context.Context, db *gorm.DB, reportID string, releaseID string, applicationID string) (Report, error) {
	reportID = strings.TrimSpace(reportID)
	releaseID = strings.TrimSpace(releaseID)
	applicationID = strings.TrimSpace(applicationID)

	if reportID == "" || releaseID == "" || applicationID == "" {
		return Report{}, gorm.ErrRecordNotFound
	}

	var report Report

	if err := db.WithContext(ctx).
		Table("reports").
		Where("id = ? AND release_id = ? AND application_id = ?", reportID, releaseID, applicationID).
		Take(&report).Error; err != nil {
		return Report{}, err
	}

	return report, nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}
