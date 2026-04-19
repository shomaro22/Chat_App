package entity

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Name              string     `gorm:"not null"`
	Email             string     `gorm:"not null"`
	PasswordHash      string     `gorm:"column:password_hash;not null"`
	Role              UserRole   `gorm:"not null"`
	Active            bool       `gorm:"not null"`
	AvatarURL         *string    `gorm:"column:avatar_url"`
	EmailVerifiedAt   *time.Time `gorm:"column:email_verified_at"`
	CreatedAt         time.Time  `gorm:"not null"`
	UpdatedAt         time.Time  `gorm:"not null"`
}

func (User) TableName() string { return "users" }
