package controller

import (
	"context"
	"io"

	"chat-app/entity"
	"chat-app/usecase"

	"github.com/google/uuid"
)

// RoomUseCase はルーム系ユースケースの抽象。
type RoomUseCase interface {
	GetJoinedRooms(ctx context.Context, userID uuid.UUID) ([]entity.Room, error)
	GetJoinedRoomSummaries(ctx context.Context, userID uuid.UUID, limit, offset int, updatedDesc bool) ([]usecase.JoinedRoomSummary, error)
	GetRoomDetail(ctx context.Context, roomID, requesterID uuid.UUID) (*usecase.RoomDetail, error)
	AddGroupMember(ctx context.Context, roomID, actorID, newUserID uuid.UUID) error
	RemoveRoomMember(ctx context.Context, roomID, actorID, targetUserID uuid.UUID) error
	CreateOrGetDirectRoom(ctx context.Context, userID, targetUserID uuid.UUID) (*entity.Room, error)
	CreateGroupRoom(ctx context.Context, createdBy uuid.UUID, name string, memberIDs []uuid.UUID) (*entity.Room, error)
	UpdateGroupName(ctx context.Context, roomID, actorID uuid.UUID, name string) error
	CloseRoom(ctx context.Context, roomID, actorID uuid.UUID) error
	IsActiveRoomMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
}

// MessageUseCase はメッセージ送受信・削除の抽象。
type MessageUseCase interface {
	SendMessage(ctx context.Context, roomID, senderID uuid.UUID, content, imageURL *string) (*entity.Message, error)
	ListMessages(ctx context.Context, roomID, userID uuid.UUID, beforeID, afterID *uuid.UUID, limit int, orderDesc, includeReadCount bool) ([]usecase.ListedMessage, error)
	DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) (uuid.UUID, error)
}

// MessageReadUseCase は既読更新・集計の抽象。
type MessageReadUseCase interface {
	MarkAsRead(ctx context.Context, userID, roomID, messageID uuid.UUID) (*usecase.MarkAsReadResult, error)
	GetRoomReadState(ctx context.Context, userID, roomID uuid.UUID) (*usecase.RoomReadState, error)
}

// PresenceJoinedRooms はプレゼンス配信先ルーム一覧の解決用抽象。
type PresenceJoinedRooms interface {
	JoinedRoomIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// AdminUseCase は管理者 API から参照する操作の抽象。
type AdminUseCase interface {
	SetUserActive(ctx context.Context, adminID, targetUserID uuid.UUID, active bool) error
	SetRoomFrozen(ctx context.Context, adminID, roomID uuid.UUID, frozen bool) error
	ForceRemoveMember(ctx context.Context, adminID, roomID, targetUserID uuid.UUID) error
}

// MediaUseCase は画像アップロード保存の抽象。
type MediaUseCase interface {
	SaveUploadedImage(contentType string, r io.Reader) (urlPath string, err error)
}
