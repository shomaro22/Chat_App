package controller

import (
	"context"

	"chat-app/entity"
	"chat-app/usecase"

	"github.com/google/uuid"
)

type RoomUseCase interface {
	GetJoinedRooms(ctx context.Context, userID uuid.UUID) ([]entity.Room, error)
	CreateOrGetDirectRoom(ctx context.Context, userID, targetUserID uuid.UUID) (*entity.Room, error)
	CreateGroupRoom(ctx context.Context, createdBy uuid.UUID, name string, memberIDs []uuid.UUID) (*entity.Room, error)
	IsActiveRoomMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
}

// MessageUseCase はメッセージ送受信・削除
type MessageUseCase interface {
	SendMessage(ctx context.Context, roomID, senderID uuid.UUID, content, imageURL *string) (*entity.Message, error)
	ListMessages(ctx context.Context, roomID, userID uuid.UUID, beforeID *uuid.UUID, limit int) ([]entity.Message, error)
	DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) error
}

// MessageReadUseCase は既読更新
type MessageReadUseCase interface {
	MarkAsRead(ctx context.Context, userID, roomID, messageID uuid.UUID) (*usecase.MarkAsReadResult, error)
}

// PresenceJoinedRooms はプレゼンス配信先ルーム一覧の解決用
type PresenceJoinedRooms interface {
	JoinedRoomIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
