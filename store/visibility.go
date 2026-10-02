package store

import "strings"

func MissingRequiredExchangeCredentialFields(exchangeType, apiKey, secretKey, passphrase string, ntDataDir ...string) []string {
	switch strings.ToLower(strings.TrimSpace(exchangeType)) {
	case "ninjatrader":
		ntDir := ""
		if len(ntDataDir) > 0 {
			ntDir = ntDataDir[0]
		}
		return missingNamedFields(
			namedField{"nt_data_dir", ntDir},
		)
	default:
		return []string{"exchange_type"}
	}
}

type namedField struct {
	name  string
	value string
}

func missingNamedFields(fields ...namedField) []string {
	missing := make([]string, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
		}
	}
	return missing
}

func IsVisibleAIModel(model *AIModel) bool {
	if model == nil {
		return false
	}
	return model.Enabled ||
		strings.TrimSpace(string(model.APIKey)) != "" ||
		strings.TrimSpace(model.CustomAPIURL) != "" ||
		strings.TrimSpace(model.CustomModelName) != ""
}

func IsVisibleExchange(exchange *Exchange) bool {
	if exchange == nil {
		return false
	}
	return exchange.Enabled ||
		strings.TrimSpace(string(exchange.APIKey)) != "" ||
		strings.TrimSpace(string(exchange.SecretKey)) != "" ||
		strings.TrimSpace(string(exchange.Passphrase)) != "" ||
		strings.TrimSpace(exchange.NTDataDir) != ""
}

func IsVisibleTrader(trader *Trader) bool {
	if trader == nil {
		return false
	}
	return strings.TrimSpace(trader.Name) != "" &&
		strings.TrimSpace(trader.AIModelID) != "" &&
		strings.TrimSpace(trader.ExchangeID) != ""
}

func IsVisibleStrategy(strategy *Strategy) bool {
	if strategy == nil {
		return false
	}
	return strings.TrimSpace(strategy.Name) != ""
}
