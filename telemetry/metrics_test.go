package telemetry

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// Plan 4 Task 25 — verify metric collectors are registered and behave as expected.

// gatherFamilies returns all metric families from the default registry.
func gatherFamilies(t *testing.T) []*dto.MetricFamily {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}
	if len(mfs) == 0 {
		t.Fatalf("expected at least one metric family registered, got 0")
	}
	return mfs
}

func familyByName(mfs []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, mf := range mfs {
		if mf.GetName() == name {
			return mf
		}
	}
	return nil
}

// TestAllCollectorsRegistered confirms the 5 Plan 4 T25 collectors are present.
func TestAllCollectorsRegistered(t *testing.T) {
	expected := []string{
		"nofx_decisions_total",
		"nofx_decision_latency_seconds",
		"nofx_fill_latency_seconds",
		"nofx_databento_errors_total",
		"nofx_risk_gate_trips_total",
	}

	// Touch each collector once so it surfaces in Gather() output.
	// Counters/Histograms with label dims only appear after at least one
	// child has been instantiated.
	DecisionsTotal.WithLabelValues("test-trader", "HOLD", "queued").Add(0)
	DecisionLatency.WithLabelValues("test-trader").Observe(0)
	FillLatency.WithLabelValues("test-exchange").Observe(0)
	DatabentoErrorsTotal.Add(0)
	RiskGateTrips.WithLabelValues("test-gate").Add(0)

	mfs := gatherFamilies(t)
	for _, name := range expected {
		if familyByName(mfs, name) == nil {
			// Dump found names for debugging.
			var names []string
			for _, mf := range mfs {
				names = append(names, mf.GetName())
			}
			t.Fatalf("collector %q not registered. registered: %s",
				name, strings.Join(names, ", "))
		}
	}
}

// TestDecisionsTotalIncrement verifies counter increment + label propagation.
func TestDecisionsTotalIncrement(t *testing.T) {
	// Use a unique trader_id so this test doesn't interfere with others.
	DecisionsTotal.WithLabelValues("t1-incr", "LONG", "queued").Inc()
	DecisionsTotal.WithLabelValues("t1-incr", "LONG", "queued").Inc()

	mfs := gatherFamilies(t)
	fam := familyByName(mfs, "nofx_decisions_total")
	if fam == nil {
		t.Fatal("nofx_decisions_total not found")
	}

	var got float64
	for _, m := range fam.Metric {
		if labelEquals(m.Label, map[string]string{
			"trader_id": "t1-incr", "action": "LONG", "status": "queued",
		}) {
			got = m.GetCounter().GetValue()
			break
		}
	}
	if got != 2 {
		t.Errorf("DecisionsTotal{t1-incr,LONG,queued} = %v, want 2", got)
	}
}

// TestDecisionLatencyObservation verifies histogram observation count.
func TestDecisionLatencyObservation(t *testing.T) {
	DecisionLatency.WithLabelValues("t1-hist").Observe(0.5)
	DecisionLatency.WithLabelValues("t1-hist").Observe(1.5)
	DecisionLatency.WithLabelValues("t1-hist").Observe(2.0)

	mfs := gatherFamilies(t)
	fam := familyByName(mfs, "nofx_decision_latency_seconds")
	if fam == nil {
		t.Fatal("nofx_decision_latency_seconds not found")
	}

	var samples uint64
	for _, m := range fam.Metric {
		if labelEquals(m.Label, map[string]string{"trader_id": "t1-hist"}) {
			samples = m.GetHistogram().GetSampleCount()
			break
		}
	}
	if samples != 3 {
		t.Errorf("DecisionLatency{t1-hist} sample count = %d, want 3", samples)
	}
}

// TestFillLatencyObservation verifies the futures-friendly bucket histogram.
func TestFillLatencyObservation(t *testing.T) {
	FillLatency.WithLabelValues("ninjatrader-test").Observe(0.2)
	FillLatency.WithLabelValues("ninjatrader-test").Observe(45.0) // slow NT8 CSV bridge

	mfs := gatherFamilies(t)
	fam := familyByName(mfs, "nofx_fill_latency_seconds")
	if fam == nil {
		t.Fatal("nofx_fill_latency_seconds not found")
	}

	var samples uint64
	for _, m := range fam.Metric {
		if labelEquals(m.Label, map[string]string{"exchange": "ninjatrader-test"}) {
			samples = m.GetHistogram().GetSampleCount()
			break
		}
	}
	if samples != 2 {
		t.Errorf("FillLatency{ninjatrader-test} sample count = %d, want 2", samples)
	}
}

// TestDatabentoErrorsTotalIncrement verifies the plain counter increments.
func TestDatabentoErrorsTotalIncrement(t *testing.T) {
	before := getCounterValue(t, "nofx_databento_errors_total")
	DatabentoErrorsTotal.Inc()
	DatabentoErrorsTotal.Inc()
	after := getCounterValue(t, "nofx_databento_errors_total")
	if after-before != 2 {
		t.Errorf("DatabentoErrorsTotal delta = %v, want 2", after-before)
	}
}

// TestRiskGateTripsForEachGate verifies all 4 expected gate-name labels.
func TestRiskGateTripsForEachGate(t *testing.T) {
	gates := []string{
		"task18_cme_closed",
		"task19_contract_roll",
		"task21_risk_limit",
		"task22_drift",
	}
	for _, g := range gates {
		RiskGateTrips.WithLabelValues(g).Inc()
	}

	mfs := gatherFamilies(t)
	fam := familyByName(mfs, "nofx_risk_gate_trips_total")
	if fam == nil {
		t.Fatal("nofx_risk_gate_trips_total not found")
	}

	got := map[string]float64{}
	for _, m := range fam.Metric {
		for _, l := range m.Label {
			if l.GetName() == "gate_name" {
				got[l.GetValue()] = m.GetCounter().GetValue()
			}
		}
	}
	for _, g := range gates {
		if got[g] < 1 {
			t.Errorf("RiskGateTrips{gate_name=%q} = %v, want >= 1", g, got[g])
		}
	}
}

// --- helpers ---

func labelEquals(labels []*dto.LabelPair, want map[string]string) bool {
	if len(labels) != len(want) {
		return false
	}
	for _, l := range labels {
		if v, ok := want[l.GetName()]; !ok || v != l.GetValue() {
			return false
		}
	}
	return true
}

func getCounterValue(t *testing.T, name string) float64 {
	t.Helper()
	mfs := gatherFamilies(t)
	fam := familyByName(mfs, name)
	if fam == nil {
		t.Fatalf("metric family %q not found", name)
	}
	// Single (non-vec) counter: one Metric entry.
	if len(fam.Metric) == 0 {
		return 0
	}
	return fam.Metric[0].GetCounter().GetValue()
}
