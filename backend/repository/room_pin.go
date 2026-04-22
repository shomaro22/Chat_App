package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoomPin struct {
	RoomID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	MessageID uuid.UUID `gorm:"type:uuid;not null"`
	PinnedBy  uuid.UUID `gorm:"type:uuid;not null"`
	PinnedAt  time.Time `gorm:"not null"`
}

func (RoomPin) TableName() string { return "room_pins" }

type RoomPinRepository interface {
	Upsert(ctx context.Context, roomID, messageID, pinnedBy uuid.UUID) error
	Delete(ctx context.Context, roomID uuid.UUID) error
	FindByRoom(ctx context.Context, roomID uuid.UUID) (*RoomPin, error)
}

type roomPinRepository struct{ db *gorm.DB }

func NewRoomPinRepository(db *gorm.DB) RoomPinRepository {
	return &roomPinRepository{db: db}
}

func (r *roomPinRepository) Upsert(ctx context.Context, roomID, messageID, pinnedBy uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO room_pins (room_id, message_id, pinned_by, pinned_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (room_id) DO UPDATE SET
			message_id = EXCLUDED.message_id,
			pinned_by = EXCLUDED.pinned_by,
			pinned_at = EXCLUDED.pinned_at
	`, roomID, messageID, pinnedBy, now).Error
}

func (r *roomPinRepository) Delete(ctx context.Context, roomID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("room_id = ?", roomID).Delete(&RoomPin{}).Error
}

func (r *roomPinRepository) FindByRoom(ctx context.Context, roomID uuid.UUID) (*RoomPin, error) {
	var row RoomPin
	if err := r.db.WithContext(ctx).Where("room_id = ?", roomID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}
