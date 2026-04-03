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
	Create(ctx context.Context, m *entity.Message) error
	ListByRoomOrderByCreatedAt(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]entity.Message, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Message, error)
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, m *entity.Message) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	row := messageToModel(m)
	return r.db.WithContext(ctx).Create(row).Error
}

// ListByRoomOrderByCreatedAt returns messages in the room ordered by created_at ASC, id ASC (tie-break).
func (r *messageRepository) ListByRoomOrderByCreatedAt(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]entity.Message, error) {
	var rows []MessageModel
	q := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("created_at ASC, id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
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

func (r *messageRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Message, error) {
	var row MessageModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return messageToEntity(&row), nil
}
