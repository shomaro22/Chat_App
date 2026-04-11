package db

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// OpenRedis は環境変数から Redis クライアントを開く。
// REDIS_ADDR が空のときはレート制限無効用に (nil, nil) を返す。
//
//	REDIS_ADDR      例: localhost:6379（必須でない場合は空で Redis レート制限をスキップ）
//	REDIS_PASSWORD  省略可
//	REDIS_DB        省略時 0
func OpenRedis(ctx context.Context) (*redis.Client, error) {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		return nil, nil
	}
	dbNum := 0
	if s := strings.TrimSpace(os.Getenv("REDIS_DB")); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		dbNum = n
	}
	pass := os.Getenv("REDIS_PASSWORD")

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pass,
		DB:       dbNum,
	})
	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctxPing).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return rdb, nil
}
