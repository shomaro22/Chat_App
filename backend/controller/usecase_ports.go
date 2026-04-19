package controller

import (
	"context"
	"io"

	"chat-app/entity"

	"github.com/google/uuid"
)

// RoomUseCase はルーム系ユースケース
type RoomUseCase interface {
	GetJoinedRooms(ctx context.Context, userID uuid.UUID) ([]entity.Room, error)
	GetJoinedRoomSummaries(ctx context.Context, userID uuid.UUID, limit, offset int, updatedDesc bool) ([]JoinedRoomSummary, error)
	GetRoomDetail(ctx context.Context, roomID, requesterID uuid.UUID) (*RoomDetail, error)
	AddGroupMember(ctx context.Context, roomID, actorID, newUserID uuid.UUID) error
	RemoveRoomMember(ctx context.Context, roomID, actorID, targetUserID uuid.UUID) error
	CreateOrGetDirectRoom(ctx context.Context, userID, targetUserID uuid.UUID) (*entity.Room, error)
	CreateGroupRoom(ctx context.Context, createdBy uuid.UUID, name string, memberIDs []uuid.UUID) (*entity.Room, error)
	ResolveMemberRef(ctx context.Context, ref string) (uuid.UUID, error)
	UpdateGroupName(ctx context.Context, roomID, actorID uuid.UUID, name string) error
	CloseRoom(ctx context.Context, roomID, actorID uuid.UUID) error
	IsActiveRoomMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	CanSubscribeWebSocket(ctx context.Context, roomID, userID uuid.UUID) error
}

// MessageUseCase はメッセージ送受信・削除
type MessageUseCase interface {
	SendMessage(ctx context.Context, roomID, senderID uuid.UUID, content, imageURL *string) (*entity.Message, error)
	ListMessages(ctx context.Context, roomID, userID uuid.UUID, beforeID, afterID *uuid.UUID, limit int, orderDesc, includeReadCount bool) ([]ListedMessage, error)
	DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) (uuid.UUID, error)
}

// MessageReadUseCase は既読更新・集計
type MessageReadUseCase interface {
	MarkAsRead(ctx context.Context, userID, roomID, messageID uuid.UUID) (*MarkAsReadResult, error)
	GetRoomReadState(ctx context.Context, userID, roomID uuid.UUID) (*RoomReadState, error)
}

// PresenceJoinedRooms はプレゼンス配信先ルーム一覧の解決用
type PresenceJoinedRooms interface {
	JoinedRoomIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// AdminUseCase は管理者 API から参照する操作
type AdminUseCase interface {
	ListUsers(ctx context.Context, adminID uuid.UUID, search string, limit, offset int) ([]entity.User, int64, error)
	ListRooms(ctx context.Context, adminID uuid.UUID, search string, limit, offset int) ([]entity.Room, int64, error)
	SetUserActive(ctx context.Context, adminID, targetUserID uuid.UUID, active bool) error
	SetRoomFrozen(ctx context.Context, adminID, roomID uuid.UUID, frozen bool) error
	ForceRemoveMember(ctx context.Context, adminID, roomID, targetUserID uuid.UUID) error
}

// MediaUseCase は画像・ファイルアップロード保存
type MediaUseCase interface {
	SaveUploadedImage(contentType string, r io.Reader) (urlPath string, err error)
	SaveUploadedFile(contentType string, originalFilename string, r io.Reader) (urlPath string, err error)
}
