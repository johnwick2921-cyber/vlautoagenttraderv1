package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ── M3 red-team H2 — the ONE retire rule (CTO ruling 1790231205208) ─────────
//
// A password change retires every token issued before it, on EVERY protected
// route (api authMiddleware), on the credential routes (api credential guard)
// and on /api/updates (Q8). The comparison is in whole seconds, because a JWT
// NumericDate carries whole seconds (golang-jwt TimePrecision) while
// users.updated_at carries whatever precision its writer stored: the rule
// correct at every precision is iat STRICTLY after the epoch truncated to the
// second (red-team red-1 #5) — a token from the change's own second, either
// side of it, is retired. The cost: a login in that same second SUCCEEDS
// (200), but its new session is refused on first use (401) and has to sign in
// again (the guide's wording, web/src/guide/content/updates.ts).

// IssuedNotAfter reports whether a token with this iat is retired by epoch:
// true unless iat is STRICTLY after epoch truncated to the second. A nil iat
// is retired (fail closed — no epoch can be compared).
func IssuedNotAfter(iat *jwt.NumericDate, epoch time.Time) bool {
	return iat == nil || iat.Time.Unix() <= epoch.Unix()
}

// CredentialEpoch is the account's retire epoch: the instant of its last
// credential change. users.updated_at is moved only by the password change
// (store.UserStore.UpdatePassword) — pinned by the users-table writer census,
// store/users_writer_census_test.go TestUsersTableWriterCensus: every other
// reviewed writer creates the row, deletes it or migrates the schema.
// Creating the row stamps updated_at == created_at, which is NOT a credential
// change — so a row that never changed (or a legacy row with a NULL/zero
// updated_at) has no epoch and retires nothing (the registration token,
// minted in the row's creation second, works at once). The converse is
// documented too (FOLD-M3-B): a row whose updated_at differs from created_at
// for any OTHER reason — a legacy value some older writer or migration left —
// IS an epoch, and retires the tokens issued at or before it although the
// password never changed; one sign-in with the same password clears it
// (api TestLegacyUpdatedAtIsAnEpochWithoutAPasswordChange).
func CredentialEpoch(createdAt, updatedAt time.Time) time.Time {
	if updatedAt.IsZero() || updatedAt.Equal(createdAt) {
		return time.Time{}
	}
	return updatedAt
}

// RetiredBy reports whether a token with this iat is retired by an account
// row carrying createdAt/updatedAt: a nil iat always is; otherwise it is when
// the row has a credential epoch and iat is not strictly after it (whole
// seconds). The ONE predicate authMiddleware, the credential guard and the
// Telegram bot's re-mint decision share.
func RetiredBy(iat *jwt.NumericDate, createdAt, updatedAt time.Time) bool {
	if iat == nil {
		return true
	}
	ep := CredentialEpoch(createdAt, updatedAt)
	return !ep.IsZero() && IssuedNotAfter(iat, ep)
}
