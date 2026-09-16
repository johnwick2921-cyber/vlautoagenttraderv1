package mcp

import "strings"

// DefaultModelForAlias maps a bare provider alias ("deepseek", "qwen", "minimax")
// to its EXACT default model string. Returns "" for anything that is not a known
// alias (i.e. it is already an exact model id). Used for model-id pinning (§125):
// the day plan must never be stamped with a provider alias.
func DefaultModelForAlias(alias string) string {
	switch strings.ToLower(strings.TrimSpace(alias)) {
	case "deepseek":
		return DefaultDeepSeekModel
	case "qwen":
		return DefaultQwenModel
	case "minimax":
		return DefaultMiniMaxModel
	default:
		return ""
	}
}

// IsProviderAlias reports whether id is a bare provider alias rather than an exact
// model string.
func IsProviderAlias(id string) bool {
	return DefaultModelForAlias(id) != ""
}
