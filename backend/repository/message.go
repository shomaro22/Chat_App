package repository

import (
	"context"
	"errors"
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *entity.Message) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Message, error)
	ListByRoomID(ctx context.Context, roomID uuid.UUID, beforeID *uuid.UUID, limit int) ([]entity.Message, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (messages *messageRepository) Create(ctx context.Context, msg *entity.Message) error {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	now := time.Now().UTC()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	if msg.UpdatedAt.IsZero() {
		msg.UpdatedAt = now
	}
	row := messageToModel(msg)
	return messages.db.WithContext(ctx).Create(row).Error
}

func (messages *messageRepository) ListByRoomID(ctx context.Context, roomID uuid.UUID, beforeID *uuid.UUID, limit int) ([]entity.Message, error) {
	var rows []MessageModel
	q := messages.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("created_at DESC, id DESC")
	if beforeID != nil {
		q = q.Where(
			"(created_at, id) < (SELECT created_at, id FROM messages WHERE id = ? AND room_id = ?)",
			*beforeID, roomID,
		)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]entity.Message, 0, len(rows))
	for i := range rows {
		if e := messageToEntity(&rows[i]); e != nil {
			out = append(out, *e)
		}
	}
	return out, nil
}

func (messages *messageRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Message, error) {
	var row MessageModel
	if err := messages.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return messageToEntity(&row), nil
}

func (messages *messageRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	return messages.db.WithContext(ctx).Model(&MessageModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"is_deleted": true,
			"updated_at": now,
		}).Error
}
