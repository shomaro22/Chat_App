package entity

import (
	"time"

	"github.com/google/uuid"
)

type RoomMember struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoomID   uuid.UUID `gorm:"type:uuid;not null"`
	UserID   uuid.UUID `gorm:"type:uuid;not null"`
	JoinedAt time.Time `gorm:"not null"`
	LeftAt   *time.Time
}

func (RoomMember) TableName() string { return "room_members" }
