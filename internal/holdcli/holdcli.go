// Package holdcli is the operator CLI for the W-ONE-BUTTON installation-wide
// maintenance hold. cmd/maintenance-hold is a thin main over Run; a test in
// the bot's own main package drives Run to prove the CLI writes the file the
// running bot reads (MUST-2).
//
//	maintenance-hold [--install-dir <dir>] set --job <id> [--reason <text>]
//	maintenance-hold [--install-dir <dir>] status
//	maintenance-hold [--install-dir <dir>] clear --job <id>
//	maintenance-hold [--install-dir <dir>] clear --force   # corrupt/foreign file, operator recovery
//
// --install-dir is the bot's WorkingDirectory (default: the current dir). The
// data dir is resolved exactly as the bot resolves it: DB_PATH from the
// process environment, else from <install-dir>/.env, else data/data.db;
// relative paths anchored on <install-dir>.
package holdcli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"nofx/internal/installpath"
	"nofx/store"
)

// DataDirFor is the CLI's half of the ONE resolver.
func DataDirFor(installDir string) string {
	return installpath.DataDir(installDir, installpath.DBPath(installpath.DotEnvGetenv(installDir)))
}

// Run executes the CLI and returns the exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	top := flag.NewFlagSet("maintenance-hold", flag.ContinueOnError)
	top.SetOutput(stderr)
	wd, _ := os.Getwd()
	installDir := top.String("install-dir", wd, "the bot's WorkingDirectory (its .env and DB_PATH decide the data dir)")
	if err := top.Parse(args); err != nil {
		return 2
	}
	rest := top.Args()
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "usage: maintenance-hold [--install-dir d] set|status|clear ...")
		return 2
	}
	dataDir := DataDirFor(*installDir)
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
		if err := store.WriteMaintenanceHold(dataDir, h); err != nil {
			fmt.Fprintln(stderr, "set:", err)
			return 1
		}
		fmt.Fprintf(stdout, "held job=%s path=%s\n", *job, store.MaintenanceHoldPath(dataDir))
		return 0
	case "status":
		st := store.ReadMaintenanceHold(dataDir)
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
			err = store.ForceClearMaintenanceHold(dataDir)
		} else {
			err = store.ClearMaintenanceHold(dataDir, *job)
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
