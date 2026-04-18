package usecase

import (
	"context"

	"chat-app/repository"

	"github.com/google/uuid"
)

const presenceJoinedRoomsLimit = 500

// PresenceUseCase はプレゼンス配信先となる参加ルーム一覧の解決。
type PresenceUseCase struct {
	Rooms repository.RoomRepository
}

func NewPresenceUseCase(rooms repository.RoomRepository) *PresenceUseCase {
	return &PresenceUseCase{Rooms: rooms}
}

// JoinedRoomIDs はユーザーがアクティブメンバーとして参加しているルーム ID の一覧を返す。
func (u *PresenceUseCase) JoinedRoomIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rooms, err := u.Rooms.ListJoinedRooms(ctx, userID, presenceJoinedRoomsLimit, 0, true)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rooms))
	for i := range rooms {
		out = append(out, rooms[i].ID)
	}
	return out, nil
}
