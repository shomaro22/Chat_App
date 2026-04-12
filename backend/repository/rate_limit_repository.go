package repository

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisCmdableIsNil は *redis.Client の typed nil を redis.Cmdable に渡したときの落とし穴を避ける。
// （インターフェース値は非 nil だが動的値が nil のとき、rdb == nil が false になり得る）
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

// RateLimitRepository は固定窓カウンタによるレート制限の永続化。
type RateLimitRepository interface {
	Allow(ctx context.Context, key string, max int64, window time.Duration) (bool, error)
}

// incrExpireOnceLua は INCR と初回のみ EXPIRE を原子的に行う（固定窓カウンタ）。
const incrExpireOnceLua = `
local c = redis.call('INCR', KEYS[1])
if c == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return c
`

// FixedWindowRateLimiter は Redis 上の固定時間窓ごとのカウントで上限を判定する。
type FixedWindowRateLimiter struct {
	rdb redis.Cmdable
	sha string
}

var _ RateLimitRepository = (*FixedWindowRateLimiter)(nil)

// NewFixedWindowRateLimiter は rdb が nil のとき Allow は常に true（レート制限オフ）。
func NewFixedWindowRateLimiter(rdb redis.Cmdable) *FixedWindowRateLimiter {
	if redisCmdableIsNil(rdb) {
		return &FixedWindowRateLimiter{}
	}
	return &FixedWindowRateLimiter{rdb: rdb}
}

// Allow は key の窓内カウントを 1 増やし、max 以下なら true。
// max は「許容する試行回数」（例 max=20 なら 20 回目まで true、21 回目 false）。
func (l *FixedWindowRateLimiter) Allow(ctx context.Context, key string, max int64, window time.Duration) (bool, error) {
	if l == nil || redisCmdableIsNil(l.rdb) {
		return true, nil
	}
	if key == "" || max <= 0 {
		return true, nil
	}
	sec := int64(window / time.Second)
	if sec < 1 {
		sec = 1
	}
	var count int64
	var err error
	if l.sha != "" {
		count, err = l.rdb.EvalSha(ctx, l.sha, []string{key}, sec).Int64()
	} else {
		count, err = l.rdb.Eval(ctx, incrExpireOnceLua, []string{key}, sec).Int64()
	}
	if err != nil {
		return false, fmt.Errorf("redis rate limit allow: %w", err)
	}
	return count <= max, nil
}

// LoadScript は本番 Redis で EVAL を避けるため SCRIPT LOAD を行う（任意）。
func (l *FixedWindowRateLimiter) LoadScript(ctx context.Context) error {
	if l == nil || redisCmdableIsNil(l.rdb) {
		return nil
	}
	s, err := l.rdb.ScriptLoad(ctx, incrExpireOnceLua).Result()
	if err != nil {
		return err
	}
	l.sha = s
	return nil
}
