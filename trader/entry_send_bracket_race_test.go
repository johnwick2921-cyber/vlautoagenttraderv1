package trader

import (
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// ── W1b FOLD-10 — ONE ENTRY-SEND SECTION SPANS SET-BRACKET → OPEN ──────────
//
// A CME broker that keeps the entry's SL/TP in shared per-(symbol, side) maps
// (SetStopLoss / SetTakeProfit, then OpenLong reads them — the legacy NT8
// placeEntry, the CSV transport) was written and read with NO lock spanning
// the set and the send. The AI entry runs on the cycle goroutine; the chat
// door runs on the HTTP goroutine. Interleaving A.set(S1,T1) → C.set(S2,T2) →
// A.send reads (S2,T2): the AI's quantity goes out on the chat's bracket (or
// the reverse). Driven at the two production call sites — the AI decision's
// executeOpenLongWithRecord (what executeDecisionWithRecord calls after
// admission) and the chat door's OpenManualEntryAt (what the agent's
// executeTradeWith calls) — on a broker that records the (sl, tp) its maps
// hold AT OPEN TIME. The interleaves are forced, not hoped for (see
// bracketMapBroker): an unguarded write lands inside the other entry's
// set → read every time; a guarded one costs the fake's hold.

// bracketMapBroker is a CME broker whose entry reads its bracket from shared
// (symbol, side) maps, like the legacy NT8 placeEntry. It does NOT carry its
// own bracket (no OpenWithBracket), so the AutoTrader sets the maps first.
//
// It forces the two interleaves an entry-send section must exclude:
//   - every open YIELDS after its own sets, until two foreign sets land (the
//     other entry's full bracket) or the hold expires — so a set that is not
//     kept out lands between this entry's set and its read;
//   - the FIRST entry's post-open stop set (identified by its price,
//     gateStop, after the first open returned) waits until the SECOND entry's
//     pre-entry stop (releaseStop) has landed, or the hold expires — so a
//     post-open set that is not kept out lands inside the second entry's
//     set → read.
type bracketMapBroker struct {
	stubTrader
	mu          sync.Mutex
	stops       map[string]float64
	targets     map[string]float64
	sets        int
	bump        chan struct{} // closed and replaced on every set (broadcast)
	entered     int
	first       chan struct{} // closed when the first open is entered
	firstDone   bool
	gateStop    float64
	releaseStop float64
	gateUsed    bool
	stopSeen    map[float64]bool
	hold        time.Duration
	sent        map[int64][2]float64
}

func newBracketMapBroker(hold time.Duration, gateStop, releaseStop float64) *bracketMapBroker {
	return &bracketMapBroker{
		stops: map[string]float64{}, targets: map[string]float64{},
		bump: make(chan struct{}), first: make(chan struct{}),
		gateStop: gateStop, releaseStop: releaseStop, stopSeen: map[float64]bool{},
		hold: hold, sent: map[int64][2]float64{},
	}
}

// waitUntil blocks until cond (read under mu) holds or the hold expires.
func (b *bracketMapBroker) waitUntil(cond func() bool) {
	deadline := time.After(b.hold)
	for {
		b.mu.Lock()
		ok, ch := cond(), b.bump
		b.mu.Unlock()
		if ok {
			return
		}
		select {
		case <-ch:
		case <-deadline:
			return
		}
	}
}

func (b *bracketMapBroker) set(stop bool, sym, side string, px float64) {
	b.mu.Lock()
	gated := stop && b.firstDone && !b.gateUsed && px == b.gateStop
	if gated {
		b.gateUsed = true
	}
	b.mu.Unlock()
	if gated {
		b.waitUntil(func() bool { return b.stopSeen[b.releaseStop] })
	}
	b.mu.Lock()
	k := sym + ":" + strings.ToUpper(side)
	if stop {
		b.stops[k] = px
		b.stopSeen[px] = true
	} else {
		b.targets[k] = px
	}
	b.sets++
	close(b.bump)
	b.bump = make(chan struct{})
	b.mu.Unlock()
}

func (b *bracketMapBroker) SetStopLoss(sym, side string, _ float64, px float64) error {
	b.set(true, sym, side, px)
	return nil
}

func (b *bracketMapBroker) SetTakeProfit(sym, side string, _ float64, px float64) error {
	b.set(false, sym, side, px)
	return nil
}

func (b *bracketMapBroker) open(sym, side string) (map[string]interface{}, error) {
	b.mu.Lock()
	b.entered++
	isFirst := b.entered == 1
	n0 := b.sets
	b.mu.Unlock()
	if isFirst {
		close(b.first)
	}
	// Yield between THIS entry's set and its read.
	b.waitUntil(func() bool { return b.sets >= n0+2 })
	b.mu.Lock()
	defer b.mu.Unlock()
	k := sym + ":" + side
	id := int64(len(b.sent) + 1)
	b.sent[id] = [2]float64{b.stops[k], b.targets[k]}
	if isFirst {
		b.firstDone = true
	}
	return map[string]interface{}{"orderId": id}, nil
}

func (b *bracketMapBroker) OpenLong(sym string, _ float64, _ int) (map[string]interface{}, error) {
	return b.open(sym, "LONG")
}

func (b *bracketMapBroker) OpenShort(sym string, _ float64, _ int) (map[string]interface{}, error) {
	return b.open(sym, "SHORT")
}

func (b *bracketMapBroker) sentFor(id int64) ([2]float64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v, ok := b.sent[id]
	return v, ok
}

func TestConcurrentAIAndChatEntriesEachSendTheirOwnBracket(t *testing.T) {
	for _, chatFirst := range []bool{false, true} {
		name := "AI sends first, the chat's set interleaves"
		if chatFirst {
			name = "chat sends first, the AI's set interleaves"
		}
		t.Run(name, func(t *testing.T) {
			w := newChatDoorWire(t, store.RiskControlConfig{})
			chatStop, chatTarget := chatBracket(t, "open_long")
			aiStop, aiTarget := chatStop-2, chatTarget+2 // distinct on both legs
			firstStop, secondStop := aiStop, chatStop
			if chatFirst {
				firstStop, secondStop = chatStop, aiStop
			}
			b := newBracketMapBroker(400*time.Millisecond, firstStop, secondStop)
			w.at.trader = b

			type result struct {
				id  int64
				err error
			}
			aiDone, chatDone := make(chan result, 1), make(chan result, 1)
			runAI := func() {
				d := &kernel.Decision{Action: "open_long", Symbol: "MNQ", Leverage: 1, StopLoss: aiStop, TakeProfit: aiTarget, PositionSizeUSD: 1000, Confidence: 80}
				rec := &store.DecisionAction{Action: "open_long", Symbol: "MNQ", Leverage: 1}
				err := w.at.executeOpenLongWithRecord(d, rec)
				aiDone <- result{rec.OrderID, err}
			}
			runChat := func() {
				order, err := w.at.OpenManualEntryAt("MNQ", "open_long", 1, 1, chatStop, chatTarget, chatDoorMidday)
				id, _ := order["orderId"].(int64)
				chatDone <- result{id, err}
			}
			firstRun, secondRun := runAI, runChat
			if chatFirst {
				firstRun, secondRun = runChat, runAI
			}
			go firstRun()
			select {
			case <-b.first:
			case <-time.After(5 * time.Second):
				t.Fatal("fixture: the first entry never reached the broker's open")
			}
			go secondRun()

			var ai, chat result
			for i := 0; i < 2; i++ {
				select {
				case ai = <-aiDone:
				case chat = <-chatDone:
				case <-time.After(10 * time.Second):
					t.Fatal("an entry never returned")
				}
			}
			if ai.err != nil || chat.err != nil || ai.id == 0 || chat.id == 0 {
				t.Fatalf("fixture: both entries must send: ai=%+v chat=%+v", ai, chat)
			}
			aiSent, _ := b.sentFor(ai.id)
			chatSent, _ := b.sentFor(chat.id)
			if aiSent != [2]float64{aiStop, aiTarget} || chatSent != [2]float64{chatStop, chatTarget} {
				t.Fatalf("each entry must go out on its OWN bracket: AI sent (SL %.2f, TP %.2f) want (%.2f, %.2f); chat sent (SL %.2f, TP %.2f) want (%.2f, %.2f)",
					aiSent[0], aiSent[1], aiStop, aiTarget, chatSent[0], chatSent[1], chatStop, chatTarget)
			}
		})
	}
}
