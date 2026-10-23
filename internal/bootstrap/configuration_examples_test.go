package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestExplicitAccountConfigurationExampleLoadsAndApplyRemainsUnavailable(t *testing.T) {
	data, err := os.ReadFile("../../examples/run.example.json")
	if err != nil {
		t.Fatal(err)
	}
	dir := testkit.TempDir(t)
	state := filepath.Join(dir, "state")
	replacements := map[string]string{"PRODUCT_BUILD": "source-test-build", "STATE_DIRECTORY": state, "ENGINE_EXECUTABLE": filepath.Join(dir, "never-run-engine"), "DEADLINE_UTC": "2026-10-23T23:00:00Z", "ACCOUNT_LOGIN_PATH": filepath.Join(dir, "never-read-account-login.json")}
	text := string(data)
	for token, value := range replacements {
		encoded, _ := json.Marshal(value)
		text = strings.ReplaceAll(text, `"`+token+`"`, string(encoded))
	}
	config := filepath.Join(dir, "run.json")
	if err = os.WriteFile(config, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadRunConfig(context.Background(), config)
	if err != nil || cfg.FormatVersion != 3 || cfg.Credentials["example-account"].Mode != "dedicated-runtime-login" {
		t.Fatal(cfg, err)
	}
	var out, stderr bytes.Buffer
	calls := 0
	code := run(context.Background(), []string{"apply", "--repo", dir, "--runtime-profile", filepath.Join(dir, "never-read-runtime.yaml"), "--run-config", config}, &out, &stderr, func(codexappserver.EngineConfig) (codexappserver.ImageReader, error) {
		calls++
		t.Fatal("unavailable Apply acquired runtime metadata")
		return nil, nil
	})
	var report map[string]any
	if err = json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if code != 3 || report["status"] != "not_completed" || report["outcome"] != "incomplete" || calls != 0 {
		t.Fatalf("code=%d report=%v stderr=%s", code, report, stderr.String())
	}
	if _, err = os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("unavailable Apply created state", err)
	}
}
