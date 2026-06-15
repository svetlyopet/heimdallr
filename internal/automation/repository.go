package automation

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context, limit int, offset int) ([]Automation, int64, error)
	FindById(ctx context.Context, automationId string) (Automation, error)
	FindByName(ctx context.Context, automationName string) (Automation, error)
	ExistsByName(ctx context.Context, automationName string) (bool, error)
	Create(ctx context.Context, automation Automation) (Automation, error)
	Update(ctx context.Context, automation Automation) (Automation, error)
	Delete(ctx context.Context, automationId string) error
}

type repository struct {
	db *gorm.DB
}

func (r repository) FindAll(ctx context.Context, limit int, offset int) ([]Automation, int64, error) {
	var automations []Automation
	var total int64

	query := r.db.WithContext(ctx).
		Table("automations").
		Select(`
			automations.id,
			automations.name,
			automations.url,
			automations.provider_id,
			providers.name AS provider,
			automations.cost_savings
		`).
		Joins("JOIN providers ON providers.id = automations.provider_id")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Find(&automations).Error; err != nil {
		return nil, 0, err
	}

	return automations, total, nil
}

func (r repository) FindById(ctx context.Context, automationId string) (Automation, error) {
	var automation Automation

	if err := r.db.WithContext(ctx).
		Table("automations").
		Select(`
			automations.id,
			automations.name,
			automations.url,
			automations.provider_id,
			providers.name AS provider,
			automations.cost_savings
		`).
		Joins("JOIN providers ON providers.id = automations.provider_id").
		Where("automations.id = ?", automationId).
		First(&automation).Error; err != nil {
		return Automation{}, err
	}

	return automation, nil
}

func (r repository) FindByName(ctx context.Context, automationName string) (Automation, error) {
	var automation Automation

	if err := r.db.WithContext(ctx).
		Table("automations").
		Select(`
			automations.id,
			automations.name,
			automations.url,
			automations.provider_id,
			providers.name AS provider,
			automations.cost_savings
		`).
		Joins("JOIN providers ON providers.id = automations.provider_id").
		Where("automations.name = ?", automationName).
		First(&automation).Error; err != nil {
		return Automation{}, err
	}

	return automation, nil
}

func (r repository) ExistsByName(ctx context.Context, automationName string) (bool, error) {
	var count int64

	if err := r.db.WithContext(ctx).
		Model(&Automation{}).
		Where("name = ?", automationName).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r repository) Create(ctx context.Context, automation Automation) (Automation, error) {
	if err := r.db.WithContext(ctx).Create(&automation).Error; err != nil {
		return Automation{}, err
	}

	return automation, nil
}

func (r repository) Update(ctx context.Context, automation Automation) (Automation, error) {
	result := r.db.WithContext(ctx).
		Model(&Automation{}).
		Where("id = ?", automation.ID).
		Select("name", "url", "provider", "provider_id", "cost_savings").
		Updates(automation)

	if result.Error != nil {
		return Automation{}, result.Error
	}

	if result.RowsAffected == 0 {
		return Automation{}, gorm.ErrRecordNotFound
	}

	return automation, nil
}

func (r repository) Delete(ctx context.Context, automationId string) error {
	if err := r.db.WithContext(ctx).
		Delete(&Automation{}, "id = ?", automationId).Error; err != nil {
		return err
	}

	return nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}
