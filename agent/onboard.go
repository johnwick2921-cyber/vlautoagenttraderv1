package agent

import (
	"fmt"
	"nofx/branding"
	"strings"

	"nofx/store"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var titleCaser = cases.Title(language.English)

const setupExchangeAccountName = "Default"

// Onboard handles first-time setup through natural language.
// When there's no trader configured, the agent guides the user.

// SetupState tracks where the user is in the setup flow.
type SetupState struct {
	Step       string // "", "await_exchange", "await_api_key", "await_api_secret", "await_passphrase", "await_ai_model", "await_ai_key"
	Exchange   string
	ExchangeID string
	APIKey     string
	APISecret  string
	Passphrase string
	AIProvider string
	AIModel    string
	AIModelID  string
	AIKey      string
	AIBaseURL  string
}

// getSetupState loads the current setup state from user preferences.
func (a *Agent) getSetupState(userID int64) *SetupState {
	if cached, ok := a.stateOwner().setupStates.Load(userID); ok {
		if state, ok := cached.(*SetupState); ok && state != nil {
			return cloneSetupState(state)
		}
	}
	step, _ := a.store.GetSystemConfig(fmt.Sprintf("setup_step_%d", userID))
	if step == "" {
		return &SetupState{}
	}
	return &SetupState{
		Step:       step,
		Exchange:   getConfig(a.store, userID, "exchange"),
		ExchangeID: getConfig(a.store, userID, "exchange_id"),
		AIProvider: getConfig(a.store, userID, "ai_provider"),
		AIModel:    getConfig(a.store, userID, "ai_model"),
		AIModelID:  getConfig(a.store, userID, "ai_model_id"),
		AIBaseURL:  getConfig(a.store, userID, "ai_base_url"),
	}
}

func (a *Agent) saveSetupState(userID int64, s *SetupState) {
	a.stateOwner().setupStates.Store(userID, cloneSetupState(s))
	a.store.SetSystemConfig(fmt.Sprintf("setup_step_%d", userID), s.Step)
	setConfig(a.store, userID, "exchange", s.Exchange)
	setConfig(a.store, userID, "exchange_id", s.ExchangeID)
	setConfig(a.store, userID, "ai_provider", s.AIProvider)
	setConfig(a.store, userID, "ai_model", s.AIModel)
	setConfig(a.store, userID, "ai_model_id", s.AIModelID)
	setConfig(a.store, userID, "ai_base_url", s.AIBaseURL)
}

func (a *Agent) clearSetupState(userID int64) {
	a.stateOwner().setupStates.Delete(userID)
	for _, k := range []string{"step", "exchange", "exchange_id", "ai_provider", "ai_model", "ai_model_id", "ai_base_url"} {
		a.store.SetSystemConfig(fmt.Sprintf("setup_%s_%d", k, userID), "")
	}
	a.store.SetSystemConfig(fmt.Sprintf("setup_step_%d", userID), "")
}

func getConfig(st *store.Store, uid int64, key string) string {
	v, _ := st.GetSystemConfig(fmt.Sprintf("setup_%s_%d", key, uid))
	return v
}

func setConfig(st *store.Store, uid int64, key, val string) {
	st.SetSystemConfig(fmt.Sprintf("setup_%s_%d", key, uid), val)
}

func cloneSetupState(s *SetupState) *SetupState {
	if s == nil {
		return &SetupState{}
	}
	copy := *s
	return &copy
}

func isDirectSetupCommand(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return false
	}
	switch text {
	case "setup", "/setup":
		return true
	default:
		return false
	}
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

var setupMessages = map[string]map[string]string{
	"welcome": {
		"zh": "👋 你好！我是 *" + branding.PersonaName() + "*，你的 AI 交易 Agent。\n\n" +
			"我发现你还没有配置交易所，让我帮你搞定吧！\n\n" +
			"发送 *开始配置* 或 *setup* 开始\n" +
			"发送 *取消* 随时退出",
		"en": "👋 Hi! I'm *" + branding.PersonaName() + "*, your AI trading agent.\n\n" +
			"I see you haven't configured an exchange yet. Let me help!\n\n" +
			"Send *setup* to begin\n" +
			"Send *cancel* to exit anytime",
	},
	"ask_exchange": {
		"zh": "🏦 *选择你的交易所*\n\n" +
			"1️⃣ Binance（币安）\n" +
			"2️⃣ OKX（欧易）\n" +
			"3️⃣ Bybit\n" +
			"4️⃣ Bitget\n" +
			"5️⃣ Gate\n" +
			"6️⃣ KuCoin（库币）\n" +
			"7️⃣ Hyperliquid\n\n" +
			"发送数字或名称选择：",
		"en": "🏦 *Choose your exchange*\n\n" +
			"1️⃣ Binance\n" +
			"2️⃣ OKX\n" +
			"3️⃣ Bybit\n" +
			"4️⃣ Bitget\n" +
			"5️⃣ Gate\n" +
			"6️⃣ KuCoin\n" +
			"7️⃣ Hyperliquid\n\n" +
			"Send number or name:",
	},
	"invalid_exchange": {
		"zh": "❓ 没有识别到交易所。请发送数字 1-7 或交易所名称。",
		"en": "❓ Exchange not recognized. Send a number 1-7 or exchange name.",
	},
	"ask_secret": {
		"zh": "🔑 收到 API Key。\n\n现在请发送你的 *API Secret*：",
		"en": "🔑 Got API Key.\n\nNow send your *API Secret*:",
	},
	"ask_passphrase": {
		"zh": "🔐 收到 API Secret。\n\n这个交易所还需要 *Passphrase*，请发送：",
		"en": "🔐 Got API Secret.\n\nThis exchange also needs a *Passphrase*. Please send it:",
	},
	"ask_ai": {
		"zh": "🤖 *选择 AI 模型*\n\n" +
			"1️⃣ DeepSeek（推荐，便宜好用）\n" +
			"2️⃣ 通义千问 (Qwen)\n" +
			"3️⃣ OpenAI (GPT-4o)\n" +
			"4️⃣ Claude\n" +
			"5️⃣ 跳过（不配置 AI）\n\n" +
			"发送数字或名称选择：",
		"en": "🤖 *Choose AI model*\n\n" +
			"1️⃣ DeepSeek (recommended, affordable)\n" +
			"2️⃣ Qwen\n" +
			"3️⃣ OpenAI (GPT-4o)\n" +
			"4️⃣ Claude\n" +
			"5️⃣ Skip (no AI)\n\n" +
			"Send number or name:",
	},
	"invalid_ai": {
		"zh": "❓ 没有识别到 AI 模型。请发送数字 1-5 或模型名称。",
		"en": "❓ AI model not recognized. Send a number 1-5 or model name.",
	},
	"cancelled": {
		"zh": "👌 配置已取消。随时发送 *开始配置* 重新开始。",
		"en": "👌 Setup cancelled. Send *setup* anytime to restart.",
	},
}
