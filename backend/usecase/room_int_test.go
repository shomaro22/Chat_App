//go:build integration

package usecase

import (
	"context"
	"os"
	"testing"
	"time"

	"chat-app/db"
	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUsecaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable"
	}
	require.NoError(t, db.Up(databaseURL))
	gormDB, err := db.Open(databaseURL)
	require.NoError(t, err)
	sqlDB, err := gormDB.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec(`TRUNCATE message_reads, messages, room_members, rooms, users RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	return gormDB
}

func testUser(t *testing.T, gormDB *gorm.DB, email string) *entity.User {
	return testUserActive(t, gormDB, email, true)
}

func testUserActive(t *testing.T, gormDB *gorm.DB, email string, active bool) *entity.User {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	u := &entity.User{
		ID:           uuid.New(),
		Name:         email,
		Email:        email,
		PasswordHash: "x",
		Role:         entity.UserRoleUser,
		Active:       active,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, repository.NewUserRepository(gormDB).Create(ctx, u))
	return u
}

func newRoomUseCase(gormDB *gorm.DB) *RoomUseCase {
	return NewRoomUseCase(
		repository.NewRoomUnitOfWork(gormDB),
		repository.NewUserRepository(gormDB),
		repository.NewRoomRepository(gormDB),
		repository.NewRoomMemberRepository(gormDB),
		repository.NewMessageRepository(gormDB),
		repository.NewMessageReadRepository(gormDB),
		nil,
	)
}

func TestRoomUseCase_GetJoinedRooms_OK(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u1 := testUser(t, gormDB, "join1@example.com")
	useCase := newRoomUseCase(gormDB)
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)

	rNow := time.Now().UTC()
	r := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("G"), CreatedBy: u1.ID, CreatedAt: rNow, UpdatedAt: rNow}
	require.NoError(t, rooms.Create(ctx, r))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: r.ID, UserID: u1.ID, JoinedAt: rNow}))

	list, err := useCase.GetJoinedRooms(ctx, u1.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, r.ID, list[0].ID)
}

func TestRoomUseCase_GetJoinedRooms_InvalidUser(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.GetJoinedRooms(ctx, uuid.Nil)
	require.ErrorIs(t, err, ErrInvalidUserID)
}

func TestRoomUseCase_CreateOrGetDirectRoom_Existing(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	a := testUser(t, gormDB, "d1a@example.com")
	b := testUser(t, gormDB, "d1b@example.com")
	useCase := newRoomUseCase(gormDB)
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)

	pa, pb := entity.OrderedUserPair(a.ID, b.ID)
	dmNow := time.Now().UTC()
	dm := &entity.Room{
		ID: uuid.New(), Type: entity.RoomTypeDirect, Name: nil, CreatedBy: a.ID,
		DirectUserAID: &pa, DirectUserBID: &pb,
		CreatedAt: dmNow, UpdatedAt: dmNow,
	}
	require.NoError(t, rooms.Create(ctx, dm))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: dm.ID, UserID: a.ID, JoinedAt: dmNow}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: dm.ID, UserID: b.ID, JoinedAt: dmNow}))

	got, err := useCase.CreateOrGetDirectRoom(ctx, a.ID, b.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, dm.ID, got.ID)
}

func TestRoomUseCase_CreateOrGetDirectRoom_New(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	a := testUser(t, gormDB, "d2a@example.com")
	b := testUser(t, gormDB, "d2b@example.com")
	useCase := newRoomUseCase(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)

	got, err := useCase.CreateOrGetDirectRoom(ctx, a.ID, b.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, entity.RoomTypeDirect, got.Type)
	require.Equal(t, a.ID, got.CreatedBy)

	n, err := members.CountActiveByRoomID(ctx, got.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	again, err := useCase.CreateOrGetDirectRoom(ctx, b.ID, a.ID)
	require.NoError(t, err)
	require.Equal(t, got.ID, again.ID)
}

func TestRoomUseCase_CreateOrGetDirectRoom_SelfRejected(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	a := testUser(t, gormDB, "self@example.com")
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateOrGetDirectRoom(ctx, a.ID, a.ID)
	require.ErrorIs(t, err, ErrSelfDirect)
}

func TestRoomUseCase_CreateOrGetDirectRoom_NilUser(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	a := testUser(t, gormDB, "nilu@example.com")
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateOrGetDirectRoom(ctx, uuid.Nil, a.ID)
	require.ErrorIs(t, err, ErrInvalidUserID)
}

func TestRoomUseCase_CreateOrGetDirectRoom_UnknownInitiator(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	b := testUser(t, gormDB, "unki@example.com")
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateOrGetDirectRoom(ctx, uuid.New(), b.ID)
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestRoomUseCase_CreateOrGetDirectRoom_UnknownTarget(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	a := testUser(t, gormDB, "unkt@example.com")
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateOrGetDirectRoom(ctx, a.ID, uuid.New())
	require.ErrorIs(t, err, ErrTargetUserNotFound)
}

func TestRoomUseCase_CreateOrGetDirectRoom_InactivePeer(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	a := testUser(t, gormDB, "ina@example.com")
	b := testUserActive(t, gormDB, "inb@example.com", false)
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateOrGetDirectRoom(ctx, a.ID, b.ID)
	require.ErrorIs(t, err, ErrInactiveUser)
}

func TestRoomUseCase_CreateOrGetDirectRoom_InactiveInitiator(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	a := testUserActive(t, gormDB, "ini@example.com", false)
	b := testUser(t, gormDB, "inib@example.com")
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateOrGetDirectRoom(ctx, a.ID, b.ID)
	require.ErrorIs(t, err, ErrInactiveUser)
}

func TestRoomUseCase_CreateGroupRoom_OK(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	owner := testUser(t, gormDB, "go@example.com")
	u2 := testUser(t, gormDB, "g2@example.com")
	u3 := testUser(t, gormDB, "g3@example.com")
	useCase := newRoomUseCase(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)

	got, err := useCase.CreateGroupRoom(ctx, owner.ID, "Team", []uuid.UUID{u2.ID, u3.ID, owner.ID, u2.ID})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, entity.RoomTypeGroup, got.Type)
	require.NotNil(t, got.Name)
	require.Equal(t, "Team", *got.Name)

	list, err := members.ListActiveMembers(ctx, got.ID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	ids := map[uuid.UUID]struct{}{list[0].UserID: {}, list[1].UserID: {}, list[2].UserID: {}}
	_, okOwner := ids[owner.ID]
	_, ok2 := ids[u2.ID]
	_, ok3 := ids[u3.ID]
	require.True(t, okOwner && ok2 && ok3)
}

func TestRoomUseCase_CreateGroupRoom_CreatorOnlyRejected(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	owner := testUser(t, gormDB, "solo@example.com")
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateGroupRoom(ctx, owner.ID, " Solo ", nil)
	require.ErrorIs(t, err, ErrGroupTooFewMembers)
}

func TestRoomUseCase_CreateGroupRoom_NameRequired(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	owner := testUser(t, gormDB, "noname@example.com")
	u2 := testUser(t, gormDB, "nn2@example.com")
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateGroupRoom(ctx, owner.ID, "   ", []uuid.UUID{u2.ID})
	require.ErrorIs(t, err, ErrGroupNameRequired)
}

func TestRoomUseCase_CreateGroupRoom_InvalidCreator(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateGroupRoom(ctx, uuid.Nil, "X", nil)
	require.ErrorIs(t, err, ErrInvalidCreatorID)
}

func TestRoomUseCase_CreateGroupRoom_UnknownMember(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	owner := testUser(t, gormDB, "ogm@example.com")
	u2 := testUser(t, gormDB, "og2@example.com")
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateGroupRoom(ctx, owner.ID, "G", []uuid.UUID{u2.ID, uuid.New()})
	require.ErrorIs(t, err, ErrInvalidGroupMembers)
}

func TestRoomUseCase_CreateGroupRoom_InactiveMember(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	owner := testUser(t, gormDB, "ogm2@example.com")
	u2 := testUserActive(t, gormDB, "ogm2b@example.com", false)
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateGroupRoom(ctx, owner.ID, "G", []uuid.UUID{u2.ID})
	require.ErrorIs(t, err, ErrInactiveUser)
}

func TestRoomUseCase_CreateGroupRoom_UnknownCreator(t *testing.T) {
	ctx := context.Background()
	gormDB := setupUsecaseTestDB(t)
	u2 := testUser(t, gormDB, "ogc2@example.com")
	useCase := newRoomUseCase(gormDB)
	_, err := useCase.CreateGroupRoom(ctx, uuid.New(), "G", []uuid.UUID{u2.ID})
	require.ErrorIs(t, err, ErrUserNotFound)
}

func strPtr(s string) *string { return &s }
