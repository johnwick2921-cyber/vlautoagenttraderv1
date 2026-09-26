package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nofx/auth"
	"nofx/logger"
	"nofx/store"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ── M3 red-team H1 — who may act on an account's CREDENTIALS ─────────────
//
// PUT /api/user/password and POST /api/reset-account act on the account the
// JWT names. The Telegram bot's JWT (telegram/agent.GenerateBotToken) carries
// the OWNER's user_id with email "bot@internal", so a route that trusts
// user_id alone lets the bot — an LLM steerable by prompt injection or by the
// first chat to /start an unbound bot — set the owner's password, log in as
// the owner and lock the owner out (red1 R1, red2 #1, reproduced at the
// production router and through the agent's own tool).
//
// credentialActorRefusal is the ONE predicate both handlers run first: not a
// machine token; the token's email must equal, byte for byte, the email on
// the stored row its user_id names (the predicate login uses to mint that
// token in the first place); and (H2) the token must not predate the row's
// last credential change. Any refusal is 403 with one body; the cause is
// logged as a category only — never the token.

// machineDeniedRoutes are the route patterns (gin FullPath) a MACHINE token
// (auth.Claims.IsMachine: any scope claim, or the bot's bot@internal email)
// may never reach — denied BY DEFAULT in authMiddleware, before any handler
// runs. An entry also covers every route registered beneath it, so a new
// /api/telegram/... or /api/updates/... route is denied without an edit here.
//   - /api/user/password, /api/reset-account: the credential routes (H1).
//   - /api/telegram: the bot's own config — a machine token must not re-token
//     the bot, change its model, or unbind it (the next /start then binds
//     whoever sends it).
//   - /api/updates: the updater (its own gate refuses machine tokens too —
//     it does not run authMiddleware).
//   - /api/reset-password (CTO ruling 1790231205208 item 2 names it): a
//     PUBLIC route (no authMiddleware) that answers 410 to everyone; a
//     machine token presented there is refused 403 by denyMachineBearer.
//   - /api/logout (PR #200 fold F4a, CTO 1790252194343): a machine token has
//     no user session to end. The handler blacklists the presented token
//     until its exp, so a bot steered into logging itself out (the agent's
//     api_request tool has no path allowlist) was locked out for the token's
//     whole life. Refused here, the handler never runs and nothing is
//     blacklisted (TestMachineTokenCannotLogOutAndIsNotBlacklisted).
var machineDeniedRoutes = []string{
	"/api/user/password",
	"/api/reset-account",
	"/api/reset-password",
	"/api/telegram",
	"/api/updates",
	"/api/logout",
}

// machineDenied reports whether fullPath (a registered route pattern) is one
// a machine token is refused on.
func machineDenied(fullPath string) bool {
	for _, r := range machineDeniedRoutes {
		if fullPath == r || strings.HasPrefix(fullPath, r+"/") {
			return true
		}
	}
	return false
}

// agentOnlyHiddenRoutes are left out of the agent's route list (GetAPIDocs)
// on top of every machine-denied route: session/account management a
// machine token has no business with. They are NOT denied (the web UI's
// login/register are public) — omitting them only stops handing the LLM the
// map (red1 R1: /login was advertised and was step 2 of the chain).
// /api/reset-password and (PR #200 F4a) /api/logout moved to
// machineDeniedRoutes (still hidden — agentHidden covers both lists).
var agentOnlyHiddenRoutes = []string{
	"/api/login",
	"/api/register",
}

// denyMachineBearer guards a PUBLIC machine-denied route (one registered
// outside authMiddleware — today only /api/reset-password): a request whose
// Authorization is a valid machine token (auth.Claims.IsMachine — any scope
// claim, or bot@internal with none) is refused 403 before the handler runs.
// A request with no token, or a token that does not validate, proceeds to
// the handler unchanged (the route is public; it proves nothing about a
// caller that presents no valid machine identity).
func denyMachineBearer(h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ah := c.Request.Header.Values("Authorization"); len(ah) > 0 {
			for _, v := range ah {
				parts := strings.Split(v, " ")
				if len(parts) != 2 || parts[0] != "Bearer" {
					continue
				}
				if cl, err := auth.ValidateJWT(parts[1]); err == nil && cl.IsMachine() {
					credentialForbid(c, "machine token on a machine-denied route")
					return
				}
			}
		}
		h(c)
	}
}

// agentHidden reports whether a registered route path is omitted from the
// Telegram agent's route list: every machine-denied route (the agent would
// only be refused there) plus agentOnlyHiddenRoutes.
func agentHidden(path string) bool {
	if machineDenied(path) {
		return true
	}
	for _, r := range agentOnlyHiddenRoutes {
		if path == r || strings.HasPrefix(path, r+"/") {
			return true
		}
	}
	return false
}

// ctxAuthClaims is the gin context key authMiddleware stores the validated
// claims under (the credential guard reads them; a handler reached WITHOUT
// authMiddleware has none and is refused).
const ctxAuthClaims = "auth_claims"

