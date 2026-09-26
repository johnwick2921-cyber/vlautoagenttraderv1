package telegram

import (
	"nofx/auth"
	"nofx/store"
)

// botTokenStale reports whether the API would refuse the bot's current JWT,
// so the bot must mint a new one before its next call:
//   - the token no longer validates (empty, expired, signed under another
//     secret) or names another user;
//   - M3 red-team H2 (CTO ruling 1790231205208): authMiddleware refuses, on
//     EVERY protected route, a token issued at or before the account's last
//     credential change (auth.RetiredBy — the same predicate, the same
//     whole-second rule). The owner's password change therefore retires the
//     bot's token too; the bot — a process on the box reading the account row
//     itself — re-mints rather than going dark until a restart;
//   - PR #200 review F4b (CTO bridge msg 1790252194343): the token is on the
//     logout blacklist (auth.IsTokenBlacklisted — the check authMiddleware
//     makes first, on every route). The bot's agent calls the API with this
//     token and apicall has no allowlist, so one prompt-injected "log out"
//     blacklists it; the bot re-mints instead of going dark until a restart
//     or the token's expiry.
func botTokenStale(token string, u store.User) bool {
	if token == "" {
		return true
	}
	if auth.IsTokenBlacklisted(token) {
		return true
	}
	cl, err := auth.ValidateJWT(token)
	if err != nil || cl == nil || cl.UserID != u.ID {
		return true
	}
	return auth.RetiredBy(cl.IssuedAt, u.CreatedAt, u.UpdatedAt)
}
