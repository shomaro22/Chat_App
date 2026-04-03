package entity

import (
	"time"

	"github.com/google/uuid"
)

type RoomMember struct {
	ID       uuid.UUID
	RoomID   uuid.UUID
	UserID   uuid.UUID
	JoinedAt time.Time
	LeftAt   *time.Time
}
