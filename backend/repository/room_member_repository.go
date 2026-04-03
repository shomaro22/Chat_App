package repository

import (
	"context"
	"errors"
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoomMemberRepository interface {
	Create(ctx context.Context, m *entity.RoomMember) error
	FindActiveByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entity.RoomMember, error)
}

type roomMemberRepository struct {
	db *gorm.DB
}

func NewRoomMemberRepository(db *gorm.DB) RoomMemberRepository {
	return &roomMemberRepository{db: db}
}

func (r *roomMemberRepository) Create(ctx context.Context, m *entity.RoomMember) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.JoinedAt.IsZero() {
		m.JoinedAt = time.Now().UTC()
	}
	row := roomMemberToModel(m)
	return r.db.WithContext(ctx).Create(row).Error
}

// FindActiveByRoomAndUser returns the membership when the user is currently in the room (left_at IS NULL).
func (r *roomMemberRepository) FindActiveByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entity.RoomMember, error) {
	var row RoomMemberModel
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return roomMemberToEntity(&row), nil
}
