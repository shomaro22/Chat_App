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

func TestRateLimiterIntegration_LoginDualPolicy(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := NewRateLimitRepository(rdb, RateLimitPolicy{
		LoginIPLimit:     2,
		LoginIPWindow:    time.Minute,
		LoginEmailLimit:  1,
		LoginEmailWindow: time.Minute,
		RegisterLimit:    100,
		RegisterWindow:   time.Minute,
		MessageLimit:     100,
		MessageWindow:    time.Minute,
		MediaLimit:       100,
		MediaWindow:      time.Minute,
		WSConnectLimit:   100,
		WSConnectWindow:  time.Minute,
	})
	ctx := context.Background()

	ok, _, err := lim.AllowLogin(ctx, "198.51.100.10", "alice@example.test")
	require.NoError(t, err)
	require.True(t, ok)

	ok, retry, err := lim.AllowLogin(ctx, "198.51.100.10", "alice@example.test")
	require.NoError(t, err)
	require.False(t, ok)
	require.GreaterOrEqual(t, retry, int64(1))

	ok, _, err = lim.AllowLogin(ctx, "198.51.100.11", "alice@example.test")
	require.NoError(t, err)
	require.False(t, ok)

	ok, _, err = lim.AllowLogin(ctx, "198.51.100.12", "bob@example.test")
	require.NoError(t, err)
	require.True(t, ok)

	ok, _, err = lim.AllowLogin(ctx, "198.51.100.20", "dave@example.test")
	require.NoError(t, err)
	require.True(t, ok)

	ok, _, err = lim.AllowLogin(ctx, "198.51.100.20", "emma@example.test")
	require.NoError(t, err)
	require.True(t, ok)

	ok, retry, err = lim.AllowLogin(ctx, "198.51.100.20", "frank@example.test")
	require.NoError(t, err)
	require.False(t, ok)
	require.GreaterOrEqual(t, retry, int64(1))
}

func TestRateLimiterIntegration_MessagePerRoomAndWSPerUser(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	lim := NewRateLimitRepository(rdb, RateLimitPolicy{
		LoginIPLimit:     100,
		LoginIPWindow:    time.Minute,
		LoginEmailLimit:  100,
		LoginEmailWindow: time.Minute,
		RegisterLimit:    100,
		RegisterWindow:   time.Minute,
		MessageLimit:     1,
		MessageWindow:    time.Minute,
		MediaLimit:       100,
		MediaWindow:      time.Minute,
		WSConnectLimit:   1,
		WSConnectWindow:  time.Minute,
	})
	ctx := context.Background()
	uid := uuid.New()

	ok, _, err := lim.AllowMessage(ctx, uid, "room-a")
	require.NoError(t, err)
	require.True(t, ok)
	ok, _, err = lim.AllowMessage(ctx, uid, "room-a")
	require.NoError(t, err)
	require.False(t, ok)
	ok, _, err = lim.AllowMessage(ctx, uid, "room-b")
	require.NoError(t, err)
	require.True(t, ok)

	ok, _, err = lim.AllowWSConnect(ctx, uid)
	require.NoError(t, err)
	require.True(t, ok)
	ok, retry, err := lim.AllowWSConnect(ctx, uid)
	require.NoError(t, err)
	require.False(t, ok)
	require.GreaterOrEqual(t, retry, int64(1))
}
