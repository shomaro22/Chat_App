package repository

import (
	"context"
	"reflect"
	"time"

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

// RateLimitRepository は token bucket 実行のみを提供する。
type RateLimitRepository interface {
	Allow(ctx context.Context, key string, max int64, window time.Duration) (bool, int64, error)
}

// RateLimiter は Redis 上の token bucket で上限を判定する。
type RateLimiter struct {
	rdb redis.Cmdable
	lua *redis.Script
}

var _ RateLimitRepository = (*RateLimiter)(nil)

// NewRateLimitRepository は rdb が nil のとき Allow は常に true（レート制限オフ）。
func NewRateLimitRepository(rdb redis.Cmdable) *RateLimiter {
	if redisCmdableIsNil(rdb) {
		return &RateLimiter{}
	}
	return &RateLimiter{
		rdb: rdb,
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

func (l *RateLimiter) Allow(ctx context.Context, key string, max int64, window time.Duration) (bool, int64, error) {
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
