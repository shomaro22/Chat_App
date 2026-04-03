package repository

import (
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
)

func userToModel(u *entity.User) *UserModel {
	return &UserModel{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
		Active:       u.Active,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func userToEntity(m *UserModel) *entity.User {
	if m == nil {
		return nil
	}
	return &entity.User{
		ID:           m.ID,
		Name:         m.Name,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		Role:         entity.UserRole(m.Role),
		Active:       m.Active,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func roomToModel(r *entity.Room) *RoomModel {
	return &RoomModel{
		ID:        r.ID,
		Type:      string(r.Type),
		Name:      copyStrPtr(r.Name),
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func roomToEntity(m *RoomModel) *entity.Room {
	if m == nil {
		return nil
	}
	return &entity.Room{
		ID:        m.ID,
		Type:      entity.RoomType(m.Type),
		Name:      copyStrPtr(m.Name),
		CreatedBy: m.CreatedBy,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func roomMemberToModel(m *entity.RoomMember) *RoomMemberModel {
	return &RoomMemberModel{
		ID:       m.ID,
		RoomID:   m.RoomID,
		UserID:   m.UserID,
		JoinedAt: m.JoinedAt,
		LeftAt:   m.LeftAt,
	}
}

func roomMemberToEntity(m *RoomMemberModel) *entity.RoomMember {
	if m == nil {
		return nil
	}
	return &entity.RoomMember{
		ID:       m.ID,
		RoomID:   m.RoomID,
		UserID:   m.UserID,
		JoinedAt: m.JoinedAt,
		LeftAt:   m.LeftAt,
	}
}

func messageToModel(m *entity.Message) *MessageModel {
	return &MessageModel{
		ID:        m.ID,
		RoomID:    m.RoomID,
		SenderID:  m.SenderID,
		Content:   copyStrPtr(m.Content),
		ImageURL:  copyStrPtr(m.ImageURL),
		IsDeleted: m.IsDeleted,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: copyTimePtr(m.DeletedAt),
	}
}

func messageToEntity(m *MessageModel) *entity.Message {
	if m == nil {
		return nil
	}
	return &entity.Message{
		ID:        m.ID,
		RoomID:    m.RoomID,
		SenderID:  m.SenderID,
		Content:   copyStrPtr(m.Content),
		ImageURL:  copyStrPtr(m.ImageURL),
		IsDeleted: m.IsDeleted,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: copyTimePtr(m.DeletedAt),
	}
}

func messageReadToModel(m *entity.MessageRead) *MessageReadModel {
	return &MessageReadModel{
		ID:            m.ID,
		RoomID:        m.RoomID,
		UserID:        m.UserID,
		ReadMessageID: copyUUIDPtr(m.ReadMessageID),
		ReadAt:        copyTimePtr(m.ReadAt),
	}
}

func messageReadToEntity(m *MessageReadModel) *entity.MessageRead {
	if m == nil {
		return nil
	}
	return &entity.MessageRead{
		ID:            m.ID,
		RoomID:        m.RoomID,
		UserID:        m.UserID,
		ReadMessageID: copyUUIDPtr(m.ReadMessageID),
		ReadAt:        copyTimePtr(m.ReadAt),
	}
}

func copyStrPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

func copyTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}

func copyUUIDPtr(u *uuid.UUID) *uuid.UUID {
	if u == nil {
		return nil
	}
	v := *u
	return &v
}
