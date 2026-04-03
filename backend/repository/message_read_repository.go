package repository

import (
	"context"
	"errors"

	"chat-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MessageReadRepository interface {
	Upsert(ctx context.Context, m *entity.MessageRead) error
	FindByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entity.MessageRead, error)
}

type messageReadRepository struct {
	db *gorm.DB
}

func NewMessageReadRepository(db *gorm.DB) MessageReadRepository {
	return &messageReadRepository{db: db}
}

func (r *messageReadRepository) Upsert(ctx context.Context, m *entity.MessageRead) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	row := messageReadToModel(m)
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "room_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"read_message_id", "read_at"}),
	}).Create(row).Error
}

func (r *messageReadRepository) FindByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entity.MessageRead, error) {
	var row MessageReadModel
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return messageReadToEntity(&row), nil
}
