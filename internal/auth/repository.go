package auth

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"
)

type Repository interface {
	FindByID(ctx context.Context, userID string) (User, error)
	FindByUsername(ctx context.Context, username string) (User, error)
	List(ctx context.Context) ([]User, error)
	Create(ctx context.Context, user User) (User, error)
	UpdateByID(ctx context.Context, userID string, user User) (User, error)
	DeleteByID(ctx context.Context, userID string) error
	CountLegacyPasswordHashes(ctx context.Context) (int64, error)
	WithTx(tx *gorm.DB) Repository
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
	if user.PasswordHash != "" && !isBcryptHash(user.PasswordHash) {
		return User{}, ErrInvalidPasswordValue
	}

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

	updates := map[string]any{}

	if user.Email != "" {
		updates["email"] = user.Email
		existing.Email = user.Email
	}
	if user.PasswordHash != "" {
		if !isBcryptHash(user.PasswordHash) {
			return User{}, ErrInvalidPasswordValue
		}
		updates["password_hash"] = user.PasswordHash
		updates["password_reset_required"] = false
		existing.PasswordHash = user.PasswordHash
		existing.PasswordResetRequired = false
	}
	if user.Roles != nil {
		rolesJSON, marshalErr := json.Marshal(user.Roles)
		if marshalErr != nil {
			return User{}, marshalErr
		}
		updates["roles"] = string(rolesJSON)
		existing.Roles = user.Roles
	}

	if len(updates) == 0 {
		return existing, nil
	}

	updates["version"] = gorm.Expr("version + 1")

	result := r.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ? AND version = ?", userID, existing.Version).
		Updates(updates)
	if result.Error != nil {
		return User{}, result.Error
	}

	if result.RowsAffected == 0 {
		return User{}, ErrConcurrentUserUpdate
	}

	existing.Version++

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

func (r repository) WithTx(tx *gorm.DB) Repository {
	return &repository{db: tx}
}

func (r repository) CountLegacyPasswordHashes(ctx context.Context) (int64, error) {
	var count int64

	if err := r.db.WithContext(ctx).
		Model(&User{}).
		Where("password_reset_required = ?", true).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}
