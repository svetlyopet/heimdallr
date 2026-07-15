package job

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context, automationId string, limit int, offset int) ([]Job, int64, error)
	FindAllGlobal(ctx context.Context, filters ListFilters, limit int, offset int) ([]Job, int64, error)
	FindById(ctx context.Context, jobId string, automationId string) (Job, error)
	Create(ctx context.Context, job Job) (Job, error)
}

type repository struct {
	db *gorm.DB
}

type jobAutomationRelation struct {
	AutomationID uuid.UUID
	Automation   string
	ProviderID   uuid.UUID
	Provider     string
}

func (r repository) FindAll(ctx context.Context, automationId string, limit int, offset int) ([]Job, int64, error) {
	automationId = strings.TrimSpace(automationId)
	if automationId == "" {
		return nil, 0, ErrInvalidInput
	}

	var jobs []Job
	var total int64

	query := r.db.WithContext(ctx).
		Table("jobs").
		Joins("JOIN automations ON automations.id = jobs.automation_id AND automations.deleted_at IS NULL").
		Joins("JOIN providers ON providers.id = automations.provider_id AND providers.deleted_at IS NULL").
		Where("jobs.deleted_at IS NULL AND jobs.automation_id = ?", automationId)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.
		Select(`
			jobs.id,
			automations.name AS automation,
			jobs.automation_id,
			providers.name AS provider,
			providers.id AS provider_id,
			jobs.status,
			jobs.location,
			jobs.url,
			jobs.created_at,
			jobs.updated_at
		`).
		Order("jobs.created_at DESC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

func (r repository) FindAllGlobal(ctx context.Context, filters ListFilters, limit int, offset int) ([]Job, int64, error) {
	var jobs []Job
	var total int64

	query := r.db.WithContext(ctx).
		Table("jobs").
		Joins("JOIN automations ON automations.id = jobs.automation_id AND automations.deleted_at IS NULL").
		Joins("JOIN providers ON providers.id = automations.provider_id AND providers.deleted_at IS NULL").
		Where("jobs.deleted_at IS NULL")

	if filters.AutomationID != "" {
		query = query.Where("jobs.automation_id = ?", filters.AutomationID)
	}

	if filters.Status != "" {
		query = query.Where("jobs.status = ?", filters.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.
		Select(`
			jobs.id,
			automations.name AS automation,
			jobs.automation_id,
			providers.name AS provider,
			providers.id AS provider_id,
			jobs.status,
			jobs.location,
			jobs.url,
			jobs.created_at,
			jobs.updated_at
		`).
		Order("jobs.created_at DESC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

func (r repository) FindById(ctx context.Context, jobId string, automationId string) (Job, error) {
	return findJobById(ctx, r.db, jobId, automationId)
}

func (r repository) Create(ctx context.Context, job Job) (Job, error) {
	job.ID = strings.TrimSpace(job.ID)
	job.Status = strings.TrimSpace(job.Status)
	job.Location = strings.TrimSpace(job.Location)
	job.Url = strings.TrimSpace(job.Url)
	job.Output = strings.TrimSpace(job.Output)

	if job.ID == "" || job.AutomationID == uuid.Nil {
		return Job{}, ErrInvalidInput
	}

	var returnedJob Job

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		relation, err := findAutomationRelation(ctx, tx, job.AutomationID)
		if err != nil {
			return err
		}

		job.Automation = relation.Automation
		job.AutomationID = relation.AutomationID
		job.Provider = relation.Provider
		job.ProviderID = relation.ProviderID

		if err := tx.Create(&job).Error; err != nil {
			return err
		}

		createdJob, err := findJobById(ctx, tx, job.ID, job.AutomationID.String())
		if err != nil {
			return err
		}

		returnedJob = createdJob
		return nil
	})

	if err != nil {
		return Job{}, err
	}

	return returnedJob, nil
}

func findAutomationRelation(ctx context.Context, db *gorm.DB, automationID uuid.UUID) (jobAutomationRelation, error) {
	if automationID == uuid.Nil {
		return jobAutomationRelation{}, ErrInvalidInput
	}

	var relation jobAutomationRelation

	if err := db.WithContext(ctx).
		Table("automations").
		Select(`
			automations.id AS automation_id,
			automations.name AS automation,
			providers.id AS provider_id,
			providers.name AS provider
		`).
		Joins("JOIN providers ON providers.id = automations.provider_id AND providers.deleted_at IS NULL").
		Where("automations.deleted_at IS NULL AND automations.id = ?", automationID).
		Take(&relation).Error; err != nil {
		return jobAutomationRelation{}, err
	}

	if relation.AutomationID == uuid.Nil || relation.ProviderID == uuid.Nil {
		return jobAutomationRelation{}, gorm.ErrRecordNotFound
	}

	return relation, nil
}

func findJobById(ctx context.Context, db *gorm.DB, jobId string, automationId string) (Job, error) {
	jobId = strings.TrimSpace(jobId)
	automationId = strings.TrimSpace(automationId)

	if jobId == "" || automationId == "" {
		return Job{}, ErrInvalidInput
	}

	var job Job

	if err := db.WithContext(ctx).
		Table("jobs").
		Select(`
			jobs.id,
			automations.name AS automation,
			jobs.automation_id,
			providers.name AS provider,
			providers.id AS provider_id,
			jobs.status,
			jobs.location,
			jobs.url,
			jobs.created_at,
			jobs.updated_at,
			jobs.metadata,
			jobs.output
		`).
		Joins("JOIN automations ON automations.id = jobs.automation_id AND automations.deleted_at IS NULL").
		Joins("JOIN providers ON providers.id = automations.provider_id AND providers.deleted_at IS NULL").
		Where("jobs.deleted_at IS NULL AND jobs.id = ? AND jobs.automation_id = ?", jobId, automationId).
		Take(&job).Error; err != nil {
		return Job{}, err
	}

	return job, nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}
