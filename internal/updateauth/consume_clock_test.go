package updateauth

// W-ONE-BUTTON M3 red-team fold (red-3 #3), deterministic: the clock reading
// Consume judges and records (consumed_at) is taken while .seen.lock is
// held. The test clock probes the lock with a non-blocking flock from its
// own descriptor each time it is read, and returns a distinct value per
// read; consumed_at must be the value of a read made under the lock.

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestConsumeTakesItsVerdictClockReadingUnderTheLock(t *testing.T) {
	d := enrolled(t)
	type read struct {
		unix   int64
		locked bool
	}
	var reads []read
	base := tNow.Unix()
	clock := func() time.Time {
		f, err := os.OpenFile(seenLockPath(d), os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		locked := false
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			if !errors.Is(err, syscall.EWOULDBLOCK) {
				t.Fatal(err)
			}
			locked = true
		} else {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		}
		r := read{unix: base + int64(len(reads))*7, locked: locked}
		reads = append(reads, r)
		return time.Unix(r.unix, 0)
	}
	if err := Consume(d, "0123456789abcdef", base+300, clock); err != nil {
		t.Fatal(err)
	}
	var underLock []int64
	for _, r := range reads {
		if r.locked {
			underLock = append(underLock, r.unix)
		}
	}
	if len(underLock) == 0 {
		t.Fatalf("Consume never read the clock while holding .seen.lock (reads %+v)", reads)
	}
	b, _ := os.ReadFile(SeenPath(d))
	want := `"consumed_at":` + strconv.FormatInt(underLock[len(underLock)-1], 10)
	if !strings.Contains(string(b), want) {
		t.Fatalf("consumed_at is not the under-lock reading (%s): %s (reads %+v)", want, b, reads)
	}
	// an expiry that only happens under the lock is refused there
	reads = nil
	late := func() time.Time {
		tm := clock()
		if reads[len(reads)-1].locked {
			return tm.Add(time.Hour)
		}
		return tm
	}
	if err := Consume(d, "0123456789abcdee", base+300, late); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired only under the lock: %v, want ErrExpired", err)
	}
}
