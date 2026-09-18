package trader

import (
	"fmt"
	"strings"

	"nofx/kernel"
	"nofx/store"
)

// T1CurrenciesBootLine (W-T1-CURRENCIES, 2026-09-18) renders the resolved
// red-news hard-block currency set at trader load — READ from the ONE resolver
// (store.DayPlanConfig.T1CurrenciesFor), never a literal: "USD(default)" when
// the strategy saved nothing, "USD,EUR(saved)" / "ALL(saved)" otherwise.
func T1CurrenciesBootLine(dp *store.DayPlanConfig) string {
	src := "default"
	if dp.T1CurrenciesSaved() {
		src = "saved"
	}
	return fmt.Sprintf("🔴 t1_blackout=%s(%s) (W-T1-CURRENCIES)", strings.TrimSpace(kernel.T1CurrencySetLabel(dp.T1CurrenciesFor())), src)
}
