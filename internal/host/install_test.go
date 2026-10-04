package host

import (
	"bytes"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestInstalledPinsOutsideDeclaredAreaAreNotTypedResources(t *testing.T) {
	root := installTestRepo(t, "feature/area-pins")
	project := projectResource(projectNS)
	project.Spec.Areas = []authoring.Area{{Name: "engineering", Path: ".markitect/areas/engineering"}}
	rule := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion,
		Kind:     "Rule",
		Metadata: core.Metadata{Name: "deployment", Namespace: "engineering"}}, Spec: authoring.Spec{Text: "Follow the deployment process."},
	}
	writeFixture(t, root, map[string][]byte{
		"markitect.yaml": encodeResource(t, project),
		".markitect/areas/engineering/deployment.yaml": encodeResource(t, rule),
	})
	commitInstallPins(t, root, "consumer project with a hidden control-plane area")
	if _, err := Install(root, installTestBundle(t, "f", "0.9.0", "area-pins"), true); err != nil {
		t.Fatalf("install pinned tool: %v", err)
	}

	installed, err := Load(root, "")
	if err != nil {
		t.Fatalf("load installed project: %v", err)
	}
	if len(installed.Diagnostics) != 0 {
		t.Fatalf("installed project diagnostics: %#v", installed.Diagnostics)
	}
	if len(installed.Resources) != 2 || len(installed.Inventory) != 2 {
		t.Fatalf("installed project parsed unexpected resources: resources=%#v inventory=%#v", installed.Resources, installed.Inventory)
	}
	for _, entry := range installed.Inventory {
		if entry.Path == ".markitect/tool/lock.yaml" || entry.Path == ".markitect/tool/release.yaml" {
			t.Fatalf("pinned distribution metadata became a typed resource: %#v", entry)
		}
	}
	if got := installed.Graph.Resources["engineering/Rule/deployment"]; got == nil || got.Path != ".markitect/areas/engineering/deployment.yaml" {
		t.Fatalf("declared hidden area resource was not retained: %#v", got)
	}
}

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
