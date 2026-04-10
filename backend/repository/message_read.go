package repository

import (
	"context"
	"errors"
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageReadRepository interface {
	FindByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entity.MessageRead, error)
	Create(ctx context.Context, m *entity.MessageRead) error
	UpdateLastReadMessage(ctx context.Context, roomID, userID, messageID uuid.UUID) error
}

type messageReadRepository struct {
	db *gorm.DB
}

func NewMessageReadRepository(db *gorm.DB) MessageReadRepository {
	return &messageReadRepository{db: db}
}

func (readRows *messageReadRepository) Create(ctx context.Context, m *entity.MessageRead) error {
	return readRows.db.WithContext(ctx).Create(m).Error
}

func (readRows *messageReadRepository) UpdateLastReadMessage(ctx context.Context, roomID, userID, messageID uuid.UUID) error {
	now := time.Now().UTC()
	res := readRows.db.WithContext(ctx).Model(&entity.MessageRead{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Updates(map[string]interface{}{
			"read_message_id": messageID,
			"read_at":         now,
			"updated_at":      now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (readRows *messageReadRepository) FindByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entity.MessageRead, error) {
	var row entity.MessageRead
	err := readRows.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}
