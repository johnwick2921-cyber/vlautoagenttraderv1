package telegram

// PR #200 F7 verify note 3: refresh fails closed when a fresh mint would be
// refused — but when the FIRST account CHANGED (users[0] is a different
// user), failing closed left the bot acting for the PREVIOUS user: its token
// was still admitted, so the next message's agent calls ran as that user
// while the box's first account was someone else. "Nothing installed" must
// mean nothing: a refresh that fails on a user change acts for nobody.

import (
	"testing"
	"time"

	"nofx/auth"
	"nofx/store"
)

func TestBotRefreshFailingClosedOnAUserChangeActsForNobody(t *testing.T) {
	_, st := btBoot(t)
	ident := newBotIdentity(st, 0)
	if !ident.refresh() || ident.userID != btOwnerID {
		t.Fatalf("setup: refresh with the owner on the box = userID %q", ident.userID)
	}
	btRecordWaits(t, false) // the one wait never really sleeps; both mints are refused
	// A new FIRST account (created earlier than the owner) whose credential
	// epoch is an hour ahead of the clock: every token minted for it now is
	// retired at birth (auth.RetiredBy), so refresh must fail closed.
	hash, err := auth.HashPassword("second-account-pass")
	if err != nil {
		t.Fatal(err)
	}
	earlier, ahead := time.Now().Add(-2*time.Hour).UTC(), time.Now().Add(time.Hour).UTC()
	if err := st.User().Create(&store.User{ID: "ffffffff-1111-2222-3333-444444444444", Email: "second@example.test", PasswordHash: hash, CreatedAt: earlier, UpdatedAt: ahead}); err != nil {
		t.Fatal(err)
	}
	if users, err := st.User().GetAll(); err != nil || len(users) != 2 || users[0].ID == btOwnerID {
		t.Fatalf("setup: the new account must be users[0] (GetAll orders by created_at): %v %v", users, err)
	}
	if ident.refresh() {
		t.Fatal("refresh reported success for a first account whose every token the API refuses")
	}
	if ident.userID != "" || ident.email != "" || ident.token != "" || ident.agents != nil {
		t.Fatalf("refresh failed closed on a user change but the bot still acts for %q (token set=%v, manager set=%v) — it must act for nobody",
			ident.userID, ident.token != "", ident.agents != nil)
	}
}
