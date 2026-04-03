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
	gdb := setupDB(t)
	ctx := context.Background()

	err := gdb.WithContext(ctx).Create(&UserModel{
		ID:           uuid.New(),
		Name:         "x",
		Email:        "badrole@example.com",
		PasswordHash: "h",
		Role:         "superadmin",
		Active:       true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}).Error
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "check") || strings.Contains(err.Error(), "chk_users_role"))
}

func TestConstraints_InvalidRoomType(t *testing.T) {
	gdb := setupDB(t)
	ctx := context.Background()

	u := &entity.User{
		ID:           uuid.New(),
		Name:         "u",
		Email:        "roomtype@example.com",
		PasswordHash: "h",
		Role:         entity.UserRoleUser,
		Active:       true,
	}
	require.NoError(t, NewUserRepository(gdb).Create(ctx, u))

	err := gdb.WithContext(ctx).Create(&RoomModel{
		ID:        uuid.New(),
		Type:      "channel",
		Name:      nil,
		CreatedBy: u.ID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}).Error
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "check") || strings.Contains(err.Error(), "chk_rooms_type"))
}

func TestConstraints_MessageBothEmptyRejected(t *testing.T) {
	gdb := setupDB(t)
	ctx := context.Background()

	u := &entity.User{ID: uuid.New(), Name: "u", Email: "m1@example.com", PasswordHash: "h", Role: entity.UserRoleUser, Active: true}
	require.NoError(t, NewUserRepository(gdb).Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g"), CreatedBy: u.ID}
	require.NoError(t, NewRoomRepository(gdb).Create(ctx, room))

	err := gdb.WithContext(ctx).Create(&MessageModel{
		ID:        uuid.New(),
		RoomID:    room.ID,
		SenderID:  u.ID,
		Content:   nil,
		ImageURL:  nil,
		IsDeleted: false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}).Error
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "check") || strings.Contains(err.Error(), "chk_messages_content_or_image"))
}

func TestConstraints_MessageDeletedAllowsEmptyBody(t *testing.T) {
	gdb := setupDB(t)
	ctx := context.Background()

	u := &entity.User{ID: uuid.New(), Name: "u", Email: "m2@example.com", PasswordHash: "h", Role: entity.UserRoleUser, Active: true}
	require.NoError(t, NewUserRepository(gdb).Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g2"), CreatedBy: u.ID}
	require.NoError(t, NewRoomRepository(gdb).Create(ctx, room))

	now := time.Now().UTC()
	err := gdb.WithContext(ctx).Create(&MessageModel{
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
	gdb := setupDB(t)
	ctx := context.Background()

	u := &entity.User{ID: uuid.New(), Name: "u", Email: "duprm@example.com", PasswordHash: "h", Role: entity.UserRoleUser, Active: true}
	require.NoError(t, NewUserRepository(gdb).Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g"), CreatedBy: u.ID}
	require.NoError(t, NewRoomRepository(gdb).Create(ctx, room))

	rm := &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u.ID}
	require.NoError(t, NewRoomMemberRepository(gdb).Create(ctx, rm))

	dup := &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u.ID}
	err := NewRoomMemberRepository(gdb).Create(ctx, dup)
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate"))
}

func TestConstraints_DuplicateMessageReadInsert(t *testing.T) {
	gdb := setupDB(t)
	ctx := context.Background()

	u := &entity.User{ID: uuid.New(), Name: "u", Email: "dupmr@example.com", PasswordHash: "h", Role: entity.UserRoleUser, Active: true}
	require.NoError(t, NewUserRepository(gdb).Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g"), CreatedBy: u.ID}
	require.NoError(t, NewRoomRepository(gdb).Create(ctx, room))
	txt := "hi"
	msg := &entity.Message{ID: uuid.New(), RoomID: room.ID, SenderID: u.ID, Content: &txt}
	require.NoError(t, NewMessageRepository(gdb).Create(ctx, msg))

	readRepo := NewMessageReadRepository(gdb)
	now := time.Now().UTC()
	mr1 := &entity.MessageRead{
		ID:            uuid.New(),
		RoomID:        room.ID,
		UserID:        u.ID,
		ReadMessageID: &msg.ID,
		ReadAt:        &now,
	}
	require.NoError(t, readRepo.Upsert(ctx, mr1))

	dup := &entity.MessageRead{
		ID:            uuid.New(),
		RoomID:        room.ID,
		UserID:        u.ID,
		ReadMessageID: &msg.ID,
		ReadAt:        &now,
	}
	err := gdb.WithContext(ctx).Create(messageReadToModel(dup)).Error
	require.Error(t, err)
	require.True(t, strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate"))
}

func TestConstraints_MessageReadsUpsertUpdatesRow(t *testing.T) {
	gdb := setupDB(t)
	ctx := context.Background()

	u := &entity.User{ID: uuid.New(), Name: "u", Email: "upsert@example.com", PasswordHash: "h", Role: entity.UserRoleUser, Active: true}
	require.NoError(t, NewUserRepository(gdb).Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g"), CreatedBy: u.ID}
	require.NoError(t, NewRoomRepository(gdb).Create(ctx, room))
	a, b := "a", "b"
	m1 := &entity.Message{ID: uuid.New(), RoomID: room.ID, SenderID: u.ID, Content: &a}
	m2 := &entity.Message{ID: uuid.New(), RoomID: room.ID, SenderID: u.ID, Content: &b}
	require.NoError(t, NewMessageRepository(gdb).Create(ctx, m1))
	require.NoError(t, NewMessageRepository(gdb).Create(ctx, m2))

	readRepo := NewMessageReadRepository(gdb)
	now := time.Now().UTC()
	require.NoError(t, readRepo.Upsert(ctx, &entity.MessageRead{
		ID: uuid.New(), RoomID: room.ID, UserID: u.ID, ReadMessageID: &m1.ID, ReadAt: &now,
	}))
	got1, err := readRepo.FindByRoomAndUser(ctx, room.ID, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got1)
	require.Equal(t, m1.ID, *got1.ReadMessageID)

	now2 := time.Now().UTC()
	require.NoError(t, readRepo.Upsert(ctx, &entity.MessageRead{
		ID: got1.ID, RoomID: room.ID, UserID: u.ID, ReadMessageID: &m2.ID, ReadAt: &now2,
	}))
	got2, err := readRepo.FindByRoomAndUser(ctx, room.ID, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got2)
	require.Equal(t, m2.ID, *got2.ReadMessageID)
}
