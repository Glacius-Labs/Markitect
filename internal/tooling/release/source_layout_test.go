package release

import (
	"bytes"
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
		"internal/project/prepared.go":                "package project\n",
		"internal/realization/registry.go":            "package realization\n",
		"internal/analysis/impact.go":                 "package analysis\n",
		"internal/adapters/repo/git.go":               "package repo\n",
		"cmd/markitect-agentd/main.go":                "package main\n",
		"internal/agentd/agentd.go":                   "package agentd\n",
		"internal/project/testdata/fixture.json":      "literal historical fixture\n",
		"internal/adapters/cli/cli.go":                "package cli\n",
		"internal/libraries/libraries.go":             "package libraries\n",
		"internal/libraries/std/Rule.md":              "---\nkind: Kind\n---\n# Rule\n",
		"internal/skills/skills.go":                   "package skills\n",
		"internal/skills/markitect-modeling/SKILL.md": "# Modeling\n",
		"internal/tooling/release/install.go":         "package release\n",
		"internal/testkit/testdata/hostile.gitconfig": "[core]\n hooksPath=blocked\n",
	} {
		writeLayoutFile(t, root, name, content)
	}
	return root
}

func TestEmbeddedFilesSurviveLiveAndFixedPackaging(t *testing.T) {
	// The standard Libraries and the skills are built into the binary, so a
	// packaged source must carry them; other text below those packages stays out.
	root := rootLayoutFixture(t)
	writeLayoutFile(t, root, "internal/libraries/notes.txt", "do not package\n")
	live, _, err := Package(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := snapshotModuleFiles(fixedSourceSnapshot(t, root))
	if err != nil {
		t.Fatal(err)
	}
	files, err := collectModuleSnapshot(selected)
	if err != nil {
		t.Fatal(err)
	}
	fixed, err := makeArchive(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range [][]byte{live, fixed} {
		contents := archiveContents(t, archive)
		for _, name := range []string{"internal/libraries/std/Rule.md", "internal/skills/markitect-modeling/SKILL.md"} {
			if _, ok := contents[name]; !ok {
				t.Errorf("embedded file omitted: %s", name)
			}
		}
		if _, found := contents["internal/libraries/notes.txt"]; found {
			t.Error("packaged unrelated text")
		}
	}
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
	for _, name := range []string{"cmd/markitect/main.go", "internal/model/vendor.go", "internal/project/prepared.go", "internal/realization/registry.go", "internal/analysis/impact.go", "internal/adapters/repo/git.go", "cmd/markitect-agentd/main.go", "internal/agentd/agentd.go", "internal/project/testdata/fixture.json", "internal/adapters/cli/cli.go", "internal/libraries/std/Rule.md", "internal/skills/markitect-modeling/SKILL.md", "internal/tooling/release/install.go", "internal/testkit/testdata/hostile.gitconfig"} {
		if _, ok := contents[name]; !ok {
			t.Errorf("omitted %s", name)
		}
	}
	for _, name := range []string{"internal/model/vendor/ignored.go", "internal/model/notes.txt"} {
		if _, ok := contents[name]; ok {
			t.Errorf("included %s", name)
		}
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

// Use the real compiler package bytes so both live and fixed source selections
// must carry the code and its tests, not a successful placeholder package.
func TestModelSourceLiveAndFixedSourceBytes(t *testing.T) {
	root := rootLayoutFixture(t)
	originals := make(map[string][]byte)
	for _, name := range []string{"compile.go", "markdown.go", "compile_test.go"} {
		path := "internal/model/" + name
		data, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		originals[path] = data
		writeLayoutFile(t, root, path, string(data))
	}
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
	archive, err := makeArchive(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, contents := range []map[string][]byte{archiveContents(t, live), archiveContents(t, archive)} {
		for path, want := range originals {
			if !bytes.Equal(contents[path], want) {
				t.Errorf("source bytes lost or changed: %s", path)
			}
		}
	}
}

func TestRootLayoutRejectsUnsupportedInputsBeforeFiltering(t *testing.T) {
	for _, name := range []string{"internal/adapters/cockpit/web/index.html", "internal/authoring/session.go", "src/cmd/markitect/main.go", "internal/other/x.txt", "internal/model-extra/x.go", "internal/run-extra/x.go", "internal/adapters/direct.go", "internal/adapters/repo-extra/git.go", "internal/adapters/cli-extra/cli.go", "internal/harness/harness.go", "internal/apply/apply.go", "internal/adapters/unowned/codec.go", "internal/tooling-extra/release/x.go", "internal/tooling/other/x.go", "cmd/other/main.go", "internal/model/nested/go.mod"} {
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
	for _, name := range []string{"internal/model/model.go", "internal/model/notes.txt", "cmd/markitect/main.go"} {
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
	for _, name := range []string{"internal/model/a.go", "internal/tooling/release/a.go", "internal/adapters/cli/a.go", "internal/adapters/repo/git.go", "cmd/markitect/main.go"} {
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

func fixedSourceSnapshot(t *testing.T, root string) *Input {
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
		modes[relative] = RegularMode
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return &Input{Files: files, Modes: modes}
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
