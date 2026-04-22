package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"chat-app/auth"
	"chat-app/controller"
	"chat-app/db"
	"chat-app/entity"
	"chat-app/middleware"
	"chat-app/repository"
	"chat-app/router"
	"chat-app/usecase"
	"chat-app/wshub"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	labmw "github.com/labstack/echo/v4/middleware"
)

func buildApp() (*echo.Echo, string, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable"
	}
	if err := db.Up(databaseURL); err != nil {
		return nil, "", err
	}
	gormDB, err := db.Open(databaseURL)
	if err != nil {
		return nil, "", err
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, "", err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, "", err
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		jwtSecret = "dev-insecure-jwt-secret-do-not-use-in-prod!!"
	}
	bootstrapAdmin := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	publicRoot := os.Getenv("PUBLIC_ROOT")
	if publicRoot == "" {
		publicRoot = "data/public"
	}
	if err := os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755); err != nil {
		return nil, "", err
	}

	rdb := db.OpenRedis()
	rateLim := repository.NewRateLimitRepository(rdb)

	hub := wshub.NewConnectionManager()
	users := repository.NewUserRepository(gormDB)
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)
	roomTx := repository.NewRoomUnitOfWork(gormDB)
	msgRepo := repository.NewMessageRepository(gormDB)
	readRepo := repository.NewMessageReadRepository(gormDB)
	tokRepo := repository.NewAuthTokenRepository(gormDB)
	reactionRepo := repository.NewMessageReactionRepository(gormDB)
	pinRepo := repository.NewRoomPinRepository(gormDB)
	bookmarkRepo := repository.NewUserBookmarkRepository(gormDB)
	blockRepo := repository.NewUserBlockRepository(gormDB)

	tokenSvc := auth.NewTokenService([]byte(jwtSecret), auth.DefaultTokenTTL)
	authUC := usecase.NewAuthUseCase(users, tokenSvc)
	appPublicURL := strings.TrimSpace(os.Getenv("PUBLIC_APP_URL"))
	if appPublicURL == "" {
		appPublicURL = "http://localhost:5173"
	}
	requireVerify := strings.TrimSpace(os.Getenv("REQUIRE_EMAIL_VERIFY")) == "1"
	authUC = authUC.WithEmailFlow(tokRepo, usecase.LogMailer{}, requireVerify, appPublicURL)
	roomUC := usecase.NewRoomUseCase(roomTx, users, rooms, members, msgRepo, readRepo, hubOnlineLookup{hub}, blockRepo)
	mediaUC := usecase.NewMediaUseCase(publicRoot)
	msgUC := usecase.NewMessageUseCase(msgRepo, users, rooms, mediaUC, blockRepo).
		WithEventHook(controller.NewMessageEventPublisher(hub, blockRepo))
	readUC := usecase.NewMessageReadUseCase(msgRepo, readRepo, rooms, users, blockRepo)
	presenceUC := usecase.NewPresenceUseCase(rooms)
	adminUC := usecase.NewAdminUseCase(users, rooms, members)
	profileUC := usecase.NewProfileUseCase(users)
	socialUC := usecase.NewSocialUseCase(msgRepo, reactionRepo, pinRepo, bookmarkRepo, rooms, blockRepo)

	roomCtl := controller.NewRoomController(controller.NewRoomUseCaseAdapter(roomUC))
	validateSession := func(c echo.Context, uid uuid.UUID, _ entity.UserRole) error {
		u, err := users.FindByID(c.Request().Context(), uid)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		if u == nil || !u.Active {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		c.Set(middleware.ContextUserRoleKey, string(u.Role))
		return nil
	}

	msgCtl := controller.NewMessageController(controller.NewMessageUseCaseAdapter(msgUC), mediaUC)
	readCtl := controller.NewMessageReadController(controller.NewMessageReadUseCaseAdapter(readUC), hub)
	wsCtl := controller.NewWSController(wsHubAdapter{hub}, controller.NewRoomUseCaseAdapter(roomUC), presenceUC)
	authCtl := controller.NewAuthController(authUC, profileUC, bootstrapAdmin)
	mediaCtl := controller.NewMediaController(mediaUC)
	socialCtl := controller.NewSocialController(socialUC)
	adminCtl := controller.NewAdminController(adminUC, wsHubAdapter{hub}, hub, presenceUC)

	rateUC := usecase.NewRateLimitUseCase(rateLim)

	corsRaw := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if corsRaw == "" && strings.TrimSpace(os.Getenv("CORS_DISABLE")) != "1" {
		du := strings.ToLower(databaseURL)
		if strings.Contains(du, "127.0.0.1") || strings.Contains(du, "localhost") {
			corsRaw = "http://127.0.0.1:5173,http://localhost:5173,http://127.0.0.1:4173,http://localhost:4173"
		}
	}
	var preAuth []echo.MiddlewareFunc
	if corsRaw != "" {
		var origins []string
		for _, p := range strings.Split(corsRaw, ",") {
			if s := strings.TrimSpace(p); s != "" {
				origins = append(origins, s)
			}
		}
		if len(origins) > 0 {
			preAuth = append(preAuth, labmw.CORSWithConfig(labmw.CORSConfig{
				AllowOrigins: origins,
				AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
				AllowMethods: []string{
					http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch,
					http.MethodPost, http.MethodDelete, http.MethodOptions,
				},
			}))
		}
	}

	e := router.New(router.Config{
		PreAuthMiddleware: preAuth,
		Auth:              authCtl,
		Room:              roomCtl,
		Messages:          msgCtl,
		Reads:             readCtl,
		WS:                wsCtl,
		Media:             mediaCtl,
		Social:            socialCtl,
		Admin:             adminCtl,
		ParseJWT:          authUC.ParseToken,
		ValidateSession:   validateSession,
		StaticDir:         publicRoot,
		RateLimitLogin: []echo.MiddlewareFunc{
			middleware.LoginRateLimit(rateUC),
		},
		RateLimitRegister: []echo.MiddlewareFunc{
			middleware.RegisterRateLimit(rateUC),
		},
		RateLimitAfterJWT: []echo.MiddlewareFunc{
			middleware.AfterAuthRateLimits(rateUC),
		},
	})
	return e, addr, nil
}

