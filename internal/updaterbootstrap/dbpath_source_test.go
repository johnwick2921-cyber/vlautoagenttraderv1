package updaterbootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/internal/holdcli"
	"nofx/internal/updateauth"
	"nofx/store"
)

// PR #200 fold F3 (CTO 1790252194343): installpath.DotEnvGetenv lets a
// DB_PATH in the PROCESS environment win over <install>/.env. That is right
// for the bot (it is the process) and wrong for this CLI: the process is the
// OPERATOR's shell, while the bot started by the shipped systemd unit has no
// DB_PATH in its environment and resolves <install>/.env, else data/data.db.
// An exported DB_PATH silently diverted enroll/authorize. The pins drive the
// real entry (Run, existing seams) with every diverted target EXISTING and
// ACCEPTING — a seeded bot DB, and for authorize a valid enrollment — so a
// refusal can only be caused by the divergence.

const dotEnvVarDB = "DB_PATH=var/db/data.db\n"

// seedBotDB creates a production-schema bot DB at dbFile holding the
// fixture user (install()'s shape, at any path).
func seedBotDB(t *testing.T, dbFile string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dbFile), 0o700); err != nil {
		t.Fatal(err)
	}
	st, err := store.New(dbFile)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	past := bNow.Add(-time.Hour)
	if err := st.User().Create(&store.User{ID: bUser, Email: bEmail, PasswordHash: "$2a$10$not-a-real-hash-but-non-empty", CreatedAt: past, UpdatedAt: past}); err != nil {
		t.Fatal(err)
	}
	st.Plan().Close()
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
}

// noProcessDBPath removes DB_PATH from the process env for the test (and
// restores it after).
func noProcessDBPath(t *testing.T) {
	t.Helper()
	t.Setenv("DB_PATH", "placeholder")
	os.Unsetenv("DB_PATH")
}

func writeDotEnvFile(t *testing.T, inst, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(inst, ".env"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// divergence is one installation plus one place a diverted DB_PATH lands.
type divergence struct {
	name       string
	dotenv     string // "" = no .env file
	procVal    string // the operator shell's DB_PATH (set, possibly empty)
	instRel    string // the installation's own DB, relative to inst
	instSource string // the installation's source as the refusal names it (inst-relative %s)
	divertAbs  bool   // procVal names another installation (filled at run)
	divertRel  string // else: the diverted DB relative to inst
}

var divergences = []divergence{
	{name: ".env DB_PATH vs shell default", dotenv: dotEnvVarDB, procVal: "data/data.db",
		instRel: "var/db/data.db", instSource: `DB_PATH="var/db/data.db" in %s/.env`, divertRel: "data/data.db"},
	{name: ".env DB_PATH vs shell DB_PATH set EMPTY (resolves the default)", dotenv: dotEnvVarDB, procVal: "",
		instRel: "var/db/data.db", instSource: `DB_PATH="var/db/data.db" in %s/.env`, divertRel: "data/data.db"},
	{name: ".env DB_PATH vs shell absolute elsewhere", dotenv: dotEnvVarDB, divertAbs: true,
		instRel: "var/db/data.db", instSource: `DB_PATH="var/db/data.db" in %s/.env`},
	{name: ".env without DB_PATH vs shell elsewhere (the bot uses the default)", dotenv: "OTHER=1\n", divertAbs: true,
		instRel: "data/data.db", instSource: "the default data/data.db (%s/.env sets no DB_PATH)"},
	{name: "no .env vs shell elsewhere (the bot uses the default)", dotenv: "", divertAbs: true,
		instRel: "data/data.db", instSource: "the default data/data.db (no %s/.env)"},
	{name: "no .env vs shell var/db", dotenv: "", procVal: "var/db/data.db",
		instRel: "data/data.db", instSource: "the default data/data.db (no %s/.env)", divertRel: "var/db/data.db"},
}

// layout builds the installation for d with BOTH candidate DBs seeded (and,
// when enrolled, both enrolled through Run with no process DB_PATH), and
// returns the install dir and the diverted DB file.
func (d divergence) layout(t *testing.T, enrolled bool) (inst, divertedDB string) {
	t.Helper()
	noProcessDBPath(t)
	inst = t.TempDir()
	if d.divertAbs {
		other := install(t) // another installation: <other>/data/data.db
		if enrolled {
			if rc, _, errb := run(other, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
				t.Fatalf("enroll the other installation: rc=%d %s", rc, errb)
			}
		}
		divertedDB = filepath.Join(other, "data", "data.db")
	} else {
		divertedDB = filepath.Join(inst, d.divertRel)
		seedBotDB(t, divertedDB)
		if enrolled { // enroll the diverted dir as if it were the installation's
			if d.divertRel != "data/data.db" {
				writeDotEnvFile(t, inst, "DB_PATH="+d.divertRel+"\n")
			}
			if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
				t.Fatalf("enroll the diverted dir: rc=%d %s", rc, errb)
			}
			_ = os.Remove(filepath.Join(inst, ".env"))
		}
	}
	seedBotDB(t, filepath.Join(inst, d.instRel))
	if d.dotenv != "" {
		writeDotEnvFile(t, inst, d.dotenv)
	}
	if enrolled {
		if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
			t.Fatalf("enroll the installation: rc=%d %s", rc, errb)
		}
	}
	if got := DBFileFor(inst); got != filepath.Join(inst, d.instRel) {
		t.Fatalf("precondition: with no process DB_PATH the installation resolves %s, want %s", got, filepath.Join(inst, d.instRel))
	}
	return inst, divertedDB
}

