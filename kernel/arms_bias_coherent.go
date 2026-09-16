// D2 (arms-follow-bias, 2026-09-04) — a plan should be able to trade its own bias.
//
// With plan_mode strict the decision path is closed and ARMS ARE THE ONLY
// ENTRY, so a plan whose bias direction carries no armed scenario cannot act on
// the direction it just argued for. NY 2026-09-03 v7 authored a long AND a
// short on the identical level 29543.75, both confirms true at 11:58 CT, and
// armed only the short.
//
// WARN-FIRST (owner ruling 2026-09-04). Measured across 171 stored directional
// plans, a hard reject would have refused 50/68 longs and 66/103 shorts —
// roughly two thirds of everything the planner has ever written. A reject that
// fires that often teaches the model nothing, because it cannot produce a
// compliant plan at all: 18 of those longs lean on breakout_retest, which is
// BOTH un-armable and shadowed, and nothing ever told the model. So this counts
// per side, D1 tells the model which conditions can actually be armed, and a
// later ruling promotes it to a reject once the numbers have moved.
//
// This encodes NO market belief: not which way to lean, nor what to arm. Only
// that the direction the planner CHOSE should have a way to reach the market.

package kernel

import (
	"fmt"
	"sort"
	"strings"
)

// BiasCoherentArmsHint is the text the model reads, and the class-34/38 hint
// registry entry that guards its tokens.
const BiasCoherentArmsHint = "bias_requires_an_arm (a plan whose bias direction carries no armed scenario cannot trade its own bias — arm a scenario in the bias direction using an armable, live condition, or state a neutral bias)"

// ArmableConditionsLine renders the vocabulary the planner is allowed to arm,
// DERIVED from ArmableCondition/ArmKindFor and the resolved live|shadow
// statuses. Nothing here is hand-listed: a condition that changes status or
// gains a kind changes this line, and the prompt with it.
//
// (a), owner ruling 2026-09-04 — the 18 long plans that could never comply were
// leaning on a condition that was un-armable AND shadowed, and the prompt had
// never named either fact.
func ArmableConditionsLine(statuses map[string]string) string {
	var live, shadowed, unarmable []string
	for _, c := range KnownConditions() {
		kind := ArmKindFor(c)
		armable := ArmableCondition(c) || kind != ""
		shadow := statuses[strings.ToLower(strings.TrimSpace(c))] == ConditionShadow
		switch {
		case armable && !shadow:
			live = append(live, fmt.Sprintf("%s→%s", c, kind))
		case armable && shadow:
			shadowed = append(shadowed, c)
		default:
			unarmable = append(unarmable, c)
		}
	}
	sort.Strings(live)
	sort.Strings(shadowed)
	sort.Strings(unarmable)

	parts := []string{"ARMABLE + LIVE (these are the only conditions an arm can rest on): " + strings.Join(live, " · ")}
	if len(shadowed) > 0 {
		parts = append(parts, "armable but SHADOWED — do not arm these: "+strings.Join(shadowed, ", "))
	}
	if len(unarmable) > 0 {
		parts = append(parts, "NOT armable at all: "+strings.Join(unarmable, ", "))
	}
	return strings.Join(parts, ". ") + "."
}

// BiasArmWarning returns the D2 warning for a plan, or "" when the plan can
// trade its own bias. It names the scenarios the model chose in the bias
// direction and why they could not be armed, so the line is actionable instead
// of a restatement of the rule.
func BiasArmWarning(d *PlanDoc, statuses map[string]string) string {
	if d == nil {
		return ""
	}
	bias := strings.ToLower(strings.TrimSpace(d.Bias.Direction))
	if bias != "long" && bias != "short" {
		return "" // neutral: no direction to be incoherent with
	}

	var blockers []string
	armedInBias, armedOther := 0, 0
	for _, sc := range d.Scenarios {
		inBias := strings.EqualFold(strings.TrimSpace(sc.Direction), bias)
		armed := sc.Arm != nil && sc.Arm.Enabled
		switch {
		case inBias && armed:
			armedInBias++
		case !inBias && armed:
			armedOther++
		case inBias:
			cond := strings.ToLower(strings.TrimSpace(sc.Condition))
			switch {
			case !ArmableCondition(cond) && ArmKindFor(cond) == "":
				reason := "un-armable"
				if statuses[cond] == ConditionShadow {
					reason = "shadowed/un-armable"
				}
				blockers = append(blockers, fmt.Sprintf("%s %s is %s", sc.ID, cond, reason))
			case statuses[cond] == ConditionShadow:
				blockers = append(blockers, fmt.Sprintf("%s %s is shadowed", sc.ID, cond))
			default:
				blockers = append(blockers, fmt.Sprintf("%s %s is armable but carries no arm", sc.ID, cond))
			}
		}
	}
	if armedInBias > 0 {
		return ""
	}

	msg := fmt.Sprintf("⚠ bias=%s but no %s scenario carries an arm", bias, bias)
	if armedOther > 0 {
		msg += fmt.Sprintf(" (%d arm(s) authored on the other side)", armedOther)
	}
	if len(blockers) > 0 {
		msg += " — " + strings.Join(blockers, "; ")
	} else {
		msg += " — no scenario in the bias direction at all"
	}
	return msg + ". " + BiasCoherentArmsHint
}
