package modulecli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

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

// This uses the real v1 Model check, repository capture and complete accounting.
// A changed hook remains structurally accounted but fails its explicit byte pin.
func TestRunChecksActualV1RegistryAndHistoricalCensus(t *testing.T) {
	repo := testkit.NewRepo(t)
	err := filepath.WalkDir("../../../.markitect/model", func(name string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			return nil
		}
		relative, err := filepath.Rel("../../../.markitect/model", name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		repo.Write(".markitect/model/"+filepath.ToSlash(relative), string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"model.yaml", "model-lock.yaml", "execution.yaml"} {
		data, err := os.ReadFile("../../../.markitect/" + name)
		if err != nil {
			t.Fatal(err)
		}
		repo.Write(".markitect/"+name, string(data))
	}
	const configPath = ".markitect/modules/hooks.config"
	const hookPath = ".githooks/pre-commit"
	hook := []byte("#!/bin/sh\nexit 0\n")
	sum := sha256.Sum256(hook)
	config := fmt.Sprintf("apiVersion: markitect.example.org/git-hooks-check/v1alpha1\nhooks:\n  - name: fixture\n    stage: pre-commit\n    path: %s\n    digest: %s\n    owner: tooling/gates\n", hookPath, hex.EncodeToString(sum[:]))
	repo.Write(configPath, string(config))
	repo.Write(hookPath, string(hook))
	repo.Write(".markitect/registry/tooling/gates/registry.yaml", "apiVersion: markitect.registry/v1\nartifacts:\n  - path: "+configPath+"\n    form: config\n    realizes: [tooling/gates/source-checks]\n    why: Exact configured module input.\n  - path: "+hookPath+"\n    form: build\n    realizes: [tooling/gates/source-checks]\n    why: The hook entrypoint under test.\n")
	revision := repo.Commit("v1 module fixture")
	run := func(args []string, want int, fragment string) {
		t.Helper()
		var out, errout bytes.Buffer
		code := Run(args, &out, &errout)
		if code != want || !strings.Contains(out.String()+errout.String(), fragment) {
			t.Fatalf("code=%d want=%d stdout=%s stderr=%s", code, want, out.String(), errout.String())
		}
	}
	base := []string{"--repo", repo.Dir, "--hooks", configPath}
	run(base, 0, "status: passed")
	repo.Write(hookPath, "changed hook\n")
	run(base, 1, "hooks.digest.mismatch")
	run(append(append([]string{}, base...), "--revision", revision), 0, "status: passed")
	run([]string{"--repo", repo.Dir, "--hooks", ".markitect/modules/missing.config"}, 2, "not a registered artifact")
	repo.Write("uncovered.txt", "not silently exempt\n")
	run(base, 2, "require passed structural preparation")
}
