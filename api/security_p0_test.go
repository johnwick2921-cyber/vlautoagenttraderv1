// P0 SECURITY REGRESSION TESTS (2026-08-16).
//
// These lock the fix for the unauthenticated account-takeover chain reported in
// docs/superpowers/reports/2026-08-16-acceptance-gate-v2.md finding #1:
//
//	 public POST /api/reset-account  → delete every user
//	 public POST /api/register       → adopt the orphaned credential rows
//	 public POST /api/reset-password → own any account from an email alone
//
// If any of these tests starts failing, that chain is back.

package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() { gin.SetMode(gin.TestMode) }

// newSecurityTestServer builds a Server over a throwaway sqlite store.
func newSecurityTestServer(t *testing.T) *Server {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "sec.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &Server{store: st}
}

// --- S2: reset-password is permanently disabled -----------------------------

// TestResetPasswordIsDisabled: the endpoint must refuse unconditionally — no
// body shape, no email, nothing gets a password changed.
func TestResetPasswordIsDisabled(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.POST("/api/reset-password", s.handleResetPasswordDisabled)

	for _, body := range []string{
		`{"email":"owner@example.com","new_password":"attacker-chosen"}`,
		`{}`,
		``,
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/reset-password", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusGone {
			t.Fatalf("body %q: expected 410 Gone, got %d (%s)", body, rec.Code, rec.Body.String())
		}
	}
}

// TestDestructiveRoutesAreWiredSafely is a wiring lint over setupRoutes.
//
// The runtime tests above prove the HANDLERS refuse. This proves the ROUTER
// still points at those handlers — the part that silently regresses when someone
// restores an old line. routeRegistry is only populated by a live setupRoutes(),
// which needs a full Server, so this reads the registration source directly.
func TestDestructiveRoutesAreWiredSafely(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}
	text := string(src)

	// reset-password must route to the disabled handler, and the old permissive
	// handler must not exist to be wired at all.
	if !strings.Contains(text, "s.handleResetPasswordDisabled") {
		t.Error("/api/reset-password is no longer wired to handleResetPasswordDisabled")
	}
	if strings.Contains(text, "s.handleResetPassword)") {
		t.Error("REGRESSION: the permissive handleResetPassword is wired again")
	}

	// reset-account must be registered on the `protected` (JWT) group only.
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, `"/reset-account"`) {
			continue
		}
		if strings.Contains(line, "protected") {
			continue // correct: JWT group
		}
		if strings.Contains(line, "(api,") || strings.Contains(line, "api.POST") {
			t.Errorf("REGRESSION: /reset-account registered on a PUBLIC group: %s", strings.TrimSpace(line))
		}
	}
	if !strings.Contains(text, `s.routeWithSchema(protected, "POST", "/reset-account"`) {
		t.Error("/reset-account is not registered on the protected group as expected")
	}
}

// --- S2: reset-account needs env + confirm token ----------------------------

