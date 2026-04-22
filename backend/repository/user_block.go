package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserBlock struct {
	BlockerID uuid.UUID `gorm:"column:blocker_id;type:uuid;primaryKey"`
	BlockedID uuid.UUID `gorm:"column:blocked_id;type:uuid;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

func (UserBlock) TableName() string { return "user_blocks" }

type UserBlockRepository interface {
	Block(ctx context.Context, blockerID, blockedID uuid.UUID) error
	Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error
	IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error)
	ListBlockedUserIDs(ctx context.Context, blockerID uuid.UUID) ([]uuid.UUID, error)
	ListBlockerUserIDsInRoomForBlocked(ctx context.Context, roomID, blockedID uuid.UUID) ([]uuid.UUID, error)
}

type userBlockRepository struct {
	db *gorm.DB
}

func NewUserBlockRepository(db *gorm.DB) UserBlockRepository {
	return &userBlockRepository{db: db}
}

func (r *userBlockRepository) Block(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	row := UserBlock{
		BlockerID: blockerID,
		BlockedID: blockedID,
		CreatedAt: time.Now().UTC(),
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&row).Error
}

func (r *userBlockRepository) Unblock(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Delete(&UserBlock{}).Error
}

func (r *userBlockRepository) IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).
		Model(&UserBlock{}).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *userBlockRepository) ListBlockedUserIDs(ctx context.Context, blockerID uuid.UUID) ([]uuid.UUID, error) {
	var out []uuid.UUID
	if err := r.db.WithContext(ctx).
		Model(&UserBlock{}).
		Where("blocker_id = ?", blockerID).
		Pluck("blocked_id", &out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *userBlockRepository) ListBlockerUserIDsInRoomForBlocked(ctx context.Context, roomID, blockedID uuid.UUID) ([]uuid.UUID, error) {
	var out []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT ub.blocker_id
		FROM user_blocks ub
		INNER JOIN room_members rm ON rm.user_id = ub.blocker_id AND rm.room_id = ? AND rm.left_at IS NULL
		WHERE ub.blocked_id = ?
	`, roomID, blockedID).Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}
