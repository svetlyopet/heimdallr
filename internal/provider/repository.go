package provider

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context, limit int, offset int) ([]Provider, int64, error)
	FindById(ctx context.Context, providerId string) (Provider, error)
	FindByName(ctx context.Context, providerName string) (Provider, error)
	FindByIdWithAutomations(ctx context.Context, providerId string) (Provider, error)
	Create(ctx context.Context, provider Provider) (Provider, error)
}

type repository struct {
	db *gorm.DB
}

func (r repository) FindAll(ctx context.Context, limit int, offset int) ([]Provider, int64, error) {
	var providers []Provider
	var total int64

	query := r.db.WithContext(ctx).
		Model(&Provider{})

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

	if err := findQuery.Find(&providers).Error; err != nil {
		return nil, 0, err
	}

	return providers, total, nil
}

func (r repository) FindById(ctx context.Context, providerId string) (Provider, error) {
	var provider Provider

	if err := r.db.WithContext(ctx).
		Where("id = ?", providerId).
		First(&provider).Error; err != nil {
		return Provider{}, err
	}

	return provider, nil
}

func (r repository) FindByName(ctx context.Context, providerName string) (Provider, error) {
	var provider Provider

	if err := r.db.WithContext(ctx).
		Where("name = ?", providerName).
		First(&provider).Error; err != nil {
		return Provider{}, err
	}

	return provider, nil
}

func (r repository) FindByIdWithAutomations(ctx context.Context, providerId string) (Provider, error) {
	var provider Provider

	if err := r.db.WithContext(ctx).
		Preload("Automations").
		Where("id = ?", providerId).
		First(&provider).Error; err != nil {
		return Provider{}, err
	}

	return provider, nil
}

func (r repository) Create(ctx context.Context, provider Provider) (Provider, error) {
	if err := r.db.WithContext(ctx).Create(&provider).Error; err != nil {
		return Provider{}, err
	}

	return provider, nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}
