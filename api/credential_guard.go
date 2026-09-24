package api

import (
	"net/http"
	"strings"
	"time"

	"nofx/auth"
	"nofx/logger"
	"nofx/store"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
var machineDeniedRoutes = []string{
	"/api/user/password",
	"/api/reset-account",
	"/api/telegram",
	"/api/updates",
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
// login/register are public; logout only revokes the caller's own token;
// reset-password always answers 410) — omitting them only stops handing the
// LLM the map (red1 R1: /login was advertised and was step 2 of the chain).
var agentOnlyHiddenRoutes = []string{
	"/api/login",
	"/api/register",
	"/api/logout",
	"/api/reset-password",
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
	// H2 (red1 R2): a token issued before the row's last credential change
	// (users.updated_at — the retire epoch Q8 applies on /api/updates) cannot
	// act on the credentials again: a thief's older token cannot change the
	// password the owner just rotated, log in, and lock the owner out. A row
	// that never recorded a change (zero updated_at) retires nothing here; the
	// /updates gate stays stricter (zero ⇒ refuse).
	if cl.IssuedAt == nil || (!u.UpdatedAt.IsZero() && issuedBefore(cl.IssuedAt, u.UpdatedAt)) {
		return nil, "token predates the account's last credential change"
	}
	return u, ""
}

// issuedBefore reports whether a token's iat lies before epoch, compared in
// the whole seconds a JWT NumericDate carries (nil iat ⇒ true, fail closed).
// The ONE retire predicate: the credential guard above and the /updates gate's
// Q8 both call it.
func issuedBefore(iat *jwt.NumericDate, epoch time.Time) bool {
	return iat == nil || iat.Time.Unix() < epoch.Unix()
}

func credentialForbid(c *gin.Context, why string) {
	logger.Warnf("🔒 [credentials] refused %s %s from %s: %s", c.Request.Method, c.FullPath(), c.ClientIP(), why)
	c.AbortWithStatusJSON(http.StatusForbidden, credentialForbiddenBody)
}
