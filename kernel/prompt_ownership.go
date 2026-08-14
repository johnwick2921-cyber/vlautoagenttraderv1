package kernel

import "sort"

// A5 (G5) — PROMPT-OWNERSHIP TAGS. Each account-scoped context field the trader
// populates (account, positions, market data, …) is tagged with its owning
// trader_id in ctx.OwnerTags. Before the prompt is assembled, assertPromptOwnership
// requires EVERY tag to equal the deciding trader (ctx.TraderID). A tag owned by a
// different trader is cross-trader contamination (the "record contamination" bug
// class — one trader's context carrying another's account/positions); the caller
// skips the cycle + 🚨 rather than deciding on someone else's data.
//
// An untagged context (empty OwnerTags) or an empty TraderID → no violation, so
// legacy / test contexts and the goldens stay byte-identical (the tags are never
// rendered into the prompt).

func assertPromptOwnership(ctx *Context) (field, badOwner string, violated bool) {
	if ctx == nil || ctx.TraderID == "" || len(ctx.OwnerTags) == 0 {
		return "", "", false
	}
	// Deterministic scan so the reported violation is stable.
	fields := make([]string, 0, len(ctx.OwnerTags))
	for f := range ctx.OwnerTags {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	for _, f := range fields {
		if owner := ctx.OwnerTags[f]; owner != ctx.TraderID {
			return f, owner, true
		}
	}
	return "", "", false
}
