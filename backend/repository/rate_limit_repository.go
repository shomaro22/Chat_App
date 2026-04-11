package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

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

// NewFixedWindowRateLimiter は rdb が nil のとき Allow は常に true（レート制限オフ）。
func NewFixedWindowRateLimiter(rdb redis.Cmdable) *FixedWindowRateLimiter {
	if rdb == nil {
		return &FixedWindowRateLimiter{}
	}
	return &FixedWindowRateLimiter{rdb: rdb}
}

// Allow は key の窓内カウントを 1 増やし、max 以下なら true。
// max は「許容する試行回数」（例 max=20 なら 20 回目まで true、21 回目 false）。
func (l *FixedWindowRateLimiter) Allow(ctx context.Context, key string, max int64, window time.Duration) (bool, error) {
	if l == nil || l.rdb == nil {
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
		return true, nil // Redis 障害時はフェイルオープン
	}
	return count <= max, nil
}

// LoadScript は本番 Redis で EVAL を避けるため SCRIPT LOAD を行う（任意）。
func (l *FixedWindowRateLimiter) LoadScript(ctx context.Context) error {
	if l == nil || l.rdb == nil {
		return nil
	}
	s, err := l.rdb.ScriptLoad(ctx, incrExpireOnceLua).Result()
	if err != nil {
		return err
	}
	l.sha = s
	return nil
}
