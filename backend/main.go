package main

import (
	"log"
	"os"

	"chat-app/controller"
	"chat-app/db"
	"chat-app/repository"
	"chat-app/router"
	"chat-app/usecase"
	"chat-app/wshub"
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
	log.Printf("db ok; http %s", addr)

	users := repository.NewUserRepository(gormDB)
	rooms := repository.NewRoomRepository(gormDB)
	members := repository.NewRoomMemberRepository(gormDB)
	roomTx := repository.NewRoomUnitOfWork(gormDB)
	roomUC := usecase.NewRoomUseCase(roomTx, users, rooms, members)
	roomCtl := controller.NewRoomController(roomUC)
	hub := wshub.NewConnectionManager()
	msgRepo := repository.NewMessageRepository(gormDB)
	msgUC := usecase.NewMessageUseCase(msgRepo, members)
	msgCtl := controller.NewMessageController(msgUC, hub)
	readRepo := repository.NewMessageReadRepository(gormDB)
	readUC := usecase.NewMessageReadUseCase(msgRepo, readRepo, members)
	readCtl := controller.NewMessageReadController(readUC, hub)
	presenceUC := usecase.NewPresenceUseCase(rooms)
	wsCtl := controller.NewWSController(hub, roomUC, presenceUC)
	e := router.New(roomCtl, msgCtl, wsCtl, readCtl)
	if err := e.Start(addr); err != nil {
		log.Fatal(err)
	}
}
