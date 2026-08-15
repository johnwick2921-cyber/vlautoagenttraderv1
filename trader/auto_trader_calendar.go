package trader

import (
	"encoding/json"
	"time"

	"nofx/calendar"
	"nofx/kernel"
	"nofx/store"
)

// W3 — the calendar PRODUCER (the audit's dead wire): fetch the ForexFactory
// weekly feed and store one slice per CT trade-date, so the planner's GetSlice
// read finally has data and T1 (red) events become HARD no-trade blackouts. Gated
// on day_plan → dormant by default; idempotent (SaveSliceIfAbsent); throttled.

// maybeFetchCalendar stores the week's calendar slices when the current
// trade-date's slice is missing (covers on-boot + each new day). Network attempts
// are throttled to ≤1/hour so an outage doesn't hammer the feed.
func (at *AutoTrader) maybeFetchCalendar(now time.Time) {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	tradeDate := plannerTradeDateCT(now)
	if slice, _ := at.store.Calendar().GetSlice(tradeDate); slice != nil {
		return // already have today's slice
	}
	if !at.lastCalFetch.IsZero() && now.Sub(at.lastCalFetch) < time.Hour {
		return // throttle outage retries
	}
	at.lastCalFetch = now

	res := calendar.FetchWeek(calendar.DefaultFetch, calendarStaticLoader)
	if res.Warning != "" {
		at.logWarnf("🗓️ calendar: %s", res.Warning)
	}
	for date, evs := range res.Days {
		js, _ := json.Marshal(evs)
		if wrote, err := at.store.Calendar().SaveSliceIfAbsent(&store.CalendarSliceDB{
			TradeDate: date, Source: string(res.Source), EventsJSON: string(js), CreatedAt: now.UnixMilli(),
		}); err == nil && wrote {
			at.logInfof("🗓️ calendar slice stored %s (%d events, src %s).", date, len(evs), res.Source)
		}
	}
}

// calendarStaticLoader is the owner-editable static T1 fallback. No file yet →
// nil, so FetchWeek uses the live feed and only warns on outage (never fabricates).
func calendarStaticLoader() []calendar.Event { return nil }

// currentT1Windows returns the red-news HARD no-trade windows for the active
// session at `now` (empty when no slice / no T1 events).
func (at *AutoTrader) currentT1Windows(now time.Time) []kernel.CTWindow {
	if at.store == nil {
		return nil
	}
	reg := kernel.DefaultSessionRegistry()
	sess, ok := reg.ActiveSession(now)
	if !ok {
		return nil
	}
	tradeDate := plannerTradeDateCT(now)
	slice, err := at.store.Calendar().GetSlice(tradeDate)
	if err != nil || slice == nil {
		return nil
	}
	var evs []calendar.Event
	if json.Unmarshal([]byte(slice.EventsJSON), &evs) != nil {
		return nil
	}
	return kernel.T1BlackoutWindows(sessionPlannerEvents(evs, sess.Name))
}

// sessionPlannerEvents maps stored calendar.Event → kernel.PlannerCalendarEvent
// (session-sliced, CT time). Shared by the gate + the planner input + the plan
// no-trade injection so all three agree.
func sessionPlannerEvents(evs []calendar.Event, session string) []kernel.PlannerCalendarEvent {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		loc = time.UTC
	}
	var out []kernel.PlannerCalendarEvent
	for _, e := range calendar.EventsForSession(evs, session) {
		out = append(out, kernel.PlannerCalendarEvent{
			TimeCT: e.Time.In(loc).Format("15:04"), Currency: e.Currency,
			Title: e.Title, Impact: string(e.Impact),
		})
	}
	return out
}
