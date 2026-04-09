package usecase

import (
	"context"

	"chat-app/repository"

	"github.com/google/uuid"
)

const presenceJoinedRoomsLimit = 500

type PresenceUseCase struct {
	Rooms repository.RoomRepository
}

func NewPresenceUseCase(rooms repository.RoomRepository) *PresenceUseCase {
	return &PresenceUseCase{Rooms: rooms}
}

func (u *PresenceUseCase) JoinedRoomIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rooms, err := u.Rooms.ListJoinedRooms(ctx, userID, presenceJoinedRoomsLimit, 0)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rooms))
	for i := range rooms {
		out = append(out, rooms[i].ID)
	}
	return out, nil
}
