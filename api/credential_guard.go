package api

import (
	"net/http"

	"nofx/auth"
	"nofx/logger"
	"nofx/store"

	"github.com/gin-gonic/gin"
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
// credentialActorRefusal is the ONE predicate both handlers run first: the
// token's email must equal, byte for byte, the email on the stored row its
// user_id names (the predicate login uses to mint that token in the first
// place). Any refusal is 403 with one body; the cause is logged as a category
// only — never the token.

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
	return u, ""
}

func credentialForbid(c *gin.Context, why string) {
	logger.Warnf("🔒 [credentials] refused %s %s from %s: %s", c.Request.Method, c.FullPath(), c.ClientIP(), why)
	c.AbortWithStatusJSON(http.StatusForbidden, credentialForbiddenBody)
}
