package api

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"nofx/auth"
	"nofx/config"
	"nofx/internal/updateauth"
	"nofx/logger"
	"nofx/trader"

	"github.com/gin-gonic/gin"
)

// ── W-ONE-BUTTON M3 — update authorization ────────────────────────────────
//
// Five routes under /api/updates, mounted with their OWN gate on the bare
// /api group — deliberately NOT the protected group (whose refusals are 401s
// with distinguishing texts) and deliberately NOT through s.route /
// s.routeWithSchema, which append to routeRegistry and so to GetAPIDocs — the
// route list the Telegram LLM agent is given (CTO ruling F1). The agent holds
// a JWT whose user_id IS the owner's; it must never learn these routes exist.
//
// Every gate failure is the SAME response: 403 {"error":"forbidden"}. The
// cause is logged server-side as a category only — never the token, the MAC
// or the key.
//
// With nothing enrolled (data/updater/admin.json + device.key absent — the
// OFF state) every route refuses and nothing else in the app changes.
//
// Install = the gate (identity factor: JWT of the enrolled admin) AND an
// HMAC-SHA256 over release_id|job_id|expires_at under device.key (possession
// factor). Nothing on the API side can mint a MAC (CTO ruling Q1(a)): the
// owner runs the attended `updater-bootstrap authorize <release_id>` on the
// box and pastes its {job_id, expires_at, hmac}.

// UpdateHeader is the custom header every /api/updates request must carry
// with the exact value "1". It is NOT in the CORS Access-Control-Allow-Headers
// list, so a cross-origin page can never get a browser to send it (the
// preflight fails) — pinned by TestUpdatePreflightNeverAllowsTheUpdateHeader.
const UpdateHeader = "X-NOFX-Update"

// maxUpdateInstallBody caps the install body (a Grant is ~200 bytes).
const maxUpdateInstallBody = 4096

var errForbiddenBody = gin.H{"error": "forbidden"}

// UpdateStarter hands a fully authorized, verified install to the updater
// worker (M4). In M3 no starter exists: a verified manifest (impossible with
// StubVerifier) answers 503 rather than pretending to start anything.
type UpdateStarter func(g updateauth.Grant, m updateauth.Manifest) error

// SetUpdateVerifier replaces the manifest verifier (M4 wires the real one).
// nil restores the refusing stub.
func (s *Server) SetUpdateVerifier(v updateauth.Verifier) {
	if v == nil {
		v = updateauth.StubVerifier{}
	}
	s.updateVerifier = v
}

// SetUpdateStarter installs the M4 worker hand-off. nil = none (503).
func (s *Server) SetUpdateStarter(f UpdateStarter) { s.updateStart = f }

func (s *Server) updatesClock() time.Time {
	if s.updatesNow != nil {
		return s.updatesNow()
	}
	return time.Now()
}

// registerUpdateRoutes mounts the five update routes on the bare /api group
// with raw g.GET/g.POST — never s.route (F1: not advertised to the agent).
func (s *Server) registerUpdateRoutes(api *gin.RouterGroup) {
	upd := api.Group("/updates", s.updatesGate())
	upd.GET("", s.handleUpdatesStatus)
	upd.POST("/check", s.handleUpdatesCheck)
	upd.POST("/install", s.handleUpdatesInstall)
	upd.GET("/jobs/:id", s.handleUpdatesJob)
	upd.GET("/jobs/:id/receipt", s.handleUpdatesJob)
}

func updatesForbid(c *gin.Context, why string) {
	logger.Warnf("🔒 [updates] refused %s %.96q: %s", c.Request.Method, c.Request.URL.Path, why)
	c.AbortWithStatusJSON(http.StatusForbidden, errForbiddenBody)
}

// updatesGate is the whole identity gate. It returns the refusal category
// ("" = admitted). Order: transport checks (no I/O), then the JWT, then the
// enrollment files, then the users store.
func (s *Server) updatesGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if why := s.updatesRefusal(c); why != "" {
			updatesForbid(c, why)
			return
		}
		c.Next()
	}
}

