// Package updaterbootstrap is the attended, local CLI for W-ONE-BUTTON M3
// update authorization. cmd/updater-bootstrap is a thin main over Run.
//
//	updater-bootstrap [--install-dir <dir>] enroll [--replace] <email>
//	updater-bootstrap [--install-dir <dir>] authorize <release_id>
//
// enroll binds the installation's update administrator to the app user whose
// email is EXACTLY <email> (the predicate login uses) and writes
// <data>/updater/admin.json + device.key (32 random bytes), both 0600 in a
// 0700 dir. No API creates, resets or reads these files.
//
// authorize (CTO ruling Q1(a)) prints ONE install authorization for
// <release_id> — {release_id, job_id, expires_at, hmac}, valid 5 minutes,
// single use — computed from device.key on this box. The key never leaves the
// box and is never printed; nothing on the API side mints a MAC.
//
// Both are ATTENDED: stdin must be a terminal and the operator types an exact
// confirmation line. Both refuse to run as root (a root-owned data/updater
// would lock the bot out — the M2.1 N4 lesson) and refuse an --install-dir
// with no bot database (the M2.1 N5 lesson: an enrollment the bot never reads
// is one that does not exist).
//
// --install-dir is the bot's WorkingDirectory (default: the current dir),
// made absolute at entry (a relative one used to pass every precondition and
// fail inside, after the typed confirmation — PR #200 fold #18); the data
// dir is resolved by internal/installpath exactly as the bot and
// cmd/maintenance-hold resolve it (ONE resolver).
package updaterbootstrap

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nofx/internal/installpath"
	"nofx/internal/updateauth"

	_ "modernc.org/sqlite"
)

// Seams (tests only): root refusal, the attended check and the clock.
var (
	geteuid    = os.Geteuid
	isTerminal = stdinIsTerminal
	now        = time.Now
)

// DataDirFor is the CLI's half of the ONE resolver (identical to
// holdcli.DataDirFor; a test pins the two together).
func DataDirFor(installDir string) string {
	return installpath.DataDir(installDir, installpath.DBPath(installpath.DotEnvGetenv(installDir)))
}

// DBFileFor is the bot database the same resolver points at.
func DBFileFor(installDir string) string {
	return installpath.DBFile(installDir, installpath.DBPath(installpath.DotEnvGetenv(installDir)))
}

const usage = "usage: updater-bootstrap [--install-dir d] enroll [--replace] <email> | authorize <release_id>"

// Run executes the CLI and returns the exit code: 0 ok, 1 refused/failed,
// 2 usage or a precondition (root, no bot DB, not attended).
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	top := flag.NewFlagSet("updater-bootstrap", flag.ContinueOnError)
	top.SetOutput(stderr)
	wd, _ := os.Getwd()
	installDir := top.String("install-dir", wd, "the bot's WorkingDirectory (its .env and DB_PATH decide the data dir)")
	if err := top.Parse(args); err != nil {
		return 2
	}
	// PR #200 fold #18: absolutize at entry. A relative dir used to pass
	// every precondition (the DB stat resolves against the cwd) and fail
	// only inside updateauth, after the operator had typed the confirmation.
	abs, err := filepath.Abs(*installDir)
	if err != nil {
		fmt.Fprintf(stderr, "refusing: cannot make --install-dir %q absolute: %v\n", *installDir, err)
		return 2
	}
	*installDir = abs
	rest := top.Args()
	if len(rest) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	sub := flag.NewFlagSet(rest[0], flag.ContinueOnError)
	sub.SetOutput(stderr)
	replace := sub.Bool("replace", false, "enroll: replace an existing enrollment (rotates the device key)")
	pos, err := parseInterspersed(sub, rest[1:])
	if err != nil {
		return 2
	}
	switch rest[0] {
	case "enroll", "authorize":
	default:
		fmt.Fprintf(stderr, "unknown subcommand %q\n%s\n", rest[0], usage)
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	if rest[0] == "authorize" && *replace {
		fmt.Fprintln(stderr, "--replace applies to enroll only")
		return 2
	}
	if rc := preconditions(*installDir, stdin, stderr); rc != 0 {
		return rc
	}
	dataDir := DataDirFor(*installDir)
	if rest[0] == "enroll" {
		return enroll(dataDir, DBFileFor(*installDir), pos[0], *replace, stdin, stdout, stderr)
	}
	return authorize(dataDir, pos[0], stdin, stdout, stderr)
}

// parseInterspersed accepts flags before or after the one positional
// argument (`enroll --replace e` and `enroll e --replace`).
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

func preconditions(installDir string, stdin io.Reader, stderr io.Writer) int {
	if geteuid() == 0 {
		fmt.Fprintln(stderr, "refusing to run as root: run updater-bootstrap as the bot's own user (a root-owned data/updater locks the bot out)")
		return 2
	}
	dbFile := DBFileFor(installDir)
	if fi, err := os.Stat(dbFile); err != nil || !fi.Mode().IsRegular() {
		fmt.Fprintf(stderr, "refusing: no bot database at %s — --install-dir must be the installation the bot runs from (its WorkingDirectory)\n", dbFile)
		return 2
	}
	if !isTerminal(stdin) {
		fmt.Fprintln(stderr, "refusing: updater-bootstrap is attended — run it in a terminal (stdin is not a TTY)")
		return 2
	}
	return 0
}

