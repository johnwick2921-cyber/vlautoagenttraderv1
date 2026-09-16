package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// THE PIN THAT WOULD HAVE CAUGHT IT. On the 042ff360 boot the seam boot line
// shipped and reported "excluded=3" while StampSeamRowsExcluded was never
// called — rows 573 and 574 still read D and F in production. A python edit
// whose anchor did not match failed silently, and I reported it wired without
// opening the file.
//
// A boot line without its migration is a claim without a mechanism.
func TestSeamMigrationIsActuallyWiredIntoMain(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if !strings.Contains(src, "st.StampSeamRowsExcluded()") {
		t.Error("main.go does not call StampSeamRowsExcluded — the boot line would report a count with nothing behind it")
	}
	// And it must run BEFORE the line that reports on it, or the first boot
	// reports the pre-migration state.
	call := strings.Index(src, "st.StampSeamRowsExcluded()")
	line := strings.Index(src, "SeamExclusionBootLine")
	if call < 0 || line < 0 || call > line {
		t.Errorf("the migration must run before the boot line reports it (call at %d, line at %d)", call, line)
	}
}

// The boot line must distinguish MATCHED from STAMPED, and shout when a matched
// row still carries a grade.
func TestSeamBootLineNamesTheDefectWhenUnstamped(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "sw.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Now().UnixMilli()
	mk := func(id int64, grade string) {
		p := &TraderPosition{TraderID: "h", ExchangeID: "nt8", ExchangePositionID: "s" + grade,
			Symbol: "MNQ", Side: "SHORT", Quantity: 1, EntryQuantity: 1, EntryPrice: 29000,
			EntryTime: now, Leverage: 1, Status: "OPEN", Source: CloseReasonTestSeam,
			CreatedAt: now, UpdatedAt: now}
		if err := st.Position().CreateOpenPosition(p); err != nil {
			t.Fatal(err)
		}
		if err := st.GormDB().Exec(`UPDATE trader_positions SET id=?, status='CLOSED', adherence_grade=? WHERE id=?`, id, grade, p.ID).Error; err != nil {
			t.Fatal(err)
		}
	}
	mk(573, "D")
	mk(574, "F")

	// BEFORE the migration: the line must say the rows are still graded.
	line := st.SeamExclusionBootLine()
	if !strings.Contains(line, "matched=2") {
		t.Errorf("the line must report what it MATCHED: %s", line)
	}
	if !strings.Contains(line, "STILL GRADED") {
		t.Errorf("a matched row holding a real grade is a DEFECT and the line must say so: %s", line)
	}
	t.Logf("before: %s", line)

	// AFTER: stamped, and the warning is gone.
	st.StampSeamRowsExcluded()
	line = st.SeamExclusionBootLine()
	if strings.Contains(line, "STILL GRADED") {
		t.Errorf("after the migration nothing should remain graded: %s", line)
	}
	if !strings.Contains(line, "stamped=2") {
		t.Errorf("the line must report what it STAMPED: %s", line)
	}
	var g string
	st.GormDB().Raw(`SELECT adherence_grade FROM trader_positions WHERE id=574`).Scan(&g)
	if g != SeamExcludedNote {
		t.Errorf("574 must carry the reason in the row, got %q", g)
	}
	t.Logf("after:  %s", line)
}
