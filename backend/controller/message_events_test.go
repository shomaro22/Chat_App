package controller

import (
	"context"
	"testing"
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type eventHubStub struct {
	roomID      uuid.UUID
	payload     []byte
	exRoomID    uuid.UUID
	exPayload   []byte
	excludedIDs []uuid.UUID
}

func (h *eventHubStub) BroadcastToRoom(roomID uuid.UUID, payload []byte) {
	h.roomID = roomID
	h.payload = append([]byte(nil), payload...)
}

func (h *eventHubStub) BroadcastToRoomExceptUsers(roomID uuid.UUID, payload []byte, excludeUserIDs []uuid.UUID) {
	h.exRoomID = roomID
	h.exPayload = append([]byte(nil), payload...)
	h.excludedIDs = append([]uuid.UUID(nil), excludeUserIDs...)
}

type eventBlockStub struct {
	ids []uuid.UUID
}

func (s *eventBlockStub) ListBlockerUserIDsInRoomForBlocked(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]uuid.UUID, error) {
	return append([]uuid.UUID(nil), s.ids...), nil
}

func TestMessageEventPublisher_Created_ExcludesBlockers(t *testing.T) {
	hub := &eventHubStub{}
	blockedBy := uuid.New()
	pub := NewMessageEventPublisher(hub, &eventBlockStub{ids: []uuid.UUID{blockedBy}})

	msg := &entity.Message{
		ID:        uuid.New(),
		RoomID:    uuid.New(),
		SenderID:  uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	pub.OnMessageCreated(context.Background(), msg)

	require.Equal(t, msg.RoomID, hub.exRoomID)
	require.NotEmpty(t, hub.exPayload)
	require.Equal(t, []uuid.UUID{blockedBy}, hub.excludedIDs)
	require.Empty(t, hub.payload, "fallback broadcast should not be used when exclusion-capable hub is available")
}

func TestMessageEventPublisher_Created_FallbackBroadcast(t *testing.T) {
	hub := &eventHubStub{}
	pub := &MessageEventPublisher{Hub: roomOnlyHub{hub}}

	msg := &entity.Message{
		ID:        uuid.New(),
		RoomID:    uuid.New(),
		SenderID:  uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	pub.OnMessageCreated(context.Background(), msg)

	require.Equal(t, msg.RoomID, hub.roomID)
	require.NotEmpty(t, hub.payload)
}

type roomOnlyHub struct {
	inner *eventHubStub
}

func (h roomOnlyHub) BroadcastToRoom(roomID uuid.UUID, payload []byte) {
	h.inner.BroadcastToRoom(roomID, payload)
}
