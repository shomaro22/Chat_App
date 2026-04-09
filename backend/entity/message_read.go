package entity

import (
	"time"

	"github.com/google/uuid"
)

type MessageRead struct {
	ID                uuid.UUID
	RoomID            uuid.UUID
	UserID            uuid.UUID
	LastReadMessageID *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
