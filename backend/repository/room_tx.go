package repository

import (
	"context"

	"gorm.io/gorm"
)

// RoomUnitOfWork はルームとメンバーへの書き込みを同一トランザクションでまとめるための窓口。
type RoomUnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, rooms RoomRepository, members RoomMemberRepository) error) error
}

type gormRoomUnitOfWork struct {
	db *gorm.DB
}

// NewRoomUnitOfWork は GORM を用いた RoomUnitOfWork を返す。
func NewRoomUnitOfWork(db *gorm.DB) RoomUnitOfWork {
	return &gormRoomUnitOfWork{db: db}
}

func (u *gormRoomUnitOfWork) WithinTransaction(ctx context.Context, fn func(ctx context.Context, rooms RoomRepository, members RoomMemberRepository) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx, NewRoomRepository(tx), NewRoomMemberRepository(tx))
	})
}
