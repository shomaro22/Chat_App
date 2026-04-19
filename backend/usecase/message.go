package usecase

// メッセージと画像・添付ファイルの結合（API 契約）

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

const (
	DefaultMessageListLimit = 50
	MaxMessageListLimit     = 100
)

// ListedMessage は一覧 API 向け（送信者名・任意で既読数）。
type ListedMessage struct {
	Message    entity.Message
	SenderName string
	ReadCount  *int64
}

// MessageSendInput は POST /rooms/:id/messages の入力。
type MessageSendInput struct {
	RoomID          uuid.UUID
	SenderID        uuid.UUID
	Content         *string
	ImageURL        *string
	FileURL         *string
	FileName        *string
	FileContentType *string
	ParentMessageID *uuid.UUID
}

type PublicUploadDeleter interface {
	DeleteByPublicURL(urlPath string)
}

type MessageUseCase struct {
	MessageRepo repository.MessageRepository
	Users       repository.UserRepository
	Rooms       repository.RoomRepository
	Uploads     PublicUploadDeleter
}

func NewMessageUseCase(messages repository.MessageRepository, users repository.UserRepository, rooms repository.RoomRepository, uploads PublicUploadDeleter) *MessageUseCase {
	return &MessageUseCase{
		MessageRepo: messages,
		Users:       users,
		Rooms:       rooms,
		Uploads:     uploads,
	}
}

func validateStaticUploadPath(u *string) error {
	if u == nil || *u == "" {
		return nil
	}
	p := strings.TrimSpace(*u)
	const prefix = "/static/uploads/"
	if !strings.HasPrefix(p, prefix) {
		return ErrInvalidImageURL
	}
	name := strings.TrimPrefix(p, prefix)
	if name == "" || strings.Contains(name, "..") {
		return ErrInvalidImageURL
	}
	return nil
}

func validateMessageImageURL(u *string) error {
	return validateStaticUploadPath(u)
}

func validateMessageFileURL(u *string) error {
	return validateStaticUploadPath(u)
}

