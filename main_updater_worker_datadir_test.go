package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/config"
	"nofx/internal/holdcli"
	"nofx/internal/updaterwire"
	"nofx/internal/updaterworker"
	"nofx/store"
)

// PIN (M4 3b-B U4, C15 as ruled): the worker acts on the data dir the BOT
// reads and the hold CLI writes. The bot resolves it the way main does
// (loadDotEnv → config.Init → resolveMaintenanceDataDir) from its
// WorkingDirectory with no DB_PATH in its own environment (the shipped unit
// sets none); the operator runs the CLI and the worker from elsewhere.
// A hold the worker writes there is the hold the bot reads. A process DB_PATH
// naming another file is refused (updaterbootstrap.resolveTarget's rule).
func TestDataDirMatchesHoldCLIAndBot(t *testing.T) {
	inst := t.TempDir()
	if err := os.WriteFile(filepath.Join(inst, ".env"), []byte("DB_PATH=var/db/data.db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"var/db/data.db", "data/data.db"} {
		p := filepath.Join(inst, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("SQLite format 3\x00"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("DB_PATH", "placeholder")
	os.Unsetenv("DB_PATH")

	// ── the bot, from its WorkingDirectory ──
	t.Chdir(inst)
	loadDotEnv(".env")
	config.Init()
	botDir := resolveMaintenanceDataDir(config.Get().DBPath)
	os.Unsetenv("DB_PATH") // loadDotEnv exported it into THIS process; the operator's shell has none

	// ── the operator, from elsewhere ──
	t.Chdir(t.TempDir())
	tgt, err := updaterworker.ResolveTarget(inst)
	if err != nil {
		t.Fatal(err)
	}
	sock, err := updaterwire.SocketPathFor(inst)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(inst, "var", "db")
	if botDir != want || tgt.DataDir != botDir || holdcli.DataDirFor(inst) != botDir || filepath.Dir(filepath.Dir(sock)) != botDir {
		t.Fatalf("data dirs: bot %s, worker %s, hold CLI %s, socket %s — want all %s", botDir, tgt.DataDir, holdcli.DataDirFor(inst), sock, want)
	}
	if tgt.DBFile != filepath.Join(botDir, "data.db") || tgt.DBFile != holdcli.DBFileFor(inst) || tgt.LogDir != filepath.Join(inst, "data") {
		t.Fatalf("worker db %s log dir %s", tgt.DBFile, tgt.LogDir)
	}

	// the hold the worker writes is the hold the bot reads
	if already, err := updaterworker.HoldForJob(tgt.DataDir, "job-u4-dd000abc", "v1.2.0", time.Now()); err != nil || already {
		t.Fatalf("HoldForJob: %v (already %v)", err, already)
	}
	st := store.ReadMaintenanceHold(botDir)
	if !st.Held || st.Hold.JobID != "job-u4-dd000abc" || st.Hold.Owner != "updater" || st.Hold.WithdrawEntries {
		t.Fatalf("the bot reads %+v", st)
	}
	if err := updaterworker.ReleaseJob(tgt.DataDir, "job-u4-dd000abc"); err != nil {
		t.Fatal(err)
	}

	// a process DB_PATH naming ANOTHER file is refused; the same file, spelled otherwise, is not
	os.Setenv("DB_PATH", "data/data.db")
	if _, err := updaterworker.ResolveTarget(inst); err == nil || !strings.Contains(err.Error(), "DB_PATH differs") ||
		!strings.Contains(err.Error(), filepath.Join(inst, "data", "data.db")) || !strings.Contains(err.Error(), filepath.Join(botDir, "data.db")) {
		t.Fatalf("a diverging DB_PATH: %v", err)
	}
	os.Setenv("DB_PATH", "./var/db/../db/data.db")
	if got, err := updaterworker.ResolveTarget(inst); err != nil || got.DataDir != botDir {
		t.Fatalf("the same file spelled otherwise: %+v, %v", got, err)
	}
	os.Unsetenv("DB_PATH")
	if _, err := updaterworker.ResolveTarget("relative/dir"); err == nil {
		t.Fatal("a relative install dir was accepted")
	}
}