func (d divergence) procValue(divertedDB string) string {
	if d.divertAbs {
		return divertedDB
	}
	return d.procVal
}

// assertRefusal: rc 2 before any prompt; the message names both values with
// their sources and both resolved files.
func assertRefusal(t *testing.T, d divergence, inst, divertedDB string, rc int, out, errb string) {
	t.Helper()
	if rc != 2 {
		t.Fatalf("rc=%d (want the precondition refusal, rc 2) stdout=%q stderr=%s", rc, out, errb)
	}
	for _, want := range []string{
		"refusing",
		fmt.Sprintf("DB_PATH=%q in this process's environment", d.procValue(divertedDB)),
		divertedDB,
		fmt.Sprintf(d.instSource, inst),
		filepath.Join(inst, d.instRel),
	} {
		if !strings.Contains(errb, want) {
			t.Errorf("the refusal does not name %q:\n%s", want, errb)
		}
	}
	if strings.Contains(errb, "Type exactly") || strings.Contains(out, "hmac") || strings.Contains(out, "enrolled:") {
		t.Fatalf("refused only after prompting or writing: stdout=%q stderr=%s", out, errb)
	}
}

// enroll: refused, and NOTHING is written anywhere — neither the
// installation's data dir nor the diverted one gains an updater dir.
// Positive control: the same installation with the process DB_PATH unset
// enrolls into the installation's own data dir.
func TestProcessDBPathThatDiffersFromTheInstallationRefusesEnroll(t *testing.T) {
	for _, d := range divergences {
		t.Run(d.name, func(t *testing.T) {
			attended(t)
			inst, divertedDB := d.layout(t, false)
			os.Setenv("DB_PATH", d.procValue(divertedDB))
			for _, args := range [][]string{{"enroll", bEmail}, {"enroll", "--replace", bEmail}} {
				rc, out, errb := run(inst, enrollLine(bEmail), args...)
				assertRefusal(t, d, inst, divertedDB, rc, out, errb)
			}
			for _, dir := range []string{filepath.Dir(divertedDB), filepath.Join(inst, filepath.Dir(d.instRel))} {
				if _, err := os.Lstat(updateauth.Dir(dir)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("a refused enroll created %s", updateauth.Dir(dir))
				}
			}
			os.Unsetenv("DB_PATH") // positive control
			if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
				t.Fatalf("positive control rc=%d %s", rc, errb)
			}
			if _, err := updateauth.LoadAdmin(filepath.Join(inst, filepath.Dir(d.instRel))); err != nil {
				t.Fatalf("positive control: not enrolled in the installation's dir: %v", err)
			}
		})
	}
}

