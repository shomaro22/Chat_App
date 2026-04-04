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

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable"
	}
	return databaseURL
}

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	databaseURL := testDatabaseURL(t)
	require.NoError(t, db.Up(databaseURL))
	gormDB, err := db.Open(databaseURL)
	require.NoError(t, err)
	sqlDB, err := gormDB.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(`TRUNCATE message_reads, messages, room_members, rooms, users RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	return gormDB
}

func TestIntegration_Flow(t *testing.T) {
	gormDB := setupDB(t)
	ctx := context.Background()

	users := NewUserRepository(gormDB)
	rooms := NewRoomRepository(gormDB)
	members := NewRoomMemberRepository(gormDB)
	messages := NewMessageRepository(gormDB)
	readRows := NewMessageReadRepository(gormDB)

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
	require.NoError(t, users.Create(ctx, u1))
	require.NoError(t, users.Create(ctx, u2))

	got1, err := users.FindByEmail(ctx, "alice@example.com")
	require.NoError(t, err)
	require.NotNil(t, got1)
	require.Equal(t, u1.ID, got1.ID)

	room := &entity.Room{
		ID:        uuid.New(),
		Type:      entity.RoomTypeGroup,
		Name:      strPtr("g1"),
		CreatedBy: u1.ID,
	}
	require.NoError(t, rooms.Create(ctx, room))

	rm1 := &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u1.ID}
	rm2 := &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u2.ID}
	require.NoError(t, members.Create(ctx, rm1))
	require.NoError(t, members.Create(ctx, rm2))

	active, err := members.FindActiveByRoomAndUser(ctx, room.ID, u1.ID)
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
	require.NoError(t, messages.Create(ctx, m1))

	time.Sleep(5 * time.Millisecond)
	m2 := &entity.Message{
		ID:       uuid.New(),
		RoomID:   room.ID,
		SenderID: u2.ID,
		Content:  &txt,
	}
	require.NoError(t, messages.Create(ctx, m2))

	list, err := messages.ListByRoomOrderByCreatedAt(ctx, room.ID, 100, 0)
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
	require.NoError(t, readRows.Upsert(ctx, mr))

	gotRead, err := readRows.FindByRoomAndUser(ctx, room.ID, u1.ID)
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
	require.NoError(t, readRows.Upsert(ctx, mr2))

	gotRead2, err := readRows.FindByRoomAndUser(ctx, room.ID, u1.ID)
	require.NoError(t, err)
	require.NotNil(t, gotRead2)
	require.Equal(t, m2.ID, *gotRead2.ReadMessageID)
}

func strPtr(s string) *string { return &s }
