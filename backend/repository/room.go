package repository

import (
	"context"
	"errors"

	"chat-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoomRepository interface {
	Create(ctx context.Context, room *entity.Room) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Room, error)
	FindByIDForActiveMember(ctx context.Context, roomID, userID uuid.UUID) (*entity.Room, error)
	ListJoinedRooms(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entity.Room, error)
	FindDirectRoomByUserPair(ctx context.Context, userA, userB uuid.UUID) (*entity.Room, error)
}

type roomRepository struct {
	db *gorm.DB
}

// NewRoomRepository はルーム永続化用リポジトリを返す。
func NewRoomRepository(db *gorm.DB) RoomRepository {
	return &roomRepository{db: db}
}

func (rooms *roomRepository) Create(ctx context.Context, room *entity.Room) error {
	return rooms.db.WithContext(ctx).Create(room).Error
}

func (rooms *roomRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Room, error) {
	var m entity.Room
	if err := rooms.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (rooms *roomRepository) FindByIDForActiveMember(ctx context.Context, roomID, userID uuid.UUID) (*entity.Room, error) {
	var m entity.Room
	err := rooms.db.WithContext(ctx).
		Table("rooms").
		Joins("INNER JOIN room_members ON room_members.room_id = rooms.id AND room_members.user_id = ? AND room_members.left_at IS NULL", userID).
		Where("rooms.id = ?", roomID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (rooms *roomRepository) ListJoinedRooms(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entity.Room, error) {
	q := rooms.db.WithContext(ctx).
		Table("rooms").
		Select("rooms.*").
		Joins("INNER JOIN room_members ON room_members.room_id = rooms.id AND room_members.left_at IS NULL").
		Where("room_members.user_id = ?", userID).
		Order("rooms.updated_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	var rows []entity.Room
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (rooms *roomRepository) FindDirectRoomByUserPair(ctx context.Context, userA, userB uuid.UUID) (*entity.Room, error) {
	if userA == userB {
		return nil, nil
	}
	a, b := entity.OrderedUserPair(userA, userB)
	var m entity.Room
	err := rooms.db.WithContext(ctx).
		Where("type = ?", string(entity.RoomTypeDirect)).
		Where("direct_user_a_id = ? AND direct_user_b_id = ?", a, b).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}
