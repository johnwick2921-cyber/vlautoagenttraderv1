package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SSE auth uses short-lived, single-use tickets instead of putting the
// long-lived JWT in the EventSource URL (?token=) — a JWT in a query string
// leaks into access logs, proxy logs and browser history. The client exchanges
// its Bearer JWT (via the authMiddleware-protected mint endpoint) for an opaque
// ~30s ticket, then opens the stream with ?ticket=. Tickets are consumed on
// first use, so a leaked ticket grants at most one short-lived bar-stream read.

const streamTicketTTL = 30 * time.Second

type streamTicket struct {
	userID  string
	expires time.Time
}

var (
	streamTicketMu sync.Mutex
	streamTickets  = make(map[string]streamTicket)
)

// mintStreamTicket issues a random single-use ticket bound to userID and
// opportunistically evicts expired entries (the map only grows on mint, and
// SSE connections are infrequent, so a sweep-on-mint is sufficient).
func mintStreamTicket(userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	ticket := hex.EncodeToString(b)

	streamTicketMu.Lock()
	defer streamTicketMu.Unlock()
	now := time.Now()
	for k, v := range streamTickets {
		if now.After(v.expires) {
			delete(streamTickets, k)
		}
	}
	streamTickets[ticket] = streamTicket{userID: userID, expires: now.Add(streamTicketTTL)}
	return ticket, nil
}

// consumeStreamTicket validates and removes a ticket (single-use), returning
// the bound userID. Returns false if unknown or expired.
func consumeStreamTicket(ticket string) (string, bool) {
	if ticket == "" {
		return "", false
	}
	streamTicketMu.Lock()
	defer streamTicketMu.Unlock()
	e, ok := streamTickets[ticket]
	if !ok {
		return "", false
	}
	delete(streamTickets, ticket)
	if time.Now().After(e.expires) {
		return "", false
	}
	return e.userID, true
}

// handleBarsStreamTicket mints an SSE stream ticket for the authenticated
// caller. Protected by authMiddleware (reads user_id from context).
func (s *Server) handleBarsStreamTicket(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	ticket, err := mintStreamTicket(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not mint ticket"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ticket": ticket, "expires_in": int(streamTicketTTL.Seconds())})
}
