package trader

import (
	"testing"
	"time"
)

func TestPictureRunEpochLifecycle(t *testing.T) {
	at := &AutoTrader{id: "w5-epoch"}
	if _, ok := at.pictureRunEpoch(); ok {
		t.Fatal("no run → no epoch")
	}
	e1 := at.markPictureRunEpoch(time.Unix(100, 0))
	if got, ok := at.pictureRunEpoch(); !ok || got != e1 {
		t.Fatalf("live run epoch = %d %v, want %d", got, ok, e1)
	}
	at.clearPictureRunEpoch()
	if _, ok := at.pictureRunEpoch(); ok {
		t.Fatal("a Stop ends the epoch")
	}
	if e2 := at.markPictureRunEpoch(time.Unix(200, 0)); e2 == e1 {
		t.Fatal("a new run is a new epoch")
	}
	at.clearPictureRunEpoch()
}
