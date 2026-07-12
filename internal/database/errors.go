package database

import (
	"errors"

	"gorm.io/gorm"
)

func IsUniqueViolation(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}
