package researchsnapshot

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"
)

var active atomic.Pointer[Recorder]

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

func Start(path string, log func(string)) (closeRecorder func()) {
	closeRecorder = func() {}
	emit := func(message string) {
		defer func() { _ = recover() }()
		if log != nil {
			log(message)
		}
	}
	defer func() {
		if recover() != nil {
			if log != nil {
				emit("WARN research snapshot initialization panic; capture unavailable")
			}
		}
	}()
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
		emit("WARN research snapshot archive unavailable; capture disabled")
		return
	}
	r := NewRecorder(a, 128, log)
	Install(r)
	return func() { Install(nil); r.Close(); _ = a.Close() }
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
	r := Active()
	if r == nil {
		return BootLineAt(nil, nil, now)
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
