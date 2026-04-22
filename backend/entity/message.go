package entity

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`
	RoomID           uuid.UUID  `gorm:"type:uuid;not null"`
	SenderID         uuid.UUID  `gorm:"type:uuid;not null"`
	ParentMessageID  *uuid.UUID `gorm:"type:uuid;column:parent_message_id"`
	Content          *string
	ImageURL         *string    `gorm:"column:image_url"`
	FileURL          *string    `gorm:"column:file_url"`
	FileName         *string    `gorm:"column:file_name"`
	FileContentType  *string    `gorm:"column:file_content_type"`
	IsDeleted        bool       `gorm:"not null"`
	CreatedAt        time.Time  `gorm:"not null"`
	UpdatedAt        time.Time  `gorm:"not null"`
	DeletedAt        *time.Time
}

func (Message) TableName() string { return "messages" }
