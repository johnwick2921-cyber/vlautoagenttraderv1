// The store cannot import kernel (kernel imports store), so the detector's two
// resolved parameters are read here through the same environment contract the
// kernel resolver uses. The NAMES are kept identical on purpose: a reader
// grepping DETECTOR_K finds both sites, and a drift between them is visible.
//
// Δ is deliberately NOT mirrored — it is derived from bars, not the
// environment, and the boot line names its source rather than a value.

package store

import (
	"os"
	"strconv"
	"strings"
)

func detectorKForBoot() float64 {
	if v := strings.TrimSpace(os.Getenv("DETECTOR_K")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return 3.0
}

func detectorHorizonForBoot() int {
	if v := strings.TrimSpace(os.Getenv("DETECTOR_HORIZON_BARS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 12
}
