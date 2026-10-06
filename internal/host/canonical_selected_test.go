package host

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func TestLoadSelectedCanonicalSourceMatchesFullCompilerAndExposesScope(t *testing.T) {
	root := "../.."
	commitBytes, err := source.GitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(commitBytes))
	configPath := "examples/canonical-projection/canonical.yaml"
	full, err := LoadCanonicalSource(root, commit, configPath, true)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := LoadSelectedCanonicalSource(root, commit, configPath, true)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Model.Digest != full.Model.Digest || !reflect.DeepEqual(selected.Model.Edges, full.Model.Edges) || !reflect.DeepEqual(selected.Diagnostics, full.Diagnostics) {
		t.Fatalf("selected compiler result differs from full compiler: selected=%#v full=%#v", selected.Model, full.Model)
	}
	if selected.AcquisitionScope == nil || selected.AcquisitionScope.FullCommit != commit || selected.Snapshot.ID != commit || selected.Snapshot.Provisional {
		t.Fatalf("selected source lacks full-commit scoped evidence: %#v", selected)
	}
	if selected.AcquisitionScope.Digest != selected.Snapshot.Digest() || reflect.DeepEqual(selected.Snapshot.Digest(), full.Snapshot.Digest()) {
		t.Fatalf("selected digest must bind the sparse selected snapshot only: selected=%q full=%q scope=%q", selected.Snapshot.Digest(), full.Snapshot.Digest(), selected.AcquisitionScope.Digest)
	}
	if got, err := SelectedInputSnapshotDigest(selected); err != nil || got != selected.Snapshot.Digest() {
		t.Fatalf("selected scope digest = %q, %v", got, err)
	}
	if _, err := SelectedInputSnapshotDigest(full); err == nil {
		t.Fatal("full-source load unexpectedly claims a selected-input scope")
	}
}

func TestLoadSelectedCanonicalSourceAcquiresConfigThenExactDeclaredUnion(t *testing.T) {
	root := "../.."
	commitBytes, err := source.GitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(commitBytes))
	configPath := "examples/canonical-projection/canonical.yaml"
	var acquisitions [][]string
	loader := func(root, commit string, paths []string) (*source.SelectedSnapshot, error) {
		acquisitions = append(acquisitions, append([]string(nil), paths...))
		return source.LoadSelected(root, commit, paths)
	}
	loaded, err := loadSelectedCanonicalSourceWith(root, commit, configPath, true, loader)
	if err != nil {
		t.Fatal(err)
	}
	configBytes := loaded.Snapshot.Files[configPath]
	config, err := DecodeCanonicalSourceConfig(configBytes)
	if err != nil {
		t.Fatal(err)
	}
	want := canonicalSelectedInputPaths(configPath, config)
	if !reflect.DeepEqual(acquisitions, [][]string{{configPath}, want}) {
		t.Fatalf("selected acquisitions = %#v, want config then exact declared union %#v", acquisitions, [][]string{{configPath}, want})
	}
	if !reflect.DeepEqual(loaded.AcquisitionScope.Paths, want) {
		t.Fatalf("reported scope paths = %#v, want %#v", loaded.AcquisitionScope.Paths, want)
	}
}

func TestLoadSelectedCanonicalSourceRejectsRepositoryIdentitySubstitution(t *testing.T) {
	root := "../.."
	commitBytes, err := source.GitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(commitBytes))
	configPath := "examples/canonical-projection/canonical.yaml"
	calls := 0
	loader := func(root, commit string, paths []string) (*source.SelectedSnapshot, error) {
		calls++
		selected, err := source.LoadSelected(root, commit, paths)
		if err == nil && calls == 2 {
			selected.Identity.Digest = "substituted repository identity"
		}
		return selected, err
	}
	if _, err := loadSelectedCanonicalSourceWith(root, commit, configPath, true, loader); err == nil || !strings.Contains(err.Error(), "identity changed") {
		t.Fatalf("identity substitution error = %v", err)
	}
}
