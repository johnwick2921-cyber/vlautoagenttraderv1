package api

import (
	"testing"
	"time"

	"vl/store"
)

// seedSecondUser creates a non-owner account (created AFTER the admin) and
// returns its JWT. Shared fixture — moved out of the removed
// onboarding_owner_test.go because telegram_bindcode_test.go still needs a
// plain second user.
func seedSecondUser(t *testing.T, e *updEnv) string {
	t.Helper()
	now := time.Now().UTC()
	if err := e.st.User().Create(&store.User{
		ID:           "second-user",
		Email:        "second@example.com",
		PasswordHash: "x",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("seed second user: %v", err)
	}
	return mintJWT(t, "second-user", "second@example.com",
		time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), updSecret)
}