// authorize: refused (no grant) even though the diverted dir holds a valid
// enrollment that would mint one; the enrollment files are untouched.
// Positive control: unset, the grant verifies under the INSTALLATION's key
// and not under the diverted one.
func TestProcessDBPathThatDiffersFromTheInstallationRefusesAuthorize(t *testing.T) {
	for _, d := range divergences {
		t.Run(d.name, func(t *testing.T) {
			attended(t)
			inst, divertedDB := d.layout(t, true)
			instData, divData := filepath.Join(inst, filepath.Dir(d.instRel)), filepath.Dir(divertedDB)
			var before []string
			for _, dd := range []string{instData, divData} {
				before = append(before, fileSig(t, updateauth.AdminPath(dd)), fileSig(t, updateauth.DeviceKeyPath(dd)))
			}
			os.Setenv("DB_PATH", d.procValue(divertedDB))
			rc, out, errb := run(inst, authorizeLine(bRel), "authorize", bRel)
			assertRefusal(t, d, inst, divertedDB, rc, out, errb)
			var after []string
			for _, dd := range []string{instData, divData} {
				after = append(after, fileSig(t, updateauth.AdminPath(dd)), fileSig(t, updateauth.DeviceKeyPath(dd)))
			}
			if strings.Join(before, ";") != strings.Join(after, ";") {
				t.Fatal("a refused authorize changed an enrollment file")
			}
			os.Unsetenv("DB_PATH") // positive control
			rc, out, errb = run(inst, authorizeLine(bRel), "authorize", bRel)
			if rc != 0 {
				t.Fatalf("positive control rc=%d %s", rc, errb)
			}
			g, err := updateauth.ParseInstallRequest(strings.NewReader(out))
			if err != nil {
				t.Fatal(err)
			}
			instKey, _ := updateauth.LoadDeviceKey(instData)
			divKey, _ := updateauth.LoadDeviceKey(divData)
			if !updateauth.VerifyMAC(instKey, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC) || updateauth.VerifyMAC(divKey, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC) {
				t.Fatal("positive control: the grant is not the installation's")
			}
		})
	}
}

// A process DB_PATH that names the installation's OWN file (any spelling)
// proceeds — the refusal is about divergence, not about the variable being
// set — and the confirmation says the shell's value names the same file.
func TestProcessDBPathNamingTheInstallationsFileProceeds(t *testing.T) {
	cases := []struct{ name, dotenv, instRel, procVal string }{
		{"same relative spelling", dotEnvVarDB, "var/db/data.db", "var/db/data.db"},
		{"absolute spelling", dotEnvVarDB, "var/db/data.db", "<abs>"},
		{"unclean spelling", dotEnvVarDB, "var/db/data.db", "./var/db/../db/data.db"},
		{"no .env, shell names the default", "", "data/data.db", "data/data.db"},
		{"no .env, shell sets DB_PATH empty (the default)", "", "data/data.db", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			attended(t)
			noProcessDBPath(t)
			inst := t.TempDir()
			seedBotDB(t, filepath.Join(inst, c.instRel))
			if c.dotenv != "" {
				writeDotEnvFile(t, inst, c.dotenv)
			}
			v := c.procVal
			if v == "<abs>" {
				v = filepath.Join(inst, c.instRel)
			}
			os.Setenv("DB_PATH", v)
			rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail)
			if rc != 0 {
				t.Fatalf("rc=%d %s", rc, errb)
			}
			if _, err := updateauth.LoadAdmin(filepath.Join(inst, filepath.Dir(c.instRel))); err != nil {
				t.Fatalf("not enrolled in the installation's dir: %v", err)
			}
			if !strings.Contains(errb, fmt.Sprintf("this process's DB_PATH=%q names the same file", v)) {
				t.Fatalf("the confirmation does not say the shell's DB_PATH names the same file:\n%s", errb)
			}
			if rc, out, errb := run(inst, authorizeLine(bRel), "authorize", bRel); rc != 0 || !strings.Contains(out, "hmac") {
				t.Fatalf("authorize rc=%d %s", rc, errb)
			}
		})
	}
}

