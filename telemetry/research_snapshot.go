package telemetry

import "sync/atomic"

// researchsnapshot volume counters (dispatch 103, 2026-09-16). Ephemeral,
// in-memory process counters — the durable archive keeps every fact; these
// answer "how much narration did the recorder produce" without touching the
// research DB. Read over the counter API next to gate blocks.

var (
	researchSnapshotDrops atomic.Uint64
	researchSnapshotRows  atomic.Uint64
)

// IncResearchSnapshotDrop records one dropped capture.
func IncResearchSnapshotDrop() { researchSnapshotDrops.Add(1) }

// AddResearchSnapshotRows records rows written to the archive.
func AddResearchSnapshotRows(n uint64) { researchSnapshotRows.Add(n) }

// ResearchSnapshotDrops returns the live drop count.
func ResearchSnapshotDrops() uint64 { return researchSnapshotDrops.Load() }

// ResearchSnapshotRows returns the live rows-written count.
func ResearchSnapshotRows() uint64 { return researchSnapshotRows.Load() }
