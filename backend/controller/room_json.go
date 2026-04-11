package controller

import (
	"time"

	"chat-app/entity"

	"github.com/google/uuid"
)

type listRoomsResponse struct {
	Rooms []roomJSON `json:"rooms"`
}

type createDirectRoomRequest struct {
	UserID       string `json:"user_id"`
	TargetUserID string `json:"target_user_id"`
}

type createGroupRoomRequest struct {
	CreatedBy string   `json:"created_by"`
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids"`
}

type roomJSON struct {
	ID            uuid.UUID  `json:"id"`
	Type          string     `json:"type"`
	Name          *string    `json:"name,omitempty"`
	DirectUserAID *uuid.UUID `json:"direct_user_a_id,omitempty"`
	DirectUserBID *uuid.UUID `json:"direct_user_b_id,omitempty"`
	CreatedBy     uuid.UUID  `json:"created_by"`
	CreatedAt     string     `json:"created_at"`
	UpdatedAt     string     `json:"updated_at"`
}

func roomToJSON(r *entity.Room) roomJSON {
	return roomJSON{
		ID:            r.ID,
		Type:          string(r.Type),
		Name:          r.Name,
		DirectUserAID: r.DirectUserAID,
		DirectUserBID: r.DirectUserBID,
		CreatedBy:     r.CreatedBy,
		CreatedAt:     r.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:     r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
