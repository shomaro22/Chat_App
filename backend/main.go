package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"

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
	loadDotEnv()

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

	rdb := db.OpenRedis()
	rateLim := repository.NewRateLimitRepository(rdb)
	if rdb != nil {
		defer func() { _ = rdb.Close() }()
		log.Printf("redis rate limit: connected (%s)", strings.TrimSpace(os.Getenv("REDIS_ADDR")))
	} else {
		log.Println("redis rate limit: disabled (fail-open mode)")
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

	msgCtl := controller.NewMessageController(controller.NewMessageUseCaseAdapter(msgUC), hub, mediaUC)
	readCtl := controller.NewMessageReadController(controller.NewMessageReadUseCaseAdapter(readUC), hub)
	wsCtl := controller.NewWSController(wsHubAdapter{hub}, controller.NewRoomUseCaseAdapter(roomUC), presenceUC)
	authCtl := controller.NewAuthController(authUC, bootstrapAdmin)
	mediaCtl := controller.NewMediaController(mediaUC)
	adminCtl := controller.NewAdminController(adminUC, wsHubAdapter{hub}, hub, presenceUC)

	rateUC := usecase.NewRateLimitUseCase(rateLim)
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
			middleware.LoginRateLimit(rateUC),
		},
		RateLimitRegister: []echo.MiddlewareFunc{
			middleware.RegisterRateLimit(rateUC),
		},
		RateLimitAfterJWT: []echo.MiddlewareFunc{
			middleware.AfterAuthRateLimits(rateUC),
		},
	})
	if err := e.Start(addr); err != nil {
		log.Fatal(err)
	}
}

// loadDotEnv はカレントディレクトリの .env があれば読み込む（無ければ何もしない）。
// 既に設定されている環境変数は上書きしない（godotenv の既定）。
func loadDotEnv() {
	const envFile = ".env"
	if _, err := os.Stat(envFile); err != nil {
		return
	}
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("warning: load .env: %v", err)
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
