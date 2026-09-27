package api

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"time"

	"nofx/logger"
	"nofx/telemetry"

	"github.com/gin-gonic/gin"
)

// telegramBindCodeTTL is how long an issued bind code stays valid (P2-12).
const telegramBindCodeTTL = 10 * time.Minute

// telegramBindCodeAlphabet avoids visually confusable characters.
const telegramBindCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// telegramBindCodeLen is the code length (8 chars from a 31-char alphabet).
const telegramBindCodeLen = 8

// handleTelegramBindCode (P2-12, audit 0926-system) — issues the one-time code
// a Telegram chat must send back before the first /start may bind. The route
// is protected AND owner-gated (requireOwner): the code is shown in the app,
// so it must be issued by the owner's authenticated session, never by a
// machine token or a second account.
func (s *Server) handleTelegramBindCode(c *gin.Context) {
	if !s.requireOwner(c) {
		telemetry.IncGateBlock("", "telegram_bind_code_owner_gate")
		logger.Warnf("🔒 telegram bind-code refused: non-owner actor user_id=%q", c.GetString("user_id"))
		c.JSON(http.StatusForbidden, gin.H{"error": "telegram bind code is owner-only"})
		return
	}

	code, err := generateTelegramBindCode()
	if err != nil {
		logger.Errorf("telegram bind-code issue failed (rand): user_id=%q err=%v", c.GetString("user_id"), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate bind code"})
		return
	}
	if err := s.store.TelegramConfig().IssueBindCode(code, time.Now().Add(telegramBindCodeTTL)); err != nil {
		logger.Errorf("telegram bind-code issue failed (store): user_id=%q err=%v", c.GetString("user_id"), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store bind code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":               code,
		"expires_in_minutes": int(telegramBindCodeTTL.Minutes()),
	})
}

// generateTelegramBindCode draws telegramBindCodeLen chars from the alphabet
// with crypto/rand.
func generateTelegramBindCode() (string, error) {
	out := make([]byte, 0, telegramBindCodeLen)
	for i := 0; i < telegramBindCodeLen; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(telegramBindCodeAlphabet))))
		if err != nil {
			return "", err
		}
		out = append(out, telegramBindCodeAlphabet[n.Int64()])
	}
	return string(out), nil
}
