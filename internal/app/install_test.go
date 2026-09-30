package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/release"
	"go.yaml.in/yaml/v3"
)

func TestInstallPlansThenWritesCompleteFreshBundleAndNoops(t *testing.T) {
	root := installTestRepo(t, "feature/install")
	bundle := installTestBundle(t, "a", "0.1.0-rc.4", "new")

	plan, err := Install(root, bundle, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Kind != "install" || plan.Applied || len(plan.Files) != 5 {
		t.Fatalf("fresh plan = %+v", plan)
	}
	for _, file := range plan.Files {
		if file.Action != "create" {
			t.Errorf("fresh action for %s = %q, want create", file.Path, file.Action)
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(file.Path))); !os.IsNotExist(err) {
			t.Fatalf("planning created %s: %v", file.Path, err)
		}
	}

	written, err := Install(root, bundle, true)
	if err != nil {
		t.Fatal(err)
	}
	if written.Kind != "install" || !written.Applied || len(written.Written) != 5 || written.Recovery != "" {
		t.Fatalf("fresh write result = %+v", written)
	}
	commitInstallPins(t, root, "install release")
	noop, err := Install(root, bundle, false)
	if err != nil {
		t.Fatal(err)
	}
	if noop.Kind != "noop" || noop.Applied || len(noop.Written) != 0 {
		t.Fatalf("no-op plan = %+v", noop)
	}
}

func TestInstallUpgradesCommittedManifestPins(t *testing.T) {
	root := installTestRepo(t, "feature/upgrade")
	old := installTestBundle(t, "b", "0.1.0-rc.3", "old")
	writeBundleToRoot(t, root, old, true)
	commitInstallPins(t, root, "legacy upgrade baseline")
	newBundle := installTestBundle(t, "c", "0.1.0-rc.4", "new")

	plan, err := Install(root, newBundle, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Kind != "upgrade" || plan.Applied {
		t.Fatalf("upgrade plan = %+v", plan)
	}
	result, err := Install(root, newBundle, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != "upgrade" || !result.Applied || len(result.Written) != 5 {
		t.Fatalf("upgrade result = %+v", result)
	}
	commitInstallPins(t, root, "upgrade release")
	for name, want := range newBundle.Files {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("installed %s differs from selected bundle", name)
		}
	}
	noop, err := Install(root, newBundle, false)
	if err != nil || noop.Kind != "noop" {
		t.Fatalf("post-commit repeat = (%+v, %v), want noop", noop, err)
	}
}

func TestInstallReadbackNormalizesCoreAutocrlfCheckout(t *testing.T) {
	root := installTestRepo(t, "feature/autocrlf")
	old := installTestBundle(t, "8", "0.1.0-rc.3", "old")
	writeBundleToRoot(t, root, old, true)
	commitInstallPins(t, root, "install RC3 pins")
	runWriterGit(t, root, "config", "core.autocrlf", "true")
	recreateAutocrlfCheckout(t, root)

	plan, err := Install(root, old, false)
	if err != nil || plan.Kind != "noop" {
		t.Fatalf("CRLF checkout no-op plan = (%+v, %v)", plan, err)
	}
	result, err := Install(root, old, true)
	if err != nil || result.Kind != "noop" || !result.Applied || len(result.Written) != 0 {
		t.Fatalf("CRLF checkout no-op write = (%+v, %v)", result, err)
	}

	updated := installTestBundle(t, "9", "0.1.0-rc.4", "updated")
	upgrade, err := Install(root, updated, false)
	if err != nil || upgrade.Kind != "upgrade" {
		t.Fatalf("CRLF checkout upgrade plan = (%+v, %v)", upgrade, err)
	}
	if result, err := Install(root, updated, true); err != nil || result.Kind != "upgrade" || !result.Applied {
		t.Fatalf("CRLF checkout upgrade = (%+v, %v)", result, err)
	}
	commitInstallPins(t, root, "upgrade under core.autocrlf")
	recreateAutocrlfCheckout(t, root)
	noop, err := Install(root, updated, false)
	if err != nil || noop.Kind != "noop" {
		t.Fatalf("post-upgrade CRLF checkout no-op = (%+v, %v)", noop, err)
	}
	archive, err := os.ReadFile(filepath.Join(root, "tools", "markitect", "source.zip"))
	if err != nil || !bytes.Equal(archive, updated.Files["tools/markitect/source.zip"]) {
		t.Fatalf("binary source archive was normalized: %v", err)
	}
}

func TestInstallFreshnessComparesRawCheckoutBytes(t *testing.T) {
	root := installTestRepo(t, "feature/raw-freshness")
	bundle := installTestBundle(t, "a", "0.1.0-rc.3", "old")
	writeBundleToRoot(t, root, bundle, true)
	commitInstallPins(t, root, "install RC3 pins")
	runWriterGit(t, root, "config", "core.autocrlf", "true")
	recreateAutocrlfCheckout(t, root)
	_, state, err := buildInstallPlan(root, bundle)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "scripts", "run-markitect.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lf := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	if bytes.Equal(data, lf) {
		t.Skip("Git did not materialize CRLF on this platform")
	}
	if err := os.WriteFile(path, lf, 0644); err != nil {
		t.Fatal(err)
	}
	if err := ensureInstallStateUnchanged(root, state); err == nil || !strings.Contains(err.Error(), "changed during install") {
		t.Fatalf("raw line-ending edit error = %v, want concurrent-edit refusal", err)
	}
}

