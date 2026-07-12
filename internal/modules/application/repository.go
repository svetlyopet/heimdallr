package application

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context, limit int, offset int) ([]Application, int64, error)
	FindById(ctx context.Context, applicationID string) (Application, error)
	FindByName(ctx context.Context, name string) (Application, error)
	Create(ctx context.Context, application Application) (Application, error)
}

type repository struct {
	db *gorm.DB
}

func (r repository) FindAll(ctx context.Context, limit int, offset int) ([]Application, int64, error) {
	var applications []Application
	var total int64

	query := r.db.WithContext(ctx).Model(&Application{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	findQuery := query.Order("name ASC")

	if limit > 0 {
		findQuery = findQuery.Limit(limit)
	}

	if offset > 0 {
		findQuery = findQuery.Offset(offset)
	}

	if err := findQuery.Find(&applications).Error; err != nil {
		return nil, 0, err
	}

	return applications, total, nil
}

func (r repository) FindById(ctx context.Context, applicationID string) (Application, error) {
	var application Application

	if err := r.db.WithContext(ctx).
		Where("id = ?", applicationID).
		First(&application).Error; err != nil {
		return Application{}, err
	}

	return application, nil
}

func (r repository) FindByName(ctx context.Context, name string) (Application, error) {
	var application Application

	if err := r.db.WithContext(ctx).
		Where("name = ?", name).
		First(&application).Error; err != nil {
		return Application{}, err
	}

	return application, nil
}

func (r repository) Create(ctx context.Context, application Application) (Application, error) {
	if err := r.db.WithContext(ctx).Create(&application).Error; err != nil {
		return Application{}, err
	}

	return application, nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}
