package controller

import (
	"context"
	"encoding/json"

	"chat-app/entity"

	"github.com/google/uuid"
)

// MessageEventPublisher は MessageUseCase から呼ばれる WS 通知実装。
type MessageEventPublisher struct {
	Hub    RoomBroadcastHub
	Blocks MessageBlockLookup
}

type MessageBlockLookup interface {
	ListBlockerUserIDsInRoomForBlocked(ctx context.Context, roomID, blockedID uuid.UUID) ([]uuid.UUID, error)
}

type roomBroadcastHubWithExclusion interface {
	BroadcastToRoomExceptUsers(roomID uuid.UUID, payload []byte, excludeUserIDs []uuid.UUID)
}

func NewMessageEventPublisher(hub RoomBroadcastHub, blocks MessageBlockLookup) *MessageEventPublisher {
	return &MessageEventPublisher{Hub: hub, Blocks: blocks}
}

func (p *MessageEventPublisher) OnMessageCreated(ctx context.Context, msg *entity.Message) {
	if p == nil || p.Hub == nil || msg == nil {
		return
	}
	payload := messageCreatedWSBroadcast{
		Type:          WSEventMessageCreated,
		SchemaVersion: WSEventSchemaVersion,
		RoomID:        msg.RoomID,
		Message:       messageToJSON(msg),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	p.broadcast(ctx, msg.RoomID, msg.SenderID, b)
}

func (p *MessageEventPublisher) OnMessageUpdated(ctx context.Context, msg *entity.Message) {
	if p == nil || p.Hub == nil || msg == nil {
		return
	}
	payload := messageUpdatedWSBroadcast{
		Type:          WSEventMessageUpdated,
		SchemaVersion: WSEventSchemaVersion,
		RoomID:        msg.RoomID,
		Message:       messageToJSON(msg),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	p.broadcast(ctx, msg.RoomID, msg.SenderID, b)
}

func (p *MessageEventPublisher) OnMessageDeleted(ctx context.Context, roomID, messageID, senderID uuid.UUID) {
	if p == nil || p.Hub == nil || roomID == uuid.Nil {
		return
	}
	b, err := json.Marshal(messageDeletedWSBroadcast{
		Type:          WSEventMessageDeleted,
		SchemaVersion: WSEventSchemaVersion,
		RoomID:        roomID,
		MessageID:     messageID,
	})
	if err != nil {
		return
	}
	p.broadcast(ctx, roomID, senderID, b)
}

func (p *MessageEventPublisher) broadcast(ctx context.Context, roomID, senderID uuid.UUID, payload []byte) {
	if p == nil || p.Hub == nil {
		return
	}
	var excluded []uuid.UUID
	if p.Blocks != nil && senderID != uuid.Nil {
		if ids, err := p.Blocks.ListBlockerUserIDsInRoomForBlocked(ctx, roomID, senderID); err == nil {
			excluded = ids
		}
	}
	if len(excluded) > 0 {
		if h, ok := p.Hub.(roomBroadcastHubWithExclusion); ok {
			h.BroadcastToRoomExceptUsers(roomID, payload, excluded)
			return
		}
	}
	p.Hub.BroadcastToRoom(roomID, payload)
}
