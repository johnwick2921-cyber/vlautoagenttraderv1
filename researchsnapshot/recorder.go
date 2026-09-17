package researchsnapshot

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"nofx/telemetry"
)

// OfferBudget bounds synchronous admission work, not disk latency. Builders,
// JSON encoding and SQLite writes execute solely on the archive worker.
const OfferBudget = 5 * time.Millisecond

type Sink interface {
	Save(context.Context, []Fact) error
}
type work struct {
	name    string
	build   func() []Fact
	barrier chan struct{}
}

type Recorder struct {
	sink          Sink
	queue         chan work
	stop          chan struct{}
	done          chan struct{}
	once          sync.Once
	dropped       atomic.Uint64
	warn          func(string)
	dropNotices   chan string
	latency       [1002]atomic.Uint64 // microsecond histogram; final bucket is overflow
	info          func(string)        // INFO line sink (rollups); falls back to warn
	rowsMu        sync.Mutex
	rows          map[string]uint64
	rollupEvery   time.Duration
	lastRollupAt  time.Time
	rollupMu      sync.Mutex
	warnMu        sync.Mutex
	warnAt        time.Time
	warnDelta     uint64
	warnReason    string
	warnSince     time.Time
	dropWindow    time.Duration
	rollupT       *time.Ticker
	rollupStopped atomic.Bool // set when the rollup ticker is stopped (shutdown leak pin)
}

func NewRecorder(sink Sink, capacity int, warn func(string)) *Recorder {
	return NewRecorderWithInfo(sink, capacity, warn, nil)
}

// NewRecorderWithInfo builds the recorder with BOTH sinks attached before the
// worker goroutine starts (no data race on r.info). The INFO sink carries
// rollups and status lines; the WARN sink carries drop notices.
func NewRecorderWithInfo(sink Sink, capacity int, warn, info func(string)) *Recorder {
	if capacity < 1 {
		capacity = 1
	}
	r := &Recorder{sink: sink, queue: make(chan work, capacity), stop: make(chan struct{}), done: make(chan struct{}), warn: warn, dropNotices: make(chan string, 16),
		rollupEvery: envDur("RESEARCH_LOG_EVERY_S", 60*time.Second),
		dropWindow:  200 * time.Millisecond,
		rows:        make(map[string]uint64),
		info:        info}
	r.rollupT = time.NewTicker(r.rollupEvery)
	go r.run()
	return r
}

func (r *Recorder) Offer(name string, build func() []Fact) bool {
	return r.offerWithClock(time.Now, name, build)
}

func (r *Recorder) offerWithClock(clock func() time.Time, name string, build func() []Fact) (accepted bool) {
	start := clock()
	defer func() {
		if p := recover(); p != nil {
			r.drop("admission panic")
			accepted = false
		}
		n := clock().Sub(start).Microseconds()
		if n < 0 {
			n = 0
		}
		if n > 1001 {
			n = 1001
		}
		r.latency[n].Add(1)
	}()
	select {
	case <-r.stop:
		r.drop("recorder closed")
		return false
	default:
	}
	if clock().Sub(start) > OfferBudget {
		r.drop("capture admission budget exceeded")
		return false
	}
	select {
	case r.queue <- work{name: name, build: build}:
		return true
	default:
		r.drop("queue full: " + name)
		return false
	}
}

func (r *Recorder) drop(reason string) {
	r.dropped.Add(1)
	telemetry.IncResearchSnapshotDrop()
	select {
	case r.dropNotices <- reason:
	default:
	}
}

func (r *Recorder) log(message string) {
	defer func() { _ = recover() }()
	if r.info != nil {
		r.info(message)
	} else if r.warn != nil {
		r.warn(message)
	}
}

func (r *Recorder) perform(job work) {
	defer func() {
		if p := recover(); p != nil {
			r.drop(fmt.Sprintf("worker panic (%T): %s", p, job.name))
		}
	}()
	facts := job.build()
	if len(facts) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.sink.Save(ctx, facts); err != nil {
		r.drop(fmt.Sprintf("archive write %s: %v", job.name, err))
		return
	}
	r.noteRows(facts)
}

func (r *Recorder) run() {
	defer close(r.done)
	for {
		select {
		case job := <-r.queue:
			if job.barrier != nil {
				r.flushNotices()
				close(job.barrier)
			} else {
				r.perform(job)
			}
		case reason := <-r.dropNotices:
			r.coalesceDrop(reason)
		case <-r.rollupT.C:
			r.emitDropWarn(true)
			r.emitRollup(false)
		case <-r.stop:
			for {
				select {
				case job := <-r.queue:
					if job.barrier != nil {
						close(job.barrier)
					} else {
						r.drop("shutdown queued: " + job.name)
					}
				default:
					r.rollupT.Stop()
					r.rollupStopped.Store(true)
					r.emitDropWarn(true)
					r.emitRollup(true)
					r.log(fmt.Sprintf("research snapshot stopped: dropped=%d", r.Dropped()))
					return
				}
			}
		}
	}
}

// Flush is an explicit administrative/test wait, never called by a trading
// read or placement. Admission itself always uses select/default.
func (r *Recorder) Flush(ctx context.Context) error {
	barrier := make(chan struct{})
	select {
	case r.queue <- work{barrier: barrier}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-barrier:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Recorder) Dropped() uint64 { return r.dropped.Load() }
func (r *Recorder) Close() {
	r.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = r.Flush(ctx)
		close(r.stop)
	})
	<-r.done
}

func (r *Recorder) LatencyP50MS() *float64 {
	var n uint64
	for i := range r.latency {
		n += r.latency[i].Load()
	}
	if n == 0 {
		return nil
	}
	var seen uint64
	for i := range r.latency {
		seen += r.latency[i].Load()
		if seen >= (n+1)/2 {
			if i == len(r.latency)-1 {
				return nil
			} // overflow is unmeasured, never a plausible millisecond value
			return Value(float64(i+1) / 1000) // upper bound of this microsecond bucket
		}
	}
	return nil
}

func (r *Recorder) flushNotices() {
	for {
		select {
		case reason := <-r.dropNotices:
			r.coalesceDrop(reason)
		default:
			r.emitDropWarn(true)
			r.emitRollup(true)
			return
		}
	}
}
