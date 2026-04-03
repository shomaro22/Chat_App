package repository

import (
	"context"
	"errors"
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoomRepository interface {
	Create(ctx context.Context, room *entity.Room) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Room, error)
}

type roomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) Create(ctx context.Context, room *entity.Room) error {
	if room.ID == uuid.Nil {
		room.ID = uuid.New()
	}
	now := time.Now().UTC()
	if room.CreatedAt.IsZero() {
		room.CreatedAt = now
	}
	if room.UpdatedAt.IsZero() {
		room.UpdatedAt = now
	}
	m := roomToModel(room)
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *roomRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Room, error) {
	var m RoomModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return roomToEntity(&m), nil
}
