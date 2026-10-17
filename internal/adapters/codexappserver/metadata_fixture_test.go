package codexappserver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

type metadataFixture struct {
	exe, stable, experimental string
	pin                       offlinePin
}

func writeFixture(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}
func fixture(t *testing.T) metadataFixture {
	t.Helper()
	dir := testkit.TempDir(t)
	f := metadataFixture{exe: filepath.Join(dir, "bin", "program"), stable: filepath.Join(dir, "stable"), experimental: filepath.Join(dir, "experimental")}
	writeFixture(t, f.exe, "fixture executable\r\n")
	f.pin.executable = sha256.Sum256([]byte("fixture executable\r\n"))
	f.pin.selected = []string{"schema.json"}
	for i, boundary := range []string{f.stable, f.experimental} {
		writeFixture(t, filepath.Join(boundary, "schema.json"), `{"type":"object"}`)
		writeFixture(t, filepath.Join(boundary, "unused.json"), "{}\n")
		// Independent fixture manifest, not a production pin configuration API.
		manifest := fmt.Sprintf("schema.json\x0017\x00%x\x00unused.json\x003\x00%x\x00", sha256.Sum256([]byte(`{"type":"object"}`)), sha256.Sum256([]byte("{}\n")))
		f.pin.inventories[i] = sha256.Sum256([]byte(manifest))
	}
	return f
}
func (f metadataFixture) inline() string {
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	return fmt.Sprintf("profileVersion: 1\nmetadataPin: %s\nexecutable: %s\nschemas: {stable: %s, experimental: %s}\ninstructions: []\n", metadataPinName, quote(f.exe), quote(f.stable), quote(f.experimental))
}
func testPath(t *testing.T, s string) snapshot.Path {
	t.Helper()
	p, err := snapshot.NewPath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

type fixtureSource struct{ raw apply.RawInputs }

func (s fixtureSource) Capture(context.Context, apply.CaptureRequest) (apply.CapturedInputs, error) {
	return apply.CapturedInputs{Raw: s.raw, Head: "e96eef6b9751298fd930f7fcd43e00a251a0d86e", Guard: "controlled-source-fixture"}, nil
}

type captureProfile func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error)

func (f captureProfile) PrepareAgentProfile(c context.Context, p apply.ProfileInput) (apply.ProfileObservation, error) {
	return f(c, p)
}

// Obtain the opaque input through Apply's real public M1 consumer. The fixture
// Source supplies existing language bytes; it claims no physical Git acquisition.
func profileInput(t *testing.T, inline string, instructions []apply.SourceFile) apply.ProfileInput {
	t.Helper()
	base := filepath.Join("..", "..", "..", "docs", "design", "canonical-model")
	readModel := func(dir, boundary string) apply.ModelSource {
		area, _ := snapshot.DirectoryAt(testPath(t, "model"))
		m := apply.ModelSource{Selection: apply.ModelSelection{Boundary: boundary, ManifestPath: testPath(t, "model.yaml")}, Root: area, Source: model.SourceModel{Origin: "fixture:" + boundary, ManifestPath: "model.yaml"}}
		var files []snapshot.Entry
		err := fs.WalkDir(os.DirFS(dir), ".", func(path string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == "." {
				return nil
			}
			entry := model.Entry{Path: path, Kind: model.Directory}
			if !e.IsDir() {
				entry.Kind = model.Regular
				entry.Bytes, err = os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
				if err != nil {
					return err
				}
				f, err := snapshot.NewEntry(testPath(t, path), snapshot.Regular, entry.Bytes)
				if err != nil {
					return err
				}
				files = append(files, f)
			}
			m.Source.Entries = append(m.Source.Entries, entry)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		m.Files, err = snapshot.NewTree(files)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	raw := apply.RawInputs{Local: readModel(filepath.Join(base, "library-example", "m1"), "local"), Dependencies: []apply.ModelSource{readModel(filepath.Join(base, "library-example", "vocabulary"), "vocabulary"), readModel(filepath.Join(base, "minimal-language", "exchange-example", "dependencies", "core"), "core")}, Instructions: instructions}
	raw.Local.RepositoryArea, _ = snapshot.DirectoryAt(testPath(t, "local"))
	var census []snapshot.Entry
	for _, e := range raw.Local.Files.Entries() {
		f, err := snapshot.NewEntry(testPath(t, "local/"+e.Path().String()), e.Mode(), e.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		census = append(census, f)
	}
	var err error
	raw.Census, err = snapshot.NewTree(census)
	if err != nil {
		t.Fatal(err)
	}
	host := "apiVersion: markitect.host/v1\nagentProfiles:\n  library-agent:\n    codexAppServer:\n"
	for _, line := range strings.Split(strings.TrimSuffix(inline, "\n"), "\n") {
		host += "      " + line + "\n"
	}
	host += "checkProfiles: {}\n"
	location := apply.SourceLocation{Boundary: filepath.Join(t.TempDir(), "host"), Path: testPath(t, "profiles.yaml")}
	entry, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte(host))
	if err != nil {
		t.Fatal(err)
	}
	raw.HostProfiles = []apply.SourceFile{{Location: location, File: entry}}
	selection := apply.InputSelection{Repository: "controlled-M1-source", Local: raw.Local.Selection, HostProfiles: []apply.SourceLocation{location}}
	for _, dependency := range raw.Dependencies {
		selection.Dependencies = append(selection.Dependencies, dependency.Selection)
	}
	for _, instruction := range instructions {
		selection.Instructions = append(selection.Instructions, instruction.Location)
	}
	m, err := apply.NewModelService(fixtureSource{raw}, nil).CaptureModel(context.Background(), apply.CaptureRequest{Selection: selection}, model.Limits{})
	if err != nil || m.Compilation().Status != model.Passed {
		t.Fatalf("M1 source: %v %+v", err, m.Compilation())
	}
	var input apply.ProfileInput
	port := captureProfile(func(_ context.Context, p apply.ProfileInput) (apply.ProfileObservation, error) {
		input = p
		return apply.ProfileObservation{InputBinding: p.Binding(), Status: model.NotCompleted}, nil
	})
	_, err = m.PrepareRegistry().CheckFiles().PrepareSettings(context.Background(), m.DecodeExecution().PrepareDefaults(apply.ProductStrategyDefaults()), port, nil)
	if err != nil || input.Inline() == nil {
		t.Fatalf("real consumer input: %v", err)
	}
	return input
}
