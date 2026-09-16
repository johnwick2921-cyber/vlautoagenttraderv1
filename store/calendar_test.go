package store

import (
	"strings"
	"testing"
)

func TestCalendarSliceReplayFrozen(t *testing.T) {
	c := newLevelStateStore(t).Calendar()

	wrote, err := c.SaveSliceIfAbsent(&CalendarSliceDB{
		TradeDate: "2026-08-13", Source: "forexfactory",
		EventsJSON: `[{"title":"FOMC Statement","impact":"T1"}]`,
	})
	if err != nil || !wrote {
		t.Fatalf("first slice save: wrote=%v err=%v", wrote, err)
	}

	// A later refetch on the SAME trade date must NOT overwrite (replay-frozen).
	wrote2, err := c.SaveSliceIfAbsent(&CalendarSliceDB{
		TradeDate: "2026-08-13", Source: "static", EventsJSON: `[]`,
	})
	if err != nil || wrote2 {
		t.Fatalf("re-save must be frozen no-op: wrote=%v err=%v", wrote2, err)
	}

	got, err := c.GetSlice("2026-08-13")
	if err != nil || got == nil {
		t.Fatalf("get slice: %v %v", got, err)
	}
	if got.Source != "forexfactory" || !strings.Contains(got.EventsJSON, "FOMC") {
		t.Fatalf("frozen slice was altered: %+v", got)
	}

	miss, err := c.GetSlice("2026-08-14")
	if err != nil || miss != nil {
		t.Fatalf("absent date should return nil: %v %v", miss, err)
	}
}
