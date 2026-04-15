package usecase

import (
	"context"

	"chat-app/repository"

	"github.com/google/uuid"
)

// RateLimitResult はレート制限の判定結果。Redis 障害時は Allowed=true で fail-open し、Err を返す。
type RateLimitResult struct {
	Allowed           bool
	RetryAfterSeconds int64
	Err               error
}

// RateLimitUseCase はレート制限のキー生成・ポリシー適用を担う。
type RateLimitUseCase struct {
	repo repository.RateLimitRepository
}

func NewRateLimitUseCase(repo repository.RateLimitRepository) *RateLimitUseCase {
	return &RateLimitUseCase{repo: repo}
}

func toRateLimitResult(ok bool, retryAfterSec int64, err error) RateLimitResult {
	if err != nil {
		return RateLimitResult{Allowed: true, Err: err}
	}
	if !ok {
		return RateLimitResult{Allowed: false, RetryAfterSeconds: retryAfterSec}
	}
	return RateLimitResult{Allowed: true}
}

// CheckLogin は POST /auth/login 用。IP と email の両方を判定し、どちらかが超過なら拒否する。
func (u *RateLimitUseCase) CheckLogin(ctx context.Context, clientIP, email string) RateLimitResult {
	if u == nil || u.repo == nil {
		return RateLimitResult{Allowed: true}
	}
	ok, retryAfterSec, err := u.repo.AllowLogin(ctx, clientIP, email)
	return toRateLimitResult(ok, retryAfterSec, err)
}

// CheckRegister は POST /auth/register 用。rl:register:ip:{ip}
func (u *RateLimitUseCase) CheckRegister(ctx context.Context, clientIP string) RateLimitResult {
	if u == nil || u.repo == nil {
		return RateLimitResult{Allowed: true}
	}
	ok, retryAfterSec, err := u.repo.AllowRegister(ctx, clientIP)
	return toRateLimitResult(ok, retryAfterSec, err)
}

// CheckMessageSend は POST /rooms/:id/messages 用。
func (u *RateLimitUseCase) CheckMessageSend(ctx context.Context, userID uuid.UUID, roomID string) RateLimitResult {
	if u == nil || u.repo == nil {
		return RateLimitResult{Allowed: true}
	}
	ok, retryAfterSec, err := u.repo.AllowMessage(ctx, userID, roomID)
	return toRateLimitResult(ok, retryAfterSec, err)
}

// CheckMediaUpload は POST /media/images 用。
func (u *RateLimitUseCase) CheckMediaUpload(ctx context.Context, userID uuid.UUID) RateLimitResult {
	if u == nil || u.repo == nil {
		return RateLimitResult{Allowed: true}
	}
	ok, retryAfterSec, err := u.repo.AllowMedia(ctx, userID)
	return toRateLimitResult(ok, retryAfterSec, err)
}

// CheckWSConnect は GET /ws 用（JWT 後）。
func (u *RateLimitUseCase) CheckWSConnect(ctx context.Context, userID uuid.UUID) RateLimitResult {
	if u == nil || u.repo == nil {
		return RateLimitResult{Allowed: true}
	}
	ok, retryAfterSec, err := u.repo.AllowWSConnect(ctx, userID)
	return toRateLimitResult(ok, retryAfterSec, err)
}
