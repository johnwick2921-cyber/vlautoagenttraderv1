// Command decisive-test re-runs stored wait cycles through the live model with
// the DAY PLAN block and PLAN STATUS tail stripped from the system prompt.
// TEMPORARY diagnostic harness — not part of the product.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/joho/godotenv"

	"nofx/config"
	"nofx/crypto"
	"nofx/store"
)

func main() {
	ids := os.Args[1:]
	if len(ids) == 0 {
		ids = []string{"29222", "29218", "29215"}
	}
	_ = godotenv.Load()

	config.Init()
	cs, err := crypto.NewCryptoService()
	if err != nil {
		fmt.Fprintln(os.Stderr, "crypto:", err)
		os.Exit(1)
	}
	crypto.SetGlobalCryptoService(cs)

	gdb, err := gorm.Open(sqlite.Open("file:data/data.db?mode=ro"), &gorm.Config{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	st, err := store.NewFromGorm(gdb)
	if err != nil {
		fmt.Fprintln(os.Stderr, "store:", err)
		os.Exit(1)
	}

	// hoang trader's model (the one producing all the waits).
	modelID := "8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek"
	m, err := st.AIModel().GetByID(modelID)
	if err != nil || m == nil {
		fmt.Fprintln(os.Stderr, "model:", err)
		os.Exit(1)
	}
	apiKey := string(m.APIKey)
	fmt.Printf("model=%s keylen=%d\n", m.Provider, len(apiKey))

	type row struct {
		ID     int64  `gorm:"column:id"`
		Sp     string `gorm:"column:system_prompt"`
		Up     string `gorm:"column:input_prompt"`
		Dec    string `gorm:"column:decision_json"`
		Action string `gorm:"column:act"`
	}
	var rows []row
	q := gdb.Raw(`SELECT id, system_prompt, input_prompt, decision_json,
	  (SELECT group_concat(json_extract(value,'$.action')) FROM json_each(decision_json)) AS act
	  FROM decision_records WHERE id IN ? ORDER BY id DESC`, ids).Scan(&rows)
	if q.Error != nil {
		fmt.Fprintln(os.Stderr, "query:", q.Error)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 300 * time.Second}
	for _, r := range rows {
		stripped := stripPlan(r.Sp)
		acts := extractActions(r.Dec)
		fmt.Printf("\n===== cycle %d | stored action(s): %v | syslen %d -> %d =====\n",
			r.ID, acts, len(r.Sp), len(stripped))
		resp := call(client, apiKey, stripped, r.Up)
		fmt.Printf("STRIPPED RESPONSE:\n%s\n", first2000(resp))
	}
}

func stripPlan(sys string) string {
	lines := strings.Split(sys, "\n")
	var out []string
	inDayPlan, inStatus := false, false
	for _, ln := range lines {
		trim := strings.TrimSpace(ln)
		if strings.HasPrefix(trim, "# DAY PLAN") {
			inDayPlan = true
			continue
		}
		if inDayPlan {
			if strings.HasPrefix(trim, "# ") && !strings.HasPrefix(trim, "# DAY PLAN") {
				inDayPlan = false
			} else {
				continue
			}
		}
		if strings.HasPrefix(trim, "# PLAN STATUS") {
			inStatus = true
			continue
		}
		if inStatus {
			if strings.HasPrefix(trim, "# ") && !strings.HasPrefix(trim, "# PLAN STATUS") {
				inStatus = false
			} else {
				continue
			}
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

func extractActions(decisionJSON string) []string {
	var ds []map[string]any
	_ = json.Unmarshal([]byte(decisionJSON), &ds)
	var acts []string
	for _, d := range ds {
		if a, ok := d["action"].(string); ok {
			acts = append(acts, a)
		}
	}
	return acts
}

func call(client *http.Client, key, sys, usr string) string {
	body := map[string]any{
		"model": "deepseek-v4-pro",
		"messages": []map[string]string{
			{"role": "system", "content": sys},
			{"role": "user", "content": usr},
		},
		"temperature": 0.5,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		return "HTTP ERROR: " + err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, first2000(string(raw)))
	}
	return string(raw)
}

func first2000(s string) string {
	if len(s) > 2000 {
		return s[:2000] + "..."
	}
	return s
}
