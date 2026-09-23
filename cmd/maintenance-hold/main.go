// Command maintenance-hold is the local operator CLI for the W-ONE-BUTTON
// installation-wide maintenance hold (<data dir>/updater/hold.json). It
// writes through the same store functions every entry gate reads.
//
//	go run ./cmd/maintenance-hold [--data-dir data] set --job <id> [--reason <text>]
//	go run ./cmd/maintenance-hold [--data-dir data] status
//	go run ./cmd/maintenance-hold [--data-dir data] clear --job <id>
//	go run ./cmd/maintenance-hold [--data-dir data] clear --force   # corrupt/foreign file, operator only
//
// --data-dir defaults to the directory of DB_PATH (or "data"), i.e. the same
// directory the running bot resolves for data/data.db.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"nofx/store"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func defaultDataDir() string {
	if v := os.Getenv("DB_PATH"); v != "" {
		return filepath.Dir(v)
	}
	return "data"
}

func run(args []string, stdout, stderr io.Writer) int {
	top := flag.NewFlagSet("maintenance-hold", flag.ContinueOnError)
	top.SetOutput(stderr)
	dataDir := top.String("data-dir", defaultDataDir(), "installation data directory (holds updater/hold.json)")
	if err := top.Parse(args); err != nil {
		return 2
	}
	rest := top.Args()
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "usage: maintenance-hold [--data-dir d] set|status|clear ...")
		return 2
	}
	sub := flag.NewFlagSet(rest[0], flag.ContinueOnError)
	sub.SetOutput(stderr)
	job := sub.String("job", "", "job id that holds / releases")
	reason := sub.String("reason", "", "free-text reason (set)")
	force := sub.Bool("force", false, "clear regardless of content (operator recovery)")
	if err := sub.Parse(rest[1:]); err != nil {
		return 2
	}
	switch rest[0] {
	case "set":
		if *job == "" {
			fmt.Fprintln(stderr, "set requires --job")
			return 2
		}
		h := store.MaintenanceHold{Held: true, JobID: *job, Since: time.Now().UTC().Format(time.RFC3339), Reason: *reason, Owner: "cli"}
		if err := store.WriteMaintenanceHold(*dataDir, h); err != nil {
			fmt.Fprintln(stderr, "set:", err)
			return 1
		}
		fmt.Fprintf(stdout, "held job=%s path=%s\n", *job, store.MaintenanceHoldPath(*dataDir))
		return 0
	case "status":
		st := store.ReadMaintenanceHold(*dataDir)
		out := map[string]any{"held": st.Held, "present": st.Present, "corrupt": st.Corrupt, "path": st.Path}
		if st.Err != "" {
			out["error"] = st.Err
		}
		if st.Present && !st.Corrupt {
			out["job_id"], out["since"], out["owner"], out["reason"] = st.Hold.JobID, st.Hold.Since, st.Hold.Owner, st.Hold.Reason
		}
		b, _ := json.Marshal(out)
		fmt.Fprintln(stdout, string(b))
		return 0
	case "clear":
		var err error
		if *force {
			err = store.ForceClearMaintenanceHold(*dataDir)
		} else {
			err = store.ClearMaintenanceHold(*dataDir, *job)
		}
		if err != nil {
			fmt.Fprintln(stderr, "clear:", err)
			return 1
		}
		fmt.Fprintln(stdout, "cleared")
		return 0
	}
	fmt.Fprintf(stderr, "unknown subcommand %q\n", rest[0])
	return 2
}
