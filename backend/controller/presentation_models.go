package controller

import (
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
)

// HTTP 層が扱う閲覧用モデル（usecase パッケージの DTO とは別型にし、ポートと JSON 変換の依存向きを揃える）。

// DefaultJoinedRoomsLimit / メッセージ一覧 limit は usecase と同値（変更時は両方合わせること）。
const (
	DefaultJoinedRoomsLimit = 200
	DefaultMessageListLimit = 50
	MaxMessageListLimit     = 100
)

// JoinedRoomSummary は参加ルーム一覧1行分。
type JoinedRoomSummary struct {
	Room                  entity.Room
	LastMessage           *entity.Message
	LastPreview           *string
	LastMessageSenderName *string
	UnreadCount           int64
	PeerOnline            *bool
	DirectPeerName        *string
	DirectPeerID          *uuid.UUID
}

// RoomMemberBrief はルーム詳細のメンバー表示用。
type RoomMemberBrief struct {
	UserID   uuid.UUID
	Name     string
	JoinedAt time.Time
	Online   bool
}

// RoomDetail はルーム詳細 API 向け。
type RoomDetail struct {
	Room                entity.Room
	Members             []RoomMemberBrief
	DirectPeer          *RoomMemberBrief
	MyLastReadMessageID *uuid.UUID
	UnreadCount         int64
}

// ListedMessage はメッセージ一覧1件。
type ListedMessage struct {
	Message    entity.Message
	SenderName string
	ReadCount  *int64
}

// MarkAsReadResult は既読更新の結果。
type MarkAsReadResult struct {
	Updated           bool
	LastReadMessageID uuid.UUID
}

// MemberReadBrief はルーム内メンバーの最終既読概要。
type MemberReadBrief struct {
	UserID            uuid.UUID
	UserName          string
	LastReadMessageID *uuid.UUID
}

// RoomReadState は既読／未読集計。
type RoomReadState struct {
	RoomID              uuid.UUID
	MyLastReadMessageID *uuid.UUID
	UnreadCount         int64
	MemberReads         []MemberReadBrief
}