var credentialForbiddenBody = gin.H{"error": "forbidden"}

// setAuthContext records a validated token's identity on the request exactly
// as authMiddleware does (tests that drive a handler directly use it too).
func setAuthContext(c *gin.Context, claims *auth.Claims) {
	c.Set("user_id", claims.UserID)
	c.Set("email", claims.Email)
	c.Set(ctxAuthClaims, claims)
}

func authClaimsFrom(c *gin.Context) *auth.Claims {
	v, ok := c.Get(ctxAuthClaims)
	if !ok {
		return nil
	}
	cl, _ := v.(*auth.Claims)
	return cl
}

// credentialActorRefusal returns why this request may NOT act on its
// account's credentials ("" = it may) and, when it may, the stored row.
func (s *Server) credentialActorRefusal(c *gin.Context) (*store.User, string) {
	cl := authClaimsFrom(c)
	if cl == nil || cl.UserID == "" {
		return nil, "no authenticated identity"
	}
	// H1 layer 2 (authMiddleware already denies these routes; the handler
	// re-checks so it can never be wired without it).
	if cl.IsMachine() {
		return nil, "machine token"
	}
	if s.store == nil {
		return nil, "no user store"
	}
	u, err := s.store.User().GetByID(cl.UserID)
	if err != nil || u == nil {
		return nil, "no user row for the token"
	}
	// H1: kills the bot@internal token (owner id, different email) and any
	// other token whose email is not the row's.
	if cl.Email == "" || cl.Email != u.Email {
		return nil, "token email is not the account's email"
	}
	// H2 (red1 R2): a token issued at or before the row's last credential
	// change cannot act on the credentials again. authMiddleware already
	// refuses such a token on EVERY protected route (tokenRetirement, below);
	// the guard re-checks with the SAME predicate (auth.RetiredBy — the
	// whole-second Q8 rule; a row that never changed retires nothing) so the
	// handlers can never be wired without it. The /updates gate stays stricter
	// (zero updated_at ⇒ refuse).
	if auth.RetiredBy(cl.IssuedAt, u.CreatedAt, u.UpdatedAt) {
		return nil, retiredWhy(auth.CredentialEpoch(u.CreatedAt, u.UpdatedAt), time.Now())
	}
	return u, ""
}

// retiredWhy is the log category of an H2 (retired-token) refusal.
//
// PR #200 fold F6 (CTO 1790252194343): a credential epoch AHEAD of this
// clock — the clock stepped back after a password change — refuses every NEW
// sign-in too (a token minted now is not strictly after the epoch) until the
// clock passes it. That stays fail-closed; the line says so, by how many
// whole seconds, so an owner who cannot sign in is told why. An epoch in the
// past (or the same second) keeps the plain category.
func retiredWhy(epoch, now time.Time) string {
	const plain = "token predates the account's last credential change"
	if ahead := epoch.Unix() - now.Unix(); !epoch.IsZero() && ahead > 0 {
		return fmt.Sprintf("%s — credential epoch is %ds in the future — clock stepped back; sign-in refused until then", plain, ahead)
	}
	return plain
}

// tokenRetirement is authMiddleware's H2 check (CTO ruling 1790231205208):
// a token is refused on EVERY protected route when it was issued at or before
// its account's credential epoch (auth.RetiredBy — iat STRICTLY after
// users.updated_at truncated to the second, the /updates Q8 rule), when it
// carries no iat, or when its account row no longer exists (reset-account
// wiped it — no epoch can be established; fail closed). It returns the HTTP
// status and a log category ("" = admitted): 401 for a retired/unaccountable
// token (the web UI's 401 path signs the user out and back in), 503 when the
// users store cannot be read (the request is refused, the session is not
// ended by a transient read error).
func (s *Server) tokenRetirement(cl *auth.Claims) (int, string) {
	if cl == nil || cl.IssuedAt == nil {
		return http.StatusUnauthorized, "token carries no iat"
	}
	if s.store == nil {
		return http.StatusServiceUnavailable, "no user store"
	}
	u, err := s.store.User().GetByID(cl.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusUnauthorized, "no account row for the token"
		}
		return http.StatusServiceUnavailable, "user row unreadable"
	}
	if u == nil {
		return http.StatusUnauthorized, "no account row for the token"
	}
	if auth.RetiredBy(cl.IssuedAt, u.CreatedAt, u.UpdatedAt) {
		return http.StatusUnauthorized, retiredWhy(auth.CredentialEpoch(u.CreatedAt, u.UpdatedAt), time.Now())
	}
	return 0, ""
}

func credentialForbid(c *gin.Context, why string) {
	logger.Warnf("🔒 [credentials] refused %s %s from %s: %s", c.Request.Method, c.FullPath(), c.ClientIP(), why)
	c.AbortWithStatusJSON(http.StatusForbidden, credentialForbiddenBody)
}
