package store

import (
	"sort"
	"strings"
	"testing"
)

// E1 — REGISTRY COMPLETENESS. Every schema field reflection can reach must
// carry a classification; a "live" one must name a consumer. Adding a field and
// leaving it unclassified FAILS THE BUILD, which is the point.
func TestKnobRegistryIsComplete(t *testing.T) {
	fields := EnumerateSchemaKnobs()
	if len(fields) < 40 {
		t.Fatalf("reflection found only %d fields — the enumerator is broken, not the registry", len(fields))
	}
	var missing, liveNoConsumer []string
	for _, p := range fields {
		e, ok := LookupKnob(p)
		if !ok {
			missing = append(missing, p)
			continue
		}
		if e.Status == KnobLive && len(e.Consumers) == 0 {
			liveNoConsumer = append(liveNoConsumer, p)
		}
	}
	sort.Strings(missing)
	sort.Strings(liveNoConsumer)
	if len(missing) > 0 {
		show := missing
		if len(show) > 10 {
			show = show[:10]
		}
		t.Errorf("%d schema field(s) unclassified — a setting nobody classified may not take effect:\n  %s",
			len(missing), strings.Join(show, "\n  "))
	}
	if len(liveNoConsumer) > 0 {
		t.Errorf("%d knob(s) marked LIVE with no consumer — the audit's dead-knob signature:\n  %s",
			len(liveNoConsumer), strings.Join(liveNoConsumer, "\n  "))
	}
	t.Logf("registry: %d reflected fields, all classified · %s", len(fields), KnobRegistryBootLine())
}

// The audit's dead knobs must NOT be classified live — the registry is only
// useful if it tells the truth about the fifteen that produced it.
func TestRegistryDoesNotCallTheAuditsDeadKnobsLive(t *testing.T) {
	for _, p := range AuditDeadKnobs2026_09_03 {
		e, ok := LookupKnob(p)
		if !ok {
			t.Errorf("%s: the audit named it and the registry does not know it", p)
			continue
		}
		if e.Status == KnobLive {
			t.Errorf("%s: registry says LIVE but the audit proved it cannot take effect", p)
		}
		// An audit-dead knob is 'ineffective' (read, no effect) — never
		// 'candidate', which means nobody has checked yet.
		if e.Status == KnobCandidate {
			t.Errorf("%s: the audit CHECKED this one — it is ineffective, not unverified", p)
		}
		if e.Note == "" {
			t.Errorf("%s: a non-live knob must carry the REASON", p)
		}
	}
}

// The boot line is counted from the registry, never typed.
func TestKnobBootLineIsCounted(t *testing.T) {
	line := KnobRegistryBootLine()
	// The ruling's two labels must BOTH appear — conflating them is the defect.
	for _, want := range []string{"settings: schema=", "classified=", "live=", "ineffective=", "candidate-unverified=", "env-shadows=0"} {
		if !strings.Contains(line, want) {
			t.Errorf("boot line missing %q: %s", want, line)
		}
	}
	if strings.Contains(line, "UNCLASSIFIED") {
		t.Errorf("the registry has unclassified fields: %s", line)
	}
	t.Logf("boot: %s", line)
}
