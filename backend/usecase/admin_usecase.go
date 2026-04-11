package usecase

import (
	"context"
	"time"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

// AdminUseCase は管理者向け操作（ユーザー無効化・ルーム凍結・強制退室）。
type AdminUseCase struct {
	Users   repository.UserRepository
	Rooms   repository.RoomRepository
	Members repository.RoomMemberRepository
}

func NewAdminUseCase(users repository.UserRepository, rooms repository.RoomRepository, members repository.RoomMemberRepository) *AdminUseCase {
	return &AdminUseCase{Users: users, Rooms: rooms, Members: members}
}

func (a *AdminUseCase) requireAdmin(ctx context.Context, adminID uuid.UUID) error {
	u, err := a.Users.FindByID(ctx, adminID)
	if err != nil {
		return err
	}
	if u == nil || u.Role != entity.UserRoleAdmin || !u.Active {
		return ErrForbiddenAdmin
	}
	return nil
}

// SetUserActive はユーザーの有効／無効を切り替える。
func (a *AdminUseCase) SetUserActive(ctx context.Context, adminID, targetUserID uuid.UUID, active bool) error {
	if err := a.requireAdmin(ctx, adminID); err != nil {
		return err
	}
	if targetUserID == uuid.Nil {
		return ErrInvalidUserID
	}
	return a.Users.SetActive(ctx, targetUserID, active)
}

// SetRoomFrozen はルームの凍結／解除を行う。
func (a *AdminUseCase) SetRoomFrozen(ctx context.Context, adminID, roomID uuid.UUID, frozen bool) error {
	if err := a.requireAdmin(ctx, adminID); err != nil {
		return err
	}
	if roomID == uuid.Nil {
		return ErrInvalidRoomID
	}
	room, err := a.Rooms.FindByIDIncludingDeleted(ctx, roomID)
	if err != nil {
		return err
	}
	if room == nil || room.DeletedAt != nil {
		return ErrRoomNotFound
	}
	var at *time.Time
	if frozen {
		t := time.Now().UTC()
		at = &t
	}
	return a.Rooms.SetFrozenAt(ctx, roomID, at)
}

// ForceRemoveMember は管理者による強制退室。最後の1人ならルームを論理削除する。
func (a *AdminUseCase) ForceRemoveMember(ctx context.Context, adminID, roomID, targetUserID uuid.UUID) error {
	if err := a.requireAdmin(ctx, adminID); err != nil {
		return err
	}
	if roomID == uuid.Nil || targetUserID == uuid.Nil {
		return ErrInvalidUserID
	}
	ok, err := a.Members.IsActiveMember(ctx, roomID, targetUserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrMemberNotFound
	}
	now := time.Now().UTC()
	if err := a.Members.Leave(ctx, roomID, targetUserID, now); err != nil {
		return err
	}
	n, err := a.Members.CountActiveByRoomID(ctx, roomID)
	if err != nil {
		return err
	}
	if n == 0 {
		return a.Rooms.SoftDelete(ctx, roomID, now)
	}
	return nil
}