func normalizeOptionalString(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// SendMessage は後方互換用。SendRoomMessage に委譲する。
func (u *MessageUseCase) SendMessage(ctx context.Context, roomID, senderID uuid.UUID, content, imageURL *string) (*entity.Message, error) {
	return u.SendRoomMessage(ctx, MessageSendInput{
		RoomID:   roomID,
		SenderID: senderID,
		Content:  content,
		ImageURL: imageURL,
	})
}

// SendRoomMessage はメンバー検証のうえでメッセージを保存する（返信・添付・画像を含む）。
func (u *MessageUseCase) SendRoomMessage(ctx context.Context, in MessageSendInput) (*entity.Message, error) {
	if in.RoomID == uuid.Nil {
		return nil, ErrInvalidRoomID
	}
	if in.SenderID == uuid.Nil {
		return nil, ErrInvalidSenderID
	}
	content := normalizeOptionalString(in.Content)
	imageURL := normalizeOptionalString(in.ImageURL)
	fileURL := normalizeOptionalString(in.FileURL)
	fileName := normalizeOptionalString(in.FileName)
	fileCT := normalizeOptionalString(in.FileContentType)
	parentID := in.ParentMessageID
	if parentID != nil && *parentID == uuid.Nil {
		parentID = nil
	}
	if content == nil && imageURL == nil && fileURL == nil {
		return nil, ErrEmptyMessage
	}
	if err := validateMessageImageURL(imageURL); err != nil {
		return nil, err
	}
	if err := validateMessageFileURL(fileURL); err != nil {
		return nil, err
	}
	if fileURL != nil && fileName == nil {
		return nil, ErrEmptyMessage
	}
	if _, err := RoomForActiveMemberWritable(ctx, u.Rooms, in.RoomID, in.SenderID); err != nil {
		return nil, err
	}
	if parentID != nil {
		parent, err := u.MessageRepo.FindByID(ctx, *parentID)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.IsDeleted || parent.DeletedAt != nil {
			return nil, ErrMessageNotFound
		}
		if parent.RoomID != in.RoomID {
			return nil, ErrReadMessageRoomMismatch
		}
	}
	now := time.Now().UTC()
	msg := &entity.Message{
		ID:               uuid.New(),
		RoomID:           in.RoomID,
		SenderID:         in.SenderID,
		ParentMessageID:  parentID,
		Content:          content,
		ImageURL:         imageURL,
		FileURL:          fileURL,
		FileName:         fileName,
		FileContentType:  fileCT,
		IsDeleted:        false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := u.MessageRepo.Create(ctx, msg); err != nil {
		return nil, err
	}
	if u.Rooms != nil {
		_ = u.Rooms.UpdateUpdatedAt(ctx, in.RoomID, now)
	}
	return msg, nil
}

// EditMessage は本文のみ編集（送信者のみ）。
func (u *MessageUseCase) EditMessage(ctx context.Context, messageID, editorID uuid.UUID, content *string) (*entity.Message, error) {
	if messageID == uuid.Nil {
		return nil, ErrInvalidMessageID
	}
	if editorID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	content = normalizeOptionalString(content)
	if content == nil {
		return nil, ErrEmptyMessage
	}
	msg, err := u.MessageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, ErrMessageNotFound
	}
	if msg.IsDeleted || msg.DeletedAt != nil {
		return nil, ErrMessageNotFound
	}
	if msg.SenderID != editorID {
		return nil, ErrForbiddenMessageEdit
	}
	if _, err := RoomForActiveMemberWritable(ctx, u.Rooms, msg.RoomID, editorID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := u.MessageRepo.UpdateContent(ctx, messageID, content, now); err != nil {
		return nil, err
	}
	msg.Content = content
	msg.UpdatedAt = now
	return msg, nil
}

// ListMessages は参加メンバーのみメッセージ履歴を取得する。order_desc=false で昇順（古い順）。
func (u *MessageUseCase) ListMessages(ctx context.Context, roomID, userID uuid.UUID, beforeID, afterID *uuid.UUID, limit int, orderDesc, includeReadCount bool) ([]ListedMessage, error) {
	if roomID == uuid.Nil {
		return nil, ErrInvalidRoomID
	}
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if beforeID != nil && *beforeID == uuid.Nil {
		beforeID = nil
	}
	if afterID != nil && *afterID == uuid.Nil {
		afterID = nil
	}
	limit = normalizeMessageListLimit(limit)
	if _, err := RoomForActiveMember(ctx, u.Rooms, roomID, userID); err != nil {
		return nil, err
	}
	rows, err := u.MessageRepo.ListByRoomID(ctx, roomID, beforeID, afterID, limit, orderDesc)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	senderIDs := make([]uuid.UUID, 0, len(rows))
	seen := make(map[uuid.UUID]struct{})
	for i := range rows {
		id := rows[i].SenderID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		senderIDs = append(senderIDs, id)
	}
	var users []entity.User
	if u.Users != nil {
		users, err = u.Users.FindByIDs(ctx, senderIDs)
		if err != nil {
			return nil, err
		}
	}
	nameBy := make(map[uuid.UUID]string, len(users))
	for i := range users {
		nameBy[users[i].ID] = users[i].Name
	}
	out := make([]ListedMessage, 0, len(rows))
	for i := range rows {
		item := ListedMessage{
			Message:    rows[i],
			SenderName: nameBy[rows[i].SenderID],
		}
		if includeReadCount {
			n, err := u.MessageRepo.CountReadersAtOrAfterMessage(ctx, roomID, rows[i].ID, rows[i].SenderID)
			if err != nil {
				return nil, err
			}
			v := n
			item.ReadCount = &v
		}
		out = append(out, item)
	}
	return out, nil
}

// SearchJoined は参加ルームのメッセージを横断検索する。
func (u *MessageUseCase) SearchJoined(ctx context.Context, userID uuid.UUID, query string, limit int) ([]ListedMessage, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	q := strings.TrimSpace(query)
	if q == "" || utf8.RuneCountInString(q) < 2 {
		return nil, ErrInvalidSearchQuery
	}
	// LIKE メタ文字をエスケープ
	q = strings.ReplaceAll(q, `\`, `\\`)
	q = strings.ReplaceAll(q, `%`, `\%`)
	q = strings.ReplaceAll(q, `_`, `\_`)
	rows, err := u.MessageRepo.SearchInJoinedRooms(ctx, userID, q, limit)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	senderIDs := make([]uuid.UUID, 0, len(rows))
	seen := make(map[uuid.UUID]struct{})
	for i := range rows {
		id := rows[i].SenderID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		senderIDs = append(senderIDs, id)
	}
	var users []entity.User
	if u.Users != nil {
		users, err = u.Users.FindByIDs(ctx, senderIDs)
		if err != nil {
			return nil, err
		}
	}
	nameBy := make(map[uuid.UUID]string, len(users))
	for i := range users {
		nameBy[users[i].ID] = users[i].Name
	}
	out := make([]ListedMessage, 0, len(rows))
	for i := range rows {
		out = append(out, ListedMessage{
			Message:    rows[i],
			SenderName: nameBy[rows[i].SenderID],
		})
	}
	return out, nil
}

func normalizeMessageListLimit(limit int) int {
	if limit <= 0 {
		return DefaultMessageListLimit
	}
	if limit > MaxMessageListLimit {
		return MaxMessageListLimit
	}
	return limit
}

func (u *MessageUseCase) DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) (uuid.UUID, error) {
	if messageID == uuid.Nil {
		return uuid.Nil, ErrInvalidMessageID
	}
	if userID == uuid.Nil {
		return uuid.Nil, ErrInvalidUserID
	}
	msg, err := u.MessageRepo.FindByID(ctx, messageID)
	if err != nil {
		return uuid.Nil, err
	}
	if msg == nil {
		return uuid.Nil, ErrMessageNotFound
	}
	if msg.IsDeleted || msg.DeletedAt != nil {
		return msg.RoomID, nil
	}
	if msg.SenderID != userID {
		return uuid.Nil, ErrForbiddenMessageEdit
	}
	if _, err := RoomForActiveMemberWritable(ctx, u.Rooms, msg.RoomID, userID); err != nil {
		return uuid.Nil, err
	}
	if err := u.MessageRepo.SoftDelete(ctx, messageID, time.Now().UTC()); err != nil {
		return uuid.Nil, err
	}
	if u.Uploads != nil {
		if msg.ImageURL != nil && *msg.ImageURL != "" {
			u.Uploads.DeleteByPublicURL(*msg.ImageURL)
		}
		if msg.FileURL != nil && *msg.FileURL != "" {
			u.Uploads.DeleteByPublicURL(*msg.FileURL)
		}
	}
	return msg.RoomID, nil
}
