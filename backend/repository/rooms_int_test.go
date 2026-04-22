//go:build integration

package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRoom_ListJoinedRooms_OrderAndFilter(t *testing.T) {
	ctx := context.Background()
	gormDB := setupDB(t)
	users := NewUserRepository(gormDB)
	rooms := NewRoomRepository(gormDB)
	members := NewRoomMemberRepository(gormDB)

	u1 := newTestUser("u1@example.com")
	u2 := newTestUser("u2@example.com")
	require.NoError(t, users.Create(ctx, u1))
	require.NoError(t, users.Create(ctx, u2))

	t0 := time.Now().UTC()
	rA := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("A"), CreatedBy: u1.ID, CreatedAt: t0, UpdatedAt: t0}
	require.NoError(t, rooms.Create(ctx, rA))
	time.Sleep(5 * time.Millisecond)
	t1 := time.Now().UTC()
	rB := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("B"), CreatedBy: u1.ID, CreatedAt: t1, UpdatedAt: t1}
	require.NoError(t, rooms.Create(ctx, rB))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: rA.ID, UserID: u1.ID, JoinedAt: t0}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: rB.ID, UserID: u1.ID, JoinedAt: t1}))

	list, err := rooms.ListJoinedRooms(ctx, u1.ID, 50, 0, true)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, rB.ID, list[0].ID, "newer updated_at should be first")
	require.Equal(t, rA.ID, list[1].ID)

	leftOnly, err := rooms.ListJoinedRooms(ctx, u2.ID, 50, 0, true)
	require.NoError(t, err)
	require.Empty(t, leftOnly)
}

func TestRoomMember_IsActiveMember_FindByIDForActiveMember_Leave(t *testing.T) {
	ctx := context.Background()
	gormDB := setupDB(t)
	users := NewUserRepository(gormDB)
	rooms := NewRoomRepository(gormDB)
	members := NewRoomMemberRepository(gormDB)

	u := newTestUser("member@example.com")
	require.NoError(t, users.Create(ctx, u))
	rmNow := time.Now().UTC()
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g"), CreatedBy: u.ID, CreatedAt: rmNow, UpdatedAt: rmNow}
	require.NoError(t, rooms.Create(ctx, room))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u.ID, JoinedAt: rmNow}))

	ok, err := members.IsActiveMember(ctx, room.ID, u.ID)
	require.NoError(t, err)
	require.True(t, ok)

	got, err := rooms.FindByIDForActiveMember(ctx, room.ID, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, room.ID, got.ID)

	otherUser := newTestUser("other@example.com")
	require.NoError(t, users.Create(ctx, otherUser))
	nope, err := rooms.FindByIDForActiveMember(ctx, room.ID, otherUser.ID)
	require.NoError(t, err)
	require.Nil(t, nope)

	leftAt := time.Now().UTC()
	require.NoError(t, members.Leave(ctx, room.ID, u.ID, leftAt))

	ok2, err := members.IsActiveMember(ctx, room.ID, u.ID)
	require.NoError(t, err)
	require.False(t, ok2)

	afterLeave, err := rooms.FindByIDForActiveMember(ctx, room.ID, u.ID)
	require.NoError(t, err)
	require.Nil(t, afterLeave)

	err = members.Leave(ctx, room.ID, u.ID, time.Now().UTC())
	require.Error(t, err)
	require.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestRoomMember_ListActiveMembers(t *testing.T) {
	ctx := context.Background()
	gormDB := setupDB(t)
	users := NewUserRepository(gormDB)
	rooms := NewRoomRepository(gormDB)
	members := NewRoomMemberRepository(gormDB)

	u1 := newTestUser("p1@example.com")
	u2 := newTestUser("p2@example.com")
	require.NoError(t, users.Create(ctx, u1))
	require.NoError(t, users.Create(ctx, u2))
	partyNow := time.Now().UTC()
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("party"), CreatedBy: u1.ID, CreatedAt: partyNow, UpdatedAt: partyNow}
	require.NoError(t, rooms.Create(ctx, room))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u1.ID, JoinedAt: partyNow}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u2.ID, JoinedAt: partyNow}))

	list, err := members.ListActiveMembers(ctx, room.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	got := map[uuid.UUID]struct{}{
		list[0].UserID: {},
		list[1].UserID: {},
	}
	_, hasU1 := got[u1.ID]
	_, hasU2 := got[u2.ID]
	require.True(t, hasU1)
	require.True(t, hasU2)

	n, err := members.CountActiveByRoomID(ctx, room.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
}

func TestRoom_FindDirectRoomByUserPair(t *testing.T) {
	ctx := context.Background()
	gormDB := setupDB(t)
	users := NewUserRepository(gormDB)
	rooms := NewRoomRepository(gormDB)
	members := NewRoomMemberRepository(gormDB)

	a := newTestUser("alice@example.com")
	b := newTestUser("bob@example.com")
	require.NoError(t, users.Create(ctx, a))
	require.NoError(t, users.Create(ctx, b))

	name := "dm"
	pa, pb := entity.OrderedUserPair(a.ID, b.ID)
	dmNow := time.Now().UTC()
	direct := &entity.Room{
		ID: uuid.New(), Type: entity.RoomTypeDirect, Name: &name, CreatedBy: a.ID,
		DirectUserAID: &pa, DirectUserBID: &pb,
		CreatedAt: dmNow, UpdatedAt: dmNow,
	}
	require.NoError(t, rooms.Create(ctx, direct))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: direct.ID, UserID: a.ID, JoinedAt: dmNow}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: direct.ID, UserID: b.ID, JoinedAt: dmNow}))

	foundAB, err := rooms.FindDirectRoomByUserPair(ctx, a.ID, b.ID)
	require.NoError(t, err)
	require.NotNil(t, foundAB)
	require.Equal(t, direct.ID, foundAB.ID)
	require.Equal(t, entity.RoomTypeDirect, foundAB.Type)

	foundBA, err := rooms.FindDirectRoomByUserPair(ctx, b.ID, a.ID)
	require.NoError(t, err)
	require.NotNil(t, foundBA)
	require.Equal(t, direct.ID, foundBA.ID)

	self, err := rooms.FindDirectRoomByUserPair(ctx, a.ID, a.ID)
	require.NoError(t, err)
	require.Nil(t, self)
}