var loopbackHostNames = map[string]bool{"127.0.0.1": true, "localhost": true, "::1": true}

// requestHostName is the hostname part of r.Host, lower-cased, brackets
// stripped; "" when the port part is present but not all digits.
func requestHostName(hostport string) string {
	h := hostport
	if host, port, err := net.SplitHostPort(hostport); err == nil {
		for i := 0; i < len(port); i++ {
			if port[i] < '0' || port[i] > '9' {
				return ""
			}
		}
		h = host
	}
	return strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(h, "["), "]"))
}

func (s *Server) updatesRefusal(c *gin.Context) string {
	r := c.Request
	dataDir := trader.MaintenanceDataDir()
	if dataDir == "" {
		return "data dir unconfigured"
	}

	// F4: loopback judged on the socket peer (RemoteAddr), never ClientIP()
	// — gin trusts X-Forwarded-For by default in this repo.
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "peer unparseable"
	}
	if ip := net.ParseIP(peer); ip == nil || !ip.IsLoopback() {
		return "peer not loopback"
	}
	// F4: Host restricted to loopback names (DNS rebinding: evil.test
	// resolving to 127.0.0.1 still arrives with Host: evil.test).
	if !loopbackHostNames[requestHostName(r.Host)] {
		return "host not a loopback name"
	}
	// CSRF: the custom header, exactly one value, exactly "1".
	if v := r.Header.Values(UpdateHeader); len(v) != 1 || v[0] != "1" {
		return "update header missing or wrong"
	}
	// Origin absent or same-origin (the server speaks plain http).
	if o := r.Header.Values("Origin"); len(o) > 1 || (len(o) == 1 && o[0] != "http://"+r.Host) {
		return "cross-origin"
	}
	if sfs := r.Header.Values("Sec-Fetch-Site"); len(sfs) > 1 || (len(sfs) == 1 && sfs[0] != "same-origin" && sfs[0] != "none") {
		return "cross-site fetch"
	}

	// F2 + red-team M1: nothing signed with a secret this public repo
	// publishes (the loader default, the .env.example placeholder, the CI
	// literal) or with one shorter than 32 bytes is an identity.
	if why := config.JWTSecretUnfitForUpdates(auth.JWTSecret); why != "" {
		return why
	}
	ah := r.Header.Values("Authorization")
	if len(ah) != 1 {
		return "authorization missing"
	}
	parts := strings.Split(ah[0], " ")
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return "authorization malformed"
	}
	if auth.IsTokenBlacklisted(parts[1]) {
		return "token revoked"
	}
	claims, err := auth.ValidateJWT(parts[1])
	if err != nil || claims == nil {
		return "token invalid"
	}

	// The enrollment. Absent = the OFF state.
	admin, err := updateauth.LoadAdmin(dataDir)
	if err != nil {
		if errors.Is(err, updateauth.ErrNotEnrolled) {
			return "not enrolled"
		}
		return "enrollment unreadable"
	}
	if key, err := updateauth.LoadDeviceKey(dataDir); err != nil {
		return "device key unreadable"
	} else {
		clear(key)
	}
	// F1: BOTH user_id and email (the Telegram bot token carries the owner's
	// user_id with email bot@internal).
	if claims.UserID != admin.UserID || claims.Email != admin.Email {
		return "not the enrolled admin"
	}
	// F3: the admin user row must still exist with the enrolled email (a
	// reset-account wipe leaves pre-reset JWTs valid for up to 24h).
	if s.store == nil {
		return "no user store"
	}
	u, err := s.store.User().GetByID(admin.UserID)
	if err != nil || u == nil || u.Email != admin.Email {
		return "admin user row absent or changed"
	}
	// Q8: a token issued before the user row last changed (password change)
	// is stale for updates. Whole seconds: the JWT iat is second-precision.
	if claims.IssuedAt == nil || u.UpdatedAt.IsZero() || claims.IssuedAt.Time.Unix() < u.UpdatedAt.Unix() {
		return "token older than the user row"
	}
	return ""
}

