package releasecli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func invokeRelease(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errout bytes.Buffer
	code := Run(args, &out, &errout)
	return code, out.String(), errout.String()
}

func TestLocalReleaseCommandHelpAndArgumentValidation(t *testing.T) {
	for _, command := range []string{"package", "bundle", "install", "version", "licenses"} {
		code, out, stderr := invokeRelease(t, command, "--help")
		if code != 0 || !strings.Contains(stderr, "Usage of markitect-release "+command) || out != "" {
			t.Errorf("%s help: code=%d stdout=%q stderr=%q", command, code, out, stderr)
		}
	}
	for _, args := range [][]string{{"package", "unexpected"}, {"bundle", "--revision", "HEAD", "--unknown"}, {"install", "--repo", ".", "extra"}} {
		if code, _, _ := invokeRelease(t, args...); code != 2 {
			t.Errorf("accepted invalid arguments %v: %d", args, code)
		}
	}
}

func TestPackageBundleAndInstallUseFixedSourceAndExplicitDigest(t *testing.T) {
	sourceRepo := newReleaseSource(t)
	commit := sourceRepo.Commit("source release candidate")
	sourceRepo.Write("packaging/bootstrap/run-markitect.go", "package main\nfunc main() { panic(\"working tree must not be bundled\") }\n")
	bundlePath := filepath.Join(testkit.TempDir(t), "bundle.zip")
	code, out, stderr := invokeRelease(t, "bundle", "--repo", sourceRepo.Dir, "--revision", commit, "--output", bundlePath)
	if code != 0 || stderr != "" || !strings.Contains(out, commit) {
		t.Fatalf("bundle: code=%d stdout=%s stderr=%s", code, out, stderr)
	}
	bundleData, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(bundleData)
	wantSHA := hex.EncodeToString(digest[:])
	if !strings.Contains(out, wantSHA) {
		t.Fatalf("bundle report omitted exact bundle digest %s: %s", wantSHA, out)
	}
	if code, _, _ := invokeRelease(t, "bundle", "--repo", sourceRepo.Dir, "--revision", commit, "--output", bundlePath); code != 2 {
		t.Fatal("bundle overwrote an existing output")
	}
	consumer := testkit.NewRepo(t)
	consumer.Git("branch", "--move", "feature/install")
	consumer.Write("README.md", "consumer\n")
	consumer.Commit("consumer baseline")
	code, out, stderr = invokeRelease(t, "install", "--repo", consumer.Dir, "--bundle", bundlePath, "--sha256", wantSHA)
	if code != 0 || !strings.Contains(out, "status: planned") || stderr != "" {
		t.Fatalf("install preview: code=%d stdout=%s stderr=%s", code, out, stderr)
	}
	if _, err = os.Lstat(filepath.Join(consumer.Dir, ".markitect", "tool", "lock.yaml")); !os.IsNotExist(err) {
		t.Fatalf("preview wrote an install pin: %v", err)
	}
	if code, _, _ = invokeRelease(t, "install", "--repo", consumer.Dir, "--bundle", bundlePath, "--sha256", strings.Repeat("0", 64), "--write"); code != 2 {
		t.Fatal("incorrect bundle digest accepted")
	}
	code, out, stderr = invokeRelease(t, "install", "--repo", consumer.Dir, "--bundle", bundlePath, "--sha256", wantSHA, "--write")
	if code != 0 || !strings.Contains(out, "status: installed") || stderr != "" {
		t.Fatalf("install write: code=%d stdout=%s stderr=%s", code, out, stderr)
	}
	runner, err := os.ReadFile(filepath.Join(consumer.Dir, ".markitect", "bootstrap", "run.go"))
	if err != nil || strings.Contains(string(runner), "working tree must not be bundled") {
		t.Fatalf("bundle did not retain fixed source: %s %v", runner, err)
	}

	packageDir := filepath.Join(testkit.TempDir(t), "package")
	code, out, stderr = invokeRelease(t, "package", "--repo", sourceRepo.Dir, "--output", packageDir)
	if code != 0 || !strings.Contains(out, "status: packaged") || stderr != "" {
		t.Fatalf("package: code=%d stdout=%s stderr=%s", code, out, stderr)
	}
	for _, name := range []string{"source.zip", "lock.yaml"} {
		if _, err := os.Stat(filepath.Join(packageDir, ".markitect", "tool", name)); err != nil {
			t.Errorf("package omitted %s: %v", name, err)
		}
	}
	if code, _, _ = invokeRelease(t, "package", "--repo", sourceRepo.Dir, "--output", packageDir); code != 2 {
		t.Fatal("package overwrote an existing output directory")
	}
}

func newReleaseSource(t *testing.T) *testkit.Repo {
	t.Helper()
	repo := testkit.NewRepo(t)
	repo.Write("go.mod", "module github.com/Glacius-Labs/Markitect\n\ngo 1.27.1\n")
	repo.Write("go.sum", "")
	repo.Write("LICENSE", "Apache License\nVersion 2.0\n")
	repo.Write("internal/buildinfo/version.go", "package buildinfo\nvar version = \""+buildinfo.Version()+"\"\n")
	repo.Write("cmd/markitect/main.go", "package main\nfunc main() {}\n")
	repo.Write("internal/model/core.go", "package core\n")
	repo.Write("packaging/bootstrap/run-markitect.go", "package main\nfunc main() {}\n")
	repo.Write("packaging/bootstrap/run-markitect_test.go", "package main\n")
	return repo
}
