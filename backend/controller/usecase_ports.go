package controller

import (
	"context"
	"io"

	"chat-app/entity"
	"chat-app/repository"
	"chat-app/usecase"

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
	SendRoomMessage(ctx context.Context, in usecase.MessageSendInput) (*entity.Message, error)
	EditMessage(ctx context.Context, messageID, editorID uuid.UUID, content *string) (*entity.Message, error)
	SearchJoined(ctx context.Context, userID uuid.UUID, query string, limit int) ([]ListedMessage, error)
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

// AuthUseCase は認証系 API から参照する操作
type AuthUseCase interface {
	Login(ctx context.Context, email, password string) (string, error)
	Register(ctx context.Context, name, email, password, bootstrapAdminEmail string) (*usecase.RegisterResult, error)
	CreateTokenForUser(ctx context.Context, userID uuid.UUID) (string, error)
	CurrentUser(ctx context.Context, userID uuid.UUID) (*entity.User, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, plainToken, newPassword string) error
	VerifyEmail(ctx context.Context, plainToken string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
}

// ProfileUseCase はプロフィール更新 API から参照する操作
type ProfileUseCase interface {
	UpdateProfile(ctx context.Context, userID uuid.UUID, in usecase.UpdateProfileInput) (*entity.User, error)
}

// SocialUseCase はリアクション・ピン・ブックマーク API から参照する操作
type SocialUseCase interface {
	AddReaction(ctx context.Context, roomID, userID, messageID uuid.UUID, emoji string) error
	RemoveReaction(ctx context.Context, roomID, userID, messageID uuid.UUID, emoji string) error
	ListReactions(ctx context.Context, roomID, userID, messageID uuid.UUID) ([]repository.MessageReaction, error)
	SetPin(ctx context.Context, roomID, actorID, messageID uuid.UUID) error
	ClearPin(ctx context.Context, roomID, actorID uuid.UUID) error
	GetPin(ctx context.Context, roomID, userID uuid.UUID) (*repository.RoomPin, error)
	AddBookmark(ctx context.Context, userID, messageID uuid.UUID) error
	RemoveBookmark(ctx context.Context, userID, messageID uuid.UUID) error
	ListBookmarksWithMessages(ctx context.Context, userID uuid.UUID, limit, offset int) ([]usecase.BookmarkWithMessage, int64, error)
	BlockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error
	UnblockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error
	IsBlockedUser(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error)
}

// MediaUseCase は画像・ファイルアップロード保存
type MediaUseCase interface {
	SaveUploadedImage(contentType string, r io.Reader) (urlPath string, err error)
	SaveUploadedFile(contentType string, originalFilename string, r io.Reader) (urlPath string, err error)
}
