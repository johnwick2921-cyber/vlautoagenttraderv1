package trader

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"nofx/store"
)

// DEFAULTS-SANE: the boot line is READ, never literal — it renders the RESOLVED
// knob values, so this test derives its expectation from the same resolver the
// live path uses instead of hardcoding the numbers.
func TestPictureHtfBootLineReadsResolvedWindowAndFreshness(t *testing.T) {
	env := newPictureHtfEnv(t, store.PictureHtfConfig{Enabled: true, MinRR: 2.5})
	cfg := store.PictureHtfResolved(nil)
	want := fmt.Sprintf("window=%ds fresh=%ds", cfg.EntryWindowSec, cfg.FreshnessSec)
	line := env.at.pictureHtfBootLineAt(time.Now())
	if !strings.Contains(line, want) {
		t.Fatalf("boot line must READ the resolved window/freshness defaults, want %q in %q", want, line)
	}
}
