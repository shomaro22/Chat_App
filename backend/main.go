package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable"
	}
	if err := db.Up(databaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	gormDB, err := db.Open(databaseURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("ping: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		log.Println("warning: JWT_SECRET unset or shorter than 32 bytes; using insecure dev default")
		jwtSecret = "dev-insecure-jwt-secret-do-not-use-in-prod!!"
	}
	bootstrapAdmin := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	publicRoot := os.Getenv("PUBLIC_ROOT")
	if publicRoot == "" {
		publicRoot = "data/public"
	}
	if err := os.MkdirAll(filepath.Join(publicRoot, "uploads"), 0o755); err != nil {
		log.Fatalf("public dir: %v", err)
	}

	log.Printf("db ok; http %s", addr)

	rdb, err := db.OpenRedis(context.Background())
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	rateLim := repository.NewFixedWindowRateLimiter(rdb)
	if rdb != nil {
		defer func() { _ = rdb.Close() }()
		log.Printf("redis rate limit: connected (%s)", strings.TrimSpace(os.Getenv("REDIS_ADDR")))
		if err := rateLim.LoadScript(context.Background()); err != nil {
			log.Printf("redis SCRIPT LOAD (optional): %v", err)
		}
	} else {
		log.Println("redis rate limit: disabled (REDIS_ADDR empty; all Allow pass-through)")
	}

	hub := wshub.NewConnectionManager()
	users := repository.NewUserRepository(gormDB)
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)
	roomTx := repository.NewRoomUnitOfWork(gormDB)
	msgRepo := repository.NewMessageRepository(gormDB)
	readRepo := repository.NewMessageReadRepository(gormDB)

	tokenSvc := auth.NewTokenService([]byte(jwtSecret), auth.DefaultTokenTTL)
	authUC := usecase.NewAuthUseCase(users, tokenSvc)
	roomUC := usecase.NewRoomUseCase(roomTx, users, rooms, members, msgRepo, readRepo, hubOnlineLookup{hub})
	mediaUC := usecase.NewMediaUseCase(publicRoot)
	msgUC := usecase.NewMessageUseCase(msgRepo, users, rooms, mediaUC)
	readUC := usecase.NewMessageReadUseCase(msgRepo, readRepo, rooms, users)
	presenceUC := usecase.NewPresenceUseCase(rooms)
	adminUC := usecase.NewAdminUseCase(users, rooms, members)

	roomCtl := controller.NewRoomController(controller.NewRoomUseCaseAdapter(roomUC))
	validateSession := func(c echo.Context, uid uuid.UUID, _ entity.UserRole) error {
		u, err := auth.RequireActiveUser(c.Request().Context(), users, uid)
		if err != nil {
			if errors.Is(err, auth.ErrNotActive) {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		c.Set(middleware.ContextUserRoleKey, string(u.Role))
		return nil
	}

	msgCtl := controller.NewMessageController(controller.NewMessageUseCaseAdapter(msgUC), hub, mediaUC)
	readCtl := controller.NewMessageReadController(controller.NewMessageReadUseCaseAdapter(readUC), hub)
	wsCtl := controller.NewWSController(wsHubAdapter{hub}, controller.NewRoomUseCaseAdapter(roomUC), presenceUC)
	authCtl := controller.NewAuthController(authUC, bootstrapAdmin)
	mediaCtl := controller.NewMediaController(mediaUC)
	adminCtl := controller.NewAdminController(adminUC, wsHubAdapter{hub}, hub, presenceUC)

	win1m := time.Minute
	e := router.New(router.Config{
		Auth:            authCtl,
		Room:            roomCtl,
		Messages:        msgCtl,
		Reads:           readCtl,
		WS:              wsCtl,
		Media:           mediaCtl,
		Admin:           adminCtl,
		ParseJWT:        authUC.ParseToken,
		ValidateSession: validateSession,
		StaticDir:       publicRoot,
		RateLimitLogin: []echo.MiddlewareFunc{
			middleware.RedisLoginRateLimit(rateLim, 20, win1m),
		},
		RateLimitRegister: []echo.MiddlewareFunc{
			middleware.RedisRegisterRateLimit(rateLim, 10, win1m),
		},
		RateLimitMessage: []echo.MiddlewareFunc{
			middleware.RedisMessageSendRateLimit(rateLim, 60, win1m),
		},
		RateLimitMedia: []echo.MiddlewareFunc{
			middleware.RedisMediaUploadRateLimit(rateLim, 30, win1m),
		},
		RateLimitWS: []echo.MiddlewareFunc{
			middleware.RedisWSConnectRateLimit(rateLim, 30, win1m),
		},
	})
	if err := e.Start(addr); err != nil {
		log.Fatal(err)
	}
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

// hubOnlineLookup は WS 接続数からオンライン判定する。
type hubOnlineLookup struct{ *wshub.ConnectionManager }

func (h hubOnlineLookup) IsUserOnline(userID uuid.UUID) bool {
	return h.ConnectionManager.IsUserOnline(userID)
}
