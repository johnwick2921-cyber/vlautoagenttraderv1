package updaterwire

import (
	"path/filepath"
	"regexp"
	"strings"
)

// The id allow-lists — the ONE source for W-ONE-BUTTON M3.
//
// internal/updateauth did not exist when this package was built, so these are
// defined here and the auth layer imports them (the MAC message
// release_id|job_id|expires_at is unambiguous ONLY because neither id can
// contain "|"; a release id reaches a filesystem join in M4, so nothing
// path-shaped may pass). Allow-lists, not deny-lists: a character that is not
// listed is refused.
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
