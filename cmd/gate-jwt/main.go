// Command gate-jwt mints a session token for a LOCAL user via the SAME
// production path the /api/login handler uses (godotenv.Load → config.Init →
// auth.SetJWTSecret → auth.GenerateJWT). Used by the acceptance-gate E2E suite
// so Playwright can drive the owner's own local UI without a password prompt,
// and by any lane that needs to read a protected GET.
//
// RUN IT FROM THE REPO ROOT (/home/hoang/nofx). godotenv.Load() reads .env
// relative to the WORKING DIRECTORY, and .env is not tracked, so running this
// from a worktree or any other directory silently falls back to the default
// JWT secret and mints a perfectly well-formed token the server answers 401 to.
//
//	go run ./cmd/gate-jwt <email> data/data.db
//
// Local, single-owner, SIM-only. Prints the token to stdout and nothing else.
package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"nofx/auth"
	"nofx/config"
	"nofx/store"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gate-jwt <email> [db-path]")
		os.Exit(2)
	}
	email := os.Args[1]
	dbPath := "data/data.db"
	if len(os.Args) > 2 {
		dbPath = os.Args[2]
	}

	// THE 401 BUG (fixed 2026-09-03): config.Init() reads os.Getenv("JWT_SECRET")
	// and NOTHING ELSE — it never loads .env. The server does godotenv.Load()
	// first (main.go), so without this line the tool signed with the DEFAULT
	// secret and every token it minted was rejected. The failure looked like a
	// bad token rather than a bad secret, and it cost more than one lane a
	// blocked cutover. Reproduce the server's resolution, do not approximate it.
	_ = godotenv.Load()
	config.Init()
	cfg := config.Get()
	auth.SetJWTSecret(cfg.JWTSecret)

	st, err := store.New(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "store:", err)
		os.Exit(1)
	}
	defer st.Close()

	user, err := st.User().GetByEmail(email)
	if err != nil {
		fmt.Fprintln(os.Stderr, "no such user:", err)
		os.Exit(1)
	}
	tok, err := auth.GenerateJWT(user.ID, user.Email)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sign:", err)
		os.Exit(1)
	}
	fmt.Print(tok)
}
