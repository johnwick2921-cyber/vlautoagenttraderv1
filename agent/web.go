package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
	"vl/branding"
)

type storeUserIDContextKey struct{}
type sessionPolicyContextKey struct{}

type SessionPolicy struct {
	Authenticated           bool
	IsAdmin                 bool
	CanExecuteTrade         bool
	CanViewSensitiveSecrets bool
}

// WithStoreUserID annotates an HTTP request context with the authenticated store user ID.
func WithStoreUserID(ctx context.Context, storeUserID string) context.Context {
	return context.WithValue(ctx, storeUserIDContextKey{}, storeUserID)
}

func storeUserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(storeUserIDContextKey{}).(string); ok && v != "" {
		return v
	}
	return "default"
}

func WithSessionPolicy(ctx context.Context, policy SessionPolicy) context.Context {
	return context.WithValue(ctx, sessionPolicyContextKey{}, policy)
}

func sessionPolicyFromContext(ctx context.Context) SessionPolicy {
	if v, ok := ctx.Value(sessionPolicyContextKey{}).(SessionPolicy); ok {
		return v
	}
	return SessionPolicy{}
}

// WebHandler provides HTTP endpoints for the VL agent.
type WebHandler struct {
	agent  *Agent
	logger *slog.Logger
}

func NewWebHandler(agent *Agent, logger *slog.Logger) *WebHandler {
	return &WebHandler{agent: agent, logger: logger}
}

// HandleHealth handles GET /api/agent/health.
func (w *WebHandler) HandleHealth(rw http.ResponseWriter, r *http.Request) {
	writeJSON(rw, 200, map[string]string{"status": "ok", "agent": branding.PersonaName(), "time": time.Now().Format(time.RFC3339)})
}

// HandleChat handles POST /api/agent/chat.
func (w *WebHandler) HandleChat(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "method not allowed", 405)
		return
	}
	var req struct {
		Message string `json:"message"`
		UserID  int64  `json:"user_id"`
		UserKey string `json:"user_key"`
		Lang    string `json:"lang"`
	}
	// Limit request body to 64KB to prevent abuse
	if err := json.NewDecoder(io.LimitReader(r.Body, 64*1024)).Decode(&req); err != nil {
		writeJSON(rw, 400, map[string]string{"error": "invalid request"})
		return
	}
	if req.Message == "" {
		writeJSON(rw, 400, map[string]string{"error": "message required"})
		return
	}
	// F19 (WAVE 117 PR-D, ports #117 e39d2070) — HTTP conversation identity
	// comes from authenticated middleware, NEVER from a caller-supplied numeric
	// key into another user's persisted state.
	req.UserID = SessionUserIDFromKey(storeUserIDFromContext(r.Context()))
	msg := req.Message
	if req.Lang != "" {
		msg = "[lang:" + req.Lang + "] " + msg
	}

	ctx, cancel := context.WithTimeout(r.Context(), 55*time.Second)
	defer cancel()

	resp, err := w.agent.HandleMessageForStoreUser(ctx, storeUserIDFromContext(r.Context()), req.UserID, msg)
	if err != nil {
		w.logger.Error("agent HandleMessage failed", "error", err, "user_id", req.UserID)
		writeJSON(rw, 500, map[string]string{"error": "I ran into a problem while handling that message. Please try again."})
		return
	}
	writeJSON(rw, 200, map[string]string{"response": resp})
}

// HandleChatStream handles POST /api/agent/chat/stream — SSE streaming chat.
// Sends server-sent events with types including planning, plan, step_start,
// step_complete, replan, tool, delta, done, error.
func (w *WebHandler) HandleChatStream(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "method not allowed", 405)
		return
	}
	var req struct {
		Message string `json:"message"`
		UserID  int64  `json:"user_id"`
		UserKey string `json:"user_key"`
		Lang    string `json:"lang"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64*1024)).Decode(&req); err != nil {
		writeJSON(rw, 400, map[string]string{"error": "invalid request"})
		return
	}
	if req.Message == "" {
		writeJSON(rw, 400, map[string]string{"error": "message required"})
		return
	}
	// F19 — same rule on the SSE path: the identity is the authenticated
	// owner's, never a caller-selected key.
	req.UserID = SessionUserIDFromKey(storeUserIDFromContext(r.Context()))
	msg := req.Message
	if req.Lang != "" {
		msg = "[lang:" + req.Lang + "] " + msg
	}

	// Set SSE headers
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	rw.WriteHeader(200)

	flusher, ok := rw.(http.Flusher)
	if !ok {
		writeSSE(rw, nil, "error", "streaming not supported")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	resp, err := w.agent.HandleMessageStreamForStoreUser(ctx, storeUserIDFromContext(r.Context()), req.UserID, msg, func(event, data string) {
		if ctx.Err() != nil {
			return
		}
		writeSSE(rw, flusher, event, data)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			w.logger.Info("agent stream cancelled", "user_id", req.UserID, "error", err)
			return
		}
		w.logger.Error("agent HandleMessageStream failed", "error", err, "user_id", req.UserID)
		writeSSE(rw, flusher, "error", "I ran into a problem while handling that message. Please try again.")
		return
	}
	if ctx.Err() != nil {
		return
	}
	// Send final done event with complete response
	writeSSE(rw, flusher, "done", resp)
}

// writeSSE writes a single SSE event.
func writeSSE(w http.ResponseWriter, flusher http.Flusher, event, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, sseEscape(data))
	if flusher != nil {
		flusher.Flush()
	}
}

// sseEscape escapes newlines in SSE data (each line needs a "data: " prefix).
func sseEscape(s string) string {
	// SSE spec: multi-line data uses multiple "data:" lines
	// But we use JSON encoding to avoid this complexity
	b, _ := json.Marshal(s)
	return string(b)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	// CORS is handled by the gin middleware — no need to set it here
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
