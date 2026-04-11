package repository

import (
	"context"
	"errors"
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoomRepository interface {
	Create(ctx context.Context, room *entity.Room) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Room, error)
	FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*entity.Room, error)
	FindByIDForActiveMember(ctx context.Context, roomID, userID uuid.UUID) (*entity.Room, error)
	ListJoinedRooms(ctx context.Context, userID uuid.UUID, limit, offset int, updatedDesc bool) ([]entity.Room, error)
	FindDirectRoomByUserPair(ctx context.Context, userA, userB uuid.UUID) (*entity.Room, error)
	UpdateUpdatedAt(ctx context.Context, roomID uuid.UUID, t time.Time) error
	UpdateGroupName(ctx context.Context, roomID uuid.UUID, name string, updatedAt time.Time) error
	SoftDelete(ctx context.Context, roomID uuid.UUID, deletedAt time.Time) error
	SetFrozenAt(ctx context.Context, roomID uuid.UUID, frozenAt *time.Time) error
}

type roomRepository struct {
	db *gorm.DB
}

// NewRoomRepository はルーム永続化用リポジトリを返す。
func NewRoomRepository(db *gorm.DB) RoomRepository {
	return &roomRepository{db: db}
}

func notDeletedRoom(db *gorm.DB) *gorm.DB {
	return db.Where("rooms.deleted_at IS NULL")
}

func (rooms *roomRepository) Create(ctx context.Context, room *entity.Room) error {
	return rooms.db.WithContext(ctx).Create(room).Error
}

func (rooms *roomRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Room, error) {
	var m entity.Room
	if err := rooms.db.WithContext(ctx).Scopes(notDeletedRoom).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (rooms *roomRepository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*entity.Room, error) {
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
		Scopes(notDeletedRoom).
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

func (rooms *roomRepository) ListJoinedRooms(ctx context.Context, userID uuid.UUID, limit, offset int, updatedDesc bool) ([]entity.Room, error) {
	q := rooms.db.WithContext(ctx).
		Table("rooms").
		Select("rooms.*").
		Joins("INNER JOIN room_members ON room_members.room_id = rooms.id AND room_members.left_at IS NULL").
		Scopes(notDeletedRoom).
		Where("room_members.user_id = ?", userID)
	if updatedDesc {
		q = q.Order("rooms.updated_at DESC")
	} else {
		q = q.Order("rooms.updated_at ASC")
	}
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
		Scopes(notDeletedRoom).
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

func (rooms *roomRepository) UpdateUpdatedAt(ctx context.Context, roomID uuid.UUID, t time.Time) error {
	return rooms.db.WithContext(ctx).Model(&entity.Room{}).Where("id = ? AND deleted_at IS NULL", roomID).Update("updated_at", t).Error
}

func (rooms *roomRepository) UpdateGroupName(ctx context.Context, roomID uuid.UUID, name string, updatedAt time.Time) error {
	return rooms.db.WithContext(ctx).Model(&entity.Room{}).
		Where("id = ? AND deleted_at IS NULL AND type = ?", roomID, string(entity.RoomTypeGroup)).
		Updates(map[string]interface{}{"name": name, "updated_at": updatedAt}).Error
}

func (rooms *roomRepository) SoftDelete(ctx context.Context, roomID uuid.UUID, deletedAt time.Time) error {
	return rooms.db.WithContext(ctx).Model(&entity.Room{}).Where("id = ?", roomID).Updates(map[string]interface{}{
		"deleted_at": deletedAt,
		"updated_at": deletedAt,
	}).Error
}

func (rooms *roomRepository) SetFrozenAt(ctx context.Context, roomID uuid.UUID, frozenAt *time.Time) error {
	return rooms.db.WithContext(ctx).Model(&entity.Room{}).Where("id = ? AND deleted_at IS NULL", roomID).
		Update("frozen_at", frozenAt).Error
}

// RoomUnitOfWork はルームとメンバーへの書き込みを同一トランザクションでまとめるための窓口。
type RoomUnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, rooms RoomRepository, members RoomMemberRepository) error) error
}

type gormRoomUnitOfWork struct {
	db *gorm.DB
}

// NewRoomUnitOfWork は GORM を用いた RoomUnitOfWork を返す。
func NewRoomUnitOfWork(db *gorm.DB) RoomUnitOfWork {
	return &gormRoomUnitOfWork{db: db}
}

func (u *gormRoomUnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, rooms RoomRepository, members RoomMemberRepository) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx, NewRoomRepository(tx), NewRoomMemberRepository(tx))
	})
}
