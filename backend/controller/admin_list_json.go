package controller

import (
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
)

type adminUserListResponse struct {
	Users  []userPublicJSON `json:"users"`
	Total  int64            `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

type adminRoomListItemJSON struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Name      *string   `json:"name,omitempty"`
	IsFrozen  bool      `json:"is_frozen"`
	UpdatedAt string    `json:"updated_at"`
}

type adminRoomListResponse struct {
	Rooms  []adminRoomListItemJSON `json:"rooms"`
	Total  int64                   `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}

func roomToAdminListItemJSON(r *entity.Room) adminRoomListItemJSON {
	if r == nil {
		return adminRoomListItemJSON{}
	}
	return adminRoomListItemJSON{
		ID:        r.ID,
		Type:      string(r.Type),
		Name:      r.Name,
		IsFrozen:  r.FrozenAt != nil,
		UpdatedAt: r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
