package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func initWriterRepo(t *testing.T, root string) {
	t.Helper()
	runWriterGit(t, root, "init", "-b", "feature/test-migration")
	runWriterGit(t, root, "config", "user.name", "Markitect Test")
	runWriterGit(t, root, "config", "user.email", "markitect-test@example.invalid")
}

func runWriterGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	gitArgs := append([]string{"-c", "safe.directory=" + filepath.ToSlash(root), "-C", root}, args...)
	command := exec.Command("git", gitArgs...)
	command.Env = source.CleanGitEnv()
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func migrationProject(t *testing.T, target string) map[string][]byte {
	t.Helper()
	var targets []string
	if target != "" {
		targets = []string{target}
	}
	resource := core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Project",
		Metadata:   core.Metadata{Name: "project"},
		Spec: core.Spec{Profile: "generic", Targets: targets, Areas: []core.Area{
			{Name: "cockpit-general", Path: "docs/general"},
		}},
	}
	return map[string][]byte{"markitect.yaml": encodeResource(t, resource)}
}

func migrationFiles(t *testing.T) map[string][]byte {
	t.Helper()
	project := migrationProject(t, "codex")
	skill := core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Skill",
		Metadata:   core.Metadata{Name: "entry", Namespace: "cockpit-general"},
		Spec:       core.Spec{Text: "Skill text."},
	}
	project["docs/general/skills/entry.yaml"] = encodeResource(t, skill)
	return project
}

func TestWriteMigrationRefusesExistingProjectConfiguration(t *testing.T) {
	root := tempRoot(t)
	initWriterRepo(t, root)
	writeFixture(t, root, map[string][]byte{
		"docs/general/rules/legacy.md": []byte("Existing source.\n"),
		"markitect.yaml":               migrationProject(t, "")["markitect.yaml"],
	})
	snapshot, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteMigration(root, snapshot, migrationFiles(t)); err == nil || !strings.Contains(err.Error(), "markitect.yaml already exists") {
		t.Fatalf("WriteMigration error = %v, want refusal for existing project configuration", err)
	}
}

func TestWriteMigrationRefusesUnmanagedOutputOutsideLegacyInventory(t *testing.T) {
	root := tempRoot(t)
	initWriterRepo(t, root)
	writeFixture(t, root, map[string][]byte{
		"docs/general/rules/legacy.md":  []byte("Existing source.\n"),
		".agents/skills/entry/SKILL.md": []byte("Hand-authored provider file.\n"),
	})
	snapshot, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	canonical := migrationFiles(t)
	if _, err := WriteMigration(root, snapshot, canonical); err == nil || !strings.Contains(err.Error(), "unmanaged output outside the inventoried legacy definitions") {
		t.Fatalf("WriteMigration error = %v, want unmanaged-output refusal", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".agents/skills/entry/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Hand-authored provider file.\n" {
		t.Fatalf("unmanaged provider file changed: %q", data)
	}
}

func TestWriteMigrationRefusesSourceAdditionAfterCapture(t *testing.T) {
	root := tempRoot(t)
	initWriterRepo(t, root)
	writeFixture(t, root, map[string][]byte{"docs/general/notes.md": []byte("Initial source.\n")})
	snapshot, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, map[string][]byte{"docs/general/added.md": []byte("Added after capture.\n")})
	if _, err := WriteMigration(root, snapshot, migrationProject(t, "")); err == nil || !strings.Contains(err.Error(), "source changed since migration inventory") {
		t.Fatalf("WriteMigration error = %v, want source-inventory race refusal", err)
	}
	if _, err := os.Stat(filepath.Join(root, "markitect.yaml")); !os.IsNotExist(err) {
		t.Fatalf("migration wrote files after the source inventory changed: stat error = %v", err)
	}
}

func TestWriteMigrationRefusesExistingWriterLock(t *testing.T) {
	root := tempRoot(t)
	initWriterRepo(t, root)
	writeFixture(t, root, map[string][]byte{"docs/general/notes.md": []byte("Initial source.\n")})
	snapshot, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, ".artifacts/markitect/write.lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte("held\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteMigration(root, snapshot, migrationProject(t, "")); err == nil || !strings.Contains(err.Error(), "renderer lock is already present") {
		t.Fatalf("WriteMigration error = %v, want existing-lock refusal", err)
	}
}

func TestWriteSchemasRefusesUnmanagedTargetAndParentTraversal(t *testing.T) {
	t.Run("unmanaged target", func(t *testing.T) {
		root := tempRoot(t)
		target := filepath.Join(root, "schema/v1alpha1.json")
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		original := []byte("hand-authored schema\n")
		if err := os.WriteFile(target, original, 0644); err != nil {
			t.Fatal(err)
		}
		if err := WriteSchemas(root, map[string][]byte{"schema/v1alpha1.json": []byte("generated\n")}); err == nil || !strings.Contains(err.Error(), "unmanaged schema output") {
			t.Fatalf("WriteSchemas error = %v, want unmanaged-target refusal", err)
		}
		actual, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if string(actual) != string(original) {
			t.Fatal("unmanaged schema target was changed")
		}
	})

	t.Run("parent traversal", func(t *testing.T) {
		root := tempRoot(t)
		if err := WriteSchemas(root, map[string][]byte{"schema/../outside.json": []byte("escaped\n")}); err == nil || !strings.Contains(err.Error(), "unsafe output path") {
			t.Fatalf("WriteSchemas error = %v, want traversal refusal", err)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(root), "outside.json")); !os.IsNotExist(err) {
			t.Fatalf("schema write escaped its root: stat error = %v", err)
		}
	})
}

func TestWriteSchemasRefusesSymlinkPath(t *testing.T) {
	root := tempRoot(t)
	outside := tempRoot(t)
	if err := os.Symlink(outside, filepath.Join(root, "schema")); err != nil {
		t.Skipf("symlink creation is unavailable in this environment: %v", err)
	}
	err := WriteSchemas(root, map[string][]byte{"schema/v1alpha1.json": []byte("generated\n")})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("WriteSchemas error = %v, want symlink-path refusal", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "v1alpha1.json")); !os.IsNotExist(err) {
		t.Fatalf("schema write followed a symlink: stat error = %v", err)
	}
}
