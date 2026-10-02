package api

import (
	"github.com/gin-gonic/gin"
)

// requireOwner (N-1, verify/0926-system) — true only for a NON-machine JWT
// whose user is the first-created account, the single-user system's owner
// (registration closes after the first user). Fail closed: nil claims, a
// machine token, an empty users table, or a store error all refuse.
func (s *Server) requireOwner(c *gin.Context) bool {
	cl := authClaimsFrom(c)
	if cl == nil || cl.UserID == "" || cl.IsMachine() {
		return false
	}
	users, err := s.store.User().GetAll()
	if err != nil || len(users) == 0 {
		return false
	}
	return users[0].ID == cl.UserID
}

// apiTradingMode is a seam kept for tests: the build is futures-only (C2), so
// it always reports "futures".
var apiTradingMode = func() string {
	return "futures"
}

// tradingModeIsFutures reports whether this build is running the futures