// lineWith returns the one stderr line that contains label ("" if none).
func lineWith(s, label string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, label) {
			return l
		}
	}
	return ""
}

// The typed confirmation shows — BEFORE "Type exactly" — the resolved bot
// database, the data dir and where DB_PATH came from, for both subcommands.
func TestConfirmationShowsTheResolvedDatabaseDataDirAndSource(t *testing.T) {
	cases := []struct{ name, dotenv, instRel, source string }{
		{".env DB_PATH", dotEnvVarDB, "var/db/data.db", `DB_PATH="var/db/data.db" in %s/.env`},
		{"no .env", "", "data/data.db", "the default data/data.db (no %s/.env)"},
		{".env without DB_PATH", "OTHER=1\n", "data/data.db", "the default data/data.db (%s/.env sets no DB_PATH)"},
		{".env DB_PATH empty", "DB_PATH=\n", "data/data.db", "the default data/data.db (%s/.env sets DB_PATH empty)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			attended(t)
			noProcessDBPath(t)
			inst := t.TempDir()
			seedBotDB(t, filepath.Join(inst, c.instRel))
			if c.dotenv != "" {
				writeDotEnvFile(t, inst, c.dotenv)
			}
			dbFile := filepath.Join(inst, c.instRel)
			for _, sc := range []struct {
				stdin string
				args  []string
			}{
				{enrollLine(bEmail), []string{"enroll", bEmail}},
				{authorizeLine(bRel), []string{"authorize", bRel}},
			} {
				rc, _, errb := run(inst, sc.stdin, sc.args...)
				if rc != 0 {
					t.Fatalf("%v rc=%d %s", sc.args, rc, errb)
				}
				prompt := strings.Index(errb, "Type exactly")
				for label, want := range map[string]string{
					"installation:": inst,
					"bot database:": dbFile,
					"data dir:":     filepath.Dir(dbFile),
					"DB_PATH from:": fmt.Sprintf(c.source, inst),
				} {
					l := lineWith(errb, label)
					if l == "" || !strings.HasSuffix(strings.TrimSpace(l), want) {
						t.Errorf("%v: line %q = %q, want it to end %q\n%s", sc.args, label, l, want, errb)
					}
					if i := strings.Index(errb, label); i < 0 || prompt < 0 || i > prompt {
						t.Errorf("%v: %q is not shown before the typed confirmation\n%s", sc.args, label, errb)
					}
				}
			}
		})
	}
}

