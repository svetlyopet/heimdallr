package server

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context, limit int, offset int) ([]ServerWithCounts, int64, error)
	FindById(ctx context.Context, serverID string) (Server, error)
	FindByHostname(ctx context.Context, hostname string) (Server, error)
	Create(ctx context.Context, server Server) (Server, error)
	GetRelationCounts(ctx context.Context, serverID uuid.UUID) (RelationSummary, error)

	FindAssociatedJobs(ctx context.Context, serverID string, limit int, offset int) ([]JobAssociationRow, int64, error)
	JobExists(ctx context.Context, jobID string, automationID uuid.UUID) (bool, error)
	JobAssociationExists(ctx context.Context, serverID uuid.UUID, jobID string, automationID uuid.UUID) (bool, error)
	CreateJobAssociation(ctx context.Context, association ServerJob) error
	DeleteJobAssociation(ctx context.Context, serverID uuid.UUID, jobID string, automationID uuid.UUID) error

	FindAssociatedReleases(ctx context.Context, serverID string, limit int, offset int) ([]ReleaseAssociationRow, int64, error)
	ReleaseExists(ctx context.Context, releaseID uuid.UUID, applicationID uuid.UUID) (bool, error)
	ReleaseAssociationExists(ctx context.Context, serverID uuid.UUID, releaseID uuid.UUID) (bool, error)
	CreateReleaseAssociation(ctx context.Context, association ServerRelease) error
	DeleteReleaseAssociation(ctx context.Context, serverID uuid.UUID, releaseID uuid.UUID) error
}

type ServerWithCounts struct {
	Server
	AgentCount   int64
	JobCount     int64
	ReleaseCount int64
}

type JobAssociationRow struct {
	JobID        string
	AutomationID uuid.UUID
	Automation   string
	Provider     string
	Status       string
	Location     string
	URL          string
}

type ReleaseAssociationRow struct {
	ReleaseID     uuid.UUID
	ApplicationID uuid.UUID
	Application   string
	Version       string
	CommitSHA     string
	Branch        string
}

type repository struct {
	db *gorm.DB
}

