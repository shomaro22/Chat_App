package entity

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID            uuid.UUID
	Type          RoomType
	Name          *string
	DirectUserAID *uuid.UUID
	DirectUserBID *uuid.UUID
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
