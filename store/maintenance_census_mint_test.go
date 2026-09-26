package store

import (
	"sort"
	"strings"
	"testing"

	"nofx/internal/censuswalk"
)

// ── W-ONE-BUTTON M3 fold M4 (red-team 4 #2(b)) — the app never links the minter ──
//
// The import guard forbade only the worker's listener and the worker package.
// nofx/internal/updaterbootstrap — the attended CLI whose Run → authorize →
// updateauth.Authorize → ComputeMAC mints an install MAC — was not forbidden,
// so the trading app could link the one door that mints (CTO ruling Q1(a):
// nothing on the API side mints a MAC) with every census green. Ported from
// red-team 4's TestRedTeamRed4ImportGuardLetsTheAppLinkTheMintingCLI, on a
// synthetic module (t.TempDir), with the toolchain as ground truth.
func TestWorkerImportGuardRefusesTheMintingCLI(t *testing.T) {
	root := t.TempDir()
	censusWrite(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	censusWrite(t, root, "internal/updaterwire/dial.go", "package updaterwire\n")
	censusWrite(t, root, "internal/updaterwire/wireserver/server.go", "package wireserver\n\nimport _ \"nofx/internal/updaterwire\"\n")
	censusWrite(t, root, "internal/updaterworker/hold.go", "package updaterworker\n")
	censusWrite(t, root, "internal/updateauth/mac.go", "package updateauth\n\nfunc ComputeMAC() string { return \"\" }\n")
	censusWrite(t, root, "internal/updaterbootstrap/bootstrap.go", "package updaterbootstrap\n\nimport \"nofx/internal/updateauth\"\n\nfunc Run() string { return updateauth.ComputeMAC() }\n")
	censusWrite(t, root, "cmd/updater-bootstrap/main.go", "package main\n\nimport \"nofx/internal/updaterbootstrap\"\n\nfunc main() { _ = updaterbootstrap.Run() }\n")
	censusWrite(t, root, "api/server.go", "package api\n\nimport _ \"nofx/internal/updaterwire\"\n")
	censusWrite(t, root, "trader/t.go", "package trader\n")
	censusWrite(t, root, "main.go", "package main\n\nimport _ \"nofx/api\"\n\nfunc main() {}\n")
	// positive control: the CLI binary links its own package; the app does not
	if off, _, err := workerImportOffenders(root); err != nil || len(off) != 0 {
		t.Fatalf("clean: the attended CLI's own binary may link it: offenders=%v err=%v", off, err)
	}
	if off, _, err := toolchainWorkerLinkOffenders(root); err != nil || len(off) != 0 {
		t.Fatalf("clean (toolchain): offenders=%v err=%v", off, err)
	}
	for _, via := range []struct{ rel, body string }{
		{"api/mint.go", "package api\n\nimport \"nofx/internal/updaterbootstrap\"\n\nvar _ = updaterbootstrap.Run\n"},
		{"trader/mint.go", "package trader\n\nimport \"nofx/internal/updaterbootstrap\"\n\nvar _ = updaterbootstrap.Run\n"},
		{"main_mint.go", "package main\n\nimport _ \"nofx/internal/updaterbootstrap\"\n"},
	} {
		t.Run(via.rel, func(t *testing.T) {
			root := t.TempDir()
			for rel, body := range map[string]string{
				"go.mod":                                 "module nofx\n\ngo 1.25\n",
				"internal/updateauth/mac.go":             "package updateauth\n\nfunc ComputeMAC() string { return \"\" }\n",
				"internal/updaterbootstrap/bootstrap.go": "package updaterbootstrap\n\nimport \"nofx/internal/updateauth\"\n\nfunc Run() string { return updateauth.ComputeMAC() }\n",
				"api/server.go":                          "package api\n",
				"trader/t.go":                            "package trader\n",
				"main.go":                                "package main\n\nimport _ \"nofx/api\"\nimport _ \"nofx/trader\"\n\nfunc main() {}\n",
				via.rel:                                  via.body,
			} {
				censusWrite(t, root, rel, body)
			}
			linked, err := censuswalk.ListPackages(root, true, ".")
			if err != nil {
				t.Fatal(err)
			}
			truth := false
			for _, p := range linked {
				truth = truth || p.ImportPath == "nofx/internal/updaterbootstrap"
			}
			if !truth {
				t.Fatalf("ground truth: the app binary does not link updaterbootstrap via %s — probe broken", via.rel)
			}
			off, _, err := workerImportOffenders(root)
			if err != nil {
				t.Fatal(err)
			}
			hit := false
			for _, o := range off {
				hit = hit || strings.HasSuffix(o, "nofx/internal/updaterbootstrap")
			}
			if !hit {
				t.Fatalf("the trading app links nofx/internal/updaterbootstrap (the attended MAC minter) via %s and the import guard reports %v", via.rel, off)
			}
			toff, _, err := toolchainWorkerLinkOffenders(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(toff) != 1 || !strings.HasPrefix(toff[0], "nofx/internal/updaterbootstrap: linked by the trading app") {
				t.Fatalf("toolchain guard via %s: %v", via.rel, toff)
			}
		})
	}
}

// ── M4 3b-B U5b (f) — OQ-8/C2: the app never links the activation library ──
//
// nofx/internal/activation (Claude-103's kill/restart library: Activate,
// RollbackTo, Watch — SIGKILL by MainPID identity) is linked by the updater
// worker's binary only. A trading-app package that reached it — directly or
// through any chain — could restart the bot from inside the bot; and at
// 3b-B it also PANICS any binary that links it beside nofx/store (its
// steps.go registers glebarez/go-sqlite as database/sql "sqlite", the same
// name store/sqlitedriver's default backend registers via modernc.org/sqlite:
// "sql: Register called twice for driver sqlite" at init). Proved on a
// synthetic module, direct and transitive, with the toolchain as ground truth.
func TestWorkerImportGuardRefusesTheActivationLibrary(t *testing.T) {
	for _, via := range []struct{ rel, body string }{
		{"api/activate.go", "package api\n\nimport \"nofx/internal/activation\"\n\nvar _ = activation.Activate\n"},
		{"internal/helper/h.go", "package helper\n\nimport _ \"nofx/internal/activation\"\n"},
		{"main_activate.go", "package main\n\nimport _ \"nofx/internal/activation\"\n"},
	} {
		t.Run(via.rel, func(t *testing.T) {
			root := t.TempDir()
			for rel, body := range map[string]string{
				"go.mod":                            "module nofx\n\ngo 1.25\n",
				"internal/activation/activation.go": "package activation\n\nfunc Activate() {}\n",
				"internal/helper/h.go":              "package helper\n",
				"cmd/nofx-updater/main.go":          "package main\n\nimport _ \"nofx/internal/activation\"\n\nfunc main() {}\n",
				"api/server.go":                     "package api\n\nimport _ \"nofx/internal/helper\"\n",
				"trader/t.go":                       "package trader\n",
				"main.go":                           "package main\n\nimport _ \"nofx/api\"\nimport _ \"nofx/trader\"\n\nfunc main() {}\n",
			} {
				censusWrite(t, root, rel, body)
			}
			// positive control: only the worker binary links it — clean
			if off, _, err := workerImportOffenders(root); err != nil || len(off) != 0 {
				t.Fatalf("clean: the worker binary may link the activation library: offenders=%v err=%v", off, err)
			}
			if off, _, err := toolchainWorkerLinkOffenders(root); err != nil || len(off) != 0 {
				t.Fatalf("clean (toolchain): offenders=%v err=%v", off, err)
			}
			censusWrite(t, root, via.rel, via.body)
			linked, err := censuswalk.ListPackages(root, true, ".")
			if err != nil {
				t.Fatal(err)
			}
			truth := false
			for _, p := range linked {
				truth = truth || p.ImportPath == "nofx/internal/activation"
			}
			if !truth {
				t.Fatalf("ground truth: the app binary does not link nofx/internal/activation via %s — probe broken", via.rel)
			}
			off, _, err := workerImportOffenders(root)
			if err != nil {
				t.Fatal(err)
			}
			hit := false
			for _, o := range off {
				hit = hit || strings.HasSuffix(o, "nofx/internal/activation")
			}
			if !hit {
				t.Fatalf("the trading app links nofx/internal/activation via %s and the import guard reports %v", via.rel, off)
			}
			toff, _, err := toolchainWorkerLinkOffenders(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(toff) != 1 || !strings.HasPrefix(toff[0], "nofx/internal/activation: linked by the trading app") {
				t.Fatalf("toolchain guard via %s: %v", via.rel, toff)
			}
		})
	}
}

// The forbidden set is pinned exactly: narrowing it is a reviewed act.
func TestForbiddenWorkerPackagesArePinned(t *testing.T) {
	got := append([]string(nil), forbiddenWorkerPackages...)
	sort.Strings(got)
	want := "nofx/internal/activation,nofx/internal/updaterbootstrap,nofx/internal/updaterwire/wireserver,nofx/internal/updaterworker"
	if strings.Join(got, ",") != want {
		t.Fatalf("forbiddenWorkerPackages = %v, want exactly %s", got, want)
	}
}
