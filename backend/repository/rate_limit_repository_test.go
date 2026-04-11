package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFixedWindowRateLimiter_UnderLimitThen429ThenTTLRecovery(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := NewFixedWindowRateLimiter(rdb)
	ctx := context.Background()
	const max int64 = 5
	const win = 2 * time.Second

	for i := int64(0); i < max; i++ {
		ok, err := lim.Allow(ctx, "rl:test:one", max, win)
		require.NoError(t, err)
		require.True(t, ok, "i=%d", i)
	}
	ok, err := lim.Allow(ctx, "rl:test:one", max, win)
	require.NoError(t, err)
	require.False(t, ok)

	s.FastForward(win + time.Millisecond)
	ok, err = lim.Allow(ctx, "rl:test:one", max, win)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestFixedWindowRateLimiter_KeySeparation(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := NewFixedWindowRateLimiter(rdb)
	ctx := context.Background()
	const max int64 = 2
	const win = time.Minute

	for i := int64(0); i < max; i++ {
		ok, err := lim.Allow(ctx, "rl:test:a", max, win)
		require.NoError(t, err)
		require.True(t, ok)
	}
	ok, err := lim.Allow(ctx, "rl:test:a", max, win)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = lim.Allow(ctx, "rl:test:b", max, win)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestFixedWindowRateLimiter_LoadScriptStillWorks(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := NewFixedWindowRateLimiter(rdb)
	require.NoError(t, lim.LoadScript(context.Background()))
	ctx := context.Background()
	ok, err := lim.Allow(ctx, "rl:test:sha", 1, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = lim.Allow(ctx, "rl:test:sha", 1, time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestFixedWindowRateLimiter_NilRedisAlwaysAllow(t *testing.T) {
	lim := NewFixedWindowRateLimiter(nil)
	ok, err := lim.Allow(context.Background(), "k", 1, time.Second)
	require.NoError(t, err)
	require.True(t, ok)
}
