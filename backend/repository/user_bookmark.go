package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserMessageBookmark struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	MessageID uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoomID    uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (UserMessageBookmark) TableName() string { return "user_message_bookmarks" }

type UserBookmarkRepository interface {
	Add(ctx context.Context, userID, messageID, roomID uuid.UUID) error
	Remove(ctx context.Context, userID, messageID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]UserMessageBookmark, int64, error)
}

type userBookmarkRepository struct{ db *gorm.DB }

func NewUserBookmarkRepository(db *gorm.DB) UserBookmarkRepository {
	return &userBookmarkRepository{db: db}
}

func (r *userBookmarkRepository) Add(ctx context.Context, userID, messageID, roomID uuid.UUID) error {
	row := UserMessageBookmark{
		UserID:    userID,
		MessageID: messageID,
		RoomID:    roomID,
		CreatedAt: time.Now().UTC(),
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *userBookmarkRepository) Remove(ctx context.Context, userID, messageID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND message_id = ?", userID, messageID).Delete(&UserMessageBookmark{}).Error
}

func (r *userBookmarkRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]UserMessageBookmark, int64, error) {
	q := r.db.WithContext(ctx).Model(&UserMessageBookmark{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	var rows []UserMessageBookmark
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
