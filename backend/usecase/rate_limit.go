package usecase

import (
	"context"
	"strings"
	"time"

	"chat-app/repository"

	"github.com/google/uuid"
)

// RateLimitPolicy はエンドポイント別の固定窓レート制限（試行回数と窓幅）。
type RateLimitPolicy struct {
	LoginLimit        int64
	LoginWindow       time.Duration
	RegisterLimit     int64
	RegisterWindow    time.Duration
	MessageLimit      int64
	MessageWindow     time.Duration
	MediaLimit        int64
	MediaWindow       time.Duration
	WSConnectLimit    int64
	WSConnectWindow   time.Duration
}

func DefaultRateLimitPolicy() RateLimitPolicy {
	w := time.Minute
	return RateLimitPolicy{
		LoginLimit:      20,
		LoginWindow:     w,
		RegisterLimit:   10,
		RegisterWindow:  w,
		MessageLimit:    60,
		MessageWindow:   w,
		MediaLimit:      30,
		MediaWindow:     w,
		WSConnectLimit:  30,
		WSConnectWindow: w,
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

func retrySeconds(window time.Duration) int64 {
	sec := int64(window.Round(time.Second) / time.Second)
	if sec < 1 {
		return 1
	}
	return sec
}

func (u *RateLimitUseCase) allow(ctx context.Context, key string, max int64, window time.Duration) RateLimitResult {
	if u == nil || u.repo == nil {
		return RateLimitResult{Allowed: true}
	}
	ok, err := u.repo.Allow(ctx, key, max, window)
	if err != nil {
		return RateLimitResult{Allowed: true, Err: err}
	}
	if !ok {
		return RateLimitResult{Allowed: false, RetryAfterSeconds: retrySeconds(window)}
	}
	return RateLimitResult{Allowed: true}
}

// CheckLogin は POST /auth/login 用。email が空でなければ rl:login:email:{email}、空なら IP。
func (u *RateLimitUseCase) CheckLogin(ctx context.Context, email, clientIP string) RateLimitResult {
	email = strings.ToLower(strings.TrimSpace(email))
	var key string
	if email != "" {
		key = "rl:login:email:" + email
	} else {
		key = "rl:login:ip:" + strings.TrimSpace(clientIP)
	}
	return u.allow(ctx, key, u.policy.LoginLimit, u.policy.LoginWindow)
}

// CheckRegister は POST /auth/register 用。rl:register:ip:{ip}
func (u *RateLimitUseCase) CheckRegister(ctx context.Context, clientIP string) RateLimitResult {
	key := "rl:register:ip:" + strings.TrimSpace(clientIP)
	return u.allow(ctx, key, u.policy.RegisterLimit, u.policy.RegisterWindow)
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
	key := "rl:upload:" + userID.String()
	return u.allow(ctx, key, u.policy.MediaLimit, u.policy.MediaWindow)
}

// CheckWSConnect は GET /ws 用（JWT 後）。
func (u *RateLimitUseCase) CheckWSConnect(ctx context.Context, userID uuid.UUID) RateLimitResult {
	if userID == uuid.Nil {
		return RateLimitResult{Allowed: true}
	}
	key := "rl:ws:" + userID.String()
	return u.allow(ctx, key, u.policy.WSConnectLimit, u.policy.WSConnectWindow)
}
