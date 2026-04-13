package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_AllowUnderLimitThenReject(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := NewRateLimitRepository(rdb)
	ctx := context.Background()
	key := "rl:test:a"

	ok, _, err := lim.Allow(ctx, key, 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	ok, retrySec, err := lim.Allow(ctx, key, 1, time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
	require.GreaterOrEqual(t, retrySec, int64(1))
}

func TestRateLimiter_KeySeparation(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := NewRateLimitRepository(rdb)
	ctx := context.Background()

	ok, _, err := lim.Allow(ctx, "rl:test:a", 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	ok, _, err = lim.Allow(ctx, "rl:test:a", 1, time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
	ok, _, err = lim.Allow(ctx, "rl:test:b", 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestRateLimiter_NilRedisAlwaysAllow(t *testing.T) {
	lim := NewRateLimitRepository(nil)
	ok, retrySec, err := lim.Allow(context.Background(), "rl:test:nil", 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	require.Zero(t, retrySec)
}

func TestRateLimiter_TypedNilClientAlwaysAllow(t *testing.T) {
	var c *redis.Client
	lim := NewRateLimitRepository(c)
	ok, retrySec, err := lim.Allow(context.Background(), "rl:test:nil-typed", 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	require.Zero(t, retrySec)
}

func TestRateLimiter_RedisErrorPropagates(t *testing.T) {
	s := miniredis.RunT(t)
	addr := s.Addr()
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	lim := NewRateLimitRepository(rdb)
	s.Close()
	ok, _, err := lim.Allow(context.Background(), "rl:test:err", 1, time.Minute)
	require.Error(t, err)
	require.False(t, ok)
	_ = rdb.Close()
}
