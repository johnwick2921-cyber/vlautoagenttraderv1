package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/config"
	"nofx/internal/holdcli"
	"nofx/store"
)

// MUST-2 (CTO review of cb513079) — the production call-site proof: with a
// real installation layout (DB_PATH set ONLY in .env, as on the owner's and
// partners' machines), the data dir the running bot resolves (loadDotEnv →
// config.Init → resolveMaintenanceDataDir, the exact calls main makes) is the
// data dir the operator CLI writes — even when the CLI is run from another
// directory. A CLI that holds a file the bot never reads is a hold that does
// not exist.
func TestMaintenanceHoldCLIWritesTheFileTheBotReads(t *testing.T) {
	inst := t.TempDir()
	if err := os.WriteFile(filepath.Join(inst, ".env"), []byte("DB_PATH=var/db/data.db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DB_PATH", "placeholder")
	os.Unsetenv("DB_PATH") // DB_PATH comes ONLY from .env, as on a real install

	// ── the bot, from its WorkingDirectory ──
	t.Chdir(inst)
	loadDotEnv(".env")
	config.Init()
	botDir := resolveMaintenanceDataDir(config.Get().DBPath)
	if want := filepath.Join(inst, "var", "db"); botDir != want {
		t.Fatalf("bot data dir = %q, want %q", botDir, want)
	}
	if store.ReadMaintenanceHold(botDir).Held {
		t.Fatal("precondition: no hold")
	}

	// ── the operator, from somewhere else, pointing at the installation ──
	t.Chdir(t.TempDir())
	os.Unsetenv("DB_PATH") // the operator's shell does not have the bot's env
	var out, errb bytes.Buffer
	if rc := holdcli.Run([]string{"--install-dir", inst, "set", "--job", "job-mustwo"}, &out, &errb); rc != 0 {
		t.Fatalf("cli set rc=%d: %s", rc, errb.String())
	}
	st := store.ReadMaintenanceHold(botDir)
	if !st.Held || st.Hold.JobID != "job-mustwo" {
		t.Fatalf("the bot does not see the hold the CLI wrote (bot reads %s; cli said %q): %+v", store.MaintenanceHoldPath(botDir), strings.TrimSpace(out.String()), st)
	}
}

// main sets the data dir BEFORE traders load (they auto-start inside
// LoadTradersFromStore); a later call would leave the first cycles unguarded.
func TestMainResolvesTheMaintenanceDataDirBeforeTradersLoad(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	set := strings.Index(src, "trader.SetMaintenanceDataDir(resolveMaintenanceDataDir(cfg.DBPath))")
	load := strings.Index(src, "traderManager.LoadTradersFromStore(")
	override := strings.Index(src, "cfg.DBPath = os.Args[1]")
	if set < 0 || load < 0 || override < 0 {
		t.Fatalf("anchors missing: set=%d load=%d override=%d", set, load, override)
	}
	if !(override < set && set < load) {
		t.Fatalf("order must be: DB path override (%d) < SetMaintenanceDataDir (%d) < LoadTradersFromStore (%d)", override, set, load)
	}
}
