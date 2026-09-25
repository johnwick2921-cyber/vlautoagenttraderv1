package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// The ONE retire rule (M3 red-team H2, CTO ruling 1790231205208), as a table:
// iat must be STRICTLY after the credential epoch truncated to the second.
func TestRetiredByIsTheWholeSecondRuleAgainstTheCredentialEpoch(t *testing.T) {
	created := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	changed := time.Date(2026, 9, 24, 3, 0, 5, 900_000_000, time.UTC) // …:05.9
	at := func(t time.Time) *jwt.NumericDate { return jwt.NewNumericDate(t) }
	for _, c := range []struct {
		name               string
		iat                *jwt.NumericDate
		createdAt, updated time.Time
		want               bool
	}{
		{"nil iat is always retired", nil, created, created, true},
		{"nil iat on a changed row", nil, created, changed, true},
		{"never-changed row (updated == created) retires nothing", at(created.Add(-time.Hour)), created, created, false},
		{"legacy zero updated_at retires nothing", at(created), created, time.Time{}, false},
		{"iat a second before the change", at(changed.Add(-time.Second)), created, changed, true},
		{"iat earlier in the change's own second", at(time.Date(2026, 9, 24, 3, 0, 5, 100_000_000, time.UTC)), created, changed, true},
		{"iat later in the change's own second", at(time.Date(2026, 9, 24, 3, 0, 5, 950_000_000, time.UTC)), created, changed, true},
		{"iat the next whole second", at(time.Date(2026, 9, 24, 3, 0, 6, 0, time.UTC)), created, changed, false},
		{"iat well after", at(changed.Add(time.Hour)), created, changed, false},
	} {
		if got := RetiredBy(c.iat, c.createdAt, c.updated); got != c.want {
			t.Errorf("%s: RetiredBy = %v, want %v", c.name, got, c.want)
		}
	}
	// IssuedNotAfter is the Q8 comparison itself.
	if !IssuedNotAfter(at(changed.Truncate(time.Second)), changed) || IssuedNotAfter(at(changed.Truncate(time.Second).Add(time.Second)), changed) || !IssuedNotAfter(nil, changed) {
		t.Fatal("IssuedNotAfter is not the strict whole-second rule")
	}
}
