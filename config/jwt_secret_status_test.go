package config

// P2-13 (audit 0926-system): the boot line printed "🔑 JWT secret configured"
// unconditionally — even under the insecure default. JWTSecretIsDefault is the
// READ-state helper main.go now uses; these pins hold it honest.

import "testing"

func TestJWTSecretIsDefault(t *testing.T) {
	if !(&Config{}).JWTSecretIsDefault() {
		t.Fatal("empty secret is the insecure default")
	}
	if !(&Config{JWTSecret: "  "}).JWTSecretIsDefault() {
		t.Fatal("whitespace secret is the insecure default")
	}
	if !(&Config{JWTSecret: InsecureDefaultJWTSecret}).JWTSecretIsDefault() {
		t.Fatal("the literal default must read as default")
	}
	if (&Config{JWTSecret: "a-custom-secret"}).JWTSecretIsDefault() {
		t.Fatal("a custom secret must NOT read as default")
	}
	var nilCfg *Config
	if !nilCfg.JWTSecretIsDefault() {
		t.Fatal("a nil config must fail closed as default")
	}
}
