package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakePresenceFanout struct {
	roomIDs [][]uuid.UUID
}

func (f *fakePresenceFanout) BroadcastToUsersInRooms(roomIDs []uuid.UUID, payload []byte) {
	cp := make([]uuid.UUID, len(roomIDs))
	copy(cp, roomIDs)
	f.roomIDs = append(f.roomIDs, cp)
}

type fakeJoinedRooms struct {
	ids []uuid.UUID
	err error
}

func (f *fakeJoinedRooms) JoinedRoomIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.ids, nil
}

func TestBroadcastPresenceUpdated_NoPanicOnResolverError(t *testing.T) {
	h := &fakePresenceFanout{}
	r := &fakeJoinedRooms{err: errors.New("db")}
	uid := uuid.New()
	require.NotPanics(t, func() {
		BroadcastPresenceUpdated(h, r, context.Background(), uid, true)
	})
	require.Empty(t, h.roomIDs)
}

func TestBroadcastPresenceUpdated_NoBroadcastWhenEmptyRooms(t *testing.T) {
	h := &fakePresenceFanout{}
	r := &fakeJoinedRooms{ids: nil}
	BroadcastPresenceUpdated(h, r, context.Background(), uuid.New(), false)
	require.Empty(t, h.roomIDs)
}

func TestBroadcastPresenceUpdated_SingleFanOutCallWithAllRooms(t *testing.T) {
	h := &fakePresenceFanout{}
	r1 := uuid.New()
	r2 := uuid.New()
	r := &fakeJoinedRooms{ids: []uuid.UUID{r1, r2}}
	uid := uuid.New()
	BroadcastPresenceUpdated(h, r, context.Background(), uid, true)
	require.Len(t, h.roomIDs, 1)
	require.Equal(t, []uuid.UUID{r1, r2}, h.roomIDs[0])
}

func TestBroadcastPresenceUpdated_NilHubOrResolver(t *testing.T) {
	require.NotPanics(t, func() {
		BroadcastPresenceUpdated(nil, &fakeJoinedRooms{ids: []uuid.UUID{uuid.New()}}, context.Background(), uuid.New(), true)
	})
	h := &fakePresenceFanout{}
	require.NotPanics(t, func() {
		BroadcastPresenceUpdated(h, nil, context.Background(), uuid.New(), true)
	})
	require.Empty(t, h.roomIDs)
}
