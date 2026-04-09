package repository

import (
	"time"

	"github.com/google/uuid"
)

type UserModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name         string
	Email        string
	PasswordHash string
	Role         string
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (UserModel) TableName() string { return "users" }

type RoomModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	Type          string
	Name          *string
	DirectUserAID *uuid.UUID `gorm:"column:direct_user_a_id;type:uuid"`
	DirectUserBID *uuid.UUID `gorm:"column:direct_user_b_id;type:uuid"`
	CreatedBy     uuid.UUID  `gorm:"type:uuid"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (RoomModel) TableName() string { return "rooms" }

type RoomMemberModel struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoomID   uuid.UUID `gorm:"type:uuid"`
	UserID   uuid.UUID `gorm:"type:uuid"`
	JoinedAt time.Time
	LeftAt   *time.Time
}

func (RoomMemberModel) TableName() string { return "room_members" }

type MessageModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoomID    uuid.UUID `gorm:"type:uuid"`
	SenderID  uuid.UUID `gorm:"type:uuid"`
	Content   *string
	ImageURL  *string `gorm:"column:image_url"`
	IsDeleted bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (MessageModel) TableName() string { return "messages" }

type MessageReadModel struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey"`
	RoomID            uuid.UUID  `gorm:"type:uuid"`
	UserID            uuid.UUID  `gorm:"type:uuid"`
	LastReadMessageID *uuid.UUID `gorm:"column:read_message_id;type:uuid"`
	ReadAt            *time.Time `gorm:"column:read_at"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (MessageReadModel) TableName() string { return "message_reads" }
