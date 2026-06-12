package market

import (
	"testing"
	"time"
)

func TestIsFresh(t *testing.T) {
	now := time.Date(2026, 5, 26, 14, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		openTime time.Time
		maxAge   time.Duration
		want     bool
	}{
		{"fresh 5s ago, max 60s", now.Add(-5 * time.Second), 60 * time.Second, true},
		{"stale 120s ago, max 90s", now.Add(-120 * time.Second), 90 * time.Second, false},
		{"future 60s ahead (clock skew) → not fresh", now.Add(60 * time.Second), 90 * time.Second, false},
		{"exactly at max age", now.Add(-90 * time.Second), 90 * time.Second, true},
		{"1ms past max age", now.Add(-90*time.Second - time.Millisecond), 90 * time.Second, false},
		{"at now (0 age)", now, 60 * time.Second, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsFresh(tt.openTime.UnixMilli(), now, tt.maxAge)
			if got != tt.want {
				t.Errorf("IsFresh(%v, %v, %v) = %v, want %v", tt.openTime, now, tt.maxAge, got, tt.want)
			}
		})
	}
}

func TestIsDriftSuspicious(t *testing.T) {
	tests := []struct {
		name         string
		current      float64
		prev         float64
		elapsed      time.Duration
		thresholdPct float64
		window       time.Duration
		want         bool
	}{
		{"5% jump in 10s, threshold 5%", 21000, 20000, 10 * time.Second, 5.0, 60 * time.Second, false}, // exactly 5%, not > 5%
		{"6% jump in 10s", 21200, 20000, 10 * time.Second, 5.0, 60 * time.Second, true},
		{"0.5% in 60s", 20100, 20000, 60 * time.Second, 5.0, 60 * time.Second, false},      // elapsed == window → not in window
		{"0.5% in 59s", 20100, 20000, 59 * time.Second, 5.0, 60 * time.Second, false},      // small move
		{"10% in 10min (window expired)", 22000, 20000, 10 * time.Minute, 5.0, 60 * time.Second, false},
		{"-7% in 30s (downside)", 18600, 20000, 30 * time.Second, 5.0, 60 * time.Second, true},
		{"prev=0 returns false", 20000, 0, 10 * time.Second, 5.0, 60 * time.Second, false},
		{"prev negative returns false", 20000, -5, 10 * time.Second, 5.0, 60 * time.Second, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDriftSuspicious(tt.current, tt.prev, tt.elapsed, tt.thresholdPct, tt.window)
			if got != tt.want {
				t.Errorf("IsDriftSuspicious(%v, %v, %v, %v, %v) = %v, want %v",
					tt.current, tt.prev, tt.elapsed, tt.thresholdPct, tt.window, got, tt.want)
			}
		})
	}
}

func TestCheckDataHealth(t *testing.T) {
	cfg := DefaultFreshnessConfig()
	// Pick a Tuesday at 10:00 CT (RTH); confirmed weekday by date below.
	ct, _ := time.LoadLocation("America/Chicago")
	rthNow := time.Date(2026, 5, 26, 10, 0, 0, 0, ct) // Tue 2026-05-26 10:00 CT

	prevTime := rthNow.Add(-1 * time.Minute)
	latestFresh := rthNow.Add(-5 * time.Second)
	latestStale := rthNow.Add(-3 * time.Minute) // exceeds 90s RTH cap

	tests := []struct {
		name        string
		latestMs    int64
		latestClose float64
		prevMs      int64
		prevClose   float64
		now         time.Time
		want        DataHealth
	}{
		{
			name:        "fresh 5s ago, 0.05% move → OK",
			latestMs:    latestFresh.UnixMilli(),
			latestClose: 20010,
			prevMs:      prevTime.UnixMilli(),
			prevClose:   20000,
			now:         rthNow,
			want:        HealthOK,
		},
		{
			name:        "stale 3min during RTH → stale",
			latestMs:    latestStale.UnixMilli(),
			latestClose: 20010,
			prevMs:      prevTime.UnixMilli(),
			prevClose:   20000,
			now:         rthNow,
			want:        HealthStale,
		},
		{
			name:        "fresh but 7% jump in 1min → suspicious drift",
			latestMs:    rthNow.Add(-30 * time.Second).UnixMilli(),
			latestClose: 21400, // +7%
			prevMs:      rthNow.Add(-31 * time.Second).UnixMilli(),
			prevClose:   20000,
			now:         rthNow,
			want:        HealthSuspiciousDrift,
		},
		{
			name:        "fresh, 10% jump over 10min → OK (window expired)",
			latestMs:    rthNow.Add(-10 * time.Second).UnixMilli(),
			latestClose: 22000,
			prevMs:      rthNow.Add(-10 * time.Minute).Add(-10 * time.Second).UnixMilli(),
			prevClose:   20000,
			now:         rthNow,
			want:        HealthOK,
		},
		{
			name:        "future-dated latest bar (clock skew) → stale",
			latestMs:    rthNow.Add(30 * time.Second).UnixMilli(),
			latestClose: 20010,
			prevMs:      prevTime.UnixMilli(),
			prevClose:   20000,
			now:         rthNow,
			want:        HealthStale,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckDataHealth(tt.latestMs, tt.latestClose, tt.prevMs, tt.prevClose, tt.now, cfg)
			if got != tt.want {
				t.Errorf("CheckDataHealth = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestIsRTH(t *testing.T) {
	ct, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("load America/Chicago: %v", err)
	}
	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"Tue 09:00 CT → RTH", time.Date(2026, 5, 26, 9, 0, 0, 0, ct), true},
		{"Tue 07:00 CT → pre-market", time.Date(2026, 5, 26, 7, 0, 0, 0, ct), false},
		{"Tue 08:29 CT → just before open", time.Date(2026, 5, 26, 8, 29, 0, 0, ct), false},
		{"Tue 08:30 CT → exactly at open", time.Date(2026, 5, 26, 8, 30, 0, 0, ct), true},
		{"Tue 14:59 CT → last RTH minute", time.Date(2026, 5, 26, 14, 59, 0, 0, ct), true},
		{"Tue 15:00 CT → close (excluded)", time.Date(2026, 5, 26, 15, 0, 0, 0, ct), false},
		{"Sat 14:00 CT → weekend", time.Date(2026, 5, 23, 14, 0, 0, 0, ct), false},
		{"Sun 14:00 CT → weekend", time.Date(2026, 5, 24, 14, 0, 0, 0, ct), false},
		// Cross-zone: UTC 14:00 on Tue 2026-05-26 is 09:00 CT (during DST) → RTH
		{"UTC 14:00 mapped to 09:00 CT → RTH", time.Date(2026, 5, 26, 14, 0, 0, 0, time.UTC), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRTH(tt.t)
			if got != tt.want {
				t.Errorf("IsRTH(%v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}