// wsHubAdapter は *wshub.ConnectionManager を controller.WebSocketHub へ適合させる（Presence 戻り値の変換のみ）。
type wsHubAdapter struct{ *wshub.ConnectionManager }

func (a wsHubAdapter) Register(userID uuid.UUID, conn *websocket.Conn) (uuid.UUID, controller.HubPresenceChange) {
	id, ch := a.ConnectionManager.Register(userID, conn)
	return id, controller.HubPresenceChange{Changed: ch.Changed, UserID: ch.UserID, IsOnline: ch.IsOnline}
}

func (a wsHubAdapter) RemoveConn(connID uuid.UUID) controller.HubPresenceChange {
	ch := a.ConnectionManager.RemoveConn(connID)
	return controller.HubPresenceChange{Changed: ch.Changed, UserID: ch.UserID, IsOnline: ch.IsOnline}
}

func (a wsHubAdapter) CloseUserConnections(userID uuid.UUID) controller.HubPresenceChange {
	ch := a.ConnectionManager.CloseUserConnections(userID)
	return controller.HubPresenceChange{Changed: ch.Changed, UserID: ch.UserID, IsOnline: ch.IsOnline}
}

func (a wsHubAdapter) BroadcastToRoom(roomID uuid.UUID, payload []byte) {
	a.ConnectionManager.BroadcastToRoom(roomID, payload)
}

// hubOnlineLookup は WS 接続数からオンライン判定する。
type hubOnlineLookup struct{ *wshub.ConnectionManager }

func (h hubOnlineLookup) IsUserOnline(userID uuid.UUID) bool {
	return h.ConnectionManager.IsUserOnline(userID)
}
