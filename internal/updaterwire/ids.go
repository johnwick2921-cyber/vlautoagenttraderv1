package updaterwire

import (
	"path/filepath"
	"regexp"
	"strings"
)

// The id allow-lists the worker channel enforces.
//
// internal/updateauth (built in parallel, in another worktree) carries its
// own ValidJobID/ValidReleaseID with byte-identical patterns; the two must
// never drift — an id the app authorizes but the wire refuses (or the
// reverse) splits one decision in two. Until they are folded into one
// source, a parity test at the merged head is owed (see the M3 wire report).
// The MAC message (updateauth.Message: purpose tag|release_id|job_id|
// expires_at) is unambiguous ONLY because
// neither id can contain "|"; a release id reaches a filesystem join in M4,
// so nothing path-shaped may pass. Allow-lists, not deny-lists: a character
// that is not listed is refused.
var (
	jobIDRe     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{7,63}$`)
	releaseIDRe = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._-]{0,63}$`)
)

// ValidJobID reports whether s is a job id: 8–64 chars of [a-z0-9-], not
// starting with '-'.
func ValidJobID(s string) bool {
	return jobIDRe.MatchString(s)
}

// ValidReleaseID reports whether s is a release id: 1–64 chars of
// [0-9A-Za-z._-], starting alphanumeric, never containing "..", and a local
// path element (never absolute, never escaping its parent).
func ValidReleaseID(s string) bool {
	if !releaseIDRe.MatchString(s) {
		return false
	}
	if strings.Contains(s, "..") {
		return false
	}
	return filepath.IsLocal(s) && filepath.Base(s) == s
}