// TestResetAccountRequiresEnvFlag: without ALLOW_ACCOUNT_RESET=1 the handler
// refuses even with a valid confirm token and an authenticated context.
func TestResetAccountRequiresEnvFlag(t *testing.T) {
	t.Setenv("ALLOW_ACCOUNT_RESET", "")
	s := newSecurityTestServer(t)

	seedUser(t, s, "owner@example.com")

	rec := doResetAccount(s, `{"confirm":"RESET-ALL-DATA"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without ALLOW_ACCOUNT_RESET, got %d (%s)", rec.Code, rec.Body.String())
	}
	if n, _ := s.store.User().Count(); n != 1 {
		t.Fatalf("users must survive a refused reset, got %d", n)
	}
}

// TestResetAccountRequiresConfirmToken: even with the env flag on, an empty or
// wrong confirm body must not delete anything (the replayed-empty-POST case).
func TestResetAccountRequiresConfirmToken(t *testing.T) {
	t.Setenv("ALLOW_ACCOUNT_RESET", "1")
	s := newSecurityTestServer(t)
	seedUser(t, s, "owner@example.com")

	for _, body := range []string{``, `{}`, `{"confirm":"yes"}`, `{"confirm":"reset-all-data"}`} {
		rec := doResetAccount(s, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %q: expected 400 without the exact confirm token, got %d (%s)", body, rec.Code, rec.Body.String())
		}
		if n, _ := s.store.User().Count(); n != 1 {
			t.Fatalf("body %q: users must survive, got %d", body, n)
		}
	}
}

// TestResetAccountWorksWhenFullyAuthorized: the escape hatch still functions
// when the operator has explicitly opted in — a gate, not a removal.
func TestResetAccountWorksWhenFullyAuthorized(t *testing.T) {
	t.Setenv("ALLOW_ACCOUNT_RESET", "1")
	s := newSecurityTestServer(t)
	seedUser(t, s, "owner@example.com")

	rec := doResetAccount(s, `{"confirm":"RESET-ALL-DATA"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when fully authorized, got %d (%s)", rec.Code, rec.Body.String())
	}
	if n, _ := s.store.User().Count(); n != 0 {
		t.Fatalf("authorized reset should clear users, got %d", n)
	}
}

// --- S3: registration never inherits credentials ----------------------------

// TestRegisterDoesNotAdoptOrphanCredentials is the payoff test: after a reset
// leaves orphaned ai_models/exchanges behind, a fresh registration must see
// ZERO of them. This is the step that used to hand an attacker the owner's
// broker API keys and wallet private keys.
func TestRegisterDoesNotAdoptOrphanCredentials(t *testing.T) {
	s := newSecurityTestServer(t)

	// A previous owner's credential rows, now orphaned (their user is gone).
	// Inserted straight through GORM — the same view of the tables the deleted
	// adoptOrphanRecords had when it re-homed them.
	db := s.store.GormDB()
	ghostUserID := uuid.New().String()
	orphanModel := &store.AIModel{
		ID: uuid.New().String(), UserID: ghostUserID,
		Name: "victim deepseek", Provider: "deepseek", APIKey: "sk-VICTIM-SECRET",
	}
	if err := db.Create(orphanModel).Error; err != nil {
		t.Fatalf("seed orphan model: %v", err)
	}
	orphanExchange := &store.Exchange{
		ID: uuid.New().String(), UserID: ghostUserID,
		Name: "victim binance", ExchangeType: "binance", APIKey: "VICTIM-EXCHANGE-KEY",
	}
	if err := db.Create(orphanExchange).Error; err != nil {
		t.Fatalf("seed orphan exchange: %v", err)
	}

	// The attacker registers into the "uninitialized" system.
	router := gin.New()
	router.POST("/api/register", s.handleRegister)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/register",
		strings.NewReader(`{"email":"attacker@example.com","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("register should succeed on an empty system, got %d (%s)", rec.Code, rec.Body.String())
	}

	newUser, err := s.store.User().GetByEmail("attacker@example.com")
	if err != nil {
		t.Fatalf("registered user not found: %v", err)
	}

	models, _ := s.store.AIModel().List(newUser.ID)
	for _, m := range models {
		if string(m.APIKey) == "sk-VICTIM-SECRET" {
			t.Fatalf("REGRESSION: the new user inherited the previous owner's model API key (model %q)", m.Name)
		}
	}
	exchanges, _ := s.store.Exchange().List(newUser.ID)
	for _, e := range exchanges {
		if string(e.APIKey) == "VICTIM-EXCHANGE-KEY" {
			t.Fatalf("REGRESSION: the new user inherited the previous owner's exchange API key (exchange %q)", e.Name)
		}
	}

	// And the orphans must still belong to the ghost — not silently re-homed.
	stillOrphan, err := s.store.AIModel().GetByID(orphanModel.ID)
	if err != nil {
		t.Fatalf("orphan model disappeared: %v", err)
	}
	if stillOrphan.UserID != ghostUserID {
		t.Fatalf("REGRESSION: orphan model was re-homed from %s to %s", ghostUserID, stillOrphan.UserID)
	}
}

// TestRegisterClosedAfterFirstUser keeps the single-user invariant that makes
// the reset gate meaningful.
func TestRegisterClosedAfterFirstUser(t *testing.T) {
	s := newSecurityTestServer(t)
	seedUser(t, s, "owner@example.com")

	router := gin.New()
	router.POST("/api/register", s.handleRegister)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/register",
		strings.NewReader(`{"email":"second@example.com","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 once a user exists, got %d (%s)", rec.Code, rec.Body.String())
	}
}

// --- helpers ----------------------------------------------------------------

func seedUser(t *testing.T, s *Server, email string) {
	t.Helper()
	if err := s.store.User().Create(&store.User{
		ID: uuid.New().String(), Email: email, PasswordHash: "x",
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// doResetAccount drives the handler with an authenticated context (user_id set,
// as authMiddleware would) so the test exercises the env + token gates rather
// than the JWT layer, which the route registration covers.
func doResetAccount(s *Server, body string) *httptest.ResponseRecorder {
	router := gin.New()
	router.POST("/api/reset-account", func(c *gin.Context) {
		c.Set("user_id", "test-user")
		s.handleResetAccount(c)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/reset-account", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}
