package usecase

import (
	"context"
	"strings"
	"time"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}

const defaultJoinedRoomsLimit = 200

type RoomUseCase struct {
	Rooms   repository.RoomRepository
	Members repository.RoomMemberRepository
	Users   repository.UserRepository
	DB      *gorm.DB
}

func NewRoomUseCase(db *gorm.DB, users repository.UserRepository, rooms repository.RoomRepository, members repository.RoomMemberRepository) *RoomUseCase {
	return &RoomUseCase{
		Users:   users,
		Rooms:   rooms,
		Members: members,
		DB:      db,
	}
}

func (useCase *RoomUseCase) GetJoinedRooms(ctx context.Context, userID uuid.UUID) ([]entity.Room, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	return useCase.Rooms.ListJoinedRooms(ctx, userID, defaultJoinedRoomsLimit, 0)
}

func (useCase *RoomUseCase) CreateOrGetDirectRoom(ctx context.Context, userID, targetUserID uuid.UUID) (*entity.Room, error) {
	if userID == uuid.Nil || targetUserID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if userID == targetUserID {
		return nil, ErrSelfDirect
	}
	u, err := useCase.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	if !u.Active {
		return nil, ErrInactiveUser
	}
	peer, err := useCase.Users.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if peer == nil {
		return nil, ErrTargetUserNotFound
	}
	if !peer.Active {
		return nil, ErrInactiveUser
	}

	existing, err := useCase.Rooms.FindDirectRoomByUserPair(ctx, userID, targetUserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	pairA, pairB := entity.OrderedUserPair(userID, targetUserID)

	var created *entity.Room
	err = useCase.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		rooms := repository.NewRoomRepository(transaction)
		members := repository.NewRoomMemberRepository(transaction)

		again, err := rooms.FindDirectRoomByUserPair(ctx, userID, targetUserID)
		if err != nil {
			return err
		}
		if again != nil {
			created = again
			return nil
		}

		now := time.Now().UTC()
		room := &entity.Room{
			ID:            uuid.New(),
			Type:          entity.RoomTypeDirect,
			Name:          nil,
			DirectUserAID: uuidPtr(pairA),
			DirectUserBID: uuidPtr(pairB),
			CreatedBy:     userID,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := rooms.Create(ctx, room); err != nil {
			return err
		}
		batch := []entity.RoomMember{
			{ID: uuid.New(), RoomID: room.ID, UserID: userID, JoinedAt: now},
			{ID: uuid.New(), RoomID: room.ID, UserID: targetUserID, JoinedAt: now},
		}
		if err := members.CreateBulk(ctx, batch); err != nil {
			return err
		}
		created = room
		return nil
	})
	if err != nil {
		if isPostgresUniqueViolation(err) {
			after, err2 := useCase.Rooms.FindDirectRoomByUserPair(ctx, userID, targetUserID)
			if err2 != nil {
				return nil, err2
			}
			if after != nil {
				return after, nil
			}
		}
		return nil, err
	}
	return created, nil
}

func (useCase *RoomUseCase) CreateGroupRoom(ctx context.Context, createdBy uuid.UUID, name string, memberIDs []uuid.UUID) (*entity.Room, error) {
	if createdBy == uuid.Nil {
		return nil, ErrInvalidCreatorID
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrGroupNameRequired
	}

	participantIDs := dedupeMemberIDs(createdBy, memberIDs)
	if len(participantIDs) < 2 {
		return nil, ErrGroupTooFewMembers
	}

	creator, err := useCase.Users.FindByID(ctx, createdBy)
	if err != nil {
		return nil, err
	}
	if creator == nil {
		return nil, ErrUserNotFound
	}
	if !creator.Active {
		return nil, ErrInactiveUser
	}

	foundUsers, err := useCase.Users.FindByIDs(ctx, participantIDs)
	if err != nil {
		return nil, err
	}
	if len(foundUsers) != len(participantIDs) {
		return nil, ErrInvalidGroupMembers
	}
	for i := range foundUsers {
		if !foundUsers[i].Active {
			return nil, ErrInactiveUser
		}
	}

	var created *entity.Room
	err = useCase.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		rooms := repository.NewRoomRepository(transaction)
		members := repository.NewRoomMemberRepository(transaction)
		n := name
		now := time.Now().UTC()
		room := &entity.Room{
			ID:        uuid.New(),
			Type:      entity.RoomTypeGroup,
			Name:      &n,
			CreatedBy: createdBy,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := rooms.Create(ctx, room); err != nil {
			return err
		}
		batch := make([]entity.RoomMember, 0, len(participantIDs))
		for _, uid := range participantIDs {
			batch = append(batch, entity.RoomMember{
				ID:       uuid.New(),
				RoomID:   room.ID,
				UserID:   uid,
				JoinedAt: now,
			})
		}
		if err := members.CreateBulk(ctx, batch); err != nil {
			return err
		}
		created = room
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func dedupeMemberIDs(createdBy uuid.UUID, memberIDs []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	out := make([]uuid.UUID, 0, 1+len(memberIDs))
	add := func(id uuid.UUID) {
		if id == uuid.Nil {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	add(createdBy)
	for _, id := range memberIDs {
		add(id)
	}
	return out
}
