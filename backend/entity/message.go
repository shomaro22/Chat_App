package entity

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoomID    uuid.UUID `gorm:"type:uuid;not null"`
	SenderID  uuid.UUID `gorm:"type:uuid;not null"`
	Content   *string
	ImageURL  *string   `gorm:"column:image_url"`
	IsDeleted bool      `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
	DeletedAt *time.Time
}

func (Message) TableName() string { return "messages" }
