package repository

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func redisCmdableIsNil(c redis.Cmdable) bool {
	if c == nil {
		return true
	}
	v := reflect.ValueOf(c)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	default:
		return false
	}
}

// RateLimitPolicy はエンドポイント別のレート制限ポリシー。
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

// RateLimitRepository は業務単位の判定 API を提供する。
type RateLimitRepository interface {
	AllowLogin(ctx context.Context, clientIP, email string) (bool, int64, error)
	AllowRegister(ctx context.Context, clientIP string) (bool, int64, error)
	AllowMessage(ctx context.Context, userID uuid.UUID, roomID string) (bool, int64, error)
	AllowMedia(ctx context.Context, userID uuid.UUID) (bool, int64, error)
	AllowWSConnect(ctx context.Context, userID uuid.UUID) (bool, int64, error)
}

// RateLimiter は Redis 上の token bucket で上限を判定する。
type RateLimiter struct {
	rdb    redis.Cmdable
	lua    *redis.Script
	policy RateLimitPolicy
}

var _ RateLimitRepository = (*RateLimiter)(nil)

// NewRateLimitRepository は rdb が nil のとき常に true（レート制限オフ）。
func NewRateLimitRepository(rdb redis.Cmdable, policy ...RateLimitPolicy) *RateLimiter {
	p := DefaultRateLimitPolicy()
	if len(policy) > 0 {
		p = policy[0]
	}
	if redisCmdableIsNil(rdb) {
		return &RateLimiter{policy: p}
	}
	return &RateLimiter{
		rdb:    rdb,
		policy: p,
		lua: redis.NewScript(`
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local cap = tonumber(ARGV[2])
local cost = tonumber(ARGV[3])
local now_ms = tonumber(ARGV[4])
local ttl_sec = tonumber(ARGV[5])

if rate <= 0 or cap <= 0 or cost <= 0 then
  return {1, 0}
end

local vals = redis.call('HMGET', key, 'tokens', 'ts')
local tokens = tonumber(vals[1])
local ts = tonumber(vals[2])
if not tokens then
  tokens = cap
end
if not ts then
  ts = now_ms
end

local elapsed_ms = now_ms - ts
if elapsed_ms < 0 then
  elapsed_ms = 0
end
local refill = elapsed_ms * rate / 1000.0
tokens = math.min(cap, tokens + refill)

local allowed = 0
local retry_ms = 0
if tokens >= cost then
  tokens = tokens - cost
  allowed = 1
else
  retry_ms = math.ceil((cost - tokens) / rate * 1000.0)
end

redis.call('HMSET', key, 'tokens', tokens, 'ts', now_ms)
redis.call('EXPIRE', key, ttl_sec)
return {allowed, retry_ms}
`),
	}
}

func (l *RateLimiter) allow(ctx context.Context, key string, max int64, window time.Duration) (bool, int64, error) {
	if l == nil || redisCmdableIsNil(l.rdb) {
		return true, 0, nil
	}
	if max <= 0 || window <= 0 {
		return true, 0, nil
	}
	ratePerSec := float64(max) / window.Seconds()
	capacity := max
	cost := int64(1)
	now := time.Now()
	if key == "" || ratePerSec <= 0 || capacity <= 0 || cost <= 0 {
		return true, 0, nil
	}
	ttl := int64(float64(capacity)/ratePerSec + 2.0)
	if ttl < 1 {
		ttl = 1
	}
	res, err := l.lua.Run(ctx, l.rdb, []string{key}, ratePerSec, capacity, cost, now.UnixMilli(), ttl).Result()
	if err != nil {
		return false, 0, err
	}
	arr, ok := res.([]any)
	if !ok || len(arr) < 2 {
		return false, 0, nil
	}
	allowed, ok1 := arr[0].(int64)
	retryMs, ok2 := arr[1].(int64)
	if !ok1 || !ok2 {
		return false, 0, nil
	}
	retrySec := retryMs / 1000
	if retryMs%1000 != 0 {
		retrySec++
	}
	return allowed == 1, retrySec, nil
}

func (l *RateLimiter) AllowLogin(ctx context.Context, clientIP, email string) (bool, int64, error) {
	p := l.policy
	ip := strings.TrimSpace(clientIP)
	emailKey := strings.ToLower(strings.TrimSpace(email))
	ipOk, ipRetrySec, ipErr := l.allow(ctx, "rl:login:ip:"+ip, p.LoginIPLimit, p.LoginIPWindow)
	if emailKey == "" {
		return ipOk, ipRetrySec, ipErr
	}
	emailOk, emailRetrySec, emailErr := l.allow(ctx, "rl:login:email:"+emailKey, p.LoginEmailLimit, p.LoginEmailWindow)
	if ipErr != nil {
		return ipOk && emailOk, maxInt64(ipRetrySec, emailRetrySec), ipErr
	}
	if emailErr != nil {
		return ipOk && emailOk, maxInt64(ipRetrySec, emailRetrySec), emailErr
	}
	return ipOk && emailOk, maxInt64(ipRetrySec, emailRetrySec), nil
}

func (l *RateLimiter) AllowRegister(ctx context.Context, clientIP string) (bool, int64, error) {
	p := l.policy
	return l.allow(ctx, "rl:register:ip:"+strings.TrimSpace(clientIP), p.RegisterLimit, p.RegisterWindow)
}

func (l *RateLimiter) AllowMessage(ctx context.Context, userID uuid.UUID, roomID string) (bool, int64, error) {
	if userID == uuid.Nil || strings.TrimSpace(roomID) == "" {
		return true, 0, nil
	}
	p := l.policy
	key := "rl:message:" + userID.String() + ":" + strings.TrimSpace(roomID)
	return l.allow(ctx, key, p.MessageLimit, p.MessageWindow)
}

func (l *RateLimiter) AllowMedia(ctx context.Context, userID uuid.UUID) (bool, int64, error) {
	if userID == uuid.Nil {
		return true, 0, nil
	}
	p := l.policy
	return l.allow(ctx, "rl:upload:"+userID.String(), p.MediaLimit, p.MediaWindow)
}

func (l *RateLimiter) AllowWSConnect(ctx context.Context, userID uuid.UUID) (bool, int64, error) {
	if userID == uuid.Nil {
		return true, 0, nil
	}
	p := l.policy
	return l.allow(ctx, "rl:ws:"+userID.String(), p.WSConnectLimit, p.WSConnectWindow)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
