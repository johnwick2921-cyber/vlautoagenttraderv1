// Package installpath is the ONE resolver for where an installation keeps its
// mutable state (W-ONE-BUTTON M2, MUST-2). The running bot (main.go via
// config.Init) and the operator CLI (cmd/maintenance-hold) both resolve the
// data directory here, so a hold the CLI writes is the hold the bot reads — a
// CLI that holds a file the bot never reads is a hold that does not exist.
//
// It is a leaf package on purpose: config imports logger/telemetry/mcp and
// store imports config, so neither can host a resolver the CLI can use
// without their side effects.
package installpath

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// DefaultDBPath is the database path when DB_PATH is unset (config.Init's
// historical default).
const DefaultDBPath = "data/data.db"

// DBPath resolves the database path exactly as config.Init does: DB_PATH if
// set and non-empty, else DefaultDBPath.
func DBPath(getenv func(string) string) string {
	if v := getenv("DB_PATH"); v != "" {
		return v
	}
	return DefaultDBPath
}

// DataDir is the directory of dbPath, anchored on workDir when dbPath is
// relative (the bot resolves relative paths against its WorkingDirectory),
// always absolute.
func DataDir(workDir, dbPath string) string {
	p := dbPath
	if !filepath.IsAbs(p) {
		p = filepath.Join(workDir, p)
	}
	return filepath.Dir(filepath.Clean(p))
}

// DotEnvGetenv returns a getenv that answers the way the bot's environment
// looks after main.go's godotenv.Load("<installDir>/.env"): a variable already
// in the process environment wins; <installDir>/.env fills the gaps; an
// absent or unreadable .env means the process environment only.
func DotEnvGetenv(installDir string) func(string) string {
	vals, err := godotenv.Read(filepath.Join(installDir, ".env"))
	if err != nil {
		vals = nil
	}
	return func(k string) string {
		if v, ok := os.LookupEnv(k); ok {
			return v
		}
		return vals[k]
	}
}
