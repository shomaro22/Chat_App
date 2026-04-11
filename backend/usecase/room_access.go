package usecase

import (
	"context"
	"errors"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

// RoomForActiveMember は論理削除されていないルームに user がアクティブメンバーとして属しているとき *entity.Room を返す。
// 閲覧・送信・既読更新・WS 購読可否の共通前提（非メンバーは ErrNotRoomMember）。
func RoomForActiveMember(ctx context.Context, rooms repository.RoomRepository, roomID, userID uuid.UUID) (*entity.Room, error) {
	if roomID == uuid.Nil {
		return nil, ErrInvalidRoomID
	}
	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	r, err := rooms.FindByIDForActiveMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrNotRoomMember
	}
	return r, nil
}

// readRoomForActiveMember は既読ユースケース向けに RoomForActiveMember のエラーを既読用エラーへ寄せる。
func readRoomForActiveMember(ctx context.Context, rooms repository.RoomRepository, roomID, userID uuid.UUID) (*entity.Room, error) {
	r, err := RoomForActiveMember(ctx, rooms, roomID, userID)
	if err == nil {
		return r, nil
	}
	if errors.Is(err, ErrInvalidRoomID) {
		return nil, ErrInvalidMessageReadRoomID
	}
	if errors.Is(err, ErrInvalidUserID) {
		return nil, ErrInvalidMessageReadUserID
	}
	if errors.Is(err, ErrNotRoomMember) {
		return nil, ErrReadNotRoomMember
	}
	return nil, err
}
