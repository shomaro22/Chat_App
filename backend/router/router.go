package router

import (
	"net/http"

	"chat-app/controller"
	"chat-app/middleware"

	"github.com/labstack/echo/v4"
)

// Config は Echo に載せるハンドラとミドルウェア設定。
type Config struct {
	Auth            *controller.AuthController
	Room            *controller.RoomController
	Messages        *controller.MessageController
	Reads           *controller.MessageReadController
	WS              *controller.WSController
	Media           *controller.MediaController
	Admin           *controller.AdminController
	ParseJWT        middleware.TokenParser
	ValidateSession middleware.SessionValidator // nil 可（JWT のみ検証）
	StaticDir       string                      // ローカルパス（/static で配信）
	// PreAuthMiddleware はルート登録前に e.Use される。
	PreAuthMiddleware []echo.MiddlewareFunc
	// RateLimitLogin / RateLimitRegister は該当 POST にだけ適用（空なら未適用）。
	RateLimitLogin    []echo.MiddlewareFunc
	RateLimitRegister []echo.MiddlewareFunc
	// RateLimitAfterJWT は JWT 検証直後の api グループに Use される（例: メッセージ送信・画像・WS）。
	// 空なら未適用。
	RateLimitAfterJWT []echo.MiddlewareFunc
}

// New はルーティング済み Echo を返す。/auth は公開、それ以外の API は JWT 必須、/admin は admin ロール必須。
func New(cfg Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	for _, mw := range cfg.PreAuthMiddleware {
		e.Use(mw)
	}
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	if cfg.StaticDir != "" {
		e.Static("/static", cfg.StaticDir)
	}

	if cfg.Auth != nil {
		if len(cfg.RateLimitLogin) > 0 {
			e.POST("/auth/login", cfg.Auth.Login, cfg.RateLimitLogin...)
		} else {
			e.POST("/auth/login", cfg.Auth.Login)
		}
		if len(cfg.RateLimitRegister) > 0 {
			e.POST("/auth/register", cfg.Auth.Register, cfg.RateLimitRegister...)
		} else {
			e.POST("/auth/register", cfg.Auth.Register)
		}
	}

	jwt := middleware.JWTAuth(cfg.ParseJWT, cfg.ValidateSession)

	api := e.Group("")
	api.Use(jwt)
	for _, mw := range cfg.RateLimitAfterJWT {
		api.Use(mw)
	}
	if cfg.Room != nil {
		api.GET("/rooms", cfg.Room.ListJoinedRooms)
		api.GET("/rooms/:id", cfg.Room.GetRoomDetail)
		api.PATCH("/rooms/:id", cfg.Room.UpdateGroupRoom)
		api.DELETE("/rooms/:id", cfg.Room.CloseRoom)
		api.POST("/rooms/direct", cfg.Room.CreateDirectRoom)
		api.POST("/rooms/groups", cfg.Room.CreateGroupRoom)
		api.POST("/rooms/:id/members", cfg.Room.AddRoomMember)
		api.DELETE("/rooms/:id/members/:userId", cfg.Room.RemoveRoomMember)
	}
	if cfg.Messages != nil {
		api.POST("/rooms/:id/messages", cfg.Messages.CreateMessage)
		api.GET("/rooms/:id/messages", cfg.Messages.ListMessages)
		api.DELETE("/messages/:id", cfg.Messages.DeleteMessage)
	}
	if cfg.Reads != nil {
		api.GET("/rooms/:id/read-state", cfg.Reads.GetRoomReadState)
		api.POST("/rooms/:id/read", cfg.Reads.MarkAsRead)
	}
	if cfg.Media != nil {
		api.POST("/media/images", cfg.Media.UploadImage)
	}
	if cfg.WS != nil {
		api.GET("/ws", cfg.WS.Handle)
	}

	if cfg.Admin != nil {
		ad := e.Group("/admin")
		ad.Use(jwt)
		ad.Use(middleware.RequireAdmin())
		ad.PATCH("/users/:id/active", cfg.Admin.SetUserActive)
		ad.PATCH("/rooms/:id/frozen", cfg.Admin.SetRoomFrozen)
		ad.DELETE("/rooms/:id/members/:userId", cfg.Admin.ForceRemoveMember)
	}

	return e
}