// An <install>/.env the CLI cannot read or parse is refused (fail-closed,
// decided in the fold): the CLI cannot see the DB_PATH the bot uses. The
// refusal never echoes the file (godotenv's parse error quotes it).
// Positive control: the same installation with the .env repaired proceeds.
func TestUnreadableDotEnvIsRefused(t *testing.T) {
	type bad struct {
		name, body string
		mode       os.FileMode
		want       string
	}
	bads := []bad{{"malformed", dotEnvVarDB + "MIIEowIBAAKCAQEA+notreallyakey/AAAA\n", 0o600, "could not be read or parsed"}}
	if os.Geteuid() != 0 {
		bads = append(bads, bad{"permission denied", dotEnvVarDB, 0o000, "permission denied"})
	}
	for _, b := range bads {
		t.Run(b.name, func(t *testing.T) {
			attended(t)
			noProcessDBPath(t)
			inst := t.TempDir()
			seedBotDB(t, filepath.Join(inst, "var", "db", "data.db"))
			seedBotDB(t, filepath.Join(inst, "data", "data.db")) // where an ignored .env would divert to
			p := filepath.Join(inst, ".env")
			writeDotEnvFile(t, inst, b.body)
			_ = os.Chmod(p, b.mode)
			for _, sc := range []struct {
				stdin string
				args  []string
			}{
				{enrollLine(bEmail), []string{"enroll", bEmail}},
				{authorizeLine(bRel), []string{"authorize", bRel}},
			} {
				rc, out, errb := run(inst, sc.stdin, sc.args...)
				if rc != 2 || !strings.Contains(errb, p) || !strings.Contains(errb, b.want) {
					t.Fatalf("%v: rc=%d stderr=%s, want rc 2 naming %s (%s)", sc.args, rc, errb, p, b.want)
				}
				if strings.Contains(errb, "notreallyakey") || strings.Contains(errb, "Type exactly") || strings.Contains(out, "hmac") {
					t.Fatalf("%v: echoed the file, prompted or minted:\n%s", sc.args, errb)
				}
			}
			for _, dd := range []string{filepath.Join(inst, "data"), filepath.Join(inst, "var", "db")} {
				if _, err := os.Lstat(updateauth.Dir(dd)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("a refused run created %s", updateauth.Dir(dd))
				}
			}
			_ = os.Chmod(p, 0o600) // positive control: repaired
			writeDotEnvFile(t, inst, dotEnvVarDB)
			if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
				t.Fatalf("positive control rc=%d %s", rc, errb)
			}
			if _, err := updateauth.LoadAdmin(filepath.Join(inst, "var", "db")); err != nil {
				t.Fatalf("positive control: not enrolled where the .env points: %v", err)
			}
		})
	}
}

// With no process DB_PATH the database and data dir the CLI shows and acts
// on are the ONE resolver's (DataDirFor/DBFileFor, which
// TestDataDirIsTheMaintenanceResolver pins to cmd/maintenance-hold's) for
// every layout — the fold changes nothing there. Read from the real entry.
func TestShownTargetIsTheOneResolverWithoutAProcessDBPath(t *testing.T) {
	for _, env := range []string{"", "OTHER=1\n", "DB_PATH=var/db/data.db\n", "DB_PATH=/ABS/x.db\n", "DB_PATH=\n"} {
		attended(t)
		noProcessDBPath(t)
		inst := t.TempDir()
		elsewhere := filepath.Join(t.TempDir(), "x.db")
		env = strings.ReplaceAll(env, "/ABS/x.db", elsewhere)
		if env != "" {
			writeDotEnvFile(t, inst, env)
		}
		seedBotDB(t, DBFileFor(inst))
		rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail)
		if rc != 0 {
			t.Fatalf(".env %q: rc=%d %s", env, rc, errb)
		}
		dbLine, dirLine := lineWith(errb, "bot database:"), lineWith(errb, "data dir:")
		if dbLine == "" || dirLine == "" {
			t.Fatalf(".env %q: the confirmation shows no bot database / data dir line:\n%s", env, errb)
		}
		db := strings.TrimSpace(strings.SplitN(dbLine, "bot database:", 2)[1])
		dir := strings.TrimSpace(strings.SplitN(dirLine, "data dir:", 2)[1])
		if db != DBFileFor(inst) || dir != DataDirFor(inst) || dir != holdcli.DataDirFor(inst) {
			t.Fatalf(".env %q: shown db=%s dir=%s, resolver db=%s dir=%s hold=%s", env, db, dir, DBFileFor(inst), DataDirFor(inst), holdcli.DataDirFor(inst))
		}
		if _, err := updateauth.LoadAdmin(dir); err != nil {
			t.Fatalf(".env %q: not enrolled in the shown dir: %v", env, err)
		}
	}
}
