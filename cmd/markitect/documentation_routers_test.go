package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
)

func TestCheckDocumentationRoutersUsesSelectedSnapshot(t *testing.T) {
	repo := newCLIRepo(t, false)
	projectPath := filepath.Join(repo.root, "markitect.yaml")
	data, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	project, err := format.Parse("markitect.yaml", data)
	if err != nil {
		t.Fatal(err)
	}
	project.Spec.Documentation = &core.Documentation{Roots: []string{"docs"}}
	data, err = format.Encode(project)
	if err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo.root, "markitect.yaml", data)
	for name, text := range map[string]string{
		"docs/README.md":                "[General](general/README.md)\n",
		"docs/general/README.md":        "[Rules](rules/) [Skills](skills/)\n",
		"docs/general/rules/README.md":  "[Policy](policy.md) [Legacy](legacy.md)\n",
		"docs/general/skills/README.md": "[Entry](entry.md)\n",
	} {
		writeRepoFile(t, repo.root, name, []byte(text))
	}
	git(t, repo.root, "add", "-A")
	git(t, repo.root, "commit", "-m", "configure documentation routers")
	fixed := git(t, repo.root, "rev-parse", "HEAD")
	code, output, stderr := invoke("check", "--repo", repo.root, "--revision", fixed)
	if code != 0 {
		t.Fatalf("fixed router check exit=%d stderr=%s output=%s", code, stderr, output)
	}
	writeRepoFile(t, repo.root, "docs/general/rules/new.md", []byte("# New\n"))
	code, output, stderr = invoke("check", "--repo", repo.root)
	if code != 1 || !strings.Contains(output, "documentation.router.unlisted-file") {
		t.Fatalf("dirty router check exit=%d stderr=%s output=%s", code, stderr, output)
	}
	code, output, stderr = invoke("check", "--repo", repo.root, "--revision", fixed)
	if code != 0 {
		t.Fatalf("fixed snapshot changed with dirty file: exit=%d stderr=%s output=%s", code, stderr, output)
	}
	git(t, repo.root, "add", "docs/general/rules/new.md")
	git(t, repo.root, "commit", "-m", "add unlisted documentation")
	broken := git(t, repo.root, "rev-parse", "HEAD")
	code, output, stderr = invoke("verify", "--repo", repo.root, "--revision", broken)
	if code != 1 || !strings.Contains(output, "documentation.router.unlisted-file") {
		t.Fatalf("verify accepted broken router: exit=%d stderr=%s output=%s", code, stderr, output)
	}
}
