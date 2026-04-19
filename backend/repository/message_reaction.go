package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageReaction struct {
	MessageID uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Emoji     string    `gorm:"primaryKey;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (MessageReaction) TableName() string { return "message_reactions" }

type MessageReactionRepository interface {
	Add(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	Remove(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	ListByMessage(ctx context.Context, messageID uuid.UUID) ([]MessageReaction, error)
}

type messageReactionRepository struct{ db *gorm.DB }

func NewMessageReactionRepository(db *gorm.DB) MessageReactionRepository {
	return &messageReactionRepository{db: db}
}

func (r *messageReactionRepository) Add(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	row := MessageReaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: time.Now().UTC(),
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *messageReactionRepository) Remove(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	return r.db.WithContext(ctx).
		Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).
		Delete(&MessageReaction{}).Error
}

func (r *messageReactionRepository) ListByMessage(ctx context.Context, messageID uuid.UUID) ([]MessageReaction, error) {
	var rows []MessageReaction
	if err := r.db.WithContext(ctx).Where("message_id = ?", messageID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
