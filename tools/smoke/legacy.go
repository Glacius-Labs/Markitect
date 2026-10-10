package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// legacySteps checks the published Project/Domain line through the
// markitect-legacy tool built from the packaged source. ARCH-09 removes that
// line; delete this file and its call in source with it.
func (s *smoke) legacySteps() []struct {
	name  string
	check func() error
} {
	legacy := func(args ...string) (string, error) { return s.run(s.repo, nil, s.tool("markitect-legacy"), args...) }
	example := func(name string) string { return filepath.Join(s.repo, "examples", name) }
	passed := regexp.MustCompile(`(?m)^status: passed$`)
	checkPasses := func(repo string) error {
		out, err := legacy("check", "--repo", repo)
		if err != nil {
			return err
		}
		if !passed.MatchString(out) {
			return fmt.Errorf("check of %s did not pass:\n%s", filepath.Base(repo), tail(out))
		}
		return nil
	}
	return []struct {
		name  string
		check func() error
	}{
		{"legacy: authoring Skill is embedded", func() error {
			out, err := legacy("authoring")
			if err != nil {
				return err
			}
			return requireContains(out, "core/Skill/authoring")
		}},
		{"legacy: minimal example passes check", func() error { return checkPasses(example("minimal")) }},
		{"legacy: canonical examples check, compile and format", func() error {
			for _, c := range []struct{ example, namespace, skill string }{
				{"canonical-engineering", "engineering", "architecture-review"},
				{"software-architecture", "engineering", "implement-order"},
				{"delivery-target-equality", "engineering", "deployment-review"},
			} {
				repo := example(c.example)
				if err := checkPasses(repo); err != nil {
					return err
				}
				for _, args := range [][]string{
					{"context", "--repo", repo, "--namespace", c.namespace, "--kind", "Skill", "--name", c.skill},
					{"model", "--repo", repo},
					{"format", "--repo", repo},
				} {
					if _, err := legacy(args...); err != nil {
						return err
					}
				}
			}
			return nil
		}},
		{"legacy: reconcile renders and verifies a canonical fixture", s.legacyReconcile},
	}
}

// legacyReconcile runs observe, plan, apply and verify for the rendering
// adapter and the external .NET adapter on a committed copy of the canonical
// engineering example.
func (s *smoke) legacyReconcile() error {
	repo, err := s.gitRepo("canonical-reconcile", "feature/canonical-smoke")
	if err != nil {
		return err
	}
	if err := os.CopyFS(repo, os.DirFS(filepath.Join(s.repo, "examples", "canonical-engineering"))); err != nil {
		return err
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "--quiet", "-m", "Create isolated canonical engineering fixture"}} {
		if _, err := s.run(repo, nil, "git", args...); err != nil {
			return err
		}
	}
	legacy := func(args ...string) (string, error) { return s.run(repo, nil, s.tool("markitect-legacy"), args...) }
	evidence := filepath.Join(repo, ".artifacts", "markitect", "reconcile")
	if err := os.MkdirAll(evidence, 0o755); err != nil {
		return err
	}
	for _, adapter := range []struct {
		name  string
		apply bool
	}{{"markitect-render", true}, {"dotnet-dependencies", false}} {
		if _, err := legacy("reconcile", "--repo", repo, "--action", "observe", "--adapter", adapter.name); err != nil {
			return err
		}
		plan, err := legacy("reconcile", "--repo", repo, "--action", "plan", "--adapter", adapter.name)
		if err != nil {
			return err
		}
		planPath := filepath.Join(evidence, adapter.name+"-plan.yaml")
		if err := os.WriteFile(planPath, []byte(plan), 0o644); err != nil {
			return err
		}
		if adapter.apply {
			if _, err := legacy("reconcile", "--repo", repo, "--action", "apply", "--adapter", adapter.name, "--plan", planPath, "--write"); err != nil {
				return err
			}
		}
		if _, err := legacy("reconcile", "--repo", repo, "--action", "verify", "--adapter", adapter.name, "--plan", planPath); err != nil {
			return err
		}
	}
	return nil
}
