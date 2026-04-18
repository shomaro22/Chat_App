package wshub

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestConnectionManager_Presence_FirstOnlineThenLastOffline(t *testing.T) {
	m := NewConnectionManager()
	u := uuid.New()

	id1, ch1 := m.Register(u, nil)
	require.True(t, ch1.Changed)
	require.True(t, ch1.IsOnline)
	require.Equal(t, u, ch1.UserID)

	id2, ch2 := m.Register(u, nil)
	require.False(t, ch2.Changed)

	id3, ch3 := m.Register(u, nil)
	require.False(t, ch3.Changed)

	require.False(t, m.RemoveConn(id1).Changed)
	require.False(t, m.RemoveConn(id2).Changed)

	chLast := m.RemoveConn(id3)
	require.True(t, chLast.Changed)
	require.Equal(t, u, chLast.UserID)
	require.False(t, chLast.IsOnline)
}

func TestConnectionManager_RemoveUnknownConnID_NoOp(t *testing.T) {
	m := NewConnectionManager()
	u := uuid.New()
	_, _ = m.Register(u, nil)
	ch := m.RemoveConn(uuid.New())
	require.False(t, ch.Changed)
}

func TestConnectionManager_RemoveClearsSubscriptions_BroadcastNoPanic(t *testing.T) {
	m := NewConnectionManager()
	u := uuid.New()
	room := uuid.New()
	id, _ := m.Register(u, nil)
	m.Subscribe(id, room)
	m.RemoveConn(id)
	require.NotPanics(t, func() {
		m.BroadcastToRoom(room, []byte(`{}`))
	})
}

func TestConnectionManager_CloseUserConnections_ClearsSubsAndOfflinePresence(t *testing.T) {
	m := NewConnectionManager()
	u := uuid.New()
	room := uuid.New()
	id, _ := m.Register(u, nil)
	m.Subscribe(id, room)
	ch := m.CloseUserConnections(u)
	require.True(t, ch.Changed)
	require.False(t, ch.IsOnline)
	require.Equal(t, u, ch.UserID)
	require.NotPanics(t, func() { m.BroadcastToRoom(room, []byte(`{}`)) })
}

func TestConnectionManager_BroadcastOnlyToSubscribedRoom(t *testing.T) {
	m := NewConnectionManager()
	u := uuid.New()
	room := uuid.New()
	id, _ := m.Register(u, nil)
	m.Subscribe(id, room)
	other := uuid.New()
	// No subscribers on other room: must not panic.
	require.NotPanics(t, func() {
		m.BroadcastToRoom(other, []byte(`{}`))
	})
}
