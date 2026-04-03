//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"chat-app/db"
	"chat-app/entity"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable"
	}
	return dsn
}

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := testDSN(t)
	require.NoError(t, db.Up(dsn))
	gdb, err := db.Open(dsn)
	require.NoError(t, err)
	sqlDB, err := gdb.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(`TRUNCATE message_reads, messages, room_members, rooms, users RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	return gdb
}

func TestIntegration_Flow(t *testing.T) {
	gdb := setupDB(t)
	ctx := context.Background()

	userRepo := NewUserRepository(gdb)
	roomRepo := NewRoomRepository(gdb)
	memberRepo := NewRoomMemberRepository(gdb)
	msgRepo := NewMessageRepository(gdb)
	readRepo := NewMessageReadRepository(gdb)

	u1 := &entity.User{
		ID:           uuid.New(),
		Name:         "Alice",
		Email:        "alice@example.com",
		PasswordHash: "hash",
		Role:         entity.UserRoleUser,
		Active:       true,
	}
	u2 := &entity.User{
		ID:           uuid.New(),
		Name:         "Bob",
		Email:        "bob@example.com",
		PasswordHash: "hash",
		Role:         entity.UserRoleUser,
		Active:       true,
	}
	require.NoError(t, userRepo.Create(ctx, u1))
	require.NoError(t, userRepo.Create(ctx, u2))

	got1, err := userRepo.FindByEmail(ctx, "alice@example.com")
	require.NoError(t, err)
	require.NotNil(t, got1)
	require.Equal(t, u1.ID, got1.ID)

	room := &entity.Room{
		ID:        uuid.New(),
		Type:      entity.RoomTypeGroup,
		Name:      strPtr("g1"),
		CreatedBy: u1.ID,
	}
	require.NoError(t, roomRepo.Create(ctx, room))

	rm1 := &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u1.ID}
	rm2 := &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u2.ID}
	require.NoError(t, memberRepo.Create(ctx, rm1))
	require.NoError(t, memberRepo.Create(ctx, rm2))

	active, err := memberRepo.FindActiveByRoomAndUser(ctx, room.ID, u1.ID)
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Nil(t, active.LeftAt)

	txt := "hello"
	m1 := &entity.Message{
		ID:       uuid.New(),
		RoomID:   room.ID,
		SenderID: u1.ID,
		Content:  &txt,
	}
	require.NoError(t, msgRepo.Create(ctx, m1))

	time.Sleep(5 * time.Millisecond)
	m2 := &entity.Message{
		ID:       uuid.New(),
		RoomID:   room.ID,
		SenderID: u2.ID,
		Content:  &txt,
	}
	require.NoError(t, msgRepo.Create(ctx, m2))

	list, err := msgRepo.ListByRoomOrderByCreatedAt(ctx, room.ID, 100, 0)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, m1.ID, list[0].ID)
	require.Equal(t, m2.ID, list[1].ID)

	now := time.Now().UTC()
	mr := &entity.MessageRead{
		ID:            uuid.New(),
		RoomID:        room.ID,
		UserID:        u1.ID,
		ReadMessageID: &m1.ID,
		ReadAt:        &now,
	}
	require.NoError(t, readRepo.Upsert(ctx, mr))

	gotRead, err := readRepo.FindByRoomAndUser(ctx, room.ID, u1.ID)
	require.NoError(t, err)
	require.NotNil(t, gotRead)
	require.NotNil(t, gotRead.ReadMessageID)
	require.Equal(t, m1.ID, *gotRead.ReadMessageID)

	m2id := m2.ID
	now2 := time.Now().UTC()
	mr2 := &entity.MessageRead{
		ID:            gotRead.ID,
		RoomID:        room.ID,
		UserID:        u1.ID,
		ReadMessageID: &m2id,
		ReadAt:        &now2,
	}
	require.NoError(t, readRepo.Upsert(ctx, mr2))

	gotRead2, err := readRepo.FindByRoomAndUser(ctx, room.ID, u1.ID)
	require.NoError(t, err)
	require.NotNil(t, gotRead2)
	require.Equal(t, m2.ID, *gotRead2.ReadMessageID)
}

func strPtr(s string) *string { return &s }
