package main

import (
	"log"
	"os"

	"chat-app/db"
	"chat-app/router"
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

	e := router.New()
	if err := e.Start(addr); err != nil {
		log.Fatal(err)
	}
}
