package installpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDBPathDefaultsAndEnvOverride(t *testing.T) {
	if got := DBPath(func(string) string { return "" }); got != "data/data.db" {
		t.Fatalf("unset DB_PATH → %q, want data/data.db", got)
	}
	if got := DBPath(func(k string) string {
		if k == "DB_PATH" {
			return "var/x.db"
		}
		return ""
	}); got != "var/x.db" {
		t.Fatalf("DB_PATH override → %q", got)
	}
}

func TestDataDirIsAbsoluteAndAnchoredOnTheWorkDir(t *testing.T) {
	wd := t.TempDir()
	if got := DataDir(wd, "data/data.db"); got != filepath.Join(wd, "data") {
		t.Fatalf("relative db path → %q, want %q", got, filepath.Join(wd, "data"))
	}
	if got := DataDir(wd, "/abs/place/x.db"); got != "/abs/place" {
		t.Fatalf("absolute db path → %q", got)
	}
}

// .env in the installation dir is read the way the bot's godotenv.Load reads
// it: a variable already in the process environment WINS; .env fills gaps.
func TestDotEnvGetenvMatchesGodotenvLoadPrecedence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DB_PATH=from/dotenv.db\nOTHER=x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DB_PATH", "placeholder")
	os.Unsetenv("DB_PATH")
	if got := DotEnvGetenv(dir)("DB_PATH"); got != "from/dotenv.db" {
		t.Fatalf(".env fills an unset var: got %q", got)
	}
	t.Setenv("DB_PATH", "from/env.db")
	if got := DotEnvGetenv(dir)("DB_PATH"); got != "from/env.db" {
		t.Fatalf("process env wins over .env: got %q", got)
	}
	if got := DotEnvGetenv(t.TempDir())("OTHER"); got != "" {
		t.Fatalf("no .env → env only: got %q", got)
	}
}
