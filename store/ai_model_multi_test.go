package store

import (
	"path/filepath"
	"testing"
	"time"
)

func newAIModelTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	return st
}

// Two entries for the SAME provider must produce TWO distinct rows, both listed —
// the core of "allow multiple AI-model entries per provider".
func TestCreateEntry_MultiplePerProvider(t *testing.T) {
	st := newAIModelTestStore(t)
	id1, err := st.AIModel().CreateEntry("u1", "deepseek", "DeepSeek-main", true, "KEY1", "", "")
	if err != nil {
		t.Fatalf("CreateEntry 1: %v", err)
	}
	id2, err := st.AIModel().CreateEntry("u1", "deepseek", "DeepSeek-backup", true, "KEY2", "", "")
	if err != nil {
		t.Fatalf("CreateEntry 2: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("ids must be unique, both = %s", id1)
	}
	models, _ := st.AIModel().List("u1")
	deep := 0
	for _, m := range models {
		if m.Provider == "deepseek" {
			deep++
		}
	}
	if deep != 2 {
		t.Fatalf("want 2 deepseek rows, got %d", deep)
	}
}

// Editing by EXACT id must touch only that row — a sibling same-provider entry is
// left byte-untouched (no provider-collapse).
func TestUpdate_ExactID_Isolated(t *testing.T) {
	st := newAIModelTestStore(t)
	id1, _ := st.AIModel().CreateEntry("u1", "deepseek", "main", true, "K1", "", "")
	id2, _ := st.AIModel().CreateEntry("u1", "deepseek", "backup", true, "K2", "", "")

	if err := st.AIModel().UpdateWithName("u1", id1, "", false, "K1B", "", ""); err != nil {
		t.Fatalf("update id1: %v", err)
	}
	m1, _ := st.AIModel().GetByID(id1)
	m2, _ := st.AIModel().GetByID(id2)
	if m1.Enabled {
		t.Fatal("id1 should be disabled after update")
	}
	if string(m1.APIKey) != "K1B" {
		t.Fatalf("id1 key = %q, want K1B", string(m1.APIKey))
	}
	if !m2.Enabled || string(m2.APIKey) != "K2" {
		t.Fatalf("sibling id2 must be untouched (enabled=%v key=%q)", m2.Enabled, string(m2.APIKey))
	}
}

// A bare-provider "deepseek" save (legacy path) must keep updating the SAME single
// legacy row — never spawn a second one (backward compatibility).
func TestUpdate_LegacyBareProvider_Compat(t *testing.T) {
	st := newAIModelTestStore(t)
	if err := st.AIModel().UpdateWithName("u1", "deepseek", "DeepSeek AI", true, "K1", "", ""); err != nil {
		t.Fatalf("legacy create: %v", err)
	}
	if err := st.AIModel().UpdateWithName("u1", "deepseek", "", true, "K2", "", ""); err != nil {
		t.Fatalf("legacy update: %v", err)
	}
	models, _ := st.AIModel().List("u1")
	if len(models) != 1 {
		t.Fatalf("bare-provider save must not create a 2nd row; got %d", len(models))
	}
	if string(models[0].APIKey) != "K2" {
		t.Fatalf("legacy row key = %q, want K2", string(models[0].APIKey))
	}
	if models[0].ID != "u1_deepseek" {
		t.Fatalf("legacy id = %q, want u1_deepseek", models[0].ID)
	}
}

// Deleting one entry must leave the sibling alive; delete is by exact id.
func TestDelete_OneSurvives(t *testing.T) {
	st := newAIModelTestStore(t)
	id1, _ := st.AIModel().CreateEntry("u1", "deepseek", "main", true, "K1", "", "")
	id2, _ := st.AIModel().CreateEntry("u1", "deepseek", "backup", true, "K2", "", "")

	if err := st.AIModel().Delete("u1", id2); err != nil {
		t.Fatalf("delete id2: %v", err)
	}
	if _, err := st.AIModel().GetByID(id2); err == nil {
		t.Fatal("id2 should be gone")
	}
	if _, err := st.AIModel().GetByID(id1); err != nil {
		t.Fatalf("id1 must survive: %v", err)
	}
}

// PickProviderModel is the single deterministic rule every by-provider lookup site
// uses (#4): enabled beats disabled, then most-recently-updated, then lowest id.
func TestPickProviderModel_Determinism(t *testing.T) {
	old := time.Now().Add(-time.Hour)
	newer := time.Now()
	a := &AIModel{ID: "z_deepseek_a", Provider: "deepseek", Enabled: true, UpdatedAt: old}
	b := &AIModel{ID: "z_deepseek_b", Provider: "deepseek", Enabled: true, UpdatedAt: newer}
	c := &AIModel{ID: "z_deepseek_c", Provider: "deepseek", Enabled: false, UpdatedAt: newer}

	chosen, n := PickProviderModel([]*AIModel{a, c, b}, "deepseek")
	if n != 3 {
		t.Fatalf("candidates = %d, want 3", n)
	}
	if chosen.ID != "z_deepseek_b" {
		t.Fatalf("chose %s, want z_deepseek_b (enabled + newest)", chosen.ID)
	}

	// All disabled → tie-break on lowest id.
	lo := &AIModel{ID: "a", Provider: "deepseek", Enabled: false, UpdatedAt: newer}
	hi := &AIModel{ID: "m", Provider: "deepseek", Enabled: false, UpdatedAt: newer}
	chosen2, _ := PickProviderModel([]*AIModel{hi, lo}, "deepseek")
	if chosen2.ID != "a" {
		t.Fatalf("tie-break chose %s, want a (lowest id)", chosen2.ID)
	}

	// No match → nil, 0.
	if got, n0 := PickProviderModel([]*AIModel{a}, "qwen"); got != nil || n0 != 0 {
		t.Fatalf("qwen expected (nil,0), got (%v,%d)", got, n0)
	}
}
