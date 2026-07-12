package auth

import (
	"strings"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"gorm.io/gorm"
)

const (
	RoleAdmin  = "admin"
	RoleReader = "reader"
)

var defaultSupportedRoles = map[string]struct{}{
	RoleAdmin:  {},
	RoleReader: {},
}

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Username     string    `gorm:"type:varchar(255);uniqueIndex;not null;check:username <> ''" json:"username"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null;check:email <> ''" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null;check:password_hash <> ''" json:"-"`
	Roles        []string  `gorm:"serializer:json;type:text;not null" json:"roles"`

	model.Timestamp
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}

	return nil
}

func normalizeRoles(roles []string) []string {
	normalized := make([]string, 0, len(roles))
	seen := map[string]struct{}{}

	for _, role := range roles {
		role = strings.TrimSpace(strings.ToLower(role))
		if role == "" {
			continue
		}

		if _, ok := seen[role]; ok {
			continue
		}

		seen[role] = struct{}{}
		normalized = append(normalized, role)
	}

	return normalized
}
