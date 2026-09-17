package researchsnapshot

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"
)

var active atomic.Pointer[Recorder]
var startCalled atomic.Bool

// Install is called before any producer starts. A failed archive leaves
// telemetry disabled; it cannot fail application startup.
func Install(r *Recorder) {
	active.Store(r)
}
func Active() *Recorder {
	return active.Load()
}
func Record(name string, build func() []Fact) (accepted bool) {
	r := Active()
	if r == nil {
		return false
	}
	defer func() {
		if recover() != nil {
			r.drop("producer admission panic: " + name)
			accepted = false
		}
	}()
	return r.Offer(name, build)
}

func Start(path string, log func(string), warn func(string)) (closeRecorder func()) {
	startCalled.Store(true)
	closeRecorder = func() {}
	emit := func(message string) {
		defer func() { _ = recover() }()
		if log != nil {
			log(message)
		}
	}
	warnEmit := func(message string) {
		defer func() { _ = recover() }()
		if warn != nil {
			warn(message)
		} else if log != nil {
			log(message)
		}
	}
	defer func() {
		if recover() != nil {
			warnEmit("WARN research snapshot initialization panic; capture unavailable")
		}
	}()
	// RESEARCH_SNAPSHOT gate (review B1): OPT-OUT — the recorder keeps today's
	// behaviour (ON) unless the env is explicitly 0/false.
	if v := os.Getenv("RESEARCH_SNAPSHOT"); v == "0" || strings.EqualFold(v, "false") {
		setStatusNote("OFF (RESEARCH_SNAPSHOT=0)")
		emit("research snapshot: OFF (RESEARCH_SNAPSHOT=0)")
		return
	}
	if os.Getenv("RESEARCH_SNAPSHOT") == "" {
		emit("research snapshot: ON (default)")
	} else {
		emit("research snapshot: ON")
	}
	rev := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				rev = s.Value
			}
		}
	}
	a, err := Open(path, rev)
	if err != nil {
		setStatusNote("OFF (archive unavailable)")
		warnEmit("WARN research snapshot archive unavailable; capture disabled")
		return
	}
	// Retention (review B2): RESEARCH_RETAIN_DAYS UNSET means NO prune, ever.
	// When set, the prune runs OFF the boot path in a goroutine — batched
	// DELETEs with sleeps, one INFO line per batch — never synchronously inside
	// Start() on a ~77 GB file. VACUUM is never run automatically.
	retentionStop := make(chan struct{})
	retentionDays := retainDays()
	if retentionDays > 0 {
		go func() {
			for {
				pruneOldFacts(a, retentionDays, emit)
				select {
				case <-time.After(24 * time.Hour):
				case <-retentionStop:
					return
				}
			}
		}()
	}
	r := NewRecorderWithInfo(a, 128, warnEmit, emit)
	Install(r)
	return func() { close(retentionStop); Install(nil); r.Close(); _ = a.Close() }
}

func BootLineAt(a *Archive, r *Recorder, now time.Time) string {
	schema := "UNKNOWN"
	counts := map[string]string{}
	for _, k := range Objects {
		counts[k] = "UNKNOWN"
	}
	nulls := "UNKNOWN"
	dropped := "UNKNOWN"
	latency := "UNKNOWN"
	if r != nil {
		dropped = fmt.Sprint(r.Dropped())
		if n := r.LatencyP50MS(); n != nil {
			latency = fmt.Sprintf("≤%.3fms (admission)", *n)
		}
	}
	if a != nil {
		loc, err := time.LoadLocation("America/Chicago")
		if err == nil {
			ct := now.In(loc)
			from := time.Date(ct.Year(), ct.Month(), ct.Day(), 0, 0, 0, 0, loc)
			to := from.AddDate(0, 0, 1)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			var version int
			if a.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version) == nil {
				schema = fmt.Sprint(version)
			}
			rows, e := a.db.QueryContext(ctx, "SELECT object,count(*),sum(null_fields) FROM research_facts WHERE captured_ms>=? AND captured_ms<? GROUP BY object", from.UnixMilli(), to.UnixMilli())
			if e == nil {
				for _, k := range Objects {
					counts[k] = "0"
				}
				var total int64
				for rows.Next() {
					var k string
					var n, missing int64
					if rows.Scan(&k, &n, &missing) != nil {
						e = fmt.Errorf("scan failed")
						break
					}
					counts[k] = fmt.Sprint(n)
					total += missing
				}
				if rows.Err() != nil {
					e = rows.Err()
				}
				rows.Close()
				if e == nil {
					nulls = fmt.Sprint(total)
				} else {
					for _, k := range Objects {
						counts[k] = "UNKNOWN"
					}
				}
			}
		}
	}
	return fmt.Sprintf("🗄 research snapshot: schema=%s · objects=%d · rows today market=%s candidate=%s plan=%s scenario=%s exec=%s · null-fields=%s · dropped=%s · added latency p50=%s", schema, len(Objects), counts["market"], counts["candidate"], counts["plan"], counts["scenario"], counts["exec"], nulls, dropped, latency)
}
func CurrentBootLine() string {
	return CurrentBootLineAt(time.Now())
}
func CurrentBootLineAt(now time.Time) string {
	if !startCalled.Load() {
		return "research snapshot: n/a"
	}
	r := Active()
	if r == nil {
		return "research snapshot: " + currentStatusNote()
	}
	a, _ := r.sink.(*Archive)
	return BootLineAt(a, r, now)
}

// Contain is deferred by producer adapters as well as the worker. Even a
// custom error/string conversion fault cannot escape back into a decision.
func Contain(name string) {
	if recover() != nil {
		if r := Active(); r != nil {
			r.drop("producer panic: " + name)
		}
	}
}
