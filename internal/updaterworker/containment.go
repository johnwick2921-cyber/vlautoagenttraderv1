package updaterworker

// containment.go — W-ONE-BUTTON M4 (3b-B, unit U4G, U4F verify note 5): the
// ONE path-containment check of internal/updaterworker and cmd/nofx-updater.
// Every "is P inside D?" question in their non-test code asks PathWithin and
// nothing else; TestContainmentCensusOneHelper (an AST scan) refuses any other
// filepath.Rel result compared to ".." or given to strings.HasPrefix, and any
// strings.HasPrefix(…, os.TempDir()).
//
// The class (verifier U4F text, "path containment by text"):
//   - compare path ELEMENTS: P is outside D only when Rel(D, P) IS ".." or
//     starts with ".." + the separator — "<D>/..x" is INSIDE;
//   - resolve symlinks on BOTH sides first; for a path not yet created,
//     resolve its deepest existing ancestor and refuse a symlink (dangling, a
//     loop) among the elements that do not resolve — as plain text it would
//     pass while pointing anywhere.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// PathWithin reports whether p is dir or below it, comparing path ELEMENTS
// of the RESOLVED paths (see the file comment). Both must be absolute. An
// error (a relative path, an element that cannot be resolved or checked) is a
// refusal for every caller: containment that cannot be decided is never
// "outside".
func PathWithin(p, dir string) (bool, error) {
	if !filepath.IsAbs(p) || !filepath.IsAbs(dir) {
		return false, fmt.Errorf("containment: %q and %q must both be absolute", p, dir)
	}
	rp, err := resolveExisting(p)
	if err != nil {
		return false, err
	}
	rd, err := resolveExisting(dir)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(rd, rp)
	if err != nil {
		return false, fmt.Errorf("containment: %s against %s: %w", rp, rd, err)
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

// resolveExisting is p with its deepest EXISTING ancestor resolved through
// filepath.EvalSymlinks and the not-yet-created rest appended unchanged (p
// absolute and clean). Every element that does not resolve is Lstat'ed first
// (U4F verify note 4): only one that does not EXIST is text to append — a
// dangling symlink or a loop there would later be followed by the MkdirAll
// that creates the path, so it refuses, as does an element that cannot be
// Lstat'ed at all. With nothing resolvable it is p itself.
func resolveExisting(p string) (string, error) {
	p = filepath.Clean(p)
	var rest []string
	for cur := p; ; cur = filepath.Dir(cur) {
		real, err := filepath.EvalSymlinks(cur)
		if err == nil {
			for i := len(rest) - 1; i >= 0; i-- {
				real = filepath.Join(real, rest[i])
			}
			return real, nil
		}
		fi, lerr := os.Lstat(cur)
		switch {
		case lerr == nil && fi.Mode()&fs.ModeSymlink != 0:
			return "", fmt.Errorf("%s is a symlink that does not resolve (dangling or a loop): %w", cur, err)
		case lerr == nil:
			return "", fmt.Errorf("%s exists but cannot be resolved: %w", cur, err)
		case !errors.Is(lerr, fs.ErrNotExist):
			return "", fmt.Errorf("%s cannot be checked: %w", cur, lerr)
		}
		if filepath.Dir(cur) == cur {
			return p, nil
		}
		rest = append(rest, filepath.Base(cur))
	}
}
