package controller

import (
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
)

// --- メッセージ CRUD / 一覧 ---

// createMessageRequest は POST /rooms/:id/messages（JSON）用。
// content と image_url はどちらか一方以上必須。併用可（キャプション付き画像）。
// image_url は POST /media/images のレスポンス url（/static/uploads/... のみ許可）をそのまま渡す。
type patchMessageRequest struct {
	Content string `json:"content"`
}

type createMessageRequest struct {
	Content           *string    `json:"content"`
	ImageURL          *string    `json:"image_url"`
	FileURL           *string    `json:"file_url"`
	FileName          *string    `json:"file_name"`
	FileContentType   *string    `json:"file_content_type"`
	ParentMessageID   *uuid.UUID `json:"parent_message_id"`
}

type createMessageResponse struct {
	Message messageJSON `json:"message"`
}

type listMessagesResponse struct {
	RoomID   uuid.UUID           `json:"room_id"`
	Messages []listedMessageJSON `json:"messages"`
	Order    string              `json:"order"`
	Limit    int                 `json:"limit"`
	Count    int                 `json:"count"`
}

type listedMessageJSON struct {
	messageJSON
	SenderName string `json:"sender_name"`
	ReadCount  *int64 `json:"read_count,omitempty"`
}

type messageJSON struct {
	ID                uuid.UUID  `json:"id"`
	RoomID            uuid.UUID  `json:"room_id"`
	SenderID          uuid.UUID  `json:"sender_id"`
	ParentMessageID   *uuid.UUID `json:"parent_message_id,omitempty"`
	Content           *string    `json:"content"`
	ImageURL          *string    `json:"image_url"`
	FileURL           *string    `json:"file_url,omitempty"`
	FileName          *string    `json:"file_name,omitempty"`
	FileContentType   *string    `json:"file_content_type,omitempty"`
	IsDeleted         bool       `json:"is_deleted"`
	DeletedAt         *string    `json:"deleted_at,omitempty"`
	CreatedAt         string     `json:"created_at"`
	UpdatedAt         string     `json:"updated_at"`
}

// messageDeletedWSBroadcast は message_deleted の WS イベント用。
type messageDeletedWSBroadcast struct {
	Type          string    `json:"type"`
	SchemaVersion int       `json:"schema_version"`
	RoomID        uuid.UUID `json:"room_id"`
	MessageID     uuid.UUID `json:"message_id"`
}

// messageCreatedWSBroadcast は message_created イベント用 JSON。
type messageCreatedWSBroadcast struct {
	Type          string      `json:"type"`
	SchemaVersion int         `json:"schema_version"`
	RoomID        uuid.UUID   `json:"room_id"`
	Message       messageJSON `json:"message"`
}

// messageUpdatedWSBroadcast は message_updated イベント用。
type messageUpdatedWSBroadcast struct {
	Type          string      `json:"type"`
	SchemaVersion int         `json:"schema_version"`
	RoomID        uuid.UUID   `json:"room_id"`
	Message       messageJSON `json:"message"`
}

func listedMessageToJSON(m ListedMessage) listedMessageJSON {
	return listedMessageJSON{
		messageJSON: messageToJSON(&m.Message),
		SenderName:  m.SenderName,
		ReadCount:   m.ReadCount,
	}
}

func messageToJSON(m *entity.Message) messageJSON {
	var deleted *string
	if m.DeletedAt != nil {
		s := m.DeletedAt.UTC().Format(time.RFC3339Nano)
		deleted = &s
	}
	var parent *uuid.UUID
	if m.ParentMessageID != nil && *m.ParentMessageID != uuid.Nil {
		parent = m.ParentMessageID
	}
	return messageJSON{
		ID:              m.ID,
		RoomID:          m.RoomID,
		SenderID:        m.SenderID,
		ParentMessageID: parent,
		Content:         copyStrPtr(m.Content),
		ImageURL:        copyStrPtr(m.ImageURL),
		FileURL:         copyStrPtr(m.FileURL),
		FileName:        copyStrPtr(m.FileName),
		FileContentType: copyStrPtr(m.FileContentType),
		IsDeleted:       m.IsDeleted || m.DeletedAt != nil,
		DeletedAt:       deleted,
		CreatedAt:       m.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:       m.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func copyStrPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

// --- 既読 ---

type markAsReadRequest struct {
	MessageID string `json:"message_id"`
}

// readUpdatedPayload は read_updated イベント用 JSON。
type readUpdatedPayload struct {
	Type              string    `json:"type"`
	SchemaVersion     int       `json:"schema_version"`
	RoomID            uuid.UUID `json:"room_id"`
	UserID            uuid.UUID `json:"user_id"`
	LastReadMessageID uuid.UUID `json:"last_read_message_id"`
}

type memberReadBriefJSON struct {
	UserID            uuid.UUID  `json:"user_id"`
	UserName          string     `json:"user_name"`
	LastReadMessageID *uuid.UUID `json:"last_read_message_id,omitempty"`
}

type roomReadStateJSON struct {
	RoomID              uuid.UUID             `json:"room_id"`
	MyLastReadMessageID *uuid.UUID            `json:"my_last_read_message_id,omitempty"`
	UnreadCount         int64                 `json:"unread_count"`
	MemberReads         []memberReadBriefJSON `json:"member_reads"`
}

func roomReadStateToJSON(s *RoomReadState) roomReadStateJSON {
	if s == nil {
		return roomReadStateJSON{}
	}
	members := make([]memberReadBriefJSON, 0, len(s.MemberReads))
	for _, m := range s.MemberReads {
		members = append(members, memberReadBriefJSON{
			UserID:            m.UserID,
			UserName:          m.UserName,
			LastReadMessageID: m.LastReadMessageID,
		})
	}
	return roomReadStateJSON{
		RoomID:              s.RoomID,
		MyLastReadMessageID: s.MyLastReadMessageID,
		UnreadCount:         s.UnreadCount,
		MemberReads:         members,
	}
}
