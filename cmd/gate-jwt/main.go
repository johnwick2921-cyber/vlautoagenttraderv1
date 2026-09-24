// Command gate-jwt mints a session token for a LOCAL user via the SAME
// secret resolution the server uses (godotenv.Load → config.Init →
// auth.SetJWTSecret) and the same signer. Used by the acceptance-gate E2E suite
// so Playwright can drive the owner's own local UI without a password prompt,
// and by any lane that needs to read a protected GET.
//
// It mints a MACHINE token (scope "gate-jwt", M3 red-team H1): no password was
// proven, so the API refuses it on the credential routes (/api/user/password,
// /api/reset-account), the Telegram-config routes (/api/telegram*) and
// /api/updates*. Every other protected route answers it as before.
//
// RUN IT FROM THE REPO ROOT (/home/hoang/nofx). godotenv.Load() reads .env
// relative to the WORKING DIRECTORY, and .env is not tracked, so running this
// from a worktree or any other directory silently falls back to the default
// JWT secret and mints a perfectly well-formed token the server answers 401 to.
//
//	go run ./cmd/gate-jwt <email> data/data.db
//
// Local, single-owner, SIM-only. stdout is NOT the token alone: the logger
// writes to stdout too, so log lines come first (store.New's "✅ Database
// initialized", and config.Init's JWT_SECRET warning when the secret is
// unset). The token is the LAST line, with no trailing newline, and the only
// eyJ… segment — capture it by that, never the whole of stdout. Errors go to
// stderr with a non-zero exit, so capture it with pipefail: without it a
// failed mint leaves TOK empty and the line still exits 0.
//
//	set -o pipefail
//	TOK=$(go run ./cmd/gate-jwt <email> data/data.db | grep -oE 'eyJ[A-Za-z0-9_.-]+' | tail -1)
//	test -n "$TOK"
//
// (TestGateJWTBinaryPrintsAGateScopedTokenLast runs the built tool and pins
// this shape; TestGateJWTDocumentedCaptureFailsWhenTheMintFails runs these
// three lines against it — a failed mint must fail them.)
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

	tok, err := mintGateToken(st, email)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(tok)
}

// mintGateToken is the tool's whole mint: the user row by email, then a
// MACHINE token (scope gate-jwt) carrying that row's id and email. main()
// calls it; main_test.go drives the token it returns through the production
// server (canon 53), so the scope the tool mints and the routes the API
// admits it to are pinned together. main_pin_test.go pins that it is the
// package's ONE mint and main's only way to one, and runs the built tool.
func mintGateToken(st *store.Store, email string) (string, error) {
	user, err := st.User().GetByEmail(email)
	if err != nil {
		return "", fmt.Errorf("no such user: %w", err)
	}
	tok, err := auth.GenerateScopedJWT(user.ID, user.Email, auth.ScopeGateJWT)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	return tok, nil
}
