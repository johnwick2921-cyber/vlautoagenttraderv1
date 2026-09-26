package agent

import (
	"context"
	"strings"
	"testing"

	"nofx/mcp"
)

// ── W1b E9 repair (verifier defect 6) — no proposal the door must refuse ───
//
// execute_trade's schema calls stop_loss and take_profit REQUIRED for an
// open, but nothing checked them when the trade was PROPOSED: the tool minted
// a pending open with no bracket and asked the owner to confirm an order the
// admission chain is certain to refuse. Driven through handleToolCall, the
// LLM's production entry to the tool.
func TestExecuteTradeProposalNeedsItsOwnBracket(t *testing.T) {
	a := &Agent{config: &Config{AllowTradeExecution: true}} // no trader manager: a proposal that passes the bracket check stops at "no trader manager"
	ctx := WithSessionPolicy(context.Background(), SessionPolicy{Authenticated: true, CanExecuteTrade: true})
	call := func(args string) string {
		return a.handleToolCall(ctx, "u1", 1, "en", mcp.ToolCall{Function: mcp.ToolCallFunction{Name: "execute_trade", Arguments: args}})
	}
	for _, args := range []string{
		`{"action":"open_long","symbol":"MNQ","quantity":1}`,
		`{"action":"open_short","symbol":"MNQ","quantity":1,"stop_loss":29100}`,
		`{"action":"open_long","symbol":"MNQ","quantity":1,"take_profit":29100}`,
		`{"action":"open_long","symbol":"MNQ","quantity":1,"stop_loss":-1,"take_profit":29100}`,
	} {
		if got := call(args); !strings.Contains(got, "stop_loss and take_profit") {
			t.Fatalf("an open proposed without its own stop AND target must be refused at PROPOSAL, before any pending trade exists: %s → %s", args, got)
		}
	}
	// Both present: the bracket check passes (the next check is the trader).
	if got := call(`{"action":"open_long","symbol":"MNQ","quantity":1,"stop_loss":28950,"take_profit":29100}`); strings.Contains(got, "stop_loss and take_profit") || !strings.Contains(got, "no trader manager") {
		t.Fatalf("a bracketed open must pass the proposal check: %s", got)
	}
	// A close carries no bracket and is never refused for lacking one.
	if got := call(`{"action":"close_long","symbol":"MNQ","quantity":1}`); strings.Contains(got, "stop_loss and take_profit") {
		t.Fatalf("a close must not need a bracket: %s", got)
	}
}
