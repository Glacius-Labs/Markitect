package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpSucceedsAndShowsOnlyApplicableFlags(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"help"}, {"help", "install"}, {"install", "--help"}, {"version", "--help"}, {"authoring", "--help"}} {
		code, out, errout := invoke(args...)
		if code != 0 || !strings.Contains(out, "usage:") || errout != "" {
			t.Fatalf("help %v: code=%d out=%s err=%s", args, code, out, errout)
		}
	}
	_, out, _ := invoke("install", "--help")
	if !strings.Contains(out, "--sha256") || strings.Contains(out, "--revision") || strings.Contains(out, "--config") {
		t.Fatalf("incorrect install help: %s", out)
	}
	for _, args := range [][]string{{"help", "missing"}, {"install", "--revision", "HEAD"}, {"bundle", "--write"}, {"missing-command"}, {"migrate"}, {"check", "--profile", "generic"}} {
		code, _, _ := invoke(args...)
		if code != 2 {
			t.Fatalf("%v returned %d, want 2", args, code)
		}
	}
}

func TestBundleInstallUsesFixedSourceAndExplicitDigest(t *testing.T) {
	sourceRoot := t.TempDir()
	git(t, sourceRoot, "init", "-b", "feature/source")
	git(t, sourceRoot, "config", "user.name", "Markitect Test")
	git(t, sourceRoot, "config", "user.email", "markitect-test@example.invalid")
	for name, content := range map[string]string{
		"go.mod":                            "module github.com/Glacius-Labs/Markitect\n\ngo 1.27.1\n",
		"go.sum":                            "",
		"LICENSE":                           "Apache License\nVersion 2.0\n",
		"cmd/markitect/main.go":             "package main\nvar version = \"" + version + "\"\nfunc main() {}\n",
		"internal/core/core.go":             "package core\n",
		"integration/run-markitect.go":      "package main\nfunc main() {}\n",
		"integration/run-markitect_test.go": "package main\n",
	} {
		writeRepoFile(t, sourceRoot, name, []byte(content))
	}
	git(t, sourceRoot, "add", ".")
	git(t, sourceRoot, "commit", "-m", "source fixture")
	commit := git(t, sourceRoot, "rev-parse", "HEAD")
	writeRepoFile(t, sourceRoot, "integration/run-markitect.go", []byte("dirty working source"))
	bundlePath := filepath.Join(t.TempDir(), "bundle.zip")
	code, out, errout := invoke("bundle", "--repo", sourceRoot, "--revision", commit, "--output", bundlePath)
	if code != 0 || !strings.Contains(out, commit) {
		t.Fatalf("bundle: %d %s %s", code, out, errout)
	}
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	wantSHA := hex.EncodeToString(digest[:])
	code, _, _ = invoke("bundle", "--repo", sourceRoot, "--revision", commit, "--output", bundlePath)
	if code != 2 {
		t.Fatal("bundle overwrote existing output")
	}
	consumer := newCLIRepo(t, false)
	code, out, errout = invoke("install", "--repo", consumer.root, "--bundle", bundlePath, "--sha256", wantSHA)
	if code != 0 || !strings.Contains(out, "status: planned") {
		t.Fatalf("preview: %d %s %s", code, out, errout)
	}
	if _, err = os.Stat(filepath.Join(consumer.root, ".markitect", "tool", "lock.yaml")); !os.IsNotExist(err) {
		t.Fatal("preview wrote pin")
	}
	code, _, _ = invoke("install", "--repo", consumer.root, "--bundle", bundlePath, "--sha256", strings.Repeat("0", 64), "--write")
	if code != 2 {
		t.Fatal("incorrect bundle hash accepted")
	}
	code, out, errout = invoke("install", "--repo", consumer.root, "--bundle", bundlePath, "--sha256", wantSHA, "--write")
	if code != 0 || !strings.Contains(out, "status: installed") {
		t.Fatalf("install: %d %s %s", code, out, errout)
	}
	runner, err := os.ReadFile(filepath.Join(consumer.root, ".markitect", "bootstrap", "run.go"))
	if err != nil || strings.Contains(string(runner), "dirty") {
		t.Fatalf("fixed bundle runner: %s %v", runner, err)
	}
}
