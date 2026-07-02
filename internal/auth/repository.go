package auth

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	FindByID(ctx context.Context, userID string) (User, error)
	FindByUsername(ctx context.Context, username string) (User, error)
	List(ctx context.Context) ([]User, error)
	Create(ctx context.Context, user User) (User, error)
	UpdateByID(ctx context.Context, userID string, user User) (User, error)
	DeleteByID(ctx context.Context, userID string) error
}

type repository struct {
	db *gorm.DB
}

func (r repository) FindByID(ctx context.Context, userID string) (User, error) {
	var user User

	if err := r.db.WithContext(ctx).
		Where("id = ?", userID).
		First(&user).Error; err != nil {
		return User{}, err
	}

	return user, nil
}

func (r repository) FindByUsername(ctx context.Context, username string) (User, error) {
	var user User

	if err := r.db.WithContext(ctx).
		Where("username = ?", username).
		First(&user).Error; err != nil {
		return User{}, err
	}

	return user, nil
}

func (r repository) Create(ctx context.Context, user User) (User, error) {
	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		return User{}, err
	}

	return user, nil
}

func (r repository) List(ctx context.Context) ([]User, error) {
	var users []User

	if err := r.db.WithContext(ctx).
		Order("username ASC").
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r repository) UpdateByID(ctx context.Context, userID string, user User) (User, error) {
	existing, err := r.FindByID(ctx, userID)
	if err != nil {
		return User{}, err
	}

	if user.Email != "" {
		existing.Email = user.Email
	}
	if user.PasswordHash != "" {
		existing.PasswordHash = user.PasswordHash
	}
	if user.Roles != nil {
		existing.Roles = user.Roles
	}

	if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return User{}, err
	}

	return existing, nil
}

func (r repository) DeleteByID(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).
		Delete(&User{}, "id = ?", userID)

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