func TestRoom_FindDirectRoomByUserPair_GroupOrThirdMemberExcluded(t *testing.T) {
	ctx := context.Background()
	gormDB := setupDB(t)
	users := NewUserRepository(gormDB)
	rooms := NewRoomRepository(gormDB)
	members := NewRoomMemberRepository(gormDB)

	a := newTestUser("ga@example.com")
	b := newTestUser("gb@example.com")
	c := newTestUser("gc@example.com")
	require.NoError(t, users.Create(ctx, a))
	require.NoError(t, users.Create(ctx, b))
	require.NoError(t, users.Create(ctx, c))

	gNow := time.Now().UTC()
	group := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("trip"), CreatedBy: a.ID, CreatedAt: gNow, UpdatedAt: gNow}
	require.NoError(t, rooms.Create(ctx, group))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: group.ID, UserID: a.ID, JoinedAt: gNow}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: group.ID, UserID: b.ID, JoinedAt: gNow}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: group.ID, UserID: c.ID, JoinedAt: gNow}))

	nilGroup, err := rooms.FindDirectRoomByUserPair(ctx, a.ID, b.ID)
	require.NoError(t, err)
	require.Nil(t, nilGroup)

	pda, pdb := entity.OrderedUserPair(a.ID, b.ID)
	dmT := time.Now().UTC()
	dm := &entity.Room{
		ID: uuid.New(), Type: entity.RoomTypeDirect, Name: nil, CreatedBy: a.ID,
		DirectUserAID: &pda, DirectUserBID: &pdb,
		CreatedAt: dmT, UpdatedAt: dmT,
	}
	require.NoError(t, rooms.Create(ctx, dm))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: dm.ID, UserID: a.ID, JoinedAt: dmT}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: dm.ID, UserID: b.ID, JoinedAt: dmT}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: dm.ID, UserID: c.ID, JoinedAt: dmT}))

	nilThree, err := rooms.FindDirectRoomByUserPair(ctx, a.ID, b.ID)
	require.NoError(t, err)
	require.Nil(t, nilThree, "3 active members should not count as 1:1 direct")

	require.NoError(t, members.Leave(ctx, dm.ID, c.ID, time.Now().UTC()))
	found, err := rooms.FindDirectRoomByUserPair(ctx, a.ID, b.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, dm.ID, found.ID)
}

func newTestUser(email string) *entity.User {
	now := time.Now().UTC()
	return &entity.User{
		ID:           uuid.New(),
		Name:         email,
		Email:        email,
		PasswordHash: "x",
		Role:         entity.UserRoleUser,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
