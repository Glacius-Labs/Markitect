package cockpit

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func storeIn(t *testing.T) (*ProjectStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "markitect", "cockpit", "projects.json")
	return NewProjectStore(path), path
}

func TestProjectStoreReadsAMissingFileAsAnEmptyListAndCreatesItsDirectories(t *testing.T) {
	store, path := storeIn(t)
	list, err := store.Load()
	if err != nil || len(list.Projects) != 0 || list.Last != "" {
		t.Fatalf("missing = %+v, %v", list, err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("reading created %s: %v", path, err)
	}
	entry := ProjectEntry{Name: "alpha", Repository: filepath.Join(t.TempDir(), "alpha"), Model: DefaultModel, Dependencies: []string{"deps/core/model.yaml"}}
	if _, err := store.Update(func(l *ProjectList) error {
		l.Projects = append(l.Projects, entry)
		l.Last = "alpha"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	list, err = store.Load()
	if err != nil || len(list.Projects) != 1 || list.Last != "alpha" {
		t.Fatalf("saved = %+v, %v", list, err)
	}
	got, _ := list.Find("alpha")
	if got.Repository != entry.Repository || got.Model != DefaultModel || len(got.Dependencies) != 1 || got.Runtime != "" {
		t.Fatalf("entry = %+v", got)
	}
	data, _ := os.ReadFile(path)
	for _, want := range []string{`"formatVersion": 1`, `"projects"`, `"last": "alpha"`, `"runtime": ""`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("the file lacks %s:\n%s", want, data)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %v, %v", info.Mode(), err)
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("the atomic write left temporary files: %v", entries)
	}
}

func TestProjectStoreNeverOverwritesAFileItCannotUse(t *testing.T) {
	repo := t.TempDir()
	valid := `{"formatVersion":1,"projects":[{"name":"a","repository":` + jsonString(repo) + `,"model":"","dependencies":[],"runtime":""}],"last":""}`
	for name, content := range map[string]string{
		"not JSON":              "{",
		"no version":            `{"projects":[]}`,
		"an unknown version":    `{"formatVersion":2,"projects":[]}`,
		"an unknown field":      `{"formatVersion":1,"projects":[],"last":"","extra":1}`,
		"trailing content":      valid + " {}",
		"a relative repository": `{"formatVersion":1,"projects":[{"name":"a","repository":"rel","model":"","dependencies":[],"runtime":""}],"last":""}`,
		"a relative runtime":    `{"formatVersion":1,"projects":[{"name":"a","repository":` + jsonString(repo) + `,"model":"","dependencies":[],"runtime":"rel"}],"last":""}`,
		"a duplicate name":      `{"formatVersion":1,"projects":[{"name":"a","repository":` + jsonString(repo) + `,"model":"","dependencies":[],"runtime":""},{"name":"a","repository":` + jsonString(repo) + `,"model":"","dependencies":[],"runtime":""}],"last":""}`,
		"an invalid name":       `{"formatVersion":1,"projects":[{"name":"a/b","repository":` + jsonString(repo) + `,"model":"","dependencies":[],"runtime":""}],"last":""}`,
	} {
		t.Run(name, func(t *testing.T) {
			store, path := storeIn(t)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			var bad *ProjectsFileError
			if _, err := store.Load(); !errors.As(err, &bad) || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "does not overwrite") {
				t.Fatalf("load = %v", err)
			}
			if _, err := store.Update(func(l *ProjectList) error { l.Last = "x"; return nil }); !errors.As(err, &bad) {
				t.Fatalf("update = %v", err)
			}
			if got, _ := os.ReadFile(path); string(got) != content {
				t.Fatalf("the unusable file was changed:\n%s", got)
			}
		})
	}
}

func TestProjectStoreKeepsTheFileWhenAnUpdateRefuses(t *testing.T) {
	store, path := storeIn(t)
	if _, err := store.Update(func(l *ProjectList) error { l.Last = "keep"; return nil }); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	refusal := errors.New("refused")
	if _, err := store.Update(func(l *ProjectList) error { l.Last = "other"; return refusal }); !errors.Is(err, refusal) {
		t.Fatalf("update = %v", err)
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Fatalf("a refused update changed the file:\n%s", after)
	}
}

func TestValidateEntryChecksRepositoryModelDependenciesAndRuntime(t *testing.T) {
	repo := t.TempDir()
	write := func(path string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(repo, ".markitect", "model.yaml"))
	runtimeDir := t.TempDir()
	entry := ProjectEntry{Name: "alpha", Repository: repo, Model: DefaultModel}
	if err := ValidateEntry(entry); err != nil {
		t.Fatalf("valid entry = %v", err)
	}
	for name, mutate := range map[string]func(*ProjectEntry){
		"a bad name":              func(e *ProjectEntry) { e.Name = "../x" },
		"a relative repository":   func(e *ProjectEntry) { e.Repository = "repo" },
		"a missing repository":    func(e *ProjectEntry) { e.Repository = filepath.Join(repo, "none") },
		"a missing model":         func(e *ProjectEntry) { e.Model = "other.yaml" },
		"a model outside":         func(e *ProjectEntry) { e.Model = "../model.yaml" },
		"a missing dependency":    func(e *ProjectEntry) { e.Dependencies = []string{"dep/model.yaml"} },
		"a relative runtime":      func(e *ProjectEntry) { e.Runtime = "rt" },
		"a runtime without files": func(e *ProjectEntry) { e.Runtime = runtimeDir },
	} {
		t.Run(name, func(t *testing.T) {
			e := entry
			mutate(&e)
			if err := ValidateEntry(e); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	write(filepath.Join(runtimeDir, "run.json"))
	e := entry
	e.Runtime = runtimeDir
	if err := ValidateEntry(e); err == nil || !strings.Contains(err.Error(), "runtime.yaml") {
		t.Fatalf("runtime without runtime.yaml = %v", err)
	}
	write(filepath.Join(runtimeDir, "runtime.yaml"))
	if err := ValidateEntry(e); err != nil {
		t.Fatalf("runtime with both files = %v", err)
	}
}

func jsonString(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}
