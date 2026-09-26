package agent

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// F19 (WAVE 117 PR-D, ports #117 e39d2070) — HTTP conversation identity comes
// from authenticated middleware, never from a caller-supplied numeric user_id:
// a request carrying another owner's key must not clear (or read) that owner's
// persisted conversation state. Both the normal and the SSE handler are pinned.
func TestHTTPChatCannotSelectAnotherOwnersConversation(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "chat", true: "stream"}[stream], func(t *testing.T) {
			a := newTestAgentWithStore(t)
			a.config = &Config{Language: "en"}
			a.logger = slog.Default()
			a.history = newChatHistory(10)
			own := SessionUserIDFromKey("reader")
			foreign := SessionUserIDFromKey("other-owner")
			a.history.Add(own, "user", "own retained fixture")
			a.history.Add(foreign, "user", "foreign retained fixture")
			body, _ := json.Marshal(map[string]any{"message": "/clear", "user_id": foreign})
			req := httptest.NewRequest(http.MethodPost, "/api/agent/chat", strings.NewReader(string(body)))
			ctx := WithStoreUserID(req.Context(), "reader")
			req = req.WithContext(WithSessionPolicy(ctx, SessionPolicy{Authenticated: true}))
			rec := httptest.NewRecorder()
			h := NewWebHandler(a, slog.Default())
			if stream {
				h.HandleChatStream(rec, req)
			} else {
				h.HandleChat(rec, req)
			}
			if rec.Code != 200 {
				t.Fatalf("response %d: %s", rec.Code, rec.Body.String())
			}
			if len(a.history.Get(foreign)) != 1 {
				t.Error("request-selected identity cleared another owner's history")
			}
			if len(a.history.Get(own)) != 0 {
				t.Error("authenticated owner's clear did not address own history")
			}
		})
	}
}
