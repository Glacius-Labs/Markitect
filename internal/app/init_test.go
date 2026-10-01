package app

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/source"
)

func TestInitPreviewIsDeterministicAndReadOnly(t *testing.T) {
	root := initTempRoot(t)
	options := InitOptions{Name: "sample-project", Namespace: "general", Path: "docs/general"}

	first, err := Init(root, options, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Init(root, options, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated preview differs:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if first.Kind != "init" || first.Project != options.Name || first.Area.Name != options.Namespace || first.Area.Path != options.Path || first.Applied || len(first.Files) != 2 {
		t.Fatalf("unexpected preview: %+v", first)
	}
	for _, file := range first.Files {
		hash := sha256.Sum256([]byte(file.Text))
		if file.SHA256 != hex.EncodeToString(hash[:]) {
			t.Errorf("wrong preview digest for %s", file.Path)
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(file.Path))); !os.IsNotExist(err) {
			t.Fatalf("preview created %s: %v", file.Path, err)
		}
	}
}

func TestInitWritesValidatedProjectOnUnbornFeatureBranch(t *testing.T) {
	root := initTempRoot(t)
	result, err := Init(root, InitOptions{Name: "sample-project", Namespace: "general", Path: "docs/general"}, true)
	if err != nil {
		t.Fatalf("Init on unborn feature branch: %v", err)
	}
	if !result.Applied || len(result.Written) != 2 || result.Recovery != "" || !reflect.DeepEqual(result.CreatedDirectories, []string{"docs", "docs/general"}) {
		t.Fatalf("unexpected write result: %+v", result)
	}
	snapshot, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	project, err := Parse(snapshot)
	if err != nil {
		t.Fatalf("parse initialized Project: %v", err)
	}
	if len(project.Diagnostics) != 0 || len(CheckOutputs(project)) != 0 {
		t.Fatalf("initialized Project did not pass structural/output checks: %#v %#v", project.Diagnostics, CheckOutputs(project))
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "general", "README.md")); err != nil {
		t.Fatalf("area README missing: %v", err)
	}
}

