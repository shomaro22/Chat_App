package main

import (
	"log"
	"os"
	"path/filepath"

	"chat-app/controller"
	"chat-app/db"
	"chat-app/repository"
	"chat-app/router"
	"chat-app/usecase"
	"chat-app/wshub"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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

	hub := wshub.NewConnectionManager()
	users := repository.NewUserRepository(gormDB)
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)
	roomTx := repository.NewRoomUnitOfWork(gormDB)
	msgRepo := repository.NewMessageRepository(gormDB)
	readRepo := repository.NewMessageReadRepository(gormDB)

	authUC := usecase.NewAuthUseCase(users, []byte(jwtSecret))
	roomUC := usecase.NewRoomUseCase(roomTx, users, rooms, members, msgRepo, readRepo, hubOnlineLookup{hub})
	msgUC := usecase.NewMessageUseCase(msgRepo, members, users, rooms)
	readUC := usecase.NewMessageReadUseCase(msgRepo, readRepo, members, rooms)
	presenceUC := usecase.NewPresenceUseCase(rooms)
	mediaUC := usecase.NewMediaUseCase(publicRoot)
	adminUC := usecase.NewAdminUseCase(users, rooms, members)

	roomCtl := controller.NewRoomController(roomUC)
	msgCtl := controller.NewMessageController(msgUC, hub)
	readCtl := controller.NewMessageReadController(readUC, hub)
	wsCtl := controller.NewWSController(wsHubAdapter{hub}, roomUC, presenceUC)
	authCtl := controller.NewAuthController(authUC, bootstrapAdmin)
	mediaCtl := controller.NewMediaController(mediaUC)
	adminCtl := controller.NewAdminController(adminUC)

	e := router.New(router.Config{
		Auth:      authCtl,
		Room:      roomCtl,
		Messages:  msgCtl,
		Reads:     readCtl,
		WS:        wsCtl,
		Media:     mediaCtl,
		Admin:     adminCtl,
		ParseJWT:  authUC.ParseToken,
		StaticDir: publicRoot,
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

// hubOnlineLookup は WS 接続数からオンライン判定する。
type hubOnlineLookup struct{ *wshub.ConnectionManager }

func (h hubOnlineLookup) IsUserOnline(userID uuid.UUID) bool {
	return h.ConnectionManager.IsUserOnline(userID)
}
