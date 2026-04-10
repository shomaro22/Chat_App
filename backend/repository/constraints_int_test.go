//go:build integration

package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestConstraints_InvalidUserRole(t *testing.T) {
	gormDB := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	err := gormDB.WithContext(ctx).Create(&entity.User{
		ID:           uuid.New(),
		Name:         "x",
		Email:        "badrole@example.com",
		PasswordHash: "h",
		Role:         entity.UserRole("superadmin"),
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Error
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "check") || strings.Contains(err.Error(), "chk_users_role"))
}

func TestConstraints_InvalidRoomType(t *testing.T) {
	gormDB := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	u := &entity.User{
		ID:           uuid.New(),
		Name:         "u",
		Email:        "roomtype@example.com",
		PasswordHash: "h",
		Role:         entity.UserRoleUser,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, NewUserRepository(gormDB).Create(ctx, u))

	err := gormDB.WithContext(ctx).Create(&entity.Room{
		ID:        uuid.New(),
		Type:      entity.RoomType("channel"),
		Name:      nil,
		CreatedBy: u.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "check") || strings.Contains(err.Error(), "chk_rooms_type"))
}

func TestConstraints_MessageBothEmptyRejected(t *testing.T) {
	gormDB := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	u := &entity.User{ID: uuid.New(), Name: "u", Email: "m1@example.com", PasswordHash: "h", Role: entity.UserRoleUser, Active: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewUserRepository(gormDB).Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g"), CreatedBy: u.ID, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewRoomRepository(gormDB).Create(ctx, room))

	err := gormDB.WithContext(ctx).Create(&entity.Message{
		ID:        uuid.New(),
		RoomID:    room.ID,
		SenderID:  u.ID,
		Content:   nil,
		ImageURL:  nil,
		IsDeleted: false,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "check") || strings.Contains(err.Error(), "chk_messages_content_or_image"))
}

func TestConstraints_MessageDeletedAllowsEmptyBody(t *testing.T) {
	gormDB := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	u := &entity.User{ID: uuid.New(), Name: "u", Email: "m2@example.com", PasswordHash: "h", Role: entity.UserRoleUser, Active: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewUserRepository(gormDB).Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g2"), CreatedBy: u.ID, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewRoomRepository(gormDB).Create(ctx, room))

	err := gormDB.WithContext(ctx).Create(&entity.Message{
		ID:        uuid.New(),
		RoomID:    room.ID,
		SenderID:  u.ID,
		Content:   nil,
		ImageURL:  nil,
		IsDeleted: true,
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: &now,
	}).Error
	require.NoError(t, err)
}

func TestConstraints_DuplicateRoomMember(t *testing.T) {
	gormDB := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	u := &entity.User{ID: uuid.New(), Name: "u", Email: "duprm@example.com", PasswordHash: "h", Role: entity.UserRoleUser, Active: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewUserRepository(gormDB).Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g"), CreatedBy: u.ID, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewRoomRepository(gormDB).Create(ctx, room))

	rm := &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u.ID, JoinedAt: now}
	require.NoError(t, NewRoomMemberRepository(gormDB).Create(ctx, rm))

	dup := &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u.ID, JoinedAt: now}
	err := NewRoomMemberRepository(gormDB).Create(ctx, dup)
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate"))
}

func TestConstraints_DuplicateMessageReadInsert(t *testing.T) {
	gormDB := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	u := &entity.User{ID: uuid.New(), Name: "u", Email: "dupmr@example.com", PasswordHash: "h", Role: entity.UserRoleUser, Active: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewUserRepository(gormDB).Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g"), CreatedBy: u.ID, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewRoomRepository(gormDB).Create(ctx, room))
	txt := "hi"
	msg := &entity.Message{ID: uuid.New(), RoomID: room.ID, SenderID: u.ID, Content: &txt, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewMessageRepository(gormDB).Create(ctx, msg))

	readRows := NewMessageReadRepository(gormDB)
	mr1 := &entity.MessageRead{
		ID:                uuid.New(),
		RoomID:            room.ID,
		UserID:            u.ID,
		LastReadMessageID: &msg.ID,
		ReadAt:            &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	require.NoError(t, readRows.Create(ctx, mr1))

	dup := &entity.MessageRead{
		ID:                uuid.New(),
		RoomID:            room.ID,
		UserID:            u.ID,
		LastReadMessageID: &msg.ID,
		ReadAt:            &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	err := gormDB.WithContext(ctx).Create(dup).Error
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate"))
}

func TestConstraints_MessageReadsCreateAndUpdate(t *testing.T) {
	gormDB := setupDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	u := &entity.User{ID: uuid.New(), Name: "u", Email: "upsert@example.com", PasswordHash: "h", Role: entity.UserRoleUser, Active: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewUserRepository(gormDB).Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g"), CreatedBy: u.ID, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewRoomRepository(gormDB).Create(ctx, room))
	a, b := "a", "b"
	m1 := &entity.Message{ID: uuid.New(), RoomID: room.ID, SenderID: u.ID, Content: &a, CreatedAt: now, UpdatedAt: now}
	m2 := &entity.Message{ID: uuid.New(), RoomID: room.ID, SenderID: u.ID, Content: &b, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, NewMessageRepository(gormDB).Create(ctx, m1))
	require.NoError(t, NewMessageRepository(gormDB).Create(ctx, m2))

	readRows := NewMessageReadRepository(gormDB)
	require.NoError(t, readRows.Create(ctx, &entity.MessageRead{
		ID: uuid.New(), RoomID: room.ID, UserID: u.ID, LastReadMessageID: &m1.ID, ReadAt: &now, CreatedAt: now, UpdatedAt: now,
	}))
	got1, err := readRows.FindByRoomAndUser(ctx, room.ID, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got1)
	require.Equal(t, m1.ID, *got1.LastReadMessageID)

	require.NoError(t, readRows.UpdateLastReadMessage(ctx, room.ID, u.ID, m2.ID))
	got2, err := readRows.FindByRoomAndUser(ctx, room.ID, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got2)
	require.Equal(t, m2.ID, *got2.LastReadMessageID)
}
