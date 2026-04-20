// seed は DB 投入用の CLI（サブコマンドで用途を分ける）。
//
//	go run ./cmd/seed e2e                    # Playwright 用 e2e-admin@ / e2e-bob@
//	go run ./cmd/seed dev                    # ローカル 1 件（既定 test@example.com）
//	go run ./cmd/seed dev -team              # + alice, bob, admin
//	go run ./cmd/seed dev -reset             # 既存ならパスワード等を上書き
//	go run ./cmd/seed dev -team -reset
//
// dev の環境変数: DATABASE_URL, TEST_USER_EMAIL, TEST_USER_PASSWORD, TEST_USER_NAME,
//
//	SEED_TEST_USER_RESET=1, SEED_TEST_USER_TEAM=1
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"chat-app/auth"
	"chat-app/entity"
	"chat-app/db"
	"chat-app/repository"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "e2e":
		runE2ESeed()
	case "dev":
		runDevSeed(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/seed e2e              Playwright 用ユーザー")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/seed dev [-team] [-reset]  ローカル開発用ユーザー")
}

func runE2ESeed() {
	const (
		adminEmail = "e2e-admin@chat.test"
		bobEmail   = "e2e-bob@chat.test"
		password   = "E2ePassw0rd!"
	)
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@127.0.0.1:5432/chatapp?sslmode=disable"
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
	defer func() { _ = sqlDB.Close() }()

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	now := time.Now().UTC()
	ctx := context.Background()
	users := repository.NewUserRepository(gormDB)

	upsert := func(email, name string, role entity.UserRole) {
		existing, err := users.FindByEmail(ctx, email)
		if err != nil {
			log.Fatalf("find %s: %v", email, err)
		}
		if existing != nil {
			existing.Name = name
			existing.PasswordHash = hash
			existing.Role = role
			existing.Active = true
			existing.EmailVerifiedAt = &now
			existing.UpdatedAt = now
			if err := users.Update(ctx, existing); err != nil {
				log.Fatalf("update %s: %v", email, err)
			}
			log.Printf("updated: %s (%s)\n", email, role)
			return
		}
		u := &entity.User{
			ID:           uuid.New(),
			Name:         name,
			Email:        email,
			PasswordHash: hash,
			Role:         role,
			Active:       true,
			EmailVerifiedAt: &now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := users.Create(ctx, u); err != nil {
			log.Fatalf("create %s: %v", email, err)
		}
		log.Printf("created: %s (%s)\n", email, role)
	}

	upsert(adminEmail, "E2E Admin", entity.UserRoleAdmin)
	upsert(bobEmail, "E2E Bob", entity.UserRoleUser)
	log.Println("e2eseed: password for both:", password)
}

type account struct {
	email string
	name  string
	role  entity.UserRole
}

func parseDevFlags(args []string) (reset, team bool) {
	reset = strings.TrimSpace(os.Getenv("SEED_TEST_USER_RESET")) == "1"
	team = strings.TrimSpace(os.Getenv("SEED_TEST_USER_TEAM")) == "1"
	for _, a := range args {
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
	out := []account{{email: primaryEmail, name: primaryName, role: entity.UserRoleUser}}
	if !team {
		return out
	}
	seen := map[string]struct{}{primaryEmail: {}}
	add := func(email, name string, role entity.UserRole) {
		e := strings.ToLower(strings.TrimSpace(email))
		if e == "" {
			return
		}
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		out = append(out, account{email: e, name: name, role: role})
	}
	add("alice@example.com", "Alice", entity.UserRoleUser)
	add("bob@example.com", "Bob", entity.UserRoleUser)
	add("admin@example.com", "Admin", entity.UserRoleAdmin)
	return out
}

func runDevSeed(flagArgs []string) {
	reset, team := parseDevFlags(flagArgs)

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@127.0.0.1:5432/chatapp?sslmode=disable"
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
				"role":          acc.role,
				"active":        true,
				"email_verified_at": now,
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
			Role:         acc.role,
			Active:       true,
			EmailVerifiedAt: &now,
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
