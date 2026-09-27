package api

// W-ONE-BUTTON M3 red-team fold (red-1 #5): Q8 — a token issued before the
// admin row last changed (a password change) is not an update identity —
// must hold inside the second of the change too.
//
// Precision: a JWT iat is whole seconds (golang-jwt/v5 TimePrecision =
// time.Second; a fractional iat is truncated on parse), while users.updated_at
// carries whatever precision its writer stored (sub-second from GORM today —
// red-1 read back …:29.9Z — but not guaranteed for every writer). The only
// rule correct at EVERY stored precision is: iat must be STRICTLY after
// updated_at truncated to the second. Cost: a token issued in the same wall
// second as the change — even just after it — is refused once; the owner logs
// in again a second later.

import (
	"testing"
	"time"
)

func TestUpdatesQ8RefusesATokenIssuedInTheSameSecondAsThePasswordChange(t *testing.T) {
	e := newUpdEnv(t)
	sec := time.Now().Add(-10 * time.Second).Truncate(time.Second)
	setUpdated := func(at time.Time) {
		t.Helper()
		if err := e.st.GormDB().Exec(`UPDATE users SET updated_at = ? WHERE id = ?`, at.UTC(), updAdminID).Error; err != nil {
			t.Fatal(err)
		}
	}
	tokAt := func(iat time.Time) string {
		return mintJWT(t, updAdminID, updAdminEmail, iat, time.Now().Add(time.Hour), updSecret)
	}

	// the row changed at sec.900; a token issued at sec.100 (800 ms BEFORE
	// the change) encodes iat = sec — the red-team's case
	setUpdated(sec.Add(900 * time.Millisecond))
	e.expectAllForbidden("token issued 800 ms before the change, same second", withToken(tokAt(sec.Add(100*time.Millisecond))))
	// a token issued at sec.950 (AFTER the change, same second) is refused
	// too: iat cannot say which side of the change it was minted on
	e.expectAllForbidden("token issued 50 ms after the change, same second", withToken(tokAt(sec.Add(950*time.Millisecond))))
	e.expectAllForbidden("token a whole second older than the change", withToken(tokAt(sec.Add(-time.Second))))
	// positive control: the next whole second is admitted
	e.expectAllAdmitted("token issued the second after the change", withToken(tokAt(sec.Add(time.Second))))

	// a row stored at exactly a whole second: a token of that same second is
	// refused, the next second admitted
	setUpdated(sec)
	e.expectAllForbidden("row at sec.000, token iat = sec", withToken(tokAt(sec)))
	e.expectAllAdmitted("row at sec.000, token iat = sec+1", withToken(tokAt(sec.Add(time.Second))))
}
