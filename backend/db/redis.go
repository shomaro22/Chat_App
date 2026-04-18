package db

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func OpenRedis() *redis.Client {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		return nil
	}
	dbNum := 0
	if s := strings.TrimSpace(os.Getenv("REDIS_DB")); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			dbNum = n
		}
	}
	pass := os.Getenv("REDIS_PASSWORD")

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
		DB:       dbNum,
	})
	ctxPing, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = rdb.Ping(ctxPing).Err()
	return rdb
}
