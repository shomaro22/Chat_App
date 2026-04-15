package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_AllowUnderLimitThenReject(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := NewRateLimitRepository(rdb, RateLimitPolicy{
		LoginIPLimit:     1,
		LoginIPWindow:    defaultPolicyWindow,
		LoginEmailLimit:  1,
		LoginEmailWindow: defaultPolicyWindow,
		RegisterLimit:    10,
		RegisterWindow:   defaultPolicyWindow,
		MessageLimit:     10,
		MessageWindow:    defaultPolicyWindow,
		MediaLimit:       10,
		MediaWindow:      defaultPolicyWindow,
		WSConnectLimit:   10,
		WSConnectWindow:  defaultPolicyWindow,
	})
	ctx := context.Background()

	ok, _, err := lim.AllowLogin(ctx, "198.51.100.1", "a@x.test")
	require.NoError(t, err)
	require.True(t, ok)
	ok, retrySec, err := lim.AllowLogin(ctx, "198.51.100.1", "a@x.test")
	require.NoError(t, err)
	require.False(t, ok)
	require.GreaterOrEqual(t, retrySec, int64(1))
}

func TestRateLimiter_KeySeparation(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := NewRateLimitRepository(rdb, RateLimitPolicy{
		LoginIPLimit:     10,
		LoginIPWindow:    defaultPolicyWindow,
		LoginEmailLimit:  1,
		LoginEmailWindow: defaultPolicyWindow,
		RegisterLimit:    10,
		RegisterWindow:   defaultPolicyWindow,
		MessageLimit:     10,
		MessageWindow:    defaultPolicyWindow,
		MediaLimit:       10,
		MediaWindow:      defaultPolicyWindow,
		WSConnectLimit:   10,
		WSConnectWindow:  defaultPolicyWindow,
	})
	ctx := context.Background()

	ok, _, err := lim.AllowLogin(ctx, "198.51.100.1", "a@x.test")
	require.NoError(t, err)
	require.True(t, ok)
	ok, _, err = lim.AllowLogin(ctx, "198.51.100.1", "a@x.test")
	require.NoError(t, err)
	require.False(t, ok)
	ok, _, err = lim.AllowLogin(ctx, "198.51.100.1", "b@x.test")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestRateLimiter_NilRedisAlwaysAllow(t *testing.T) {
	lim := NewRateLimitRepository(nil)
	ok, retrySec, err := lim.AllowRegister(context.Background(), "203.0.113.10")
	require.NoError(t, err)
	require.True(t, ok)
	require.Zero(t, retrySec)
}

func TestRateLimiter_TypedNilClientAlwaysAllow(t *testing.T) {
	var c *redis.Client
	lim := NewRateLimitRepository(c)
	ok, retrySec, err := lim.AllowMessage(context.Background(), uuid.New(), "room-a")
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
	ok, _, err := lim.AllowRegister(context.Background(), "203.0.113.11")
	require.Error(t, err)
	require.False(t, ok)
	_ = rdb.Close()
}

const defaultPolicyWindow = time.Minute
