// seedtestuser はローカル開発用のテストユーザーを DB に投入する。
//
//	go run ./cmd/seedtestuser              // 1件（既定 test@example.com）
//	go run ./cmd/seedtestuser -team       // DM 用に複数アカウント（+ alice, bob）
//	go run ./cmd/seedtestuser -reset      // 既存ならパスワード等を上書き
//	go run ./cmd/seedtestuser -team -reset
//
// 既定パスワードは全アカウント共通で password123（TEST_USER_PASSWORD で変更可）。
// 上書き: TEST_USER_EMAIL, TEST_USER_PASSWORD, TEST_USER_NAME
// 環境変数: SEED_TEST_USER_RESET=1, SEED_TEST_USER_TEAM=1
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"chat-app/auth"
	"chat-app/db"
	"chat-app/entity"
	"chat-app/repository"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type account struct {
	email string
	name  string
}

func parseFlags() (reset, team bool) {
	reset = strings.TrimSpace(os.Getenv("SEED_TEST_USER_RESET")) == "1"
	team = strings.TrimSpace(os.Getenv("SEED_TEST_USER_TEAM")) == "1"
	for _, a := range os.Args[1:] {
		switch a {
		case "-reset", "--reset":
			reset = true
		case "-team", "--team":
			team = true
		}
	}
	return reset, team
}

func accountsToSeed(team bool, primaryEmail, primaryName string) []account {
	primaryEmail = strings.ToLower(strings.TrimSpace(primaryEmail))
	out := []account{{email: primaryEmail, name: primaryName}}
	if !team {
		return out
	}
	seen := map[string]struct{}{primaryEmail: {}}
	add := func(email, name string) {
		e := strings.ToLower(strings.TrimSpace(email))
		if e == "" {
			return
		}
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		out = append(out, account{email: e, name: name})
	}
	add("alice@example.com", "Alice")
	add("bob@example.com", "Bob")
	return out
}

func main() {
	_ = godotenv.Load()

	reset, team := parseFlags()

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/chatapp?sslmode=disable"
	}

	primaryEmail := strings.TrimSpace(strings.ToLower(os.Getenv("TEST_USER_EMAIL")))
	if primaryEmail == "" {
		primaryEmail = "test@example.com"
	}
	password := os.Getenv("TEST_USER_PASSWORD")
	if password == "" {
		password = "password123"
	}
	primaryName := strings.TrimSpace(os.Getenv("TEST_USER_NAME"))
	if primaryName == "" {
		primaryName = "Test"
	}
	if len(password) < 8 {
		log.Fatal("TEST_USER_PASSWORD must be at least 8 characters")
	}

	if err := db.Up(databaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	gormDB, err := db.Open(databaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	now := time.Now().UTC()

	ctx := context.Background()
	users := repository.NewUserRepository(gormDB)
	accs := accountsToSeed(team, primaryEmail, primaryName)

	var created, skipped, updated int
	for _, acc := range accs {
		existing, err := users.FindByEmail(ctx, acc.email)
		if err != nil {
			log.Fatalf("find %s: %v", acc.email, err)
		}
		if existing != nil {
			if !reset {
				fmt.Printf("skip (exists): %s\n", acc.email)
				skipped++
				continue
			}
			res := gormDB.WithContext(ctx).Model(&entity.User{}).Where("email = ?", acc.email).Updates(map[string]any{
				"password_hash": hash,
				"name":          acc.name,
				"active":        true,
				"updated_at":    now,
			})
			if res.Error != nil {
				log.Fatalf("update %s: %v", acc.email, res.Error)
			}
			if res.RowsAffected == 0 {
				log.Fatalf("update %s: no row matched", acc.email)
			}
			fmt.Printf("reset: %s (%s)\n", acc.email, acc.name)
			updated++
			continue
		}

		u := &entity.User{
			ID:           uuid.New(),
			Name:         acc.name,
			Email:        acc.email,
			PasswordHash: hash,
			Role:         entity.UserRoleUser,
			Active:       true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := users.Create(ctx, u); err != nil {
			log.Fatalf("create %s: %v", acc.email, err)
		}
		fmt.Printf("created: %s (%s)\n", acc.email, acc.name)
		created++
	}

	fmt.Println()
	if team {
		fmt.Println("team seed summary (-team): 共通パスワードは TEST_USER_PASSWORD または既定 password123")
	} else {
		fmt.Println("seed summary:")
	}
	fmt.Printf("  created: %d  updated: %d  skipped: %d\n", created, updated, skipped)
	if created+updated > 0 {
		fmt.Printf("  password (共通): %s\n", password)
	}
}
