package entity

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Type          RoomType   `gorm:"not null"`
	Name          *string
	DirectUserAID *uuid.UUID `gorm:"column:direct_user_a_id;type:uuid"`
	DirectUserBID *uuid.UUID `gorm:"column:direct_user_b_id;type:uuid"`
	CreatedBy     uuid.UUID  `gorm:"type:uuid;not null"`
	CreatedAt     time.Time  `gorm:"not null"`
	UpdatedAt     time.Time  `gorm:"not null"`
}

func (Room) TableName() string { return "rooms" }
