package model

import (
	"time"

	"gorm.io/gorm"
)

type Timestamp struct {
	CreatedAt time.Time      `gorm:"type:datetime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:datetime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
