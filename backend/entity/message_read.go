package entity

import (
	"time"

	"github.com/google/uuid"
)

// MessageRead stores per-user read progress in a room (last read position), not per-message flags.
type MessageRead struct {
	ID                uuid.UUID
	RoomID            uuid.UUID
	UserID            uuid.UUID
	LastReadMessageID *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
