package entity

import (
	"time"

	"github.com/google/uuid"
)

type MessageRead struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey"`
	RoomID            uuid.UUID  `gorm:"type:uuid;not null"`
	UserID            uuid.UUID  `gorm:"type:uuid;not null"`
	LastReadMessageID *uuid.UUID `gorm:"column:read_message_id;type:uuid"`
	ReadAt            *time.Time `gorm:"column:read_at"`
	CreatedAt         time.Time  `gorm:"not null"`
	UpdatedAt         time.Time  `gorm:"not null"`
}

func (MessageRead) TableName() string { return "message_reads" }
