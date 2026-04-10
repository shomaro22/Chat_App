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
	Create(ctx context.Context, member *entity.RoomMember) error
	FindActiveByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entity.RoomMember, error)
	IsActiveMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	ListActiveMembers(ctx context.Context, roomID uuid.UUID) ([]entity.RoomMember, error)
	CreateBulk(ctx context.Context, members []entity.RoomMember) error
	CountActiveByRoomID(ctx context.Context, roomID uuid.UUID) (int64, error)
	Leave(ctx context.Context, roomID, userID uuid.UUID) error
}

type roomMemberRepository struct {
	db *gorm.DB
}

func NewRoomMemberRepository(db *gorm.DB) RoomMemberRepository {
	return &roomMemberRepository{db: db}
}

func (members *roomMemberRepository) Create(ctx context.Context, member *entity.RoomMember) error {
	return members.db.WithContext(ctx).Create(member).Error
}

func (members *roomMemberRepository) FindActiveByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entity.RoomMember, error) {
	var row entity.RoomMember
	err := members.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (members *roomMemberRepository) IsActiveMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var n int64
	err := members.db.WithContext(ctx).Model(&entity.RoomMember{}).
		Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (members *roomMemberRepository) ListActiveMembers(ctx context.Context, roomID uuid.UUID) ([]entity.RoomMember, error) {
	var rows []entity.RoomMember
	if err := members.db.WithContext(ctx).
		Where("room_id = ? AND left_at IS NULL", roomID).
		Order("joined_at ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (members *roomMemberRepository) CreateBulk(ctx context.Context, batch []entity.RoomMember) error {
	if len(batch) == 0 {
		return nil
	}
	rows := make([]entity.RoomMember, 0, len(batch))
	for i := range batch {
		rows = append(rows, batch[i])
	}
	return members.db.WithContext(ctx).Create(&rows).Error
}

func (members *roomMemberRepository) CountActiveByRoomID(ctx context.Context, roomID uuid.UUID) (int64, error) {
	var n int64
	err := members.db.WithContext(ctx).Model(&entity.RoomMember{}).
		Where("room_id = ? AND left_at IS NULL", roomID).
		Count(&n).Error
	return n, err
}

func (members *roomMemberRepository) Leave(ctx context.Context, roomID, userID uuid.UUID) error {
	now := time.Now().UTC()
	res := members.db.WithContext(ctx).Model(&entity.RoomMember{}).
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
