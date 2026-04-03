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
	// FindByIDForActiveMember returns the room only if userID has an active membership (left_at IS NULL).
	FindByIDForActiveMember(ctx context.Context, roomID, userID uuid.UUID) (*entity.Room, error)
	// ListActiveByUserID returns rooms the user is currently in, newest room.updated_at first.
	ListActiveByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entity.Room, error)
	// FindDirectRoomBetweenUsers returns a direct room where exactly userA and userB are the only active members, or nil.
	FindDirectRoomBetweenUsers(ctx context.Context, userA, userB uuid.UUID) (*entity.Room, error)
}

type roomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) Create(ctx context.Context, room *entity.Room) error {
	if room.ID == uuid.Nil {
		room.ID = uuid.New()
	}
	now := time.Now().UTC()
	if room.CreatedAt.IsZero() {
		room.CreatedAt = now
	}
	if room.UpdatedAt.IsZero() {
		room.UpdatedAt = now
	}
	m := roomToModel(room)
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *roomRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Room, error) {
	var m RoomModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return roomToEntity(&m), nil
}

func (r *roomRepository) FindByIDForActiveMember(ctx context.Context, roomID, userID uuid.UUID) (*entity.Room, error) {
	var m RoomModel
	err := r.db.WithContext(ctx).
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
	return roomToEntity(&m), nil
}

func (r *roomRepository) ListActiveByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entity.Room, error) {
	q := r.db.WithContext(ctx).
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
	var rows []RoomModel
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]entity.Room, 0, len(rows))
	for i := range rows {
		if e := roomToEntity(&rows[i]); e != nil {
			out = append(out, *e)
		}
	}
	return out, nil
}

func (r *roomRepository) FindDirectRoomBetweenUsers(ctx context.Context, userA, userB uuid.UUID) (*entity.Room, error) {
	if userA == userB {
		return nil, nil
	}
	var m RoomModel
	err := r.db.WithContext(ctx).
		Table("rooms").
		Select("rooms.*").
		Joins(`INNER JOIN room_members a ON a.room_id = rooms.id AND a.user_id = ? AND a.left_at IS NULL`, userA).
		Joins(`INNER JOIN room_members b ON b.room_id = rooms.id AND b.user_id = ? AND b.left_at IS NULL`, userB).
		Where("rooms.type = ?", string(entity.RoomTypeDirect)).
		Where(`(SELECT COUNT(*)::bigint FROM room_members m WHERE m.room_id = rooms.id AND m.left_at IS NULL) = 2`).
		Limit(1).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return roomToEntity(&m), nil
}
