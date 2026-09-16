// Wave 1D — the per-condition expectancy read model, exposed read-only.
//
//	GET /api/expectancy?by=condition|session|kind|path|full&era=pre-0B|post-0B
//
// This endpoint RENDERS a table; it rules on nothing. Every verdict in it is
// computed from the pre-registered criterion in the model (n >= MinN and a mean
// interval that excludes zero) — the endpoint never decides a status, and there
// is no query parameter that can loosen the floor.
package api

import (
	"net/http"
	"time"

	"nofx/expectancy"

	"github.com/gin-gonic/gin"
)

// expectancyView is the projection the caller asked for. Unknown or absent
// `by` renders the condition roll-up, which is the table the shadow/promote
// rulings are actually made from.
func expectancyView(tbl *expectancy.Table, by string) ([]expectancy.Cell, string) {
	switch by {
	case "full", "cell", "cells":
		return tbl.Cells, "full"
	case "session":
		return tbl.Sessions, "session"
	case "kind":
		return tbl.Kinds, "kind"
	case "path":
		return tbl.Paths, "path"
	default:
		return tbl.Conditions, "condition"
	}
}

func (s *Server) handleExpectancy(c *gin.Context) {
	tbl, err := expectancy.LoadAndBuildAt(s.store.GormDB(), time.Now())
	if err != nil {
		// A read model never 500s the caller into thinking the system is down;
		// it says it could not compute and shows the empty ledger.
		c.JSON(http.StatusOK, gin.H{
			"error":    err.Error(),
			"rows":     []expectancy.Cell{},
			"excluded": tbl.Excluded,
			"min_n":    expectancy.MinN,
		})
		return
	}
	if era := c.Query("era"); era != "" {
		tbl = expectancy.FilterEra(tbl, era)
	}
	rows, by := expectancyView(&tbl, c.Query("by"))

	asOf := ""
	if tbl.AsOfMs > 0 {
		asOf = time.UnixMilli(tbl.AsOfMs).UTC().Format(time.RFC3339)
	}
	c.JSON(http.StatusOK, gin.H{
		"by":                by,
		"rows":              rows,
		"counterfactual_e8": tbl.Counterfactual,
		"excluded":          tbl.Excluded,
		// min_n travels with the payload so the panel cannot hold its own copy
		// of the floor and drift from the binary that computed the statuses.
		"min_n":          expectancy.MinN,
		"as_of_ms":       tbl.AsOfMs,
		"as_of_utc":      asOf,
		"built_at_ms":    tbl.BuiltAtMs,
		"era_0b_start":   expectancy.Era0BStart.UTC().Format(time.RFC3339),
		"promotion_rule": "n >= min_n AND mean > 0 AND the 95% interval on the mean excludes 0",
	})
}