func TestInitAreaCreationRejectsDirectoryAppearingAfterPreflight(t *testing.T) {
	root := initTempRoot(t)
	options := InitOptions{Name: "sample-project", Namespace: "general", Path: "docs/general"}
	state, err := buildInitPlan(root, options)
	if err != nil {
		t.Fatal(err)
	}
	state.branch, err = writeBranchName(root)
	if err != nil {
		t.Fatal(err)
	}
	areaPath := filepath.Join(root, "docs", "general")
	if err := os.MkdirAll(areaPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := createInitArea(state); err == nil || !strings.Contains(err.Error(), "exclusively") {
		t.Fatalf("createInitArea error = %v, want exclusive-create refusal", err)
	}
	entries, err := os.ReadDir(areaPath)
	if err != nil || len(entries) != 0 {
		t.Fatalf("raced empty area was changed: entries=%v err=%v", entries, err)
	}
	if len(state.plan.Written) != 0 || len(state.plan.CreatedDirectories) != 0 {
		t.Fatalf("failed exclusive create claimed ownership: %+v", state.plan)
	}
}

func TestInitRejectsExistingTargetsAndCaseAliasesWithoutWrites(t *testing.T) {
	t.Run("existing area directory", func(t *testing.T) {
		root := initTempRoot(t)
		if err := os.MkdirAll(filepath.Join(root, "docs", "general"), 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := Init(root, InitOptions{Name: "sample-project", Namespace: "general", Path: "docs/general"}, true); err == nil || !strings.Contains(err.Error(), "area directory already exists") {
			t.Fatalf("Init error = %v, want existing-area refusal", err)
		}
		if _, err := os.Stat(filepath.Join(root, "markitect.yaml")); !os.IsNotExist(err) {
			t.Fatalf("failed init created Project: %v", err)
		}
	})

	t.Run("case alias ancestor", func(t *testing.T) {
		root := initTempRoot(t)
		if err := os.MkdirAll(filepath.Join(root, "docs", "General"), 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := Init(root, InitOptions{Name: "sample-project", Namespace: "general", Path: "docs/general"}, false); err == nil || !strings.Contains(err.Error(), "case-insensitive init path collision") {
			t.Fatalf("Init error = %v, want case-alias refusal", err)
		}
		if _, err := os.Stat(filepath.Join(root, "markitect.yaml")); !os.IsNotExist(err) {
			t.Fatalf("failed preview created Project: %v", err)
		}
	})

	t.Run("existing Project file", func(t *testing.T) {
		root := initTempRoot(t)
		writeFixture(t, root, map[string][]byte{"markitect.yaml": []byte("existing\n")})
		if _, err := Init(root, InitOptions{Name: "sample-project", Namespace: "general", Path: "docs/general"}, false); err == nil {
			t.Fatal("Init accepted an existing Project file")
		}
		data, err := os.ReadFile(filepath.Join(root, "markitect.yaml"))
		if err != nil || string(data) != "existing\n" {
			t.Fatalf("existing Project changed: %q, %v", data, err)
		}
	})
}

func TestInitRejectsInvalidOptionsAndExcludedPaths(t *testing.T) {
	for _, options := range []InitOptions{
		{Name: "Bad Name", Namespace: "general", Path: "docs/general"},
		{Name: "sample-project", Namespace: "Bad Namespace", Path: "docs/general"},
		{Name: "sample-project", Namespace: "general", Path: "docs/../outside"},
		{Name: "sample-project", Namespace: "general", Path: ".artifacts/area"},
	} {
		t.Run(strings.ReplaceAll(options.Name+"-"+options.Namespace+"-"+options.Path, "/", "_"), func(t *testing.T) {
			root := initTempRoot(t)
			if _, err := Init(root, options, false); err == nil {
				t.Fatalf("Init accepted invalid options: %+v", options)
			}
			if _, err := os.Stat(filepath.Join(root, "markitect.yaml")); !os.IsNotExist(err) {
				t.Fatalf("invalid preview created Project: %v", err)
			}
		})
	}
}

func TestInitRejectsGitOwnedDeletedOrStagedTargets(t *testing.T) {
	t.Run("deleted committed target", func(t *testing.T) {
		root := initTempRoot(t)
		writeFixture(t, root, map[string][]byte{"README.md": []byte("baseline\n"), "docs/general/old.md": []byte("owned\n")})
		runWriterGit(t, root, "add", "README.md", "docs/general/old.md")
		runWriterGit(t, root, "commit", "-m", "baseline")
		if err := os.RemoveAll(filepath.Join(root, "docs", "general")); err != nil {
			t.Fatal(err)
		}
		if _, err := Init(root, InitOptions{Name: "sample-project", Namespace: "general", Path: "docs/general"}, false); err == nil || !strings.Contains(err.Error(), "Git-owned target path") {
			t.Fatalf("Init error = %v, want deleted committed target refusal", err)
		}
	})

	t.Run("staged target", func(t *testing.T) {
		root := initProjectTestRepo(t, "feature/staged-target")
		writeFixture(t, root, map[string][]byte{"docs/general/staged.md": []byte("staged\n")})
		runWriterGit(t, root, "add", "docs/general/staged.md")
		if err := os.RemoveAll(filepath.Join(root, "docs", "general")); err != nil {
			t.Fatal(err)
		}
		if _, err := Init(root, InitOptions{Name: "sample-project", Namespace: "general", Path: "docs/general"}, false); err == nil || !strings.Contains(err.Error(), "staged Git path") {
			t.Fatalf("Init error = %v, want staged target refusal", err)
		}
	})
}

func TestInitRejectsGitIgnoredArea(t *testing.T) {
	root := initProjectTestRepo(t, "feature/ignored-area")
	writeFixture(t, root, map[string][]byte{".gitignore": []byte("docs/private/\n")})
	if _, err := Init(root, InitOptions{Name: "sample-project", Namespace: "private", Path: "docs/private"}, false); err == nil || !strings.Contains(err.Error(), "ignored by Git") {
		t.Fatalf("Init error = %v, want ignored-target refusal", err)
	}
}

func TestInitRequiresWritableBranchAndSharedWriterLock(t *testing.T) {
	t.Run("protected branch", func(t *testing.T) {
		root := initProjectTestRepo(t, "feature/protected-init")
		runWriterGit(t, root, "branch", "main")
		runWriterGit(t, root, "checkout", "main")
		if _, err := Init(root, InitOptions{Name: "sample-project", Namespace: "general", Path: "docs/general"}, true); err == nil || !strings.Contains(err.Error(), "non-protected") {
			t.Fatalf("Init error = %v, want protected-branch refusal", err)
		}
	})

	t.Run("existing writer lock", func(t *testing.T) {
		root := initProjectTestRepo(t, "feature/locked-init")
		lock := filepath.Join(root, ".artifacts", "markitect", "write.lock")
		if err := os.MkdirAll(filepath.Dir(lock), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(lock, []byte("held\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Init(root, InitOptions{Name: "sample-project", Namespace: "general", Path: "docs/general"}, true); err == nil || !strings.Contains(err.Error(), "renderer lock is already present") {
			t.Fatalf("Init error = %v, want shared-lock refusal", err)
		}
	})
}

func initProjectTestRepo(t *testing.T, branch string) string {
	t.Helper()
	root := initTempRoot(t)
	writeFixture(t, root, map[string][]byte{"README.md": []byte("baseline\n")})
	runWriterGit(t, root, "add", "-A")
	runWriterGit(t, root, "commit", "-m", "baseline")
	if branch != "feature/test-writes" {
		runWriterGit(t, root, "checkout", "-b", branch)
	}
	return root
}

func initTempRoot(t *testing.T) string {
	t.Helper()
	if err := os.MkdirAll(".cache", 0755); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(".cache", "markitect-init-test-")
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := removeTestRoot(root); err != nil {
			t.Errorf("remove init test root: %v", err)
		}
	})
	initWriterRepo(t, root)
	return root
}
