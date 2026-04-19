package repository

import (
	"context"
	"errors"
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *entity.Message) error
	UpdateContent(ctx context.Context, id uuid.UUID, content *string, updatedAt time.Time) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Message, error)
	// ListByRoomID は desc=true で新しい順（before_id より古い）、desc=false で古い順（after_id より新しい）。
	ListByRoomID(ctx context.Context, roomID uuid.UUID, beforeID, afterID *uuid.UUID, limit int, desc bool) ([]entity.Message, error)
	SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error
	// LatestPerRoom は各ルームの最新1件（作成順）を返す。
	LatestPerRoom(ctx context.Context, roomIDs []uuid.UUID) ([]entity.Message, error)
	// CountUnreadForViewer は lastRead より後の、自分以外が送った未削除メッセージ数。
	CountUnreadForViewer(ctx context.Context, roomID, viewerID uuid.UUID, lastReadMessageID *uuid.UUID) (int64, error)
	// CountReadersAtOrAfterMessage は read カーソルが当該メッセージ以上のアクティブメンバー数（送信者除外）。
	CountReadersAtOrAfterMessage(ctx context.Context, roomID, messageID, excludeUserID uuid.UUID) (int64, error)
	// SearchInJoinedRooms はユーザーが参加中のルームに限定し、本文・ファイル名を部分一致検索する。
	SearchInJoinedRooms(ctx context.Context, userID uuid.UUID, query string, limit int) ([]entity.Message, error)
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (messages *messageRepository) Create(ctx context.Context, msg *entity.Message) error {
	return messages.db.WithContext(ctx).Create(msg).Error
}

func (messages *messageRepository) UpdateContent(ctx context.Context, id uuid.UUID, content *string, updatedAt time.Time) error {
	return messages.db.WithContext(ctx).Model(&entity.Message{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"content":    content,
			"updated_at": updatedAt,
		}).Error
}

func (messages *messageRepository) ListByRoomID(ctx context.Context, roomID uuid.UUID, beforeID, afterID *uuid.UUID, limit int, desc bool) ([]entity.Message, error) {
	var rows []entity.Message
	q := messages.db.WithContext(ctx).Where("room_id = ?", roomID)
	if desc {
		q = q.Order("created_at DESC, id DESC")
		if beforeID != nil {
			q = q.Where("(created_at, id) < (SELECT created_at, id FROM messages WHERE id = ? AND room_id = ?)", *beforeID, roomID)
		}
	} else {
		q = q.Order("created_at ASC, id ASC")
		if afterID != nil {
			q = q.Where("(created_at, id) > (SELECT created_at, id FROM messages WHERE id = ? AND room_id = ?)", *afterID, roomID)
		}
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (messages *messageRepository) LatestPerRoom(ctx context.Context, roomIDs []uuid.UUID) ([]entity.Message, error) {
	if len(roomIDs) == 0 {
		return nil, nil
	}
	var rows []entity.Message
	err := messages.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (room_id)
			id, room_id, sender_id, parent_message_id, content, image_url, file_url, file_name, file_content_type,
			is_deleted, created_at, updated_at, deleted_at
		FROM messages
		WHERE room_id IN ?
		ORDER BY room_id, created_at DESC, id DESC
	`, roomIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (messages *messageRepository) CountUnreadForViewer(ctx context.Context, roomID, viewerID uuid.UUID, lastReadMessageID *uuid.UUID) (int64, error) {
	q := messages.db.WithContext(ctx).Model(&entity.Message{}).
		Where("room_id = ?", roomID).
		Where("sender_id <> ?", viewerID).
		Where("is_deleted = ?", false)
	if lastReadMessageID != nil {
		q = q.Where("(created_at, id) > (SELECT created_at, id FROM messages WHERE id = ? AND room_id = ?)", *lastReadMessageID, roomID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

func (messages *messageRepository) CountReadersAtOrAfterMessage(ctx context.Context, roomID, messageID, excludeUserID uuid.UUID) (int64, error) {
	var n int64
	err := messages.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT mr.user_id)
		FROM message_reads mr
		INNER JOIN room_members rm ON rm.room_id = mr.room_id AND rm.user_id = mr.user_id AND rm.left_at IS NULL
		INNER JOIN messages cursor_msg ON cursor_msg.id = mr.read_message_id AND cursor_msg.room_id = mr.room_id
		INNER JOIN messages target ON target.id = ? AND target.room_id = ?
		WHERE mr.room_id = ?
		  AND mr.read_message_id IS NOT NULL
		  AND mr.user_id <> ?
		  AND (cursor_msg.created_at, cursor_msg.id) >= (target.created_at, target.id)
	`, messageID, roomID, roomID, excludeUserID).Scan(&n).Error
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (messages *messageRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Message, error) {
	var row entity.Message
	if err := messages.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (messages *messageRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	return messages.db.WithContext(ctx).Model(&entity.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": deletedAt,
			"is_deleted": true,
			"updated_at": deletedAt,
		}).Error
}

func (messages *messageRepository) SearchInJoinedRooms(ctx context.Context, userID uuid.UUID, query string, limit int) ([]entity.Message, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	pattern := "%" + query + "%"
	var rows []entity.Message
	err := messages.db.WithContext(ctx).Raw(`
		SELECT m.id, m.room_id, m.sender_id, m.parent_message_id, m.content, m.image_url, m.file_url, m.file_name, m.file_content_type,
			m.is_deleted, m.created_at, m.updated_at, m.deleted_at
		FROM messages m
		INNER JOIN room_members rm ON rm.room_id = m.room_id AND rm.user_id = ? AND rm.left_at IS NULL
		WHERE m.is_deleted = false
		  AND (
			(m.content IS NOT NULL AND m.content ILIKE ?)
			OR (m.file_name IS NOT NULL AND m.file_name ILIKE ?)
		  )
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT ?
	`, userID, pattern, pattern, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
