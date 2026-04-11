// Package audit は管理者操作などの監査用 JSON ログを stderr に出す。
package audit

import (
	"context"
	"log/slog"
	"os"
)

var logJSON = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

// Event は1行 JSON（slog）で監査イベントを記録する。
func Event(msg string, attrs ...slog.Attr) {
	logJSON.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
}
