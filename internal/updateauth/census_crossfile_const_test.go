package updateauth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── DS-105 CENSUS-AUTH [13] (skeptic 4c05158b) ─────────────────────────────
//
// The update-auth census folds constants NAME-BASED, per FILE: a compile-time
// constant assembled from fragments declared in a SIBLING file of the same
// package — or imported from another package (a SelectorExpr never folds) —
// spells updater/device.key with every per-literal rule green. These tests
// plant both shapes in a synthetic module and pin the offenders the census
// must report (RED before the census resolves constant VALUES, GREEN after).

func writeCensusFile(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Each fragment is clean per literal ("upd", "ater", "devi", "ce.k", "ey" —
// none spells "device", "device." or ".key"); only the FOLDED value spells
// /updater/device.key.
const fragFile = `package kernel

const (
	fA = "upd"
	fB = "ater"
	fC = "devi"
	fD = "ce.k"
	fE = "ey"
)
`

const relFile = `package kernel

// KeyRel is a compile-time constant (array length proves it) assembled from
// fragments declared in a SIBLING file — the per-file name env never folds it.
const KeyRel = "/" + fA + fB + "/" + fC + fD + fE

var _ [len(KeyRel)]byte
`

const mintFile = `package kernel

import (
	"crypto/hmac"
	"crypto/sha256"
	"os"
	"path/filepath"
)

// mint reads the enrollment the folded constant spells and HMACs it — rule 4
// must fire on this package (crypto/hmac beside the updater dir).
func mint() []byte {
	data, _ := os.ReadFile(filepath.Dir(os.Getenv("DB_PATH")) + KeyRel)
	mac := hmac.New(sha256.New, []byte("k"))
	mac.Write(data)
	return mac.Sum(nil)
}
`

func TestUpdateAuthCensusFoldsSiblingFileConstants(t *testing.T) {
	root := t.TempDir()
	writeCensusFile(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	writeCensusFile(t, root, "kernel/zz_frag.go", fragFile)
	writeCensusFile(t, root, "kernel/zz_rel.go", relFile)
	writeCensusFile(t, root, "kernel/zz_mint.go", mintFile)
	off, _, err := updateAuthOffenders(t, root)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(off, "\n")
	if !strings.Contains(got, "kernel/zz_rel.go: spells device.key") {
		t.Fatalf("the folded cross-file constant spells device.key invisibly: offenders = %v", off)
	}
	if !strings.Contains(got, `names the path element "updater"`) {
		t.Fatalf("the folded cross-file constant names the updater dir invisibly: offenders = %v", off)
	}
	if !strings.Contains(got, "kernel/zz_mint.go: imports crypto/hmac") {
		t.Fatalf("rule 4 never fired on the package that MACs the spelled path: offenders = %v", off)
	}
}

const apiReaderFile = `package api

import (
	"os"
	"path/filepath"

	"nofx/kernel"
)

// read uses the exported constant of ANOTHER package — a SelectorExpr the
// census never folds — to spell the enrollment.
func read() []byte {
	data, _ := os.ReadFile(filepath.Dir(os.Getenv("DB_PATH")) + kernel.KeyRel)
	return data
}
`

func TestUpdateAuthCensusFoldsExportedConstantsOfAnotherPackage(t *testing.T) {
	root := t.TempDir()
	writeCensusFile(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	writeCensusFile(t, root, "kernel/zz_frag.go", fragFile)
	writeCensusFile(t, root, "kernel/zz_rel.go", relFile)
	writeCensusFile(t, root, "api/zz_read.go", apiReaderFile)
	off, _, err := updateAuthOffenders(t, root)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(off, "\n")
	if !strings.Contains(got, "api/zz_read.go: spells device.key") {
		t.Fatalf("the exported cross-package constant spells device.key invisibly: offenders = %v", off)
	}
}
