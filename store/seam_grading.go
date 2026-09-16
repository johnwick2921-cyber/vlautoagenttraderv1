package store

import (
	"fmt"
	"strings"

	"nofx/logger"
)

// ── SEAM ROWS ARE NEVER GRADED (owner ruling 2026-09-03) ─────────────────────
//
// Row 572 is an ARMED_TEST_SEAM experiment run against the live wire. It sat in
// the graded population holding a D; the 2026-09-03 15:02 boot recomputed it and
// W5 promoted it to an A — a test harness outscoring most real trades and
// counting toward the plan-adherence rate. Checklist 29's family: a test row
// inside a production aggregate, healthy-looking precisely because it is wrong.
//
// Enforced at the WRITE, not only at the caller. A grader that forgets to check
// cannot reintroduce the row, because SetAdherence refuses it.

// SeamExcludedNote is the stamp a seam row carries instead of a grade, so the
// reason is visible in the row rather than only in a log line.
const SeamExcludedNote = "excluded: test seam"

// IsSeamSource reports whether a position's source is a TEST HARNESS rather
// than a real trade. It matches the known seam and, by convention, any future
// one — a new harness should not have to be remembered in five places.
func IsSeamSource(source string) bool {
	s := strings.ToLower(strings.TrimSpace(source))
	if s == "" {
		return false
	}
	if s == CloseReasonTestSeam {
		return true
	}
	return strings.Contains(s, "seam") || strings.Contains(s, "_test") || strings.HasPrefix(s, "test_")
}

// isSeamPosition reads the row's source and answers the same question.
func (s *PositionStore) isSeamPosition(id int64) bool {
	var src string
	if err := s.db.Raw(`SELECT COALESCE(source,'') FROM trader_positions WHERE id = ?`, id).Scan(&src).Error; err != nil {
		return false // fail-open on a read error: never silently drop a REAL grade
	}
	return IsSeamSource(src)
}

// AdherenceDistributionExcludingSeam renders the closed-position grade histogram
// with seam rows removed, and returns HOW MANY it excluded. A distribution that
// cannot say what it dropped is not evidence (A24: no rate without its n).
func (s *Store) AdherenceDistributionExcludingSeam() (string, int, error) {
	type row struct {
		Grade string
		N     int
	}
	var rows []row
	if err := s.gdb.Raw(`SELECT COALESCE(NULLIF(adherence_grade,''),'(none)') AS grade, COUNT(*) AS n
FROM trader_positions WHERE status != 'OPEN' AND NOT (` + seamSQLPredicate + `)
GROUP BY grade ORDER BY grade`).Scan(&rows).Error; err != nil {
		return "", 0, err
	}
	var excluded int
	if err := s.gdb.Raw(`SELECT COUNT(*) FROM trader_positions WHERE status != 'OPEN' AND (` + seamSQLPredicate + `)`).
		Scan(&excluded).Error; err != nil {
		return "", 0, err
	}
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		parts = append(parts, fmt.Sprintf("%s=%d", r.Grade, r.N))
	}
	return strings.Join(parts, " "), excluded, nil
}

// seamSQLPredicate mirrors IsSeamSource for the SQL side. Kept beside it so the
// two cannot drift apart unnoticed — the class-53 lesson: one question, two
// answers is how a predicate ends up disagreeing with itself.
const seamSQLPredicate = `LOWER(COALESCE(source,'')) = '` + CloseReasonTestSeam +
	`' OR LOWER(COALESCE(source,'')) LIKE '%seam%' OR LOWER(COALESCE(source,'')) LIKE '%\_test%' ESCAPE '\' OR LOWER(COALESCE(source,'')) LIKE 'test\_%' ESCAPE '\'`

// SeamExclusionBootLine reports the count READ from the table, never a literal.
func (s *Store) SeamExclusionBootLine() string {
	_, excluded, err := s.AdherenceDistributionExcludingSeam()
	if err != nil {
		return "adherence: seam rows excluded=n/a (count unavailable)"
	}
	// HONEST WORDING (2026-09-03). This used to read "excluded=%d", which reads
	// as "%d rows were excluded" when it counts rows MATCHING the predicate. On
	// the 042ff360 boot it printed 3 while nothing had been excluded, because
	// the migration was not wired. The count and the action are now named
	// separately: a matched row that still holds a grade is a DEFECT, and the
	// line says so rather than implying success.
	stamped, unstamped, err := s.seamStampCounts()
	if err != nil {
		return fmt.Sprintf("adherence: seam rows matched=%d (stamp state unavailable)", excluded)
	}
	warn := ""
	if unstamped > 0 {
		warn = fmt.Sprintf(" · ⚠ %d STILL GRADED — the migration has not run", unstamped)
	}
	return fmt.Sprintf("adherence: seam rows matched=%d · stamped=%d%s (never graded; source matched the seam predicate)",
		excluded, stamped, warn)
}

// StampSeamRowsExcluded clears any grade a seam row still carries and stamps the
// reason. Idempotent; logs the ids it touched.
func (s *Store) StampSeamRowsExcluded() {
	var ids []int64
	if err := s.gdb.Raw(`SELECT id FROM trader_positions
WHERE status != 'OPEN' AND (`+seamSQLPredicate+`) AND COALESCE(adherence_grade,'') NOT IN ('', ?)`, SeamExcludedNote).
		Scan(&ids).Error; err != nil {
		logger.Warnf("🧪 seam exclusion: scan failed: %v", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	if err := s.gdb.Exec(`UPDATE trader_positions SET adherence_grade = ?
WHERE status != 'OPEN' AND (`+seamSQLPredicate+`)`, SeamExcludedNote).Error; err != nil {
		logger.Warnf("🧪 seam exclusion: stamp failed: %v", err)
		return
	}
	logger.Infof("🧪 seam exclusion: %d test-seam row(s) cleared and stamped %q (ids %v) — a harness must never score in the adherence table",
		len(ids), SeamExcludedNote, ids)
}

// seamStampCounts splits matched seam rows into those already excluded and
// those still carrying a real grade — the second number is the defect signal.
func (s *Store) seamStampCounts() (stamped, unstamped int, err error) {
	if e := s.gdb.Raw(`SELECT COUNT(*) FROM trader_positions WHERE status != 'OPEN' AND (`+seamSQLPredicate+`) AND adherence_grade = ?`, SeamExcludedNote).
		Scan(&stamped).Error; e != nil {
		return 0, 0, e
	}
	if e := s.gdb.Raw(`SELECT COUNT(*) FROM trader_positions WHERE status != 'OPEN' AND (`+seamSQLPredicate+`) AND COALESCE(adherence_grade,'') NOT IN ('', ?)`, SeamExcludedNote).
		Scan(&unstamped).Error; e != nil {
		return 0, 0, e
	}
	return stamped, unstamped, nil
}
