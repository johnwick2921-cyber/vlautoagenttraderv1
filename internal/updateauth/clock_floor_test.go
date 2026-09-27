package updateauth

// W-ONE-BUTTON M3 red-team fold (red-3 #2): the seen store's clock floor.
// Consume records max(floor, now) and refuses expires_at at or below the
// floor as ErrExpiredAtFloor (an expiry, not a replay); NoteExpired raises
// it, never lowers it, and never writes what its reader refuses.

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"
)

func TestClockFloorIsRecordedByConsumeAndNoteExpiredAndBindsExpiry(t *testing.T) {
	d := enrolled(t)
	floor := func() int64 {
		t.Helper()
		st, err := readSeen(d)
		if err != nil {
			t.Fatal(err)
		}
		return st.ClockFloor
	}
	if f := floor(); f != 0 {
		t.Fatalf("fresh enrollment: clock_floor = %d, want 0", f)
	}
	T := tNow
	if err := Consume(d, "aaaaaaaaaaaaaaaa", T.Unix()+300, clockAt(T)); err != nil {
		t.Fatal(err)
	}
	if f := floor(); f != T.Unix() {
		t.Fatalf("after a consumption at T: clock_floor = %d, want %d", f, T.Unix())
	}
	// a refusal of a genuine expired code at T+400 raises it; an earlier
	// reading never lowers it
	if err := NoteExpired(d, T.Add(400*time.Second)); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(SeenPath(d))
	if err := NoteExpired(d, T.Add(100*time.Second)); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(SeenPath(d)); !bytes.Equal(before, after) {
		t.Fatal("NoteExpired at an earlier clock rewrote the store")
	}
	if f := floor(); f != T.Unix()+400 {
		t.Fatalf("clock_floor = %d, want %d", f, T.Unix()+400)
	}
	// at a clock stepped back to T+200: exp T+400 (at the floor) is expired,
	// not a replay; exp T+401 is admitted
	back := T.Add(200 * time.Second)
	if err := Consume(d, "bbbbbbbbbbbbbbbb", T.Unix()+400, clockAt(back)); !errors.Is(err, ErrExpiredAtFloor) || !errors.Is(err, ErrExpired) || errors.Is(err, ErrReplay) {
		t.Fatalf("exp at the floor: %v, want ErrExpiredAtFloor", err)
	}
	if err := Consume(d, "cccccccccccccccc", T.Unix()+401, clockAt(back)); err != nil {
		t.Fatalf("positive control: exp one above the floor: %v", err)
	}
	if f := floor(); f != T.Unix()+400 {
		t.Fatalf("a consumption at an earlier clock moved the floor to %d", f)
	}
	// never writes what its reader refuses; never touches a corrupt store;
	// once enrolled a missing store is corrupt
	if err := NoteExpired(d, time.Unix(0, 0)); !errors.Is(err, ErrBadClock) {
		t.Fatalf("epoch clock: %v, want ErrBadClock", err)
	}
	if err := os.WriteFile(SeenPath(d), []byte("junk"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := NoteExpired(d, T.Add(time.Hour)); !errors.Is(err, ErrSeenCorrupt) {
		t.Fatalf("corrupt store: %v, want ErrSeenCorrupt", err)
	}
	if b, _ := os.ReadFile(SeenPath(d)); string(b) != "junk" {
		t.Fatal("NoteExpired rewrote a corrupt store")
	}
	_ = os.Remove(SeenPath(d))
	if err := NoteExpired(d, T.Add(time.Hour)); !errors.Is(err, ErrSeenCorrupt) {
		t.Fatalf("missing store after enrollment: %v, want ErrSeenCorrupt", err)
	}
	if _, err := os.Lstat(SeenPath(d)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("NoteExpired re-created a missing store")
	}
}
