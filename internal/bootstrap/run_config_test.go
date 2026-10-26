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
		"formatVersion": 4, "repositoryIdentity": "source-test-repository", "productBuild": "explicit-frozen-source-test-build",
		"stateDirectory": filepath.Join(dir, "state"), "engineExecutable": filepath.Join(dir, "engine-not-read-during-parsing"),
		"deadline": "2026-10-23T23:00:00Z", "maxManagers": 64, "maxMessages": 128, "providerIdleTimeoutSeconds": 120, "cleanupTimeoutSeconds": 30, "persistenceTimeoutSeconds": 30,
		"credentials": map[string]any{"dedicated": map[string]any{"provider": "codex-app-server", "mode": "dedicated-runtime-login", "path": filepath.Join(dir, "dedicated-credential-not-read")}},
	}
}

func TestAssignmentInvocationBoundsAreClosedAndFinite(t *testing.T) {
	document := runConfigFixture(t)
	document["maxMessages"], document["providerIdleTimeoutSeconds"] = 80, 120
	if cfg, err := decodeRunConfig(configBytes(t, document)); err != nil || cfg.MaxMessages != 80 || cfg.ProviderIdleTimeoutSeconds != 120 {
		t.Fatalf("assignment bounds were not retained: %+v %v", cfg, err)
	}
	for _, value := range []any{0, -1, 4097} {
		document["maxMessages"] = value
		if _, err := decodeRunConfig(configBytes(t, document)); err == nil {
			t.Fatalf("admitted maxMessages %v", value)
		}
	}
	document["maxMessages"] = 80
	for _, value := range []any{0, -1, 3601} {
		document["providerIdleTimeoutSeconds"] = value
		if _, err := decodeRunConfig(configBytes(t, document)); err == nil {
			t.Fatalf("admitted providerIdleTimeoutSeconds %v", value)
		}
	}
}

// Format 3 bounded all provider work of one response, format 4 bounds provider
// work without a served tool call. A format 3 file is refused by name, and no
// format reads the old limit with the new meaning.
func TestRunConfigRefusesFormat3InsteadOfChangingItsMeaning(t *testing.T) {
	old := runConfigFixture(t)
	delete(old, "providerIdleTimeoutSeconds")
	old["formatVersion"], old["messageTimeoutSeconds"] = 3, 600
	renamed := runConfigFixture(t)
	renamed["formatVersion"] = 3
	for name, document := range map[string]map[string]any{"format 3": old, "format 3 with the new field": renamed} {
		_, err := decodeRunConfig(configBytes(t, document))
		if !errors.Is(err, errRunConfigFormat3) || !errors.Is(err, errRunConfig) || !strings.Contains(err.Error(), "formatVersion 3 is no longer accepted") || !strings.Contains(err.Error(), "providerIdleTimeoutSeconds") {
			t.Fatalf("%s was not refused by name: %v", name, err)
		}
	}
	old["formatVersion"] = 4
	if _, err := decodeRunConfig(configBytes(t, old)); !errors.Is(err, errRunConfig) || errors.Is(err, errRunConfigFormat3) {
		t.Fatalf("format 4 read the old message limit: %v", err)
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
	if err != nil || cfg.ProductBuild != "explicit-frozen-source-test-build" || cfg.MaxManagers != 64 || cfg.MaxMessages != 128 {
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

func TestRunConfigDeadlineIsRequiredAbsoluteUTC(t *testing.T) {
	document := runConfigFixture(t)
	delete(document, "deadline")
	if _, err := decodeRunConfig(configBytes(t, document)); err == nil {
		t.Fatal("unbounded invocation admitted")
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
		{"legacy workspace helper", func(m map[string]any) { m["workspaceHelper"] = map[string]any{} }},
		{"legacy harvest", func(m map[string]any) { m["harvest"] = map[string]any{} }},
		{"unmodeled owner prompt", func(m map[string]any) { m["ownerRequest"] = "Use euro." }},
		{"missing manager bound", func(m map[string]any) { delete(m, "maxManagers") }},
		{"too few messages", func(m map[string]any) { m["maxMessages"] = 1 }},
		{"unbounded cleanup", func(m map[string]any) { m["cleanupTimeoutSeconds"] = 0 }},
		{"unbounded persistence", func(m map[string]any) { m["persistenceTimeoutSeconds"] = 61 }},
		{"invalid credential provider", func(m map[string]any) {
			m["credentials"].(map[string]any)["dedicated"].(map[string]any)["provider"] = "ambient"
		}},
		{"retired owner decisions", func(m map[string]any) { m["ownerDecisions"] = []any{"Use euro."} }},
		{"retired empty owner decisions", func(m map[string]any) { m["ownerDecisions"] = []any{} }},
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
		strings.Replace(base, `"formatVersion":4`, `"formatVersion":4,"formatVersion":4`, 1),
		strings.Replace(base, `"maxManagers":64`, `"maxManagers":64,"maxManagers":1`, 1),
		strings.Replace(base, `"mode":"dedicated-runtime-login"`, `"mode":"dedicated-runtime-login","mode":"dedicated-runtime-login"`, 1),
		strings.Replace(base, `"maxManagers":64`, `"maxManagers":9223372036854775808`, 1),
		base + `{}`, string([]byte{0xff}),
	} {
		if _, err := decodeRunConfig([]byte(malformed)); !errors.Is(err, errRunConfig) {
			t.Fatal("duplicate, overflowing or malformed JSON was admitted")
		}
	}
}

func TestRunConfigWorkDeadlineReservesTimeInsideTheWholeRun(t *testing.T) {
	d := runConfigFixture(t)
	d["deadline"] = "2026-10-21T22:42:00Z"
	d["workDeadline"] = "2026-10-21T22:30:00Z"
	cfg, err := decodeRunConfig(configBytes(t, d))
	if err != nil || cfg.deadline().Sub(cfg.workDeadline()) != 12*time.Minute {
		t.Fatalf("reserve lost: %+v, %v", cfg, err)
	}
	for _, bad := range []any{"", nil, "2026-10-21T22:42:00Z", "2026-10-21T22:43:00Z", "2026-10-21T22:30:00+00:00"} {
		d["workDeadline"] = bad
		if _, err := decodeRunConfig(configBytes(t, d)); !errors.Is(err, errRunConfig) {
			t.Errorf("accepted cutoff %v", bad)
		}
	}
	d["workDeadline"] = "2026-10-21T22:30:00Z"
	delete(d, "deadline")
	if _, err := decodeRunConfig(configBytes(t, d)); !errors.Is(err, errRunConfig) {
		t.Fatal("work deadline without whole-run bound accepted")
	}
}
