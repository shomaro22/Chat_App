package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	loadDotEnv()
	e, addr, err := buildApp()
	if err != nil {
		log.Fatal(err)
	}
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
