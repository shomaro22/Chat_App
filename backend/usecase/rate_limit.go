package usecase

import (
	"context"
	"strings"
	"time"

	"chat-app/repository"

	"github.com/google/uuid"
)

// RateLimitPolicy はエンドポイント別のしきい値ポリシー。
type RateLimitPolicy struct {
	LoginIPLimit     int64
	LoginIPWindow    time.Duration
	LoginEmailLimit  int64
	LoginEmailWindow time.Duration
	RegisterLimit    int64
	RegisterWindow   time.Duration
	MessageLimit     int64
	MessageWindow    time.Duration
	MediaLimit       int64
	MediaWindow      time.Duration
	WSConnectLimit   int64
	WSConnectWindow  time.Duration
}

func DefaultRateLimitPolicy() RateLimitPolicy {
	w := time.Minute
	return RateLimitPolicy{
		LoginIPLimit:     20,
		LoginIPWindow:    w,
		LoginEmailLimit:  10,
		LoginEmailWindow: w,
		RegisterLimit:    10,
		RegisterWindow:   w,
		MessageLimit:     60,
		MessageWindow:    w,
		MediaLimit:       30,
		MediaWindow:      w,
		WSConnectLimit:   30,
		WSConnectWindow:  w,
	}
}

// RateLimitResult はレート制限の判定結果。Redis 障害時は Allowed=true で fail-open し、Err を返す。
type RateLimitResult struct {
	Allowed           bool
	RetryAfterSeconds int64
	Err               error
}

// RateLimitUseCase はレート制限のキー生成・ポリシー適用を担う。
type RateLimitUseCase struct {
	repo   repository.RateLimitRepository
	policy RateLimitPolicy
}

func NewRateLimitUseCase(repo repository.RateLimitRepository, policy RateLimitPolicy) *RateLimitUseCase {
	return &RateLimitUseCase{repo: repo, policy: policy}
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

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (u *RateLimitUseCase) allow(ctx context.Context, key string, max int64, window time.Duration) RateLimitResult {
	if u == nil || u.repo == nil {
		return RateLimitResult{Allowed: true}
	}
	ok, retryAfterSec, err := u.repo.Allow(ctx, key, max, window)
	return toRateLimitResult(ok, retryAfterSec, err)
}

// CheckLogin は POST /auth/login 用。IP と email の両方を判定し、どちらかが超過なら拒否する。
func (u *RateLimitUseCase) CheckLogin(ctx context.Context, clientIP, email string) RateLimitResult {
	ip := strings.TrimSpace(clientIP)
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	ipRes := u.allow(ctx, "rl:login:ip:"+ip, u.policy.LoginIPLimit, u.policy.LoginIPWindow)
	if normalizedEmail == "" {
		return ipRes
	}
	emailRes := u.allow(ctx, "rl:login:email:"+normalizedEmail, u.policy.LoginEmailLimit, u.policy.LoginEmailWindow)
	return RateLimitResult{
		Allowed:           ipRes.Allowed && emailRes.Allowed,
		RetryAfterSeconds: maxInt64(ipRes.RetryAfterSeconds, emailRes.RetryAfterSeconds),
		Err: func() error {
			if ipRes.Err != nil {
				return ipRes.Err
			}
			return emailRes.Err
		}(),
	}
}

// CheckRegister は POST /auth/register 用。rl:register:ip:{ip}
func (u *RateLimitUseCase) CheckRegister(ctx context.Context, clientIP string) RateLimitResult {
	return u.allow(ctx, "rl:register:ip:"+strings.TrimSpace(clientIP), u.policy.RegisterLimit, u.policy.RegisterWindow)
}

// CheckMessageSend は POST /rooms/:id/messages 用。
func (u *RateLimitUseCase) CheckMessageSend(ctx context.Context, userID uuid.UUID, roomID string) RateLimitResult {
	if userID == uuid.Nil || strings.TrimSpace(roomID) == "" {
		return RateLimitResult{Allowed: true}
	}
	key := "rl:message:" + userID.String() + ":" + strings.TrimSpace(roomID)
	return u.allow(ctx, key, u.policy.MessageLimit, u.policy.MessageWindow)
}

// CheckMediaUpload は POST /media/images 用。
func (u *RateLimitUseCase) CheckMediaUpload(ctx context.Context, userID uuid.UUID) RateLimitResult {
	if userID == uuid.Nil {
		return RateLimitResult{Allowed: true}
	}
	return u.allow(ctx, "rl:upload:"+userID.String(), u.policy.MediaLimit, u.policy.MediaWindow)
}

// CheckWSConnect は GET /ws 用（JWT 後）。
func (u *RateLimitUseCase) CheckWSConnect(ctx context.Context, userID uuid.UUID) RateLimitResult {
	if userID == uuid.Nil {
		return RateLimitResult{Allowed: true}
	}
	return u.allow(ctx, "rl:ws:"+userID.String(), u.policy.WSConnectLimit, u.policy.WSConnectWindow)
}
