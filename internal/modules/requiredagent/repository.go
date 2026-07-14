package requiredagent

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context, limit int, offset int) ([]RequiredAgent, int64, error)
	FindById(ctx context.Context, requiredAgentID string) (RequiredAgent, error)
	FindByName(ctx context.Context, agentName string) (RequiredAgent, error)
	Create(ctx context.Context, requiredAgent RequiredAgent) (RequiredAgent, error)
	Update(ctx context.Context, requiredAgent RequiredAgent) (RequiredAgent, error)
	Delete(ctx context.Context, requiredAgentID string) error
}

type repository struct {
	db *gorm.DB
}

func (r repository) FindAll(ctx context.Context, limit int, offset int) ([]RequiredAgent, int64, error) {
	var requiredAgents []RequiredAgent
	var total int64

	query := r.db.WithContext(ctx).Model(&RequiredAgent{}).Where("deleted_at IS NULL")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.Order("agent_name ASC")
	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}
	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Find(&requiredAgents).Error; err != nil {
		return nil, 0, err
	}

	return requiredAgents, total, nil
}

func (r repository) FindById(ctx context.Context, requiredAgentID string) (RequiredAgent, error) {
	var requiredAgent RequiredAgent

	if err := r.db.WithContext(ctx).
		Where("id = ?", requiredAgentID).
		First(&requiredAgent).Error; err != nil {
		return RequiredAgent{}, err
	}

	return requiredAgent, nil
}

func (r repository) FindByName(ctx context.Context, agentName string) (RequiredAgent, error) {
	var requiredAgent RequiredAgent

	if err := r.db.WithContext(ctx).
		Where("agent_name = ?", agentName).
		First(&requiredAgent).Error; err != nil {
		return RequiredAgent{}, err
	}

	return requiredAgent, nil
}

func (r repository) Create(ctx context.Context, requiredAgent RequiredAgent) (RequiredAgent, error) {
	if err := r.db.WithContext(ctx).Create(&requiredAgent).Error; err != nil {
		return RequiredAgent{}, err
	}

	return requiredAgent, nil
}

func (r repository) Update(ctx context.Context, requiredAgent RequiredAgent) (RequiredAgent, error) {
	if err := r.db.WithContext(ctx).Save(&requiredAgent).Error; err != nil {
		return RequiredAgent{}, err
	}

	return requiredAgent, nil
}

func (r repository) Delete(ctx context.Context, requiredAgentID string) error {
	result := r.db.WithContext(ctx).Where("id = ?", requiredAgentID).Delete(&RequiredAgent{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func normalizeAgentName(name string) string {
	return strings.TrimSpace(name)
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}
