package installpath

import (
	"os"
	"path/filepath"
	"testing"
)

// PR #200 fold F3: DBPathOrigin sits BESIDE DotEnvGetenv and must agree with
// it — Effective() is the ONE resolver's answer for this process, and with
// no DB_PATH in the process environment FromInstallation() is too — for every
// .env shape and every process-environment shape. It adds the split (which
// place the value came from); it never changes the answer.
func TestDBPathOriginAgreesWithDotEnvGetenv(t *testing.T) {
	type dotenv struct {
		name, body string // body "" with name "none" = no file
		mode       os.FileMode
		wantState  string
		wantIn     bool
		wantVal    string
	}
	dotenvs := []dotenv{
		{"none", "", 0, DotEnvAbsent, false, ""},
		{"no DB_PATH", "OTHER=1\n", 0o600, DotEnvRead, false, ""},
		{"DB_PATH set", "DB_PATH=from/dotenv.db\n", 0o600, DotEnvRead, true, "from/dotenv.db"},
		{"export DB_PATH", "export DB_PATH=/abs/dotenv.db\n", 0o600, DotEnvRead, true, "/abs/dotenv.db"},
		{"DB_PATH empty", "DB_PATH=\n", 0o600, DotEnvRead, true, ""},
		{"malformed", "DB_PATH=from/dotenv.db\nMIIEowIBAAKCAQEA+notreallyakey/AAAA\n", 0o600, DotEnvUnreadable, false, ""},
	}
	if os.Geteuid() != 0 { // root reads a 0000 file; the denied shape needs a real user
		dotenvs = append(dotenvs, dotenv{"denied", "DB_PATH=from/dotenv.db\n", 0o000, DotEnvDenied, false, ""})
	}
	procs := []struct {
		name string
		set  bool
		val  string
	}{{"unset", false, ""}, {"empty", true, ""}, {"set", true, "from/env.db"}}

	for _, de := range dotenvs {
		for _, pe := range procs {
			t.Run(de.name+"/"+pe.name, func(t *testing.T) {
				dir := t.TempDir()
				if de.name != "none" {
					p := filepath.Join(dir, ".env")
					if err := os.WriteFile(p, []byte(de.body), 0o600); err != nil {
						t.Fatal(err)
					}
					if err := os.Chmod(p, de.mode); err != nil {
						t.Fatal(err)
					}
				}
				t.Setenv("DB_PATH", "placeholder")
				if pe.set {
					os.Setenv("DB_PATH", pe.val)
				} else {
					os.Unsetenv("DB_PATH")
				}
				o := ReadDBPathOrigin(dir)
				if o.DotEnvFile != filepath.Join(dir, ".env") || o.DotEnvState != de.wantState ||
					o.InDotEnv != de.wantIn || o.DotEnvValue != de.wantVal {
					t.Fatalf("origin %+v, want state=%s in=%v val=%q", o, de.wantState, de.wantIn, de.wantVal)
				}
				if o.InProcess != pe.set || o.ProcessValue != pe.val {
					t.Fatalf("process side %+v, want set=%v val=%q", o, pe.set, pe.val)
				}
				one := DBPath(DotEnvGetenv(dir))
				if o.Effective() != one {
					t.Fatalf("Effective()=%q, DBPath(DotEnvGetenv)=%q", o.Effective(), one)
				}
				if !pe.set && o.FromInstallation() != one {
					t.Fatalf("no process DB_PATH: FromInstallation()=%q, DBPath(DotEnvGetenv)=%q", o.FromInstallation(), one)
				}
				// FromInstallation never reads the process environment.
				os.Setenv("DB_PATH", "/somewhere/else.db")
				if again := ReadDBPathOrigin(dir).FromInstallation(); again != o.FromInstallation() {
					t.Fatalf("FromInstallation moved with the process env: %q -> %q", o.FromInstallation(), again)
				}
			})
		}
	}
}
