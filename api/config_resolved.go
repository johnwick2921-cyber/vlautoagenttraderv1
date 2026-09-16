// Settings integrity (D5) — GET /api/config/resolved.
//
// One place a client can ask "what does this build actually honour?". The
// answer is the registry's own classification and the registry's own label:
// the UI renders what is here, it does not re-derive a status from a field
// name or invent wording for one.
//
// The owner's two-label ruling is the reason this endpoint exists rather than
// a boolean "active" flag. "No consumer" and "reads it but cannot take effect"
// are different findings and must stay legible as different findings:
//
//	ineffective            read; does not take effect (<reason>)
//	candidate-unverified   no known reader — pending verification
//
// Neither reads as dead. A candidate stays listed until someone runs the
// method-level grep and quotes it.
//
// A25: KnobEntry holds no values, and this payload adds none. Infra knobs are
// ports, paths and keys; classification is safe to serve, values are not.

package api

import (
	"net/http"
	"strconv"
	"strings"

	"nofx/store"

	"github.com/gin-gonic/gin"
)

type resolvedKnob struct {
	Path      string   `json:"path"`
	Status    string   `json:"status"`
	UILabel   string   `json:"ui_label"`
	Consumers []string `json:"consumers"`
	DualLevel bool     `json:"dual_level"`
	Clamp     string   `json:"clamp,omitempty"`
	Note      string   `json:"note,omitempty"`
}

type resolvedSummary struct {
	Schema              int      `json:"schema"`
	Classified          int      `json:"classified"`
	Live                int      `json:"live"`
	Ineffective         int      `json:"ineffective"`
	CandidateUnverified int      `json:"candidate_unverified"`
	Suspended           int      `json:"suspended"`
	Advisory            int      `json:"advisory"`
	DisplayOnly         int      `json:"display_only"`
	Infra               int      `json:"infra"`
	EnvShadows          int      `json:"env_shadows"`
	EnvShadowPaths      []string `json:"env_shadow_paths"`
}

// resolvedField is one "saved → resolved · source" line. Saved is what is in
// the config as written; Resolved is what the engine will actually use; Source
// says which rule produced the difference. An absent saved value renders
// "(unset)" and never borrows the resolved value — collapsing those two is the
// defect this line exists to expose.
type resolvedField struct {
	Path     string `json:"path"`
	Saved    string `json:"saved"`
	Resolved string `json:"resolved"`
	Source   string `json:"source"`
	Line     string `json:"line"`
}

const unsetSaved = "(unset)"

func renderLine(saved, resolved, source string) string {
	return saved + " → " + resolved + " · " + source
}

// buildResolvedFields narrates R1/R2/R3 by calling the SAME resolvers the arm
// seam and the planner call. It never re-implements a rule: if a resolution
// changes, this line changes with it.
func buildResolvedFields(cfg *store.StrategyConfig, session string) []resolvedField {
	if cfg == nil {
		return []resolvedField{}
	}
	out := make([]resolvedField, 0, 3)

	// R1 — one R:R floor for the arm seam and the decision path.
	rr, rrSrc := store.ResolveMinRiskReward(cfg)
	savedRR := unsetSaved
	if cfg.RiskControl.MinRiskRewardRatio > 0 {
		savedRR = strconv.FormatFloat(cfg.RiskControl.MinRiskRewardRatio, 'f', -1, 64)
	}
	resolvedRR := strconv.FormatFloat(rr, 'f', -1, 64)
	out = append(out, resolvedField{
		Path: "risk_control.min_risk_reward_ratio", Saved: savedRR,
		Resolved: resolvedRR, Source: rrSrc,
		Line: renderLine(savedRR, resolvedRR, rrSrc),
	})

	// R2 — plan mode at the arm seam, per session.
	var dp *store.DayPlanConfig
	if cfg.DayPlan != nil {
		dp = cfg.DayPlan
	}
	mode, modeSrc := store.ResolvePlanMode(dp, session)
	savedMode := unsetSaved
	if dp != nil && strings.TrimSpace(dp.PlanMode) != "" {
		savedMode = dp.PlanMode
	}
	out = append(out, resolvedField{
		Path: "day_plan.plan_mode", Saved: savedMode,
		Resolved: mode, Source: modeSrc,
		Line: renderLine(savedMode, mode, modeSrc),
	})

	// R3 — one htf_veto, shipped default ON when the block is absent.
	veto, vetoSrc := store.ResolveHTFVeto(cfg)
	savedVeto := unsetSaved
	if cfg.Regime != nil && cfg.Regime.HTFVeto != nil {
		savedVeto = strconv.FormatBool(*cfg.Regime.HTFVeto)
	}
	resolvedVeto := strconv.FormatBool(veto)
	out = append(out, resolvedField{
		Path: "regime.htf_veto", Saved: savedVeto,
		Resolved: resolvedVeto, Source: vetoSrc,
		Line: renderLine(savedVeto, resolvedVeto, vetoSrc),
	})

	return out
}

// handleConfigResolved serves the knob registry. The counts are taken from the
// same KnobStatusSummary the ⚙ boot line prints, so the page and the log can
// never disagree about how many knobs take effect.
// configResolvedPayload assembles the response. Pure, so the trader-context
// path is testable without standing up a manager: cfg == nil means there was
// nothing to resolve against and the key is omitted entirely.
func configResolvedPayload(cfg *store.StrategyConfig, session string) gin.H {
	sum := store.KnobStatusSummary()

	entries := store.AllKnobs()
	knobs := make([]resolvedKnob, 0, len(entries))
	for _, e := range entries {
		consumers := e.Consumers
		if consumers == nil {
			// An empty computed list is [], never null: a knob with no
			// recorded consumer is a finding, not missing data.
			consumers = []string{}
		}
		knobs = append(knobs, resolvedKnob{
			Path:      e.Path,
			Status:    string(e.Status),
			UILabel:   e.UILabel(),
			Consumers: consumers,
			DualLevel: e.DualLevel,
			Clamp:     e.Clamp,
			Note:      e.Note,
		})
	}

	paths := sum.EnvShadowPaths
	if paths == nil {
		paths = []string{}
	}

	payload := gin.H{
		"summary": resolvedSummary{
			Schema:              len(store.EnumerateSchemaKnobs()),
			Classified:          sum.Total,
			Live:                sum.Live,
			Ineffective:         sum.Ineffective,
			CandidateUnverified: sum.Candidate,
			Suspended:           sum.Suspended,
			Advisory:            sum.Advisory,
			DisplayOnly:         sum.DisplayOnly,
			Infra:               sum.Infra,
			EnvShadows:          sum.EnvShadows,
			EnvShadowPaths:      paths,
		},
		"knobs": knobs,
	}

	// An uncomputed list is ABSENT; an empty computed one is []. Without a
	// trader there is no config to resolve against, so the key is omitted
	// rather than rendered as an empty result the operator would read as
	// "nothing resolves here".
	if cfg != nil {
		payload["resolved"] = buildResolvedFields(cfg, session)
	}
	return payload
}

// handleConfigResolved serves the registry, plus the "saved → resolved · source"
// lines when a trader is named (?trader_id=&session=).
func (s *Server) handleConfigResolved(c *gin.Context) {
	var cfg *store.StrategyConfig
	session := c.Query("session")
	if id := c.Query("trader_id"); id != "" && s.traderManager != nil {
		if at, err := s.traderManager.GetTrader(id); err == nil && at != nil {
			cfg = at.GetStrategyConfig()
		}
	}
	c.JSON(http.StatusOK, configResolvedPayload(cfg, session))
}
