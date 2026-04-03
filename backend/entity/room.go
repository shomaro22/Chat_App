package entity

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID        uuid.UUID
	Type      RoomType
	Name      *string
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}
