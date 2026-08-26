// Command gate-jwt mints a session token for a LOCAL user via the SAME
// production path the /api/login handler uses (config.Init → auth.SetJWTSecret
// → auth.GenerateJWT). Used by the acceptance-gate E2E suite so Playwright can
// drive the owner's own local UI without a password prompt. Prints the token to
// stdout and nothing else. Local, single-owner, SIM-only.
package main

import (
	"fmt"
	"os"

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