// handleUpdatesStatus — GET /api/updates. Only reached by the enrolled admin,
// so enrolled is true; it never returns any field or derivative of
// admin.json or device.key.
func (s *Server) handleUpdatesStatus(c *gin.Context) {
	_, stub := s.updateVerifier.(updateauth.StubVerifier)
	verifier := "configured"
	if stub {
		verifier = "stub"
	}
	c.JSON(http.StatusOK, gin.H{
		"enrolled":          true,
		"manifest_verifier": verifier,
		"install_enabled":   !stub && s.updateStart != nil,
	})
}

// handleUpdatesCheck — POST /api/updates/check. M3 has no release source and
// makes no network call; it says so rather than inventing a result.
func (s *Server) handleUpdatesCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"checked": false, "reason": "no release source in this build"})
}

// handleUpdatesInstall — POST /api/updates/install
// {release_id, job_id, expires_at, hmac}. Order (each refusal final):
// strict parse 400 → expiry window 403 → HMAC 403 → consume job id (replay
// 409) → verified manifest (M3 stub: 422 "release not verified").
func (s *Server) handleUpdatesInstall(c *gin.Context) {
	dataDir := trader.MaintenanceDataDir()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUpdateInstallBody)
	g, err := updateauth.ParseInstallRequest(c.Request.Body)
	if err != nil {
		logger.Warnf("🔒 [updates] install: bad request")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}
	now := s.updatesClock()
	if err := updateauth.CheckExpiry(g.ExpiresAt, now); err != nil {
		updatesForbid(c, "install: outside the validity window")
		return
	}
	key, err := updateauth.LoadDeviceKey(dataDir)
	if err != nil {
		updatesForbid(c, "install: device key unreadable")
		return
	}
	ok := updateauth.VerifyMAC(key, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC)
	clear(key)
	if !ok {
		updatesForbid(c, "install: MAC mismatch")
		return
	}
	// The job id is spent from here on, whatever follows.
	if err := updateauth.Consume(dataDir, g.JobID, g.ExpiresAt, now); err != nil {
		if errors.Is(err, updateauth.ErrReplay) {
			if errors.Is(err, updateauth.ErrPrunedReplay) {
				// M3-RT-F1: expires_at at or below the store's pruned-through
				// watermark — a consumed-and-pruned id, or a fresh grant minted
				// under a clock stepped back past a prune. Indistinguishable,
				// so it is a replay (fail closed).
				logger.Warnf("🔒 [updates] install: job id at or below the pruned-through watermark — refused as a replay (clock stepped back?)")
			} else {
				logger.Warnf("🔒 [updates] install: job id replay")
			}
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "job already used"})
			return
		}
		logger.Errorf("🔒 [updates] install: job-id store refused: %v", err)
		c.AbortWithStatusJSON(http.StatusForbidden, errForbiddenBody)
		return
	}
	m, err := s.updateVerifier.VerifiedManifest(g.ReleaseID)
	if err != nil || m.ReleaseID != g.ReleaseID {
		logger.Warnf("🔒 [updates] install: release %q not verified", g.ReleaseID)
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": "release not verified"})
		return
	}
	if s.updateStart == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "installer unavailable"})
		return
	}
	if err := s.updateStart(g, m); err != nil {
		logger.Errorf("🔒 [updates] install: hand-off refused: %v", err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "installer unavailable"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"job_id": g.JobID})
}

// handleUpdatesJob — GET /api/updates/jobs/:id and /jobs/:id/receipt. M3 has
// no jobs (no worker): every id is unknown ⇒ 404. It never touches the
// filesystem, so no id — however shaped — can reach a path.
func (s *Server) handleUpdatesJob(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}