func TestInstallUpgradesOnlyCompleteCommittedLegacyRC3(t *testing.T) {
	root := installTestRepo(t, "feature/legacy-upgrade")
	legacy := installTestBundle(t, "d", "0.1.0-rc.3", "legacy")
	writeBundleToRoot(t, root, legacy, false)
	commitInstallPins(t, root, "legacy RC3 pins")
	updated := installTestBundle(t, "e", "0.1.0-rc.4", "updated")

	plan, err := Install(root, updated, false)
	if err != nil {
		t.Fatal(err)
	}
	manifestAction := ""
	for _, file := range plan.Files {
		if file.Path == releaseManifestPath {
			manifestAction = file.Action
		}
	}
	if plan.Kind != "legacy-upgrade" || manifestAction != "create" {
		t.Fatalf("legacy plan = %+v", plan)
	}
	result, err := Install(root, updated, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != "legacy-upgrade" || !result.Applied {
		t.Fatalf("legacy write result = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(releaseManifestPath))); err != nil {
		t.Fatalf("release manifest was not installed: %v", err)
	}
}

func TestInstallRefusesPartialPinSetAndUnmanagedCollision(t *testing.T) {
	t.Run("partial pin", func(t *testing.T) {
		root := installTestRepo(t, "feature/partial")
		writeFixture(t, root, map[string][]byte{"markitect.lock.yaml": []byte("hand-authored\n")})
		if _, err := Install(root, installTestBundle(t, "f", "0.1.0-rc.4", "candidate"), false); err == nil || !strings.Contains(err.Error(), "partial") {
			t.Fatalf("partial pin error = %v", err)
		}
		data, err := os.ReadFile(filepath.Join(root, "markitect.lock.yaml"))
		if err != nil || string(data) != "hand-authored\n" {
			t.Fatalf("partial pin changed: %q, %v", data, err)
		}
	})

	t.Run("ancestor collision", func(t *testing.T) {
		root := installTestRepo(t, "feature/collision")
		if err := os.WriteFile(filepath.Join(root, "scripts"), []byte("unmanaged file"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(root, installTestBundle(t, "1", "0.1.0-rc.4", "candidate"), false); err == nil {
			t.Fatal("unmanaged file at scripts ancestor was accepted")
		}
		data, err := os.ReadFile(filepath.Join(root, "scripts"))
		if err != nil || string(data) != "unmanaged file" {
			t.Fatalf("unmanaged ancestor changed: %q, %v", data, err)
		}
	})
}

func TestInstallRefusesModifiedBootstrapAndChangedPlanInputs(t *testing.T) {
	t.Run("modified committed bootstrap", func(t *testing.T) {
		root := installTestRepo(t, "feature/modified")
		bundle := installTestBundle(t, "2", "0.1.0-rc.3", "old")
		writeBundleToRoot(t, root, bundle, true)
		commitInstallPins(t, root, "install old release")
		bootstrap := filepath.Join(root, "scripts", "run-markitect.go")
		if err := os.WriteFile(bootstrap, []byte("modified bootstrap\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(root, installTestBundle(t, "3", "0.1.0-rc.4", "new"), false); err == nil || !strings.Contains(err.Error(), "unchanged committed") {
			t.Fatalf("modified bootstrap error = %v", err)
		}
	})

	t.Run("destination changes after plan", func(t *testing.T) {
		root := installTestRepo(t, "feature/concurrent-edit")
		bundle := installTestBundle(t, "4", "0.1.0-rc.4", "new")
		plan, err := Install(root, bundle, false)
		if err != nil || plan.Kind != "install" {
			t.Fatalf("initial plan = (%+v, %v)", plan, err)
		}
		writeFixture(t, root, map[string][]byte{"tools/markitect/release.yaml": []byte("unexpected\n")})
		if _, err := Install(root, bundle, true); err == nil {
			t.Fatal("install accepted target changes made after the plan")
		}
		if got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(releaseManifestPath))); err != nil || string(got) != "unexpected\n" {
			t.Fatalf("changed manifest was overwritten: %q, %v", got, err)
		}
	})
}

func TestInstallWriteRequiresNonProtectedBranchAndSharedLock(t *testing.T) {
	t.Run("protected branch", func(t *testing.T) {
		root := installTestRepo(t, "feature/protected")
		runWriterGit(t, root, "checkout", "-b", "main")
		if _, err := Install(root, installTestBundle(t, "5", "0.1.0-rc.4", "new"), true); err == nil || !strings.Contains(err.Error(), "non-protected") {
			t.Fatalf("protected branch error = %v", err)
		}
	})

	t.Run("switch to protected branch at same commit", func(t *testing.T) {
		root := installTestRepo(t, "feature/branch-race")
		bundle := installTestBundle(t, "5", "0.1.0-rc.4", "new")
		_, state, err := buildInstallPlan(root, bundle)
		if err != nil {
			t.Fatal(err)
		}
		state.branch = "feature/branch-race"
		before := state.head.Revision
		runWriterGit(t, root, "branch", "main")
		runWriterGit(t, root, "checkout", "main")
		after := runWriterGit(t, root, "rev-parse", "HEAD")
		if after != before {
			t.Fatalf("test branch switch changed commit: before %s, after %s", before, after)
		}
		if err := ensureInstallStateUnchanged(root, state); err == nil || !strings.Contains(strings.ToLower(err.Error()), "branch") {
			t.Fatalf("same-SHA protected branch switch error = %v", err)
		}
	})

	t.Run("existing shared writer lock", func(t *testing.T) {
		root := installTestRepo(t, "feature/locked")
		lock := filepath.Join(root, ".artifacts", "markitect", "write.lock")
		if err := os.MkdirAll(filepath.Dir(lock), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(lock, []byte("held\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(root, installTestBundle(t, "6", "0.1.0-rc.4", "new"), true); err == nil || !strings.Contains(err.Error(), "renderer lock is already present") {
			t.Fatalf("lock error = %v", err)
		}
	})
}

func recreateAutocrlfCheckout(t *testing.T, root string) {
	t.Helper()
	for _, name := range installPaths {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Fatal(err)
		}
	}
	args := append([]string{"checkout", "--"}, installPaths...)
	runWriterGit(t, root, args...)
	for _, name := range installPaths {
		if name == "tools/markitect/source.zip" {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(name))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS == "windows" && !bytes.Contains(data, []byte("\r\n")) {
			t.Fatalf("core.autocrlf=true did not materialize CRLF in %s", name)
		}
		if !bytes.Contains(data, []byte("\r\n")) {
			data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
			if err := os.WriteFile(path, data, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestInstallRejectsSymlinkedPinPath(t *testing.T) {
	root := installTestRepo(t, "feature/symlink")
	outside := tempRoot(t)
	if err := os.Symlink(outside, filepath.Join(root, "scripts")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := Install(root, installTestBundle(t, "7", "0.1.0-rc.4", "new"), false); err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("symlinked destination error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "run-markitect.go")); !os.IsNotExist(err) {
		t.Fatalf("installer wrote through symlink: %v", err)
	}
}

func installTestRepo(t *testing.T, branch string) string {
	t.Helper()
	root := tempRoot(t)
	runWriterGit(t, root, "init", "-b", branch)
	runWriterGit(t, root, "config", "user.name", "Markitect Test")
	runWriterGit(t, root, "config", "user.email", "markitect-test@example.invalid")
	writeFixture(t, root, map[string][]byte{"README.md": []byte("test consumer\n")})
	commitInstallPins(t, root, "consumer baseline")
	return root
}

func commitInstallPins(t *testing.T, root, message string) {
	t.Helper()
	runWriterGit(t, root, "add", "-A")
	runWriterGit(t, root, "commit", "-m", message)
}

func installTestBundle(t *testing.T, sourceID, version, marker string) *release.Bundle {
	t.Helper()
	// Keep the fixture unmistakably binary so core.autocrlf never treats the
	// byte-exact archive pin as text during checkout.
	archive := []byte("PK\x03\x04source archive " + marker + "\x00\n")
	archiveHash := digestInstallBytes(archive)
	files := map[string][]byte{
		"markitect.lock.yaml":                 []byte("version: \"" + version + "\"\nsource: \"tools/markitect/source.zip\"\nsha256: \"" + archiveHash + "\"\n"),
		"scripts/markitect-bootstrap_test.go": []byte("package scripts // " + marker + "\n"),
		"scripts/run-markitect.go":            []byte("package main // " + marker + "\n"),
		"tools/markitect/source.zip":          archive,
	}
	manifest := release.BundleManifest{SchemaVersion: 1, Version: version, SourceCommit: strings.Repeat(sourceID, 40), SourceRepository: "github.com/Glacius-Labs/Markitect"}
	for _, name := range installPinPaths {
		manifest.Files = append(manifest.Files, release.BundleFile{Path: name, SHA256: digestInstallBytes(files[name])})
	}
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	files[releaseManifestPath] = manifestBytes
	bundle := &release.Bundle{Manifest: manifest, Files: files, SHA256: digestInstallBytes([]byte("outer bundle " + marker))}
	if err := validateIncomingBundle(bundle); err != nil {
		t.Fatalf("test bundle invalid: %v", err)
	}
	return bundle
}

func writeBundleToRoot(t *testing.T, root string, bundle *release.Bundle, includeManifest bool) {
	t.Helper()
	for _, name := range installPaths {
		if name == releaseManifestPath && !includeManifest {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(name))), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), bundle.Files[name], 0644); err != nil {
			t.Fatal(err)
		}
	}
}
