package token

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	FindAll(ctx context.Context) ([]APIToken, error)
	FindByHash(ctx context.Context, tokenHash string) (APIToken, error)
	Create(ctx context.Context, token APIToken) (APIToken, error)
	DeleteByID(ctx context.Context, tokenID string) error
	DeleteByCreatedBy(ctx context.Context, userID string) error
	DeleteSessionTokensByCreatedBy(ctx context.Context, userID string) error
}

type repository struct {
	db *gorm.DB
}

func (r repository) FindAll(ctx context.Context) ([]APIToken, error) {
	var tokens []APIToken

	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&tokens).Error; err != nil {
		return nil, err
	}

	return tokens, nil
}

func (r repository) FindByHash(ctx context.Context, tokenHash string) (APIToken, error) {
	var token APIToken

	if err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&token).Error; err != nil {
		return APIToken{}, err
	}

	return token, nil
}

func (r repository) Create(ctx context.Context, token APIToken) (APIToken, error) {
	if err := r.db.WithContext(ctx).Create(&token).Error; err != nil {
		return APIToken{}, err
	}

	return token, nil
}

func (r repository) DeleteByID(ctx context.Context, tokenID string) error {
	result := r.db.WithContext(ctx).
		Delete(&APIToken{}, "id = ?", tokenID)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r repository) DeleteByCreatedBy(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Delete(&APIToken{}, "created_by = ?", userID).Error
}

func (r repository) DeleteSessionTokensByCreatedBy(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Delete(&APIToken{}, "created_by = ? AND kind = ?", userID, TokenKindSession).Error
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}
