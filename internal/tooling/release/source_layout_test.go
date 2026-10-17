package release

import (
	"bytes"
	"github.com/Glacius-Labs/Markitect/internal/tooling/source"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rootLayoutFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module github.com/Glacius-Labs/Markitect\n", "go.sum": "", "LICENSE": "license\n",
		"cmd/markitect/main.go": "package main\n", "internal/model/model.go": "package model\n",
		"internal/run/state.go":                       "package run\n",
		"internal/adapters/documents/writer.go":       "package documents\n",
		"internal/tooling/release/install/install.go": "package releaseinstall\n",
		"internal/testkit/testdata/hostile.gitconfig": "[core]\n hooksPath=blocked\n",
		"internal/apply/strategy_defaults.yaml":       "exact defaults\r\n",
	} {
		writeLayoutFile(t, root, name, content)
	}
	return root
}

func TestRootLayoutLiveAndFixedMatch(t *testing.T) {
	root := rootLayoutFixture(t)
	writeLayoutFile(t, root, "internal/model/vendor/ignored.go", "package ignored\n")
	writeLayoutFile(t, root, "internal/model/vendor.go", "package model\n")
	writeLayoutFile(t, root, "internal/model/notes.txt", "omit non-runtime text\n")
	fixed := fixedSourceSnapshot(t, root)
	live, _, err := Package(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := snapshotModuleFiles(fixed)
	if err != nil {
		t.Fatal(err)
	}
	files, err := collectModuleSnapshot(selected)
	if err != nil {
		t.Fatal(err)
	}
	historical, err := makeArchive(files)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(live, historical) {
		t.Fatal("live and fixed root selections differ")
	}
	contents := archiveContents(t, live)
	for _, name := range []string{"cmd/markitect/main.go", "internal/model/vendor.go", "internal/run/state.go", "internal/adapters/documents/writer.go", "internal/tooling/release/install/install.go", "internal/testkit/testdata/hostile.gitconfig", "internal/apply/strategy_defaults.yaml"} {
		if _, ok := contents[name]; !ok {
			t.Errorf("omitted %s", name)
		}
	}
	for _, name := range []string{"internal/model/vendor/ignored.go", "internal/model/notes.txt"} {
		if _, ok := contents[name]; ok {
			t.Errorf("included %s", name)
		}
	}
	if !bytes.Equal(contents[embeddedStrategyDefaultsPath], fixed.Files[embeddedStrategyDefaultsPath]) {
		t.Fatal("embedded strategy bytes changed")
	}
	fixed.Files["internal/model/model.go"] = []byte("package model\n// fixed change\n")
	changed, err := snapshotModuleFiles(fixed)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(changed["internal/model/model.go"], contents["internal/model/model.go"]) {
		t.Fatal("fixed selection read current disk")
	}
}

func TestRootLayoutRejectsUnsupportedInputsBeforeFiltering(t *testing.T) {
	for _, name := range []string{"src/cmd/markitect/main.go", "internal/other/x.txt", "internal/model-extra/x.go", "internal/run-extra/x.go", "internal/adapters/direct.go", "internal/adapters/checks-extra/profile.go", "internal/tooling-extra/release/x.go", "internal/tooling/other/x.go", "cmd/other/main.go", "internal/model/nested/go.mod"} {
		t.Run(name, func(t *testing.T) {
			root := rootLayoutFixture(t)
			writeLayoutFile(t, root, name, "unexpected\n")
			if _, _, err := Package(root, "1.2.3"); err == nil {
				t.Fatal("live unsupported source accepted")
			}
			if _, err := snapshotModuleFiles(fixedSourceSnapshot(t, root)); err == nil {
				t.Fatal("fixed unsupported source accepted")
			}
		})
	}
	for _, name := range []string{"internal/model/model.go", "internal/model/notes.txt", "cmd/markitect/main.go", "internal/apply/strategy_defaults.yaml"} {
		t.Run("mode "+name, func(t *testing.T) {
			root := rootLayoutFixture(t)
			writeLayoutFile(t, root, name, "bytes\n")
			fixed := fixedSourceSnapshot(t, root)
			fixed.Modes[name] = "120000"
			if _, err := snapshotModuleFiles(fixed); err == nil {
				t.Fatal("nonregular source omitted or followed")
			}
		})
	}
	root := rootLayoutFixture(t)
	writeLayoutFile(t, root, "internal/model/nested/go.mod", "module nested\n")
	if _, _, err := Package(root, "1.2.3"); err == nil || !strings.Contains(err.Error(), "nested Go module") {
		t.Fatalf("%v", err)
	}
}

func TestRootLayoutRequiresCommandAndInternalGoSources(t *testing.T) {
	for _, removed := range []string{"cmd/markitect/main.go", "internal/model/model.go"} {
		root := t.TempDir()
		writeLayoutFile(t, root, "go.mod", "module github.com/Glacius-Labs/Markitect\n")
		writeLayoutFile(t, root, "go.sum", "")
		writeLayoutFile(t, root, "LICENSE", "license\n")
		for _, name := range []string{"cmd/markitect/main.go", "internal/model/model.go"} {
			if name != removed {
				writeLayoutFile(t, root, name, "package fixture\n")
			}
		}
		if _, _, err := Package(root, "1.2.3"); err == nil {
			t.Fatalf("live missing %s accepted", removed)
		}
		fixed := fixedSourceSnapshot(t, root)
		if files, err := snapshotModuleFiles(fixed); err == nil {
			if _, err = collectModuleSnapshot(files); err == nil {
				t.Fatalf("fixed missing %s accepted", removed)
			}
		}
	}
}

func TestRootResponsibilityPrefixesAreExact(t *testing.T) {
	for _, name := range []string{"internal/model/a.go", "internal/tooling/release/install/a.go", "internal/adapters/documents/a.go", "cmd/markitect/main.go"} {
		if !sourceRoot(name) {
			t.Errorf("rejected %s", name)
		}
	}
	for _, name := range []string{"internal/modelx/a.go", "internal/toolingx/a.go", "internal/adapters/direct.go", "src/internal/model/a.go", "cmd/markitect-extra/main.go"} {
		if sourceRoot(name) {
			t.Errorf("accepted %s", name)
		}
	}
}

func TestRawLegacyAndSchemaBoundariesMatchLiveAndFixed(t *testing.T) {
	for _, name := range []string{"src", "schema"} {
		t.Run(name, func(t *testing.T) {
			root := rootLayoutFixture(t)
			writeLayoutFile(t, root, name, "not a directory\n")
			if _, _, err := Package(root, "1.2.3"); err == nil {
				t.Fatal("live file at a directory boundary was accepted")
			}
			fixed := fixedSourceSnapshot(t, root)
			for _, mode := range []string{"100644", "120000"} {
				fixed.Modes[name] = mode
				if _, err := snapshotModuleFiles(fixed); err == nil {
					t.Fatalf("fixed %s mode %s was silently filtered", name, mode)
				}
			}
		})
	}
	for _, name := range []string{"schema/selected.yaml", "schema/unselected.txt"} {
		t.Run(name, func(t *testing.T) {
			root := rootLayoutFixture(t)
			fixed := fixedSourceSnapshot(t, root)
			fixed.Files[name] = []byte("outside")
			fixed.Modes[name] = "120000"
			if _, err := snapshotModuleFiles(fixed); err == nil {
				t.Fatal("fixed schema redirect was filtered before mode validation")
			}
			external := filepath.Join(t.TempDir(), "outside")
			if err := os.WriteFile(external, []byte("outside"), 0644); err != nil {
				t.Fatal(err)
			}
			full := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(external, full); err != nil {
				t.Logf("live symlink creation unavailable; fixed raw-mode assertion passed: %v", err)
				return
			}
			if _, _, err := Package(root, "1.2.3"); err == nil {
				t.Fatal("live schema redirect was silently filtered")
			}
		})
	}
}

func fixedSourceSnapshot(t *testing.T, root string) *source.Input {
	t.Helper()
	files := map[string][]byte{}
	modes := map[string]string{}
	err := filepath.WalkDir(root, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		files[relative] = data
		modes[relative] = source.RegularMode
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return &source.Input{Files: files, Modes: modes}
}

func writeLayoutFile(t *testing.T, root, relative, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