// confirm reads ONE line and requires it to be exactly want (a trailing
// "\n" or "\r\n" is the only thing stripped).
func confirm(stdin io.Reader, stderr io.Writer, want string) bool {
	fmt.Fprintf(stderr, "Type exactly:  %s\n> ", want)
	line, err := bufio.NewReader(io.LimitReader(stdin, 1024)).ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if line != want {
		fmt.Fprintln(stderr, "confirmation did not match — nothing written")
		return false
	}
	return true
}

func enroll(dataDir, dbFile, email string, replace bool, stdin io.Reader, stdout, stderr io.Writer) int {
	userID, passwordHash, err := lookupUserReadOnly(dbFile, email)
	if err != nil {
		fmt.Fprintf(stderr, "refusing: %v\n", err)
		return 1
	}
	if !replace {
		for _, p := range []string{updateauth.AdminPath(dataDir), updateauth.DeviceKeyPath(dataDir)} {
			if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintln(stderr, "refusing: this installation is already enrolled — re-run with --replace to replace it (rotates the device key)")
				return 1
			}
		}
	}
	fmt.Fprintf(stderr, "Enroll the app user with this exact email as the installation's update administrator.\n")
	if !confirm(stdin, stderr, "ENROLL "+email) {
		return 1
	}
	// passwordHash binds the enrollment to the row's CURRENT password
	// (H1/H2 belt): any later password change un-enrolls until --replace.
	if err := updateauth.Enroll(dataDir, userID, email, passwordHash, now(), replace); err != nil {
		if errors.Is(err, updateauth.ErrAlreadyEnrolled) {
			fmt.Fprintln(stderr, "refusing: already enrolled — re-run with --replace")
		} else {
			fmt.Fprintf(stderr, "enroll failed: %v\n", err)
		}
		return 1
	}
	fmt.Fprintf(stdout, "enrolled: user_id=%s… dir=%s (both enrollment files 0600; the key is never printed)\n",
		shortID(userID), updateauth.Dir(dataDir))
	return 0
}

func authorize(dataDir, releaseID string, stdin io.Reader, stdout, stderr io.Writer) int {
	if !updateauth.ValidReleaseID(releaseID) {
		fmt.Fprintln(stderr, "refusing: a release id is an identifier (letters, digits, . _ -), never a URL, path or command")
		return 1
	}
	fmt.Fprintf(stderr, "Authorize ONE install of release %s (valid %s, single use).\n", releaseID, updateauth.MaxAuthorizationWindow)
	if !confirm(stdin, stderr, "AUTHORIZE "+releaseID) {
		return 1
	}
	g, err := updateauth.Authorize(dataDir, releaseID, now())
	if err != nil {
		fmt.Fprintf(stderr, "refusing: %v\n", err)
		return 1
	}
	b, err := json.Marshal(g)
	if err != nil {
		fmt.Fprintf(stderr, "authorize failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, string(b))
	fmt.Fprintf(stderr, "valid until %s — paste it into the Updates page; it works once.\n",
		time.Unix(g.ExpiresAt, 0).UTC().Format(time.RFC3339))
	return 0
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// openReadOnly opens the bot DB so that no statement can write it: the URI
// is mode=ro (the file is opened O_RDONLY) and query_only is on. SQLite may
// still create an EMPTY -wal/-shm beside a WAL-mode database it opens
// read-only in a writable directory (the wal-index); data.db is never
// written. A path the URI cannot carry verbatim is refused.
func openReadOnly(dbFile string) (*sql.DB, error) {
	if strings.ContainsAny(dbFile, "?#%") {
		return nil, fmt.Errorf("database path %q has a character the read-only URI cannot carry", dbFile)
	}
	db, err := sql.Open("sqlite", "file:"+dbFile+"?mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open database read-only: %w", err)
	}
	return db, nil
}

// errNoSuchUser is returned when no user has exactly this email.
var errNoSuchUser = errors.New("no app user has exactly this email (it is matched exactly, as login matches it) — register in the web app first")

// lookupUserReadOnly resolves the user id — and its CURRENT password_hash,
// which the enrollment is bound to (H1/H2 belt) — for EXACTLY email from the
// bot DB, opened read-only (mode=ro + query_only): store.New migrates on
// open, and the store has no read-only constructor, so the CLI opens SQLite
// itself the way cmd/picture_htf_replay does. The hash never leaves this
// process except as Enroll's HMAC under the new device key.
func lookupUserReadOnly(dbFile, email string) (userID, passwordHash string, err error) {
	db, err := openReadOnly(dbFile)
	if err != nil {
		return "", "", err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT id, password_hash FROM users WHERE email = ?`, email)
	if err != nil {
		return "", "", fmt.Errorf("read users: %w", err)
	}
	defer rows.Close()
	var ids, hashes []string
	emptyHash := false
	for rows.Next() {
		var id, hash sql.NullString
		if err := rows.Scan(&id, &hash); err != nil {
			return "", "", fmt.Errorf("read users: %w", err)
		}
		if !id.Valid || id.String == "" {
			return "", "", errors.New("a users row has no id")
		}
		if !hash.Valid || hash.String == "" {
			emptyHash = true
		}
		ids = append(ids, id.String)
		hashes = append(hashes, hash.String)
	}
	if err := rows.Err(); err != nil {
		return "", "", fmt.Errorf("read users: %w", err)
	}
	switch {
	case len(ids) == 0:
		return "", "", errNoSuchUser
	case len(ids) > 1:
		return "", "", errors.New("more than one app user has this email")
	case emptyHash:
		return "", "", errors.New("that user has no password (not a login identity) — refusing")
	}
	return ids[0], hashes[0], nil
}
