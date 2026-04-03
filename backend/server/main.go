package main

import (
	"log"
	"os"

	"chat-app/db"
	"chat-app/router"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable"
	}
	if err := db.Up(dsn); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	gdb, err := db.Open(dsn)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	sqlDB, err := gdb.DB()
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
