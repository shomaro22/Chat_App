package usecase

import (
	"context"

	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
)

// RoomForActiveMember は論理削除されていないルームに user がアクティブメンバーとして属しているとき *entity.Room を返す。
// ルーム詳細・メッセージ一覧・既読状態取得など「閲覧」系の共通前提（非メンバーは ErrNotRoomMember）。
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

// RoomForActiveMemberWritable はアクティブメンバーかつルームが凍結していないことを検証する。
// 凍結ルーム（FrozenAt != nil）のときの統一ルール:
//
//	許可（RoomForActiveMember）: メッセージ一覧・ルーム詳細・read-state 取得など閲覧系 HTTP
//	拒否 ErrRoomFrozen（本関数経由）: メッセージ送信・論理削除・既読更新・WS subscribe
//
// WebSocket の購読可否は RoomForWebSocketSubscribe が同じく ErrRoomFrozen を返す。
func RoomForActiveMemberWritable(ctx context.Context, rooms repository.RoomRepository, roomID, userID uuid.UUID) (*entity.Room, error) {
	r, err := RoomForActiveMember(ctx, rooms, roomID, userID)
	if err != nil {
		return nil, err
	}
	if r.FrozenAt != nil {
		return nil, ErrRoomFrozen
	}
	return r, nil
}

// RoomForWebSocketSubscribe は WS でルームを購読してよいかを返す（nil で可）。RoomForActiveMemberWritable と同じ判定。
func RoomForWebSocketSubscribe(ctx context.Context, rooms repository.RoomRepository, roomID, userID uuid.UUID) error {
	_, err := RoomForActiveMemberWritable(ctx, rooms, roomID, userID)
	return err
}
