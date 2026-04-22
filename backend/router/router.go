package router

import (
	"net/http"

	"chat-app/controller"
	"chat-app/middleware"

	"github.com/labstack/echo/v4"
)

// Config は Echo に載せるハンドラとミドルウェア設定。
type Config struct {
	Auth              *controller.AuthController
	Room              *controller.RoomController
	Messages          *controller.MessageController
	Reads             *controller.MessageReadController
	WS                *controller.WSController
	Media             *controller.MediaController
	Social            *controller.SocialController
	Admin             *controller.AdminController
	ParseJWT          middleware.TokenParser
	ValidateSession   middleware.SessionValidator // nil 可（JWT のみ検証）
	StaticDir         string                      // ローカルパス（/static で配信）
	PreAuthMiddleware []echo.MiddlewareFunc
	// RateLimitLogin / RateLimitRegister は該当 POST にだけ適用（空なら未適用）。
	RateLimitLogin    []echo.MiddlewareFunc
	RateLimitRegister []echo.MiddlewareFunc
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
		e.POST("/auth/forgot-password", cfg.Auth.ForgotPassword)
		e.POST("/auth/reset-password", cfg.Auth.ResetPassword)
		e.POST("/auth/verify-email", cfg.Auth.VerifyEmail)
	}

	jwt := middleware.JWTAuth(cfg.ParseJWT, cfg.ValidateSession)

	api := e.Group("")
	api.Use(jwt)
	for _, mw := range cfg.RateLimitAfterJWT {
		api.Use(mw)
	}
	if cfg.Auth != nil {
		api.GET("/me", cfg.Auth.Me)
		api.PATCH("/me", cfg.Auth.PatchMe)
		api.POST("/me/change-password", cfg.Auth.ChangePassword)
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
		api.PATCH("/rooms/:id/messages/:messageId", cfg.Messages.PatchMessage)
		api.POST("/rooms/:id/messages", cfg.Messages.CreateMessage)
		api.GET("/rooms/:id/messages", cfg.Messages.ListMessages)
		api.DELETE("/messages/:id", cfg.Messages.DeleteMessage)
		api.GET("/search/messages", cfg.Messages.SearchMessages)
	}
	if cfg.Reads != nil {
		api.GET("/rooms/:id/read-state", cfg.Reads.GetRoomReadState)
		api.POST("/rooms/:id/read", cfg.Reads.MarkAsRead)
	}
	if cfg.Media != nil {
		api.POST("/media/images", cfg.Media.UploadImage)
		api.POST("/media/files", cfg.Media.UploadFile)
	}
	if cfg.Social != nil {
		api.POST("/rooms/:id/messages/:messageId/reactions", cfg.Social.PostReaction)
		api.DELETE("/rooms/:id/messages/:messageId/reactions", cfg.Social.DeleteReaction)
		api.GET("/rooms/:id/messages/:messageId/reactions", cfg.Social.ListReactions)
		api.POST("/rooms/:id/pin", cfg.Social.PostPin)
		api.DELETE("/rooms/:id/pin", cfg.Social.DeletePin)
		api.GET("/rooms/:id/pin", cfg.Social.GetPin)
		api.POST("/messages/:id/bookmark", cfg.Social.PostBookmark)
		api.DELETE("/messages/:id/bookmark", cfg.Social.DeleteBookmark)
		api.GET("/me/bookmarks", cfg.Social.ListBookmarks)
		api.POST("/users/:id/block", cfg.Social.PostBlockUser)
		api.DELETE("/users/:id/block", cfg.Social.DeleteBlockUser)
		api.GET("/users/:id/block", cfg.Social.GetBlockUser)
	}
	if cfg.WS != nil {
		api.GET("/ws", cfg.WS.Handle)
	}

	// /admin/* は JWT 済み api グループ配下（トークン検証・セッション確認を他 API と同じにする）
	if cfg.Admin != nil {
		ad := api.Group("/admin")
		ad.Use(middleware.RequireAdmin())
		ad.GET("/users", cfg.Admin.ListUsers)
		ad.GET("/rooms", cfg.Admin.ListRooms)
		ad.PATCH("/users/:id/active", cfg.Admin.SetUserActive)
		ad.PATCH("/rooms/:id/frozen", cfg.Admin.SetRoomFrozen)
		ad.DELETE("/rooms/:id/members/:userId", cfg.Admin.ForceRemoveMember)
	}

	return e
}
