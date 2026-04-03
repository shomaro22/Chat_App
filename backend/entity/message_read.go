package entity

import (
	"time"

	"github.com/google/uuid"
)

type MessageRead struct {
	ID            uuid.UUID
	RoomID        uuid.UUID
	UserID        uuid.UUID
	ReadMessageID *uuid.UUID
	ReadAt        *time.Time
}
