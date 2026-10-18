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
		"formatVersion": 1, "repositoryIdentity": "source-test-repository", "productBuild": "explicit-frozen-source-test-build", "ownerRequest": "Apply the current model.",
		"stateDirectory": filepath.Join(dir, "state"), "engineExecutable": filepath.Join(dir, "engine-not-read-during-parsing"),
		"allocation":      map[string]any{"runs": 10, "sessionStarts": 1000, "steps": 2000, "executionTime": int64(time.Hour)},
		"restoreLimits":   map[string]any{"pageSize": 256, "batches": 100000, "events": 1000000, "bytes": 256 << 20},
		"workspaceHelper": map[string]any{"name": "source-test-helper", "image": "sha256:" + strings.Repeat("a", 64), "labels": map[string]any{"markitect.store": "explicit-source-test-store"}, "user": "1000:1000", "cpus": 1.5, "memoryBytes": 128 << 20, "pids": 32, "tmpBytes": 16 << 20, "network": ""},
		"engineLimits": map[string]any{"startTimeout": int64(time.Second), "stopTimeout": int64(time.Second), "harvestEntries": 100000, "harvestBytes": 64 << 20,
			"record": map[string]any{"encodedBytes": 16 << 20, "fileBytes": 16 << 20, "manifestEntries": 100000, "metadataRows": 100000}},
		"credentials": map[string]any{"dedicated": map[string]any{"provider": "codex-app-server", "mode": "api-key", "path": filepath.Join(dir, "dedicated-credential-not-read")}},
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
	if err != nil || cfg.ProductBuild != "explicit-frozen-source-test-build" || cfg.Allocation.ExecutionTime != time.Hour || cfg.WorkspaceHelper.Network != "" || cfg.WorkspaceHelper.Mounts != nil {
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

func TestRunConfigRefusesImplicitOrUnknownValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"missing product assertion", func(m map[string]any) { delete(m, "productBuild") }},
		{"empty product assertion", func(m map[string]any) { m["productBuild"] = "" }},
		{"unknown version", func(m map[string]any) { m["formatVersion"] = 2 }},
		{"unknown field", func(m map[string]any) { m["fake-secret-must-not-leak"] = "secret" }},
		{"relative state", func(m map[string]any) { m["stateDirectory"] = "state" }},
		{"implicit network", func(m map[string]any) { delete(m["workspaceHelper"].(map[string]any), "network") }},
		{"null network", func(m map[string]any) { m["workspaceHelper"].(map[string]any)["network"] = nil }},
		{"mount injection", func(m map[string]any) { m["workspaceHelper"].(map[string]any)["mounts"] = []any{} }},
		{"unbounded time", func(m map[string]any) { m["allocation"].(map[string]any)["executionTime"] = 0 }},
		{"duration text", func(m map[string]any) { m["engineLimits"].(map[string]any)["startTimeout"] = "1s" }},
		{"missing record limit", func(m map[string]any) {
			delete(m["engineLimits"].(map[string]any)["record"].(map[string]any), "metadataRows")
		}},
		{"unknown record limit", func(m map[string]any) { m["engineLimits"].(map[string]any)["record"].(map[string]any)["default"] = 1 }},
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
		strings.Replace(base, `"formatVersion":1`, `"formatVersion":1,"formatVersion":1`, 1),
		strings.Replace(base, `"startTimeout":1000000000`, `"startTimeout":1000000000,"startTimeout":1`, 1),
		strings.Replace(base, `"mode":"api-key"`, `"mode":"api-key","mode":"api-key"`, 1),
		strings.Replace(base, `"runs":10`, `"runs":9223372036854775808`, 1),
		base + `{}`, string([]byte{0xff}),
	} {
		if _, err := decodeRunConfig([]byte(malformed)); !errors.Is(err, errRunConfig) {
			t.Fatal("duplicate, overflowing or malformed JSON was admitted")
		}
	}
}
