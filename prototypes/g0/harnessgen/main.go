//go:build g0proto

// Command harnessgen renders the existing Markitect harness context of every
// Manager for one mandate, so the G0 prototype gives its Managers exactly the
// model context arm M gives. Throwaway prototype code; the build tag keeps it
// out of the product build and the import gate.
//
//	go run -tags g0proto ./prototypes/g0/harnessgen -repo R -mandate implement -commit C -tree T -out DIR \
//	  -dependency .markitect/dependencies/core/model.yaml -dependency .markitect/dependencies/software/model.yaml
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type deps []string

func (d *deps) String() string     { return strings.Join(*d, ",") }
func (d *deps) Set(v string) error { *d = append(*d, v); return nil }

func main() {
	repo := flag.String("repo", "", "project repository")
	manifest := flag.String("model", ".markitect/model.yaml", "local manifest")
	mandate := flag.String("mandate", "implement", "assessment, implement or verify")
	commit := flag.String("commit", "", "bound input commit")
	tree := flag.String("tree", "", "bound input tree")
	findingsPath := flag.String("findings", "", "optional JSON list of initial Findings")
	out := flag.String("out", "", "output directory")
	var dependencies deps
	flag.Var(&dependencies, "dependency", "dependency manifest (repeat)")
	flag.Parse()
	if err := run(*repo, *manifest, dependencies, *mandate, *commit, *tree, *findingsPath, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func selection(root, literal string) (project.ModelSelection, error) {
	p, err := snapshot.NewPath(literal)
	return project.ModelSelection{Boundary: ".", ManifestPath: p}, err
}

func run(repo, manifest string, dependencies []string, mandate, commit, tree, findingsPath, out string) error {
	root, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	if err := os.Chdir(root); err != nil {
		return err
	}
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		return err
	}
	preparation := project.NewPreparationService(project.NewModelService(repository, repository))
	sel := project.InputSelection{Repository: root}
	if sel.Local, err = selection(root, manifest); err != nil {
		return err
	}
	for _, d := range dependencies {
		s, err := selection(root, d)
		if err != nil {
			return err
		}
		sel.Dependencies = append(sel.Dependencies, s)
	}
	captured, err := preparation.Capture(context.Background(), project.CaptureRequest{Selection: sel}, model.Limits{})
	if err != nil {
		return err
	}
	attempts := captured.Attempts()
	if len(attempts) == 0 {
		return fmt.Errorf("no complete acquisition")
	}
	prepared, err := project.Prepare(attempts[len(attempts)-1])
	if err != nil {
		return err
	}
	var findings []harness.Finding
	if findingsPath != "" {
		data, err := os.ReadFile(findingsPath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &findings); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	type entry struct {
		Package  string   `json:"package"`
		Parent   *string  `json:"parent"`
		Children []string `json:"children"`
		Harness  string   `json:"harness"`
	}
	var managers []entry
	slug := regexp.MustCompile(`[^a-z0-9]+`)
	for _, m := range prepared.Topology().Managers() {
		var mf []harness.Finding
		for _, f := range findings {
			if f.Manager == m.Package {
				mf = append(mf, f)
			}
		}
		doc, err := harness.Build(prepared, m, mandate, mf, harness.Origin{Commit: commit, Tree: tree})
		if err != nil {
			return fmt.Errorf("%s: %w", m.Package, err)
		}
		name := strings.Trim(slug.ReplaceAllString(strings.ToLower(m.Package), "-"), "-")
		if name == "" {
			name = "root"
		}
		file := filepath.Join(out, name+".developer.md")
		if err := os.WriteFile(file, []byte(doc.Developer), 0o644); err != nil {
			return err
		}
		e := entry{Package: m.Package, Children: m.Children, Harness: file}
		if m.Package != "." {
			p := m.Parent
			e.Parent = &p
		}
		managers = append(managers, e)
	}
	data, err := json.MarshalIndent(managers, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "managers.json"), data, 0o644)
}
