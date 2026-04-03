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
	IsActiveMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	ListActiveByRoomID(ctx context.Context, roomID uuid.UUID) ([]entity.RoomMember, error)
	CountActiveByRoomID(ctx context.Context, roomID uuid.UUID) (int64, error)
	// Leave sets left_at for the active membership row; returns gorm.ErrRecordNotFound if none matched.
	Leave(ctx context.Context, roomID, userID uuid.UUID) error
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

func (r *roomMemberRepository) IsActiveMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&RoomMemberModel{}).
		Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *roomMemberRepository) ListActiveByRoomID(ctx context.Context, roomID uuid.UUID) ([]entity.RoomMember, error) {
	var rows []RoomMemberModel
	if err := r.db.WithContext(ctx).
		Where("room_id = ? AND left_at IS NULL", roomID).
		Order("joined_at ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]entity.RoomMember, 0, len(rows))
	for i := range rows {
		if e := roomMemberToEntity(&rows[i]); e != nil {
			out = append(out, *e)
		}
	}
	return out, nil
}

func (r *roomMemberRepository) CountActiveByRoomID(ctx context.Context, roomID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&RoomMemberModel{}).
		Where("room_id = ? AND left_at IS NULL", roomID).
		Count(&n).Error
	return n, err
}

func (r *roomMemberRepository) Leave(ctx context.Context, roomID, userID uuid.UUID) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&RoomMemberModel{}).
		Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).
		Update("left_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
