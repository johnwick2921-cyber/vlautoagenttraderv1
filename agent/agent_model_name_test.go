package agent

import (
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"nofx/mcp"
	"nofx/store"
)

// Regression for the AgentBeta 400: an empty CustomModelName (the legal
// "use the provider default" state) must NEVER be replaced by the DB row id
// ("<uuid>_deepseek") — providers reject a row id as the model name. The
// agent must pass the empty name through so the provider client keeps its
// default (deepseek → deepseek-v4-pro), like every other AI path, and report
// that effective default as the selected model name.
func TestLoadAIClientEmptyCustomModelNameNeverSendsRowID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "agent-model-name.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	// id == provider ("deepseek") makes the store mint the row id
	// "<userID>_deepseek" with provider "deepseek" and an EMPTY
	// custom_model_name — the exact shape that triggered the bug.
	if err := st.AIModel().UpdateWithName("default", "deepseek", "", true, "sk-test", "", ""); err != nil {
		t.Fatalf("create deepseek model: %v", err)
	}

	models, err := st.AIModel().List("default")
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	rowID := ""
	for _, m := range models {
		if m.Provider == "deepseek" {
			rowID = m.ID
			if strings.TrimSpace(m.CustomModelName) != "" {
				t.Fatalf("precondition: custom_model_name must be empty, got %q", m.CustomModelName)
			}
		}
	}
	if !strings.HasSuffix(rowID, "_deepseek") {
		t.Fatalf("precondition: expected a \"<uuid>_deepseek\"-shaped row id, got %q", rowID)
	}

	a := New(nil, st, DefaultConfig(), slog.Default())
	client, modelName, ok := a.loadAIClientFromStoreUser("default")
	if !ok {
		t.Fatalf("expected model selection to succeed")
	}
	if modelName == rowID || strings.Contains(modelName, "_deepseek") {
		t.Fatalf("DB row id leaked as the model name: %q", modelName)
	}
	if modelName != mcp.DefaultDeepSeekModel {
		t.Fatalf("expected the provider default %q as the selected model, got %q", mcp.DefaultDeepSeekModel, modelName)
	}

	embedder, isEmbedder := client.(mcp.ClientEmbedder)
	if !isEmbedder {
		t.Fatalf("provider client does not expose its base client")
	}
	if got := embedder.BaseClient().Model; got != mcp.DefaultDeepSeekModel {
		t.Fatalf("wire model = %q, want %q (the row id must never reach the request body)", got, mcp.DefaultDeepSeekModel)
	}
}
