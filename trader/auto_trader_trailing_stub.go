package trader

// currentTrailLevel reports the armed trailing-stop level for a position key
// ("symbol_side"), 0 when not armed. Placeholder until Phase 3B lands the
// trailing engine; the watcher only READS this for its prompt context.
func (at *AutoTrader) currentTrailLevel(key string) float64 { return 0 }
