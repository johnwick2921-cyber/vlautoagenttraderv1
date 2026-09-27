//go:build r24harness

package main

// stats.go — exact two-sided binomial p against the D1′ IID null
// p(hold) = 0.5067 (the calibrated coin flip), normal approximation above
// n = 2000 (continuity-corrected).

import (
	"math"
)

func binomTwoSidedP(x, n int, p0 float64) float64 {
	if n <= 0 {
		return 1
	}
	if n > 2000 {
		// continuity-corrected two-sided normal approximation
		pHat := float64(x) / float64(n)
		sd := math.Sqrt(p0 * (1 - p0) / float64(n))
		z := (pHat - p0) / sd
		if z < 0 {
			z = -z
		}
		zc := z - 0.5/float64(n)/sd // continuity correction toward the null
		if zc < 0 {
			zc = 0
		}
		return 2 * (1 - normCDF(zc))
	}
	px := binomPmf(x, n, p0)
	var sum float64
	for k := 0; k <= n; k++ {
		pk := binomPmf(k, n, p0)
		if pk <= px*(1+1e-12) {
			sum += pk
		}
	}
	if sum > 1 {
		sum = 1
	}
	return sum
}

func binomPmf(k, n int, p float64) float64 {
	if p == 0 {
		if k == 0 {
			return 1
		}
		return 0
	}
	if p == 1 {
		if k == n {
			return 1
		}
		return 0
	}
	l := lgamma(float64(n+1)) - lgamma(float64(k+1)) - lgamma(float64(n-k+1))
	return math.Exp(l + float64(k)*math.Log(p) + float64(n-k)*math.Log(1-p))
}

func lgamma(x float64) float64 {
	l, _ := math.Lgamma(x)
	return l
}

// normCDF is the standard normal cumulative (Abramowitz–Stegun 7.1.26).
func normCDF(z float64) float64 {
	if z < -8 {
		return 0
	}
	if z > 8 {
		return 1
	}
	t := 1 / (1 + 0.2316419*math.Abs(z))
	d := 0.3989422804014327 * math.Exp(-z*z/2)
	p := 1 - d*(0.319381530*t-0.356563782*t*t+1.781477937*t*t*t-1.821255978*t*t*t*t+1.330274429*t*t*t*t*t)
	if z < 0 {
		return 1 - p
	}
	return p
}
