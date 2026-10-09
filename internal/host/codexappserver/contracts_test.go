package codexappserver

import (
	"path/filepath"
	"testing"
	"time"
)

func TestConfigurationIsExplicitAndBounded(t *testing.T) {
	cfg := Config{Command: filepath.Join(t.TempDir(), "codex.exe"), ProviderVersion: "codex-cli 0.162.0", Model: "gpt-6-luna", ReasoningEffort: "high", Timeout: time.Minute, MaxEventBytes: 1 << 20}
	baseline, err := cfg.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PermissionProfile != "" {
		t.Fatal("profile was invented")
	}
	changed := cfg
	changed.ReasoningEffort = "low"
	other, err := changed.Digest()
	if err != nil || baseline == other {
		t.Fatalf("effort not bound: %v", err)
	}
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Model = "" }, func(c *Config) { c.ReasoningEffort = "" },
		func(c *Config) { c.Command = "codex" }, func(c *Config) { c.Timeout = 0 },
		func(c *Config) { c.MaxEventBytes = 0 }, func(c *Config) { c.Helpers.Enabled = true },
		func(c *Config) { c.Helpers.MaxStartRequests = 1 },
		func(c *Config) { c.Model = string([]byte{0xff}) },
		func(c *Config) { c.PermissionProfile = string([]byte{0xfe}) },
	} {
		invalid := cfg
		mutate(&invalid)
		if invalid.Validate() == nil {
			t.Fatalf("invalid configuration accepted: %+v", invalid)
		}
	}
	cfg.Helpers = HelperPolicy{Enabled: true, MaxStartRequests: 2, MaxDepth: 1}
	if _, err := cfg.Digest(); err != nil {
		t.Fatal(err)
	}
}
