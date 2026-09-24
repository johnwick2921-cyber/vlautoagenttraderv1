package updateauth

import (
	"strings"
	"testing"
)

// ── M3 census-repair verify #3 (N1/N2): directives the COMPILER reads ──────
//
// Rule 5 read comments for ONE compiler directive (//go:linkname). A cgo
// preamble is a comment too, and it reads files at COMPILE time: GNU as's
// .incbin, C23 #embed and #cgo LDFLAGS all pull a file into the binary with
// no Go string literal, no updateauth import, no crypto/hmac and no
// LoadDeviceKey. The verifier's N2 below built, vetted and minted a MAC the
// production VerifyMAC accepted with every census green. Each probe here is
// the verifier's file WORD FOR WORD, planted in a synthetic module (t.TempDir
// — never the real tree, and no key file exists anywhere: the census is
// syntactic) and judged by the PRODUCTION census function updateAuthOffenders,
// the one TestUpdateAuthCensus runs over the real tree.

// N2 — verifier probe zz_vcc3_incbin.go, verbatim.
const probeN2Incbin = `package main

// VERIFIER PROBE N2 (census repair; scratch copy only — never a worktree).
// A cgo preamble is a COMMENT: the census never reads it for anything. GNU
// as's .incbin pulls the key file into the binary at compile time (the C
// compiler runs in the package directory), with no Go string literal, no
// updateauth import, no crypto/hmac and no LoadDeviceKey. The module has no
// cgo today.

/*
__asm__(".section .rodata\n.global vcc3key\n.balign 8\nvcc3key:\n.incbin \"data/upd" "ater/dev" "ice.k" "ey\"\n.previous\n");
extern const unsigned char vcc3key[32];
*/
import "C"

import (
	"encoding/hex"
	"strconv"
	"unsafe"

	"github.com/golang-jwt/jwt/v5"
)

func vcc3Mint(releaseID, jobID string, expiresAt int64) (string, error) {
	k := C.GoBytes(unsafe.Pointer(&C.vcc3key[0]), 32)
	sig, err := jwt.SigningMethodHS256.Sign("nofx-update-install/v1|"+releaseID+"|"+jobID+"|"+strconv.FormatInt(expiresAt, 10), k)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sig), nil
}
`

// PIN (verify #3 N2): `import "C"` is refused in ANY non-test file of the
// module (none exists today). No cgo at all is the fail-closed rule: .incbin
// takes ../ and absolute paths, so the preamble reaches the data dir from any
// package, and the preamble's C is not something this census can read.
func TestUpdateAuthCensusRefusesCgo(t *testing.T) {
	root := mintBase(t)
	const rel = "zz_vcc3_incbin.go"
	mintWrite(t, root, rel, probeN2Incbin)
	requirePrefixes(t, mintOffenders(t, root), rel+`: imports "C"`)

	// the same import in a non-root package, under an alias, and in a file a
	// build tag excludes (the walk ignores build tags: it can only over-report)
	for rel, body := range map[string]string{
		"kernel/zz_cgo.go":        "package kernel\n\n// #include <stdlib.h>\nimport \"C\"\n",
		"api/zz_cgo_alias.go":     "package api\n\nimport (\n\t\"fmt\"\n\tc \"C\"\n)\n\nvar _ = fmt.Sprint\nvar _ = c.int(0)\n",
		"trader/zz_cgo_tagged.go": "//go:build never\n\npackage trader\n\nimport \"C\"\n",
	} {
		t.Run(rel, func(t *testing.T) {
			root := mintBase(t)
			mintWrite(t, root, rel, body)
			requirePrefixes(t, mintOffenders(t, root), rel+`: imports "C"`)
		})
	}

	// controls: prose naming the import in a comment or a string, and cgo in a
	// _test.go file (not walked: a test file is never linked into the app
	// binary — and the toolchain refuses cgo in tests anyway), stay clean
	root = mintBase(t)
	mintWrite(t, root, "kernel/doc.go", "// Package kernel never writes import \"C\"; cgo would read files at compile time.\npackage kernel\n\nconst note = `import \"C\"`\n")
	mintWrite(t, root, "kernel/zz_cgo_test.go", "package kernel\n\n// #include <stdlib.h>\nimport \"C\"\n")
	if off := mintOffenders(t, root); len(off) != 0 {
		t.Fatalf("prose and a test file must stay clean:\n%s", strings.Join(off, "\n"))
	}
}