func (r repository) FindAll(ctx context.Context, limit int, offset int) ([]ServerWithCounts, int64, error) {
	var servers []ServerWithCounts
	var total int64

	query := r.db.WithContext(ctx).Model(&Server{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := r.db.WithContext(ctx).
		Table("servers").
		Select(`
			servers.*,
			(SELECT COUNT(*) FROM agents WHERE agents.server_id = servers.id AND agents.deleted_at IS NULL) AS agent_count,
			(SELECT COUNT(*) FROM server_jobs WHERE server_jobs.server_id = servers.id) AS job_count,
			(SELECT COUNT(*) FROM server_releases WHERE server_releases.server_id = servers.id) AS release_count
		`).
		Where("servers.deleted_at IS NULL").
		Order("servers.hostname ASC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Scan(&servers).Error; err != nil {
		return nil, 0, err
	}

	return servers, total, nil
}

func (r repository) FindById(ctx context.Context, serverID string) (Server, error) {
	var server Server

	if err := r.db.WithContext(ctx).
		Where("id = ?", serverID).
		First(&server).Error; err != nil {
		return Server{}, err
	}

	return server, nil
}

func (r repository) FindByHostname(ctx context.Context, hostname string) (Server, error) {
	var server Server

	if err := r.db.WithContext(ctx).
		Where("hostname = ?", hostname).
		First(&server).Error; err != nil {
		return Server{}, err
	}

	return server, nil
}

func (r repository) Create(ctx context.Context, server Server) (Server, error) {
	if err := r.db.WithContext(ctx).Create(&server).Error; err != nil {
		return Server{}, err
	}

	return server, nil
}

func (r repository) GetRelationCounts(ctx context.Context, serverID uuid.UUID) (RelationSummary, error) {
	var summary RelationSummary

	if err := r.db.WithContext(ctx).
		Table("agents").
		Where("server_id = ? AND deleted_at IS NULL", serverID).
		Count(&summary.AgentCount).Error; err != nil {
		return RelationSummary{}, err
	}

	if err := r.db.WithContext(ctx).
		Model(&ServerJob{}).
		Where("server_id = ?", serverID).
		Count(&summary.JobCount).Error; err != nil {
		return RelationSummary{}, err
	}

	if err := r.db.WithContext(ctx).
		Model(&ServerRelease{}).
		Where("server_id = ?", serverID).
		Count(&summary.ReleaseCount).Error; err != nil {
		return RelationSummary{}, err
	}

	return summary, nil
}

func (r repository) FindAssociatedJobs(ctx context.Context, serverID string, limit int, offset int) ([]JobAssociationRow, int64, error) {
	var rows []JobAssociationRow
	var total int64

	query := r.db.WithContext(ctx).
		Table("server_jobs").
		Joins("JOIN jobs ON jobs.id = server_jobs.job_id AND jobs.automation_id = server_jobs.automation_id").
		Joins("JOIN automations ON automations.id = jobs.automation_id").
		Joins("JOIN providers ON providers.id = automations.provider_id").
		Where("server_jobs.server_id = ?", serverID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.
		Select(`
			server_jobs.job_id,
			server_jobs.automation_id,
			automations.name AS automation,
			providers.name AS provider,
			jobs.status,
			jobs.location,
			jobs.url
		`).
		Order("jobs.created_at DESC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r repository) JobExists(ctx context.Context, jobID string, automationID uuid.UUID) (bool, error) {
	var count int64

	if err := r.db.WithContext(ctx).
		Table("jobs").
		Where("id = ? AND automation_id = ?", jobID, automationID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r repository) JobAssociationExists(ctx context.Context, serverID uuid.UUID, jobID string, automationID uuid.UUID) (bool, error) {
	var count int64

	if err := r.db.WithContext(ctx).
		Model(&ServerJob{}).
		Where("server_id = ? AND job_id = ? AND automation_id = ?", serverID, jobID, automationID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r repository) CreateJobAssociation(ctx context.Context, association ServerJob) error {
	return r.db.WithContext(ctx).Create(&association).Error
}

func (r repository) DeleteJobAssociation(ctx context.Context, serverID uuid.UUID, jobID string, automationID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("server_id = ? AND job_id = ? AND automation_id = ?", serverID, jobID, automationID).
		Delete(&ServerJob{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r repository) FindAssociatedReleases(ctx context.Context, serverID string, limit int, offset int) ([]ReleaseAssociationRow, int64, error) {
	var rows []ReleaseAssociationRow
	var total int64

	query := r.db.WithContext(ctx).
		Table("server_releases").
		Joins("JOIN releases ON releases.id = server_releases.release_id").
		Where("server_releases.server_id = ?", serverID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.
		Select(`
			server_releases.release_id,
			server_releases.application_id,
			releases.application,
			releases.version,
			releases.commit_sha,
			releases.branch
		`).
		Order("releases.created_at DESC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r repository) ReleaseExists(ctx context.Context, releaseID uuid.UUID, applicationID uuid.UUID) (bool, error) {
	var count int64

	if err := r.db.WithContext(ctx).
		Table("releases").
		Where("id = ? AND application_id = ?", releaseID, applicationID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r repository) ReleaseAssociationExists(ctx context.Context, serverID uuid.UUID, releaseID uuid.UUID) (bool, error) {
	var count int64

	if err := r.db.WithContext(ctx).
		Model(&ServerRelease{}).
		Where("server_id = ? AND release_id = ?", serverID, releaseID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r repository) CreateReleaseAssociation(ctx context.Context, association ServerRelease) error {
	return r.db.WithContext(ctx).Create(&association).Error
}

func (r repository) DeleteReleaseAssociation(ctx context.Context, serverID uuid.UUID, releaseID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("server_id = ? AND release_id = ?", serverID, releaseID).
		Delete(&ServerRelease{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, gorm.ErrDuplicatedKey)
}
