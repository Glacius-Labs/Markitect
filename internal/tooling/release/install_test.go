package release

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallPlansThenWritesCompleteFreshBundleAndNoops(t *testing.T) {
	root := installTestRepo(t, "feature/install")
	bundle := installTestBundle(t, "a", "0.9.0", "new")

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
	if got := strings.Join(written.Written, ","); got != ".markitect/bootstrap/run.go,.markitect/bootstrap/run_test.go,.markitect/tool/lock.yaml,.markitect/tool/source.zip,.markitect/tool/release.yaml" {
		t.Fatalf("complete pin must publish its manifest last: %s", got)
	}
	commitInstallPins(t, root, "install release")
	noop, err := Install(root, bundle, false)
	if err != nil {
		t.Fatal(err)
	}
	if noop.Kind != "noop" || noop.Applied || len(noop.Written) != 0 {
		t.Fatalf("no-op plan = %+v", noop)
	}
	noop, err = Install(root, bundle, true)
	if err != nil || noop.Kind != "noop" || !noop.Applied || len(noop.Written) != 0 {
		t.Fatalf("verified write-mode no-op = %+v, %v", noop, err)
	}
}

func TestInstallUpgradesCommittedManifestPins(t *testing.T) {
	root := installTestRepo(t, "feature/upgrade")
	old := installTestBundle(t, "b", "0.8.0", "old")
	writeBundleToRoot(t, root, old, true)
	commitInstallPins(t, root, "legacy upgrade baseline")
	newBundle := installTestBundle(t, "c", "0.9.0", "new")

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

func TestInstallPreservesUnchangedExecutableModeDuringUpgrade(t *testing.T) {
	root := installTestRepo(t, "feature/preserve-mode")
	old := installTestBundle(t, "d", "0.8.0", "stable-bootstrap")
	writeBundleToRoot(t, root, old, true)
	path := filepath.Join(root, ".markitect", "bootstrap", "run.go")
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if before.Mode().Perm()&0111 == 0 {
		t.Skip("filesystem does not expose executable permission bits")
	}
	commitInstallPins(t, root, "install executable release pin")

	updated := installTestBundle(t, "e", "0.9.0", "stable-bootstrap")
	plan, err := Install(root, updated, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, file := range plan.Files {
		if file.Path == ".markitect/bootstrap/run.go" {
			found = true
			if file.Action != "unchanged" {
				t.Fatalf("executable bootstrap action = %q, want unchanged", file.Action)
			}
		}
	}
	if !found {
		t.Fatal("plan omitted executable bootstrap file")
	}
	if _, err := Install(root, updated, true); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode().Perm() != before.Mode().Perm() {
		t.Fatalf("installed mode = %04o, want captured mode %04o", after.Mode().Perm(), before.Mode().Perm())
	}
}

func TestInstallCanPreviewWithoutGitCommit(t *testing.T) {
	t.Run("outside git", func(t *testing.T) {
		root := tempRoot(t)
		plan, err := Install(root, installTestBundle(t, "a", "0.1.0", "preview"), false)
		if err != nil || plan.Kind != "install" || plan.Applied {
			t.Fatalf("standalone preview = (%+v, %v)", plan, err)
		}
		if _, err := os.Stat(filepath.Join(root, ".markitect")); !os.IsNotExist(err) {
			t.Fatalf("standalone preview created an install target: %v", err)
		}
	})

	t.Run("unborn branch", func(t *testing.T) {
		root := tempRoot(t)
		runWriterGit(t, root, "init", "-b", "feature/unborn-preview")
		plan, err := Install(root, installTestBundle(t, "b", "0.1.0", "preview"), false)
		if err != nil || plan.Kind != "install" || plan.Applied {
			t.Fatalf("unborn branch preview = (%+v, %v)", plan, err)
		}
		if _, err := os.Stat(filepath.Join(root, ".markitect")); !os.IsNotExist(err) {
			t.Fatalf("unborn branch preview created an install target: %v", err)
		}
	})
}

func TestInstallReadbackNormalizesCoreAutocrlfCheckout(t *testing.T) {
	root := installTestRepo(t, "feature/autocrlf")
	old := installTestBundle(t, "8", "0.8.0", "old")
	writeBundleToRoot(t, root, old, true)
	commitInstallPins(t, root, "install release pins")
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
	archive, err := os.ReadFile(filepath.Join(root, ".markitect", "tool", "source.zip"))
	if err != nil || !bytes.Equal(archive, updated.Files[".markitect/tool/source.zip"]) {
		t.Fatalf("binary source archive was normalized: %v", err)
	}
}

func TestInstallUpgradesUnchangedCRLFPins(t *testing.T) {
	root := installTestRepo(t, "feature/autocrlf-unchanged")
	old := installTestBundle(t, "a", "0.8.0", "stable-bootstrap")
	writeBundleToRoot(t, root, old, true)
	commitInstallPins(t, root, "install release pins")
	runWriterGit(t, root, "config", "core.autocrlf", "true")
	recreateAutocrlfCheckout(t, root)

	// Keep both bootstrap files byte-identical while changing the lock and the
	// release manifest. This exercises verification of unchanged checkout pins.
	updated := installTestBundle(t, "b", "0.9.0", "stable-bootstrap")
	plan, err := Install(root, updated, false)
	if err != nil || plan.Kind != "upgrade" {
		t.Fatalf("CRLF upgrade plan = (%+v, %v), want upgrade", plan, err)
	}
	result, err := Install(root, updated, true)
	if err != nil || result.Kind != "upgrade" || !result.Applied {
		t.Fatalf("CRLF upgrade with unchanged text pins = (%+v, %v)", result, err)
	}
}

func TestInstallFreshnessComparesRawCheckoutBytes(t *testing.T) {
	root := installTestRepo(t, "feature/raw-freshness")
	bundle := installTestBundle(t, "a", "0.8.0", "old")
	writeBundleToRoot(t, root, bundle, true)
	commitInstallPins(t, root, "install release pins")
	runWriterGit(t, root, "config", "core.autocrlf", "true")
	recreateAutocrlfCheckout(t, root)
	_, state, err := buildInstallPlan(root, bundle)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".markitect", "bootstrap", "run.go")
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
