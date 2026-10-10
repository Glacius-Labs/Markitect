package modulecli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/consumers/githooks"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

func TestRunRequiresExplicitModuleConfigAndRejectsUnknownArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"--unknown"}, {"--hooks", "hooks.yaml", "unexpected"}} {
		var out, errout bytes.Buffer
		if code := Run(args, &out, &errout); code != 2 || out.Len() != 0 || errout.Len() == 0 {
			t.Fatalf("Run(%v): code=%d stdout=%q stderr=%q", args, code, out.String(), errout.String())
		}
	}
	var out, errout bytes.Buffer
	if code := Run([]string{"--help"}, &out, &errout); code != 0 || !strings.Contains(errout.String(), "-hooks") || !strings.Contains(errout.String(), "-pipelines") {
		t.Fatalf("help: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
}

func TestRunReportsPassFindingsAndInvalidInputExitCodes(t *testing.T) {
	root := t.TempDir()
	configPath := "docs/module-config/hooks.config"
	hookPath := ".githooks/pre-commit"
	hookBytes := []byte("#!/bin/sh\nexit 0\n")
	sum := sha256.Sum256(hookBytes)
	configBytes, err := yaml.Marshal(githooks.Config{APIVersion: githooks.ConfigVersion, Hooks: []githooks.Hook{{Name: "pre-commit-tests", Stage: "pre-commit", Path: hookPath, Digest: hex.EncodeToString(sum[:]), Owner: "docs/Rule/repository-checks"}}})
	if err != nil {
		t.Fatal(err)
	}
	project := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "sample"}}, Spec: authoring.Spec{Areas: []authoring.Area{{Name: "docs", Path: "docs", Imports: []string{"hooks"}}, {Name: "hooks", Path: ".githooks"}}}}
	rule := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "repository-checks", Namespace: "docs"}}, Spec: authoring.Spec{Text: "Keep repository checks explicit.", Files: []string{configPath, hookPath}}}
	projectBytes, err := authoring.Encode(&project)
	if err != nil {
		t.Fatal(err)
	}
	ruleBytes, err := authoring.Encode(&rule)
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("markitect.yaml", projectBytes)
	write("docs/general/rules/checks.yaml", ruleBytes)
	write(configPath, configBytes)
	write(hookPath, hookBytes)

	args := []string{"--repo", root, "--hooks", configPath}
	var out, errout bytes.Buffer
	if code := Run(args, &out, &errout); code != 0 || !strings.Contains(out.String(), "status: passed") || !strings.Contains(out.String(), "status: not-configured") || errout.Len() != 0 {
		t.Fatalf("passing configured module: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	write(hookPath, []byte("changed hook\n"))
	if code := Run(args, &out, &errout); code != 1 || !strings.Contains(out.String(), "status: findings") || !strings.Contains(out.String(), "hooks.digest.mismatch") || errout.Len() != 0 {
		t.Fatalf("stale digest finding: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"--repo", root, "--hooks", "docs/module-config/missing.config"}, &out, &errout); code != 2 || out.Len() != 0 || !strings.Contains(errout.String(), "absent from the selected snapshot") {
		t.Fatalf("invalid module input: code=%d stdout=%q stderr=%q", code, out.String(), errout.String())
	}
}
