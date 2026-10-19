package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func runConfigFixture(t *testing.T) map[string]any {
	t.Helper()
	dir := testkit.TempDir(t)
	return map[string]any{
		"formatVersion": 2, "repositoryIdentity": "source-test-repository", "productBuild": "explicit-frozen-source-test-build", "ownerRequest": "Apply the current model.",
		"stateDirectory": filepath.Join(dir, "state"), "engineExecutable": filepath.Join(dir, "engine-not-read-during-parsing"),
		"harvest":         map[string]any{"maxEntries": 100000, "maxBytes": 64 << 20},
		"workspaceHelper": map[string]any{"name": "source-test-helper", "image": "sha256:" + strings.Repeat("a", 64), "labels": map[string]any{"markitect.store": "explicit-source-test-store"}, "user": "1000:1000", "cpus": 1.5, "memoryBytes": 128 << 20, "pids": 32, "tmpBytes": 16 << 20, "network": ""},
		"credentials":     map[string]any{"dedicated": map[string]any{"provider": "codex-app-server", "mode": "api-key", "path": filepath.Join(dir, "dedicated-credential-not-read")}},
	}
}

func configBytes(t *testing.T, document map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRunConfigIsExplicitAndDoesNotReadReferencedCredentials(t *testing.T) {
	document := runConfigFixture(t)
	cfg, err := decodeRunConfig(configBytes(t, document))
	if err != nil || cfg.ProductBuild != "explicit-frozen-source-test-build" || cfg.Harvest != (harvestLimits{MaxEntries: 100000, MaxBytes: 64 << 20}) || cfg.WorkspaceHelper.Network != "" || cfg.WorkspaceHelper.Mounts != nil {
		t.Fatalf("explicit producer values were not retained: %v", err)
	}
	// None of the referenced engine, state or credential paths exists. Only
	// loading the explicitly selected JSON file is permitted at this stage.
	path := filepath.Join(testkit.TempDir(t), "run.json")
	if err := os.WriteFile(path, configBytes(t, document), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRunConfig(context.Background(), path); err != nil {
		t.Fatal("parsing tried to acquire a referenced runtime input", err)
	}
	if _, err := loadRunConfig(context.Background(), filepath.Join(testkit.TempDir(t), "missing.json")); !errors.Is(err, errRunConfigIO) {
		t.Fatal("missing configuration did not retain an I/O refusal")
	}
}

// Owner decisions are optional; when present they are retained in order
// (DEC-053).
func TestRunConfigRetainsOptionalOwnerDecisions(t *testing.T) {
	document := runConfigFixture(t)
	if cfg, err := decodeRunConfig(configBytes(t, document)); err != nil || cfg.OwnerDecisions != nil {
		t.Fatal("a run config without owner decisions changed", err, cfg.OwnerDecisions)
	}
	document["ownerDecisions"] = []any{"Unknown argument fields return invalidRequest.", "A due date after 9999-12-31 refuses the borrow."}
	cfg, err := decodeRunConfig(configBytes(t, document))
	if err != nil || len(cfg.OwnerDecisions) != 2 || cfg.OwnerDecisions[1] != "A due date after 9999-12-31 refuses the borrow." {
		t.Fatal("the owner decisions were not retained", err, cfg.OwnerDecisions)
	}
}

func TestRunConfigDeadlineIsOptionalAbsoluteUTC(t *testing.T) {
	document := runConfigFixture(t)
	if cfg, err := decodeRunConfig(configBytes(t, document)); err != nil || !cfg.deadline().IsZero() {
		t.Fatal("a run config without a deadline changed", err)
	}
	document["deadline"] = "2026-10-19T18:30:00Z"
	cfg, err := decodeRunConfig(configBytes(t, document))
	if err != nil || !cfg.deadline().Equal(time.Date(2026, 10, 19, 18, 30, 0, 0, time.UTC)) {
		t.Fatal("the deadline was not retained", err)
	}
	for _, bad := range []any{"", "tomorrow", "2026-10-19", "2026-10-19T18:30:00", "2026-10-19T20:30:00+02:00", "2026-10-19T18:30:00+00:00", "0001-01-01T00:00:00Z", 1792434600} {
		document["deadline"] = bad
		if _, err := decodeRunConfig(configBytes(t, document)); !errors.Is(err, errRunConfig) {
			t.Errorf("deadline %v accepted", bad)
		}
	}
	document["deadline"] = nil
	if _, err := decodeRunConfig(configBytes(t, document)); !errors.Is(err, errRunConfig) {
		t.Error("null deadline accepted")
	}
}

func TestRunConfigRefusesImplicitOrUnknownValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"missing product assertion", func(m map[string]any) { delete(m, "productBuild") }},
		{"empty product assertion", func(m map[string]any) { m["productBuild"] = "" }},
		{"old version", func(m map[string]any) { m["formatVersion"] = 1 }},
		{"old allocation section", func(m map[string]any) { m["allocation"] = map[string]any{"runs": 1} }},
		{"old restore limits", func(m map[string]any) { m["restoreLimits"] = map[string]any{"pageSize": 1} }},
		{"old engine limits", func(m map[string]any) { m["engineLimits"] = map[string]any{"startTimeout": 1} }},
		{"unknown field", func(m map[string]any) { m["fake-secret-must-not-leak"] = "secret" }},
		{"relative state", func(m map[string]any) { m["stateDirectory"] = "state" }},
		{"implicit network", func(m map[string]any) { delete(m["workspaceHelper"].(map[string]any), "network") }},
		{"null network", func(m map[string]any) { m["workspaceHelper"].(map[string]any)["network"] = nil }},
		{"mount injection", func(m map[string]any) { m["workspaceHelper"].(map[string]any)["mounts"] = []any{} }},
		{"missing harvest", func(m map[string]any) { delete(m, "harvest") }},
		{"missing harvest bound", func(m map[string]any) { delete(m["harvest"].(map[string]any), "maxBytes") }},
		{"unknown harvest field", func(m map[string]any) { m["harvest"].(map[string]any)["default"] = 1 }},
		{"zero harvest entries", func(m map[string]any) { m["harvest"].(map[string]any)["maxEntries"] = 0 }},
		{"unbounded harvest bytes", func(m map[string]any) { m["harvest"].(map[string]any)["maxBytes"] = int64(1) << 40 }},
		{"text harvest bound", func(m map[string]any) { m["harvest"].(map[string]any)["maxBytes"] = "64MiB" }},
		{"invalid credential provider", func(m map[string]any) {
			m["credentials"].(map[string]any)["dedicated"].(map[string]any)["provider"] = "ambient"
		}},
		{"blank owner decision", func(m map[string]any) { m["ownerDecisions"] = []any{" "} }},
		{"null owner decisions", func(m map[string]any) { m["ownerDecisions"] = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document := runConfigFixture(t)
			tc.change(document)
			if _, err := decodeRunConfig(configBytes(t, document)); !errors.Is(err, errRunConfig) || strings.Contains(err.Error(), "fake-secret") {
				t.Fatalf("invalid config was accepted or exposed unknown input: %v", err)
			}
		})
	}
	base := string(configBytes(t, runConfigFixture(t)))
	for _, malformed := range []string{
		strings.Replace(base, `"formatVersion":2`, `"formatVersion":2,"formatVersion":2`, 1),
		strings.Replace(base, `"maxEntries":100000`, `"maxEntries":100000,"maxEntries":1`, 1),
		strings.Replace(base, `"mode":"api-key"`, `"mode":"api-key","mode":"api-key"`, 1),
		strings.Replace(base, `"maxEntries":100000`, `"maxEntries":9223372036854775808`, 1),
		base + `{}`, string([]byte{0xff}),
	} {
		if _, err := decodeRunConfig([]byte(malformed)); !errors.Is(err, errRunConfig) {
			t.Fatal("duplicate, overflowing or malformed JSON was admitted")
		}
	}
}
