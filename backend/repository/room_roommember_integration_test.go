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

func TestRoom_ListActiveByUserID_OrderAndFilter(t *testing.T) {
	ctx := context.Background()
	gdb := setupDB(t)
	users := NewUserRepository(gdb)
	rooms := NewRoomRepository(gdb)
	members := NewRoomMemberRepository(gdb)

	u1 := newTestUser("u1@example.com")
	u2 := newTestUser("u2@example.com")
	require.NoError(t, users.Create(ctx, u1))
	require.NoError(t, users.Create(ctx, u2))

	rA := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("A"), CreatedBy: u1.ID}
	rB := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("B"), CreatedBy: u1.ID}
	require.NoError(t, rooms.Create(ctx, rA))
	time.Sleep(5 * time.Millisecond)
	require.NoError(t, rooms.Create(ctx, rB))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: rA.ID, UserID: u1.ID}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: rB.ID, UserID: u1.ID}))

	list, err := rooms.ListActiveByUserID(ctx, u1.ID, 50, 0)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, rB.ID, list[0].ID, "newer updated_at should be first")
	require.Equal(t, rA.ID, list[1].ID)

	leftOnly, err := rooms.ListActiveByUserID(ctx, u2.ID, 50, 0)
	require.NoError(t, err)
	require.Empty(t, leftOnly)
}

func TestRoomMember_IsActiveMember_FindByIDForActiveMember_Leave(t *testing.T) {
	ctx := context.Background()
	gdb := setupDB(t)
	users := NewUserRepository(gdb)
	rooms := NewRoomRepository(gdb)
	members := NewRoomMemberRepository(gdb)

	u := newTestUser("member@example.com")
	require.NoError(t, users.Create(ctx, u))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("g"), CreatedBy: u.ID}
	require.NoError(t, rooms.Create(ctx, room))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u.ID}))

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

	require.NoError(t, members.Leave(ctx, room.ID, u.ID))

	ok2, err := members.IsActiveMember(ctx, room.ID, u.ID)
	require.NoError(t, err)
	require.False(t, ok2)

	afterLeave, err := rooms.FindByIDForActiveMember(ctx, room.ID, u.ID)
	require.NoError(t, err)
	require.Nil(t, afterLeave)

	err = members.Leave(ctx, room.ID, u.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestRoomMember_ListActiveByRoomID(t *testing.T) {
	ctx := context.Background()
	gdb := setupDB(t)
	users := NewUserRepository(gdb)
	rooms := NewRoomRepository(gdb)
	members := NewRoomMemberRepository(gdb)

	u1 := newTestUser("p1@example.com")
	u2 := newTestUser("p2@example.com")
	require.NoError(t, users.Create(ctx, u1))
	require.NoError(t, users.Create(ctx, u2))
	room := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("party"), CreatedBy: u1.ID}
	require.NoError(t, rooms.Create(ctx, room))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u1.ID}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: room.ID, UserID: u2.ID}))

	list, err := members.ListActiveByRoomID(ctx, room.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, u1.ID, list[0].UserID)
	require.Equal(t, u2.ID, list[1].UserID)

	n, err := members.CountActiveByRoomID(ctx, room.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
}

func TestRoom_FindDirectRoomBetweenUsers(t *testing.T) {
	ctx := context.Background()
	gdb := setupDB(t)
	users := NewUserRepository(gdb)
	rooms := NewRoomRepository(gdb)
	members := NewRoomMemberRepository(gdb)

	a := newTestUser("alice@example.com")
	b := newTestUser("bob@example.com")
	require.NoError(t, users.Create(ctx, a))
	require.NoError(t, users.Create(ctx, b))

	name := "dm"
	direct := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeDirect, Name: &name, CreatedBy: a.ID}
	require.NoError(t, rooms.Create(ctx, direct))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: direct.ID, UserID: a.ID}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: direct.ID, UserID: b.ID}))

	foundAB, err := rooms.FindDirectRoomBetweenUsers(ctx, a.ID, b.ID)
	require.NoError(t, err)
	require.NotNil(t, foundAB)
	require.Equal(t, direct.ID, foundAB.ID)
	require.Equal(t, entity.RoomTypeDirect, foundAB.Type)

	foundBA, err := rooms.FindDirectRoomBetweenUsers(ctx, b.ID, a.ID)
	require.NoError(t, err)
	require.NotNil(t, foundBA)
	require.Equal(t, direct.ID, foundBA.ID)

	// self: no match
	self, err := rooms.FindDirectRoomBetweenUsers(ctx, a.ID, a.ID)
	require.NoError(t, err)
	require.Nil(t, self)
}

func TestRoom_FindDirectRoomBetweenUsers_GroupOrThirdMemberExcluded(t *testing.T) {
	ctx := context.Background()
	gdb := setupDB(t)
	users := NewUserRepository(gdb)
	rooms := NewRoomRepository(gdb)
	members := NewRoomMemberRepository(gdb)

	a := newTestUser("ga@example.com")
	b := newTestUser("gb@example.com")
	c := newTestUser("gc@example.com")
	require.NoError(t, users.Create(ctx, a))
	require.NoError(t, users.Create(ctx, b))
	require.NoError(t, users.Create(ctx, c))

	group := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeGroup, Name: strPtr("trip"), CreatedBy: a.ID}
	require.NoError(t, rooms.Create(ctx, group))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: group.ID, UserID: a.ID}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: group.ID, UserID: b.ID}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: group.ID, UserID: c.ID}))

	nilGroup, err := rooms.FindDirectRoomBetweenUsers(ctx, a.ID, b.ID)
	require.NoError(t, err)
	require.Nil(t, nilGroup)

	dm := &entity.Room{ID: uuid.New(), Type: entity.RoomTypeDirect, Name: nil, CreatedBy: a.ID}
	require.NoError(t, rooms.Create(ctx, dm))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: dm.ID, UserID: a.ID}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: dm.ID, UserID: b.ID}))
	require.NoError(t, members.Create(ctx, &entity.RoomMember{ID: uuid.New(), RoomID: dm.ID, UserID: c.ID}))

	nilThree, err := rooms.FindDirectRoomBetweenUsers(ctx, a.ID, b.ID)
	require.NoError(t, err)
	require.Nil(t, nilThree, "3 active members should not count as 1:1 direct")

	require.NoError(t, members.Leave(ctx, dm.ID, c.ID))
	found, err := rooms.FindDirectRoomBetweenUsers(ctx, a.ID, b.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, dm.ID, found.ID)
}

func newTestUser(email string) *entity.User {
	return &entity.User{
		ID:           uuid.New(),
		Name:         email,
		Email:        email,
		PasswordHash: "x",
		Role:         entity.UserRoleUser,
		Active:       true,
	}
}
